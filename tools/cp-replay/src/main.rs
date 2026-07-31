//! `cp-replay` — deterministic replay harness for CasperProver ml-attest-v0
//! attestations. See `lib.rs` for the honesty invariant and design notes.
//!
//! Exit codes (kept stable — auditors will script this):
//!
//!   0  every requested check passed
//!   1  at least one check failed (mismatch / tamper / unknown scheme)
//!   2  I/O or parse error (missing file, bad JSON)

use anyhow::{Context, Result};
use clap::{Parser, Subcommand};
use cp_replay::{
    all_artefacts_ok, attest, load_attestation, load_attestation_strict, parse_digest_hex,
    replay_artefacts, verify_attestation, AttestInput, SCHEME_ML_ATTEST_V0,
};
use std::path::PathBuf;
use std::process::ExitCode;

const DISCLOSURE_BANNER: &str =
    "cp-replay verifies an ATTESTATION of (model_id, weights_digest, inputs_digest, outputs_digest). \
It is NOT a cryptographic proof that the model was executed on the inputs. \
See docs/ZKML_HONEST_VERDICT.md.";

#[derive(Parser, Debug)]
#[command(
    name = "cp-replay",
    version,
    about = "Deterministic replay harness for CasperProver ml-attest-v0 attestations.",
    long_about = "Auditor tool. Given an Attestation JSON envelope emitted by the CasperProver engine at POST /v1/ml/attest, re-derive its commit_hex from raw inputs and confirm the envelope has not been tampered with. Optionally, re-hash the physical weights / inputs / outputs artefacts and confirm they match the envelope digests.\n\nThis is not a ZK-ML proof. See docs/ZKML_HONEST_VERDICT.md for the durable decision record."
)]
struct Cli {
    #[command(subcommand)]
    command: Command,

    /// Emit machine-readable JSON on stdout instead of the human report.
    #[arg(long, global = true)]
    json: bool,

    /// Reject envelopes with unknown top-level JSON fields or missing
    /// required fields, instead of silently defaulting them.
    ///
    /// Trade-off: paranoid schema check for auditors who need to detect a
    /// renamed / misspelled field (which would otherwise deserialise to an
    /// empty digest and fail later as an opaque "commit mismatch"). Off by
    /// default so newer emitters that add backwards-compatible fields do
    /// not brick older auditors.
    #[arg(long, global = true)]
    strict: bool,
}

#[derive(Subcommand, Debug)]
enum Command {
    /// Verify an Attestation JSON envelope: recompute commit_hex from the
    /// hex digests inside the envelope and confirm it matches.
    Verify(VerifyArgs),

    /// Verify the envelope AND re-hash the raw weights/inputs/outputs
    /// artefacts on disk, confirming they match the digests the envelope
    /// committed to. This is a stronger physical-integrity signal.
    ReplayArtefacts(ReplayArgs),

    /// Compute a commit_hex from raw inputs without producing a full
    /// envelope. Useful for cross-checking against a Go-emitted commit.
    CommitOnly(CommitOnlyArgs),
}

#[derive(clap::Args, Debug)]
struct VerifyArgs {
    /// Path to the attestation.json envelope.
    #[arg(long)]
    attestation: PathBuf,
}

#[derive(clap::Args, Debug)]
struct ReplayArgs {
    /// Path to the attestation.json envelope.
    #[arg(long)]
    attestation: PathBuf,
    /// Path to the raw weights blob referenced by weights_digest_hex.
    #[arg(long)]
    weights: PathBuf,
    /// Path to the raw inputs tensor referenced by inputs_digest_hex.
    #[arg(long)]
    inputs: PathBuf,
    /// Path to the raw outputs tensor referenced by outputs_digest_hex.
    #[arg(long)]
    outputs: PathBuf,
}

#[derive(clap::Args, Debug)]
struct CommitOnlyArgs {
    /// Opaque model identifier (e.g. "mnist-mlp-8x8-v0").
    #[arg(long)]
    model_id: String,
    /// SHA-256 of the weights, hex.
    #[arg(long)]
    weights_digest_hex: String,
    /// SHA-256 of the inputs, hex.
    #[arg(long)]
    inputs_digest_hex: String,
    /// SHA-256 of the outputs, hex.
    #[arg(long)]
    outputs_digest_hex: String,
}

fn main() -> ExitCode {
    // We manage exit codes by hand: `?` bubbles anyhow errors up here, and
    // we translate the two distinct failure classes (mismatch vs I/O) into
    // distinct codes so scripts can tell them apart.
    match run() {
        Ok(0) => ExitCode::from(0),
        Ok(code) => ExitCode::from(code),
        Err(err) => {
            eprintln!("cp-replay: error: {err:#}");
            ExitCode::from(2)
        }
    }
}

fn run() -> Result<u8> {
    let cli = Cli::parse();
    match cli.command {
        Command::Verify(args) => cmd_verify(args, cli.json, cli.strict),
        Command::ReplayArtefacts(args) => cmd_replay_artefacts(args, cli.json, cli.strict),
        Command::CommitOnly(args) => cmd_commit_only(args, cli.json),
    }
}

/// Load the envelope, honouring `--strict` when the caller asked for it.
/// Kept as a helper so both `verify` and `replay-artefacts` behave the
/// same on strict-mode failure: exit code 1 (mismatch class), not 2 (I/O).
fn load_envelope(path: &std::path::Path, strict: bool) -> Result<cp_replay::Attestation> {
    if strict {
        load_attestation_strict(path)
    } else {
        load_attestation(path)
    }
}

fn cmd_verify(args: VerifyArgs, json: bool, strict: bool) -> Result<u8> {
    let att = match load_envelope(&args.attestation, strict) {
        Ok(att) => att,
        Err(err) if strict => {
            // Strict-mode schema failure is a *validation* failure, not an
            // I/O failure — auditors script exit code 1 to mean "envelope
            // rejected". Route it through the same channel as a commit
            // mismatch instead of the exit-code-2 error path.
            // Use the alternate ({:#}) formatter so the FULL anyhow
            // context chain is preserved in BOTH JSON and human output.
            // The outer wrap ("strict-check <path>") alone is unhelpful
            // without the root cause naming which field is missing/unknown.
            let full_err = format!("{err:#}");
            if json {
                println!(
                    "{}",
                    serde_json::json!({
                        "ok": false,
                        "strict": true,
                        "error": full_err,
                    })
                );
            } else {
                eprintln!("cp-replay: envelope FAILED ✗ (strict schema)");
                eprintln!("  reason: {full_err}");
                eprintln!();
                eprintln!("note: {DISCLOSURE_BANNER}");
            }
            return Ok(1);
        }
        Err(err) => return Err(err),
    };
    match verify_attestation(&att) {
        Ok(()) => {
            if json {
                println!(
                    "{}",
                    serde_json::json!({
                        "ok": true,
                        "scheme": att.scheme,
                        "model_id": att.model_id,
                        "commit_hex": att.commit_hex,
                        "disclosure": DISCLOSURE_BANNER,
                    })
                );
            } else {
                println!("cp-replay: envelope verified ✓");
                println!("  scheme    : {}", att.scheme);
                println!("  model_id  : {}", att.model_id);
                println!("  commit    : {}", att.commit_hex);
                println!();
                println!("note: {DISCLOSURE_BANNER}");
            }
            Ok(0)
        }
        Err(err) => {
            if json {
                println!(
                    "{}",
                    serde_json::json!({
                        "ok": false,
                        "error": err.to_string(),
                        "scheme": att.scheme,
                        "commit_hex": att.commit_hex,
                    })
                );
            } else {
                eprintln!("cp-replay: envelope FAILED ✗");
                eprintln!("  reason: {err:#}");
                eprintln!();
                eprintln!("note: {DISCLOSURE_BANNER}");
            }
            Ok(1)
        }
    }
}

fn cmd_replay_artefacts(args: ReplayArgs, json: bool, strict: bool) -> Result<u8> {
    let att = match load_envelope(&args.attestation, strict) {
        Ok(att) => att,
        Err(err) if strict => {
            // Use the alternate ({:#}) formatter so the FULL anyhow
            // context chain is preserved in BOTH JSON and human output.
            // The outer wrap ("strict-check <path>") alone is unhelpful
            // without the root cause naming which field is missing/unknown.
            let full_err = format!("{err:#}");
            if json {
                println!(
                    "{}",
                    serde_json::json!({
                        "ok": false,
                        "strict": true,
                        "error": full_err,
                    })
                );
            } else {
                eprintln!("cp-replay: envelope FAILED ✗ (strict schema)");
                eprintln!("  reason: {full_err}");
                eprintln!();
                eprintln!("note: {DISCLOSURE_BANNER}");
            }
            return Ok(1);
        }
        Err(err) => return Err(err),
    };
    // Step 1: verify the envelope first. If commit is bogus we do not want
    // to give the impression that "matching digests" alone is enough.
    let envelope_ok = verify_attestation(&att);
    let checks = replay_artefacts(&att, &args.weights, &args.inputs, &args.outputs)
        .context("replaying artefact hashes")?;
    let artefacts_ok = all_artefacts_ok(&checks);
    let overall_ok = envelope_ok.is_ok() && artefacts_ok;

    if json {
        let checks_json: Vec<_> = checks
            .iter()
            .map(|c| {
                serde_json::json!({
                    "label": c.label,
                    "file_digest_hex": c.file_digest_hex,
                    "envelope_digest_hex": c.envelope_digest_hex,
                    "matches": c.matches,
                })
            })
            .collect();
        let envelope_error = envelope_ok.as_ref().err().map(|e| e.to_string());
        println!(
            "{}",
            serde_json::json!({
                "ok": overall_ok,
                "envelope_ok": envelope_ok.is_ok(),
                "envelope_error": envelope_error,
                "artefacts_ok": artefacts_ok,
                "checks": checks_json,
                "disclosure": DISCLOSURE_BANNER,
            })
        );
    } else {
        println!(
            "cp-replay: envelope    : {}",
            if envelope_ok.is_ok() { "✓" } else { "✗" }
        );
        if let Err(err) = &envelope_ok {
            println!("           reason: {err:#}");
        }
        println!(
            "cp-replay: artefacts   : {}",
            if artefacts_ok {
                "✓ all match"
            } else {
                "✗ mismatch below"
            }
        );
        for c in &checks {
            println!(
                "  {:8}: file={} envelope={} {}",
                c.label,
                c.file_digest_hex,
                c.envelope_digest_hex,
                if c.matches { "✓" } else { "✗" }
            );
        }
        println!();
        println!("note: {DISCLOSURE_BANNER}");
    }

    Ok(if overall_ok { 0 } else { 1 })
}

fn cmd_commit_only(args: CommitOnlyArgs, json: bool) -> Result<u8> {
    let input = AttestInput {
        model_id: args.model_id,
        weights_digest: parse_digest_hex("weights_digest_hex", &args.weights_digest_hex)?,
        inputs_digest: parse_digest_hex("inputs_digest_hex", &args.inputs_digest_hex)?,
        outputs_digest: parse_digest_hex("outputs_digest_hex", &args.outputs_digest_hex)?,
    };
    let att = attest(&input)?;
    if json {
        println!("{}", serde_json::to_string(&att)?);
    } else {
        println!("cp-replay: scheme     : {SCHEME_ML_ATTEST_V0}");
        println!("           model_id   : {}", att.model_id);
        println!("           commit_hex : {}", att.commit_hex);
        println!();
        println!("note: {DISCLOSURE_BANNER}");
    }
    Ok(0)
}
