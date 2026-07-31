//! Deterministic replay harness for CasperProver ml-attest-v0 attestations.
//!
//! This crate is a host-side auditor tool. It re-implements the exact
//! commit function that lives in `engine/internal/mlattest/harness.go`
//! (see `HashMLAttestor.commit()`) so a third party can, offline and
//! without pulling the Go engine, take a published `Attestation` JSON
//! envelope and re-derive `commit_hex` from the raw digest triple.
//!
//! Honesty invariant carried from `docs/ZKML_HONEST_VERDICT.md`:
//!
//!   This crate verifies an attestation of `(model_id, weights_digest,
//!   inputs_digest, outputs_digest)`. It does NOT prove that the named
//!   model was actually executed on the named inputs. The `replay-artefacts`
//!   subcommand adds an *artefact-level* check (SHA-256 the raw weights /
//!   inputs / outputs files and confirm they match the envelope digests)
//!   which is a meaningful integrity signal but still not a cryptographic
//!   proof of inference.
//!
//! The reserved scheme label `zkml-fixed-v0` is deliberately rejected here
//! for exactly the same reason it is rejected in the Go verifier: a future
//! real ZK-ML implementation must land as a new named scheme with its own
//! verifier, not by relabelling this hash chain.

use anyhow::{anyhow, bail, Context, Result};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs;
use std::path::Path;

/// The only scheme this crate is willing to verify.
///
/// Kept as a `pub const` (not an enum) so a caller who deserialises an
/// unknown scheme still lands in `verify_attestation()`'s reject path with
/// a legible error message, instead of failing at JSON parse time.
pub const SCHEME_ML_ATTEST_V0: &str = "ml-attest-v0";

/// Reserved scheme label. If we ever see it in the wild before the four
/// gates in `docs/ZKML_HONEST_VERDICT.md` are met, we refuse to verify —
/// same policy as the Go `HashMLAttestor.Verify()`.
pub const SCHEME_ZKML_FIXED_V0_RESERVED: &str = "zkml-fixed-v0";

/// SHA-256 output length in bytes.
pub const DIGEST_LEN: usize = 32;

/// On-wire envelope. Matches `mlattest.Attestation` in the Go engine
/// field-for-field. Extra JSON fields are ignored so future backwards-
/// compatible additions on the emitter side do not brick auditors.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct Attestation {
    pub scheme: String,
    pub model_id: String,
    #[serde(rename = "weights_digest_hex")]
    pub weights_digest_hex: String,
    #[serde(rename = "inputs_digest_hex")]
    pub inputs_digest_hex: String,
    #[serde(rename = "outputs_digest_hex")]
    pub outputs_digest_hex: String,
    #[serde(rename = "commit_hex")]
    pub commit_hex: String,
    #[serde(default)]
    pub disclosure: String,
}

/// Raw material for `commit()`. All three digest fields must be exactly
/// 32 bytes; the model_id is any non-empty opaque string.
#[derive(Debug, Clone)]
pub struct AttestInput {
    pub model_id: String,
    pub weights_digest: [u8; DIGEST_LEN],
    pub inputs_digest: [u8; DIGEST_LEN],
    pub outputs_digest: [u8; DIGEST_LEN],
}

/// Bit-exact re-implementation of `HashMLAttestor.commit()`.
///
/// The step-by-step must match the Go code exactly, byte-for-byte,
/// or every future attestation will fail to reverify from this crate:
///
///   step_a = SHA256( model_id_utf8_bytes || weights_digest )
///   step_b = SHA256( inputs_digest || outputs_digest )
///   seed   = SHA256( scheme_label_utf8_bytes )
///   commit = SHA256( seed || step_a || step_b )
///
/// See `engine/internal/mlattest/harness.go` for the reference and
/// `commit_matches_reference_vector()` in `tests/` for the pinned KAT.
pub fn commit(input: &AttestInput) -> [u8; DIGEST_LEN] {
    let mut step_a_hasher = Sha256::new();
    step_a_hasher.update(input.model_id.as_bytes());
    step_a_hasher.update(input.weights_digest);
    let step_a = step_a_hasher.finalize();

    let mut step_b_hasher = Sha256::new();
    step_b_hasher.update(input.inputs_digest);
    step_b_hasher.update(input.outputs_digest);
    let step_b = step_b_hasher.finalize();

    let seed = Sha256::digest(SCHEME_ML_ATTEST_V0.as_bytes());

    let mut final_hasher = Sha256::new();
    final_hasher.update(seed);
    final_hasher.update(step_a);
    final_hasher.update(step_b);
    final_hasher.finalize().into()
}

/// Parse a hex string into a 32-byte digest. Rejects wrong length up front
/// with a legible error — length checks are cheap and worth doing before
/// any commit recomputation.
pub fn parse_digest_hex(field: &str, hex_str: &str) -> Result<[u8; DIGEST_LEN]> {
    let raw =
        hex::decode(hex_str.trim()).with_context(|| format!("field {field}: not valid hex"))?;
    if raw.len() != DIGEST_LEN {
        bail!(
            "field {field}: expected {DIGEST_LEN}-byte SHA-256 digest, got {} bytes",
            raw.len()
        );
    }
    let mut out = [0u8; DIGEST_LEN];
    out.copy_from_slice(&raw);
    Ok(out)
}

/// Full envelope verification. Returns `Ok(())` iff:
///
///   - the scheme label is exactly `ml-attest-v0`;
///   - every hex-encoded digest in the envelope decodes to 32 bytes;
///   - the recomputed commit equals `att.commit_hex`.
///
/// Deliberately rejects `zkml-fixed-v0`, matching Go-side policy.
pub fn verify_attestation(att: &Attestation) -> Result<()> {
    if att.scheme == SCHEME_ZKML_FIXED_V0_RESERVED {
        bail!(
            "scheme {} is reserved and MUST NOT verify — see docs/ZKML_HONEST_VERDICT.md",
            SCHEME_ZKML_FIXED_V0_RESERVED
        );
    }
    if att.scheme != SCHEME_ML_ATTEST_V0 {
        bail!(
            "unsupported scheme {:?} (this harness only verifies {:?})",
            att.scheme,
            SCHEME_ML_ATTEST_V0
        );
    }
    if att.model_id.trim().is_empty() {
        bail!("model_id is required");
    }
    let input = AttestInput {
        model_id: att.model_id.clone(),
        weights_digest: parse_digest_hex("weights_digest_hex", &att.weights_digest_hex)?,
        inputs_digest: parse_digest_hex("inputs_digest_hex", &att.inputs_digest_hex)?,
        outputs_digest: parse_digest_hex("outputs_digest_hex", &att.outputs_digest_hex)?,
    };
    let recomputed = commit(&input);
    let recomputed_hex = hex::encode(recomputed);
    if recomputed_hex != att.commit_hex.trim().to_ascii_lowercase() {
        bail!(
            "commit mismatch: envelope={} recomputed={}",
            att.commit_hex,
            recomputed_hex
        );
    }
    Ok(())
}

/// SHA-256 an arbitrary file, streaming — safe for weights blobs that
/// won't fit in RAM.
pub fn sha256_file(path: &Path) -> Result<[u8; DIGEST_LEN]> {
    use std::io::Read;
    let mut file = fs::File::open(path).with_context(|| format!("open {}", path.display()))?;
    let mut hasher = Sha256::new();
    let mut buf = [0u8; 64 * 1024];
    loop {
        let n = file
            .read(&mut buf)
            .with_context(|| format!("read {}", path.display()))?;
        if n == 0 {
            break;
        }
        hasher.update(&buf[..n]);
    }
    Ok(hasher.finalize().into())
}

/// Load a JSON envelope from disk.
pub fn load_attestation(path: &Path) -> Result<Attestation> {
    let raw = fs::read_to_string(path).with_context(|| format!("read {}", path.display()))?;
    let att: Attestation = serde_json::from_str(&raw)
        .with_context(|| format!("parse {} as Attestation JSON", path.display()))?;
    Ok(att)
}

/// The set of top-level JSON keys `--strict` requires to be present, and
/// the exhaustive set of keys it will accept. Anything outside this set
/// is either a missing required field or an unknown field, both of which
/// `strict_check_envelope` rejects with a legible error.
///
/// Kept as a `const` so drift between this list and the `Attestation`
/// struct is caught by `strict_key_lists_match_struct` in the unit tests.
pub const STRICT_REQUIRED_KEYS: &[&str] = &[
    "scheme",
    "model_id",
    "weights_digest_hex",
    "inputs_digest_hex",
    "outputs_digest_hex",
    "commit_hex",
];

/// Keys that are permitted but not required. Used to distinguish a
/// missing-optional (fine) from an unknown-key (rejected) in strict mode.
pub const STRICT_OPTIONAL_KEYS: &[&str] = &["disclosure"];

/// Strict schema check against the raw JSON of an envelope.
///
/// The default `load_attestation` path uses serde's usual behaviour:
/// unknown JSON fields are silently ignored, and any required field that
/// is missing lands with a default (empty string for the digest fields,
/// etc.). That is deliberately permissive so auditors on old versions of
/// the tool do not brick when the emitter adds a backwards-compatible
/// field.
///
/// Strict mode reverses that trade-off for auditors who need paranoid
/// integrity: EVERY required key must be present (explicit `null` still
/// counts as missing), and any unknown top-level key is a hard error. The
/// intent is that a renamed or misspelled field can never silently be
/// treated as missing-with-a-zero-value, which is otherwise indistinguishable
/// from a genuine digest mismatch downstream.
///
/// Returns `Ok(())` on a clean envelope, or an error naming the exact
/// missing / unknown field. This function does NOT re-verify the commit;
/// call `verify_attestation` after it for full validation.
pub fn strict_check_envelope(raw_json: &str) -> Result<()> {
    let value: serde_json::Value = serde_json::from_str(raw_json)
        .context("strict: envelope is not syntactically valid JSON")?;
    let obj = value
        .as_object()
        .ok_or_else(|| anyhow!("strict: envelope root must be a JSON object"))?;

    // Missing required keys — collect them all so the operator sees the
    // whole diff, not one-at-a-time whack-a-mole.
    let mut missing: Vec<&str> = Vec::new();
    for key in STRICT_REQUIRED_KEYS {
        match obj.get(*key) {
            None => missing.push(key),
            Some(v) if v.is_null() => missing.push(key),
            Some(serde_json::Value::String(s)) if s.trim().is_empty() => missing.push(key),
            _ => {}
        }
    }
    if !missing.is_empty() {
        bail!(
            "strict: envelope missing required field(s): {}",
            missing.join(", ")
        );
    }

    // Unknown keys — anything not in the required-or-optional union.
    let mut unknown: Vec<&str> = Vec::new();
    for key in obj.keys() {
        let k = key.as_str();
        let allowed = STRICT_REQUIRED_KEYS.contains(&k) || STRICT_OPTIONAL_KEYS.contains(&k);
        if !allowed {
            unknown.push(k);
        }
    }
    if !unknown.is_empty() {
        bail!(
            "strict: envelope has unknown top-level field(s): {} — refusing to silently ignore",
            unknown.join(", ")
        );
    }

    Ok(())
}

/// Load AND strict-check an envelope in one call. Returns the parsed
/// `Attestation` after the strict check passes.
pub fn load_attestation_strict(path: &Path) -> Result<Attestation> {
    let raw = fs::read_to_string(path).with_context(|| format!("read {}", path.display()))?;
    strict_check_envelope(&raw).with_context(|| format!("strict-check {}", path.display()))?;
    let att: Attestation = serde_json::from_str(&raw)
        .with_context(|| format!("parse {} as Attestation JSON", path.display()))?;
    Ok(att)
}

/// Result row for `replay-artefacts`: for each of the three artefact files
/// we hashed locally, does its digest match the envelope?
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ArtefactCheck {
    pub label: &'static str,
    pub file_digest_hex: String,
    pub envelope_digest_hex: String,
    pub matches: bool,
}

impl ArtefactCheck {
    /// True iff the physical file matches what the envelope committed to.
    pub fn ok(&self) -> bool {
        self.matches
    }
}

/// Given an already-verified attestation and paths to the three raw
/// artefacts, hash each file and compare against the envelope digests.
/// Callers should verify the attestation envelope first — this is a
/// physical integrity check on top of the cryptographic one.
pub fn replay_artefacts(
    att: &Attestation,
    weights_path: &Path,
    inputs_path: &Path,
    outputs_path: &Path,
) -> Result<Vec<ArtefactCheck>> {
    let expected = |field: &str, hex_str: &str| -> Result<String> {
        // Cheap sanity: make sure the envelope's own digest at least
        // decodes as a 32-byte digest, otherwise the "match" below is
        // meaningless.
        parse_digest_hex(field, hex_str)?;
        Ok(hex_str.trim().to_ascii_lowercase())
    };
    let checks = [
        (
            "weights",
            weights_path,
            expected("weights_digest_hex", &att.weights_digest_hex)?,
        ),
        (
            "inputs",
            inputs_path,
            expected("inputs_digest_hex", &att.inputs_digest_hex)?,
        ),
        (
            "outputs",
            outputs_path,
            expected("outputs_digest_hex", &att.outputs_digest_hex)?,
        ),
    ];
    let mut out = Vec::with_capacity(checks.len());
    for (label, path, envelope_hex) in checks {
        let file_digest = sha256_file(path)
            .with_context(|| format!("hash {label} file at {}", path.display()))?;
        let file_hex = hex::encode(file_digest);
        out.push(ArtefactCheck {
            label,
            matches: file_hex == envelope_hex,
            file_digest_hex: file_hex,
            envelope_digest_hex: envelope_hex,
        });
    }
    Ok(out)
}

/// Small helper for callers that want a single "everything matched"
/// signal without walking the vec.
pub fn all_artefacts_ok(checks: &[ArtefactCheck]) -> bool {
    !checks.is_empty() && checks.iter().all(|c| c.ok())
}

/// Build the Attestation envelope from scratch. Useful for cross-checking
/// against a Go-emitted envelope on the same inputs — the two must agree
/// byte-for-byte on `commit_hex`, or the two implementations have drifted.
pub fn attest(input: &AttestInput) -> Result<Attestation> {
    if input.model_id.trim().is_empty() {
        return Err(anyhow!("model_id is required"));
    }
    let commit_bytes = commit(input);
    Ok(Attestation {
        scheme: SCHEME_ML_ATTEST_V0.to_string(),
        model_id: input.model_id.clone(),
        weights_digest_hex: hex::encode(input.weights_digest),
        inputs_digest_hex: hex::encode(input.inputs_digest),
        outputs_digest_hex: hex::encode(input.outputs_digest),
        commit_hex: hex::encode(commit_bytes),
        disclosure: "This is an attestation of (model_id, weights_digest, inputs_digest, outputs_digest) — NOT a cryptographic proof that the named model was executed on the named inputs. See docs/ZKML_HONEST_VERDICT.md.".to_string(),
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    // Pinned known-answer vector. Regenerate from the Go side ONLY if the
    // commit function itself changes intentionally — otherwise this KAT
    // is a canary for silent drift between Go and Rust.
    //
    // Vector: model_id="mnist-mlp-8x8-v0", each digest = SHA-256("weights"),
    // SHA-256("inputs"), SHA-256("outputs") respectively.
    fn kat_input() -> AttestInput {
        AttestInput {
            model_id: "mnist-mlp-8x8-v0".to_string(),
            weights_digest: Sha256::digest(b"weights").into(),
            inputs_digest: Sha256::digest(b"inputs").into(),
            outputs_digest: Sha256::digest(b"outputs").into(),
        }
    }

    #[test]
    fn commit_is_deterministic() {
        let a = commit(&kat_input());
        let b = commit(&kat_input());
        assert_eq!(a, b);
    }

    /// PINNED KAT: exercises byte-for-byte equality with the Go
    /// implementation in engine/internal/mlattest/harness.go.
    ///
    /// The expected commit_hex below was produced by running
    ///   go run ./engine/cmd/cross_check/
    /// on this branch against the same input. If this test ever fails,
    /// the two implementations have drifted and any published Attestation
    /// will fail to reverify from either side — investigate before
    /// touching either constant.
    #[test]
    fn commit_matches_pinned_go_reference_vector() {
        let got = commit(&kat_input());
        let expected = "d384b504fb72a340c972b8ab3ceb15fa388dda59a5548ea411023ff204e0a24a";
        assert_eq!(hex::encode(got), expected);
    }

    #[test]
    fn commit_changes_on_any_field_change() {
        let baseline = commit(&kat_input());
        let mut modified = kat_input();
        modified.model_id = "mnist-mlp-8x8-v1".to_string();
        assert_ne!(commit(&modified), baseline);

        let mut modified = kat_input();
        modified.weights_digest[0] ^= 0x01;
        assert_ne!(commit(&modified), baseline);

        let mut modified = kat_input();
        modified.inputs_digest[31] ^= 0x80;
        assert_ne!(commit(&modified), baseline);

        let mut modified = kat_input();
        modified.outputs_digest[15] ^= 0x0f;
        assert_ne!(commit(&modified), baseline);
    }

    #[test]
    fn attest_then_verify_roundtrip() {
        let input = kat_input();
        let att = attest(&input).unwrap();
        verify_attestation(&att).unwrap();
        assert_eq!(att.scheme, SCHEME_ML_ATTEST_V0);
    }

    #[test]
    fn verify_rejects_reserved_zkml_scheme() {
        let input = kat_input();
        let mut att = attest(&input).unwrap();
        att.scheme = SCHEME_ZKML_FIXED_V0_RESERVED.to_string();
        let err = verify_attestation(&att).unwrap_err().to_string();
        assert!(err.contains("reserved"), "got: {err}");
    }

    #[test]
    fn verify_rejects_tampered_commit() {
        let input = kat_input();
        let mut att = attest(&input).unwrap();
        // Flip one nibble in the commit — this is exactly the class of
        // attack the harness exists to catch.
        let mut chars: Vec<char> = att.commit_hex.chars().collect();
        chars[0] = if chars[0] == '0' { '1' } else { '0' };
        att.commit_hex = chars.into_iter().collect();
        assert!(verify_attestation(&att).is_err());
    }

    #[test]
    fn verify_rejects_bad_hex() {
        let input = kat_input();
        let mut att = attest(&input).unwrap();
        att.weights_digest_hex = "zzzz".to_string();
        assert!(verify_attestation(&att).is_err());
    }

    #[test]
    fn verify_rejects_wrong_length_digest() {
        let input = kat_input();
        let mut att = attest(&input).unwrap();
        // 16 hex chars = 8 bytes, not 32.
        att.inputs_digest_hex = "aabbccddeeff0011".to_string();
        assert!(verify_attestation(&att).is_err());
    }

    #[test]
    fn verify_rejects_empty_model_id() {
        let input = kat_input();
        let mut att = attest(&input).unwrap();
        att.model_id = "   ".to_string();
        assert!(verify_attestation(&att).is_err());
    }

    #[test]
    fn all_artefacts_ok_false_on_empty() {
        assert!(!all_artefacts_ok(&[]));
    }

    // ---------------------------------------------------------------
    // Strict schema tests. Every field of the Attestation struct that
    // downstream code reads must be a required key in STRICT_REQUIRED_KEYS,
    // otherwise a renamed/misspelled field would deserialise to a
    // zero-value and only fail later at commit recomputation with a
    // confusing "digest mismatch" instead of a legible "missing field X".
    // ---------------------------------------------------------------

    fn valid_envelope_json() -> String {
        let att = attest(&kat_input()).unwrap();
        serde_json::to_string(&att).unwrap()
    }

    #[test]
    fn strict_accepts_valid_envelope() {
        let json = valid_envelope_json();
        strict_check_envelope(&json).expect("valid envelope must pass strict check");
    }

    #[test]
    fn strict_rejects_missing_required_field() {
        // Drop weights_digest_hex from a valid envelope. Serde would
        // silently give us String::default() here; strict must not.
        let json = valid_envelope_json();
        let mut v: serde_json::Value = serde_json::from_str(&json).unwrap();
        v.as_object_mut().unwrap().remove("weights_digest_hex");
        let mangled = serde_json::to_string(&v).unwrap();
        let err =
            strict_check_envelope(&mangled).expect_err("missing required field must fail strict");
        let msg = format!("{err:#}");
        assert!(
            msg.contains("weights_digest_hex"),
            "error must name field: {msg}"
        );
        assert!(
            msg.contains("missing required field"),
            "error must be legible: {msg}"
        );
    }

    #[test]
    fn strict_rejects_null_required_field() {
        // Explicit `null` counts as missing, not as "present with empty
        // value". This is what catches an emitter that panics mid-flight
        // and writes a partial envelope with placeholder nulls.
        let json = valid_envelope_json();
        let mut v: serde_json::Value = serde_json::from_str(&json).unwrap();
        v.as_object_mut()
            .unwrap()
            .insert("inputs_digest_hex".to_string(), serde_json::Value::Null);
        let mangled = serde_json::to_string(&v).unwrap();
        let err = strict_check_envelope(&mangled).expect_err("null field must fail strict");
        let msg = format!("{err:#}");
        assert!(
            msg.contains("inputs_digest_hex"),
            "error must name field: {msg}"
        );
    }

    #[test]
    fn strict_rejects_empty_string_required_field() {
        // Empty string is what serde would land on for a genuinely
        // missing digest field — in strict mode, that is also rejected.
        let json = valid_envelope_json();
        let mut v: serde_json::Value = serde_json::from_str(&json).unwrap();
        v.as_object_mut().unwrap().insert(
            "outputs_digest_hex".to_string(),
            serde_json::Value::String("".to_string()),
        );
        let mangled = serde_json::to_string(&v).unwrap();
        let err = strict_check_envelope(&mangled).expect_err("empty string must fail strict");
        let msg = format!("{err:#}");
        assert!(
            msg.contains("outputs_digest_hex"),
            "error must name field: {msg}"
        );
    }

    #[test]
    fn strict_reports_all_missing_fields_together() {
        // Whack-a-mole is a bad auditor experience — all missing fields
        // must be reported in a single error.
        let json = valid_envelope_json();
        let mut v: serde_json::Value = serde_json::from_str(&json).unwrap();
        v.as_object_mut().unwrap().remove("model_id");
        v.as_object_mut().unwrap().remove("commit_hex");
        let mangled = serde_json::to_string(&v).unwrap();
        let err = strict_check_envelope(&mangled).unwrap_err();
        let msg = format!("{err:#}");
        assert!(msg.contains("model_id"), "must mention model_id: {msg}");
        assert!(msg.contains("commit_hex"), "must mention commit_hex: {msg}");
    }

    #[test]
    fn strict_rejects_unknown_field() {
        // This is the whole point of --strict: an envelope with a typo
        // like `weights_digets_hex` (transposed) would otherwise be read
        // as "weights_digest_hex missing, unknown field weights_digets_hex
        // ignored" and fail later with an opaque "commit mismatch". Strict
        // names the typo directly.
        let json = valid_envelope_json();
        let mut v: serde_json::Value = serde_json::from_str(&json).unwrap();
        v.as_object_mut().unwrap().insert(
            "weights_digets_hex".to_string(),
            serde_json::Value::String("deadbeef".to_string()),
        );
        let mangled = serde_json::to_string(&v).unwrap();
        let err = strict_check_envelope(&mangled).expect_err("unknown field must fail strict");
        let msg = format!("{err:#}");
        assert!(
            msg.contains("weights_digets_hex"),
            "error must name typo: {msg}"
        );
        assert!(msg.contains("unknown"), "error must say unknown: {msg}");
    }

    #[test]
    fn strict_accepts_missing_optional_disclosure() {
        // disclosure is optional — dropping it must not trip strict.
        let json = valid_envelope_json();
        let mut v: serde_json::Value = serde_json::from_str(&json).unwrap();
        v.as_object_mut().unwrap().remove("disclosure");
        let mangled = serde_json::to_string(&v).unwrap();
        strict_check_envelope(&mangled).expect("missing optional field must not trip strict");
    }

    #[test]
    fn strict_rejects_non_object_root() {
        let err = strict_check_envelope("[1,2,3]").expect_err("array root must fail");
        assert!(format!("{err:#}").contains("JSON object"));
    }

    #[test]
    fn strict_rejects_syntactic_garbage() {
        let err = strict_check_envelope("{not json").expect_err("garbage must fail");
        assert!(format!("{err:#}").contains("not syntactically valid JSON"));
    }

    #[test]
    fn strict_key_lists_match_struct() {
        // Belt-and-braces: if someone adds a field to Attestation but
        // forgets to update STRICT_REQUIRED_KEYS / STRICT_OPTIONAL_KEYS,
        // strict mode would happily reject a valid envelope from the
        // new emitter as "unknown field". Serialize a full envelope and
        // confirm every top-level key it produces is covered.
        let att = attest(&kat_input()).unwrap();
        let json = serde_json::to_string(&att).unwrap();
        let v: serde_json::Value = serde_json::from_str(&json).unwrap();
        for key in v.as_object().unwrap().keys() {
            let k = key.as_str();
            let covered = STRICT_REQUIRED_KEYS.contains(&k) || STRICT_OPTIONAL_KEYS.contains(&k);
            assert!(
                covered,
                "Attestation struct has key {k:?} not covered by STRICT_REQUIRED_KEYS or STRICT_OPTIONAL_KEYS — strict mode would reject a valid envelope; update the lists in lib.rs"
            );
        }
    }

    #[test]
    fn load_strict_round_trip() {
        // End-to-end: attest -> serialise to disk -> load_attestation_strict
        // must round-trip a valid envelope byte-for-byte.
        use std::io::Write;
        let att = attest(&kat_input()).unwrap();
        let json = serde_json::to_string_pretty(&att).unwrap();
        let tmp = tempfile::NamedTempFile::new().unwrap();
        tmp.as_file().write_all(json.as_bytes()).unwrap();
        let loaded = load_attestation_strict(tmp.path()).expect("strict load of valid envelope");
        assert_eq!(loaded, att);
    }
}
