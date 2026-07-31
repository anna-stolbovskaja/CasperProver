//! End-to-end tests for the `cp-replay` CLI. We build our own envelopes
//! via the library entry points, then shell out to the compiled binary
//! (via `env!("CARGO_BIN_EXE_cp-replay")`) to exercise the exact code
//! path an auditor would use.

use cp_replay::{attest, AttestInput};
use sha2::{Digest, Sha256};
use std::fs;
use std::process::Command;

fn kat_input() -> AttestInput {
    AttestInput {
        model_id: "mnist-mlp-8x8-v0".to_string(),
        weights_digest: Sha256::digest(b"weights").into(),
        inputs_digest: Sha256::digest(b"inputs").into(),
        outputs_digest: Sha256::digest(b"outputs").into(),
    }
}

fn bin() -> String {
    env!("CARGO_BIN_EXE_cp-replay").to_string()
}

#[test]
fn cli_verify_valid_envelope_exits_zero() {
    let dir = tempfile::tempdir().unwrap();
    let att = attest(&kat_input()).unwrap();
    let att_path = dir.path().join("attestation.json");
    fs::write(&att_path, serde_json::to_string_pretty(&att).unwrap()).unwrap();

    let output = Command::new(bin())
        .args(["verify", "--attestation"])
        .arg(&att_path)
        .output()
        .unwrap();

    assert!(
        output.status.success(),
        "expected exit 0, got {:?}\nstdout: {}\nstderr: {}",
        output.status.code(),
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr),
    );
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("verified"), "stdout: {stdout}");
    assert!(
        stdout.contains("NOT a cryptographic proof"),
        "stdout: {stdout}"
    );
}

#[test]
fn cli_verify_tampered_envelope_exits_one() {
    let dir = tempfile::tempdir().unwrap();
    let mut att = attest(&kat_input()).unwrap();
    // Flip one hex nibble in commit — classic tamper attempt.
    let mut chars: Vec<char> = att.commit_hex.chars().collect();
    chars[0] = if chars[0] == '0' { '1' } else { '0' };
    att.commit_hex = chars.into_iter().collect();
    let att_path = dir.path().join("attestation.json");
    fs::write(&att_path, serde_json::to_string_pretty(&att).unwrap()).unwrap();

    let output = Command::new(bin())
        .args(["verify", "--attestation"])
        .arg(&att_path)
        .output()
        .unwrap();

    assert_eq!(output.status.code(), Some(1), "expected exit 1");
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("FAILED") || stderr.contains("commit mismatch"),
        "stderr: {stderr}"
    );
}

#[test]
fn cli_verify_reserved_scheme_rejected() {
    let dir = tempfile::tempdir().unwrap();
    let mut att = attest(&kat_input()).unwrap();
    att.scheme = "zkml-fixed-v0".to_string();
    let att_path = dir.path().join("attestation.json");
    fs::write(&att_path, serde_json::to_string_pretty(&att).unwrap()).unwrap();

    let output = Command::new(bin())
        .args(["verify", "--attestation"])
        .arg(&att_path)
        .output()
        .unwrap();

    assert_eq!(output.status.code(), Some(1));
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("reserved"), "stderr: {stderr}");
}

#[test]
fn cli_verify_missing_file_exits_two() {
    let output = Command::new(bin())
        .args(["verify", "--attestation", "/nonexistent/path/x.json"])
        .output()
        .unwrap();

    assert_eq!(
        output.status.code(),
        Some(2),
        "expected exit 2 (I/O error), got {:?}",
        output.status.code()
    );
}

#[test]
fn cli_replay_artefacts_happy_path() {
    let dir = tempfile::tempdir().unwrap();
    // Write physical artefacts whose SHA-256s match the KAT input digests.
    let weights_path = dir.path().join("weights.bin");
    fs::write(&weights_path, b"weights").unwrap();
    let inputs_path = dir.path().join("inputs.bin");
    fs::write(&inputs_path, b"inputs").unwrap();
    let outputs_path = dir.path().join("outputs.bin");
    fs::write(&outputs_path, b"outputs").unwrap();

    let att = attest(&kat_input()).unwrap();
    let att_path = dir.path().join("attestation.json");
    fs::write(&att_path, serde_json::to_string_pretty(&att).unwrap()).unwrap();

    let output = Command::new(bin())
        .args(["replay-artefacts", "--attestation"])
        .arg(&att_path)
        .arg("--weights")
        .arg(&weights_path)
        .arg("--inputs")
        .arg(&inputs_path)
        .arg("--outputs")
        .arg(&outputs_path)
        .output()
        .unwrap();

    assert!(
        output.status.success(),
        "expected exit 0, got {:?}\nstdout: {}\nstderr: {}",
        output.status.code(),
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr),
    );
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("all match"), "stdout: {stdout}");
    assert!(stdout.contains("envelope    : ✓"), "stdout: {stdout}");
}

#[test]
fn cli_replay_artefacts_detects_swapped_file() {
    let dir = tempfile::tempdir().unwrap();
    let weights_path = dir.path().join("weights.bin");
    fs::write(&weights_path, b"weights").unwrap();
    let inputs_path = dir.path().join("inputs.bin");
    fs::write(&inputs_path, b"inputs").unwrap();
    // Deliberately wrong content — hashes to something other than the
    // envelope's outputs_digest.
    let outputs_path = dir.path().join("outputs.bin");
    fs::write(&outputs_path, b"OUTPUTS-BUT-DIFFERENT").unwrap();

    let att = attest(&kat_input()).unwrap();
    let att_path = dir.path().join("attestation.json");
    fs::write(&att_path, serde_json::to_string_pretty(&att).unwrap()).unwrap();

    let output = Command::new(bin())
        .args(["replay-artefacts", "--attestation"])
        .arg(&att_path)
        .arg("--weights")
        .arg(&weights_path)
        .arg("--inputs")
        .arg(&inputs_path)
        .arg("--outputs")
        .arg(&outputs_path)
        .output()
        .unwrap();

    assert_eq!(output.status.code(), Some(1));
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("mismatch below"), "stdout: {stdout}");
    // The per-artefact row for outputs must be present and marked ✗.
    let outputs_row = stdout
        .lines()
        .find(|l| l.trim_start().starts_with("outputs"))
        .expect("expected an 'outputs' row in the mismatch report");
    assert!(outputs_row.contains('✗'), "outputs row: {outputs_row}");
}

#[test]
fn cli_json_output_is_valid_json() {
    let dir = tempfile::tempdir().unwrap();
    let att = attest(&kat_input()).unwrap();
    let att_path = dir.path().join("attestation.json");
    fs::write(&att_path, serde_json::to_string_pretty(&att).unwrap()).unwrap();

    let output = Command::new(bin())
        .args(["--json", "verify", "--attestation"])
        .arg(&att_path)
        .output()
        .unwrap();

    assert!(output.status.success());
    let stdout = String::from_utf8(output.stdout).unwrap();
    let parsed: serde_json::Value =
        serde_json::from_str(stdout.trim()).expect("machine-readable output must be JSON");
    assert_eq!(parsed["ok"], serde_json::Value::Bool(true));
    assert_eq!(
        parsed["scheme"],
        serde_json::Value::String("ml-attest-v0".into())
    );
}

#[test]
fn cli_commit_only_matches_full_attest() {
    let input = kat_input();
    let full = attest(&input).unwrap();

    let output = Command::new(bin())
        .args([
            "--json",
            "commit-only",
            "--model-id",
            &input.model_id,
            "--weights-digest-hex",
            &hex::encode(input.weights_digest),
            "--inputs-digest-hex",
            &hex::encode(input.inputs_digest),
            "--outputs-digest-hex",
            &hex::encode(input.outputs_digest),
        ])
        .output()
        .unwrap();

    assert!(output.status.success());
    let stdout = String::from_utf8(output.stdout).unwrap();
    let parsed: cp_replay::Attestation = serde_json::from_str(stdout.trim()).unwrap();
    assert_eq!(parsed.commit_hex, full.commit_hex);
}

// ---------------------------------------------------------------------
// --strict CLI tests. Focus on the *behavioural difference* between the
// default (permissive) and --strict runs on the same tampered envelope.
// ---------------------------------------------------------------------

/// Build a valid envelope, drop `weights_digest_hex` from the JSON, and
/// return the file path. Serde would silently deserialise this to
/// `weights_digest_hex = ""` and fall over later at hex decode with an
/// opaque "not valid hex" error; --strict must catch it at load time and
/// name the field.
fn write_envelope_missing_weights_digest_hex(dir: &tempfile::TempDir) -> std::path::PathBuf {
    let att = attest(&kat_input()).unwrap();
    let mut v: serde_json::Value = serde_json::to_value(&att).unwrap();
    v.as_object_mut().unwrap().remove("weights_digest_hex");
    let path = dir.path().join("attestation.json");
    fs::write(&path, serde_json::to_string_pretty(&v).unwrap()).unwrap();
    path
}

/// Build a valid envelope, then insert a typo'd `weights_digets_hex` key
/// (note the transposed letters) alongside the real one. Permissive mode
/// silently ignores the typo, --strict must reject it and name the typo.
fn write_envelope_with_typo_key(dir: &tempfile::TempDir) -> std::path::PathBuf {
    let att = attest(&kat_input()).unwrap();
    let mut v: serde_json::Value = serde_json::to_value(&att).unwrap();
    v.as_object_mut().unwrap().insert(
        "weights_digets_hex".to_string(),
        serde_json::Value::String("cafebabe".to_string()),
    );
    let path = dir.path().join("attestation.json");
    fs::write(&path, serde_json::to_string_pretty(&v).unwrap()).unwrap();
    path
}

#[test]
fn cli_verify_strict_rejects_missing_required_field() {
    let dir = tempfile::tempdir().unwrap();
    let att_path = write_envelope_missing_weights_digest_hex(&dir);

    let output = Command::new(bin())
        .args(["--strict", "verify", "--attestation"])
        .arg(&att_path)
        .output()
        .unwrap();

    assert_eq!(
        output.status.code(),
        Some(1),
        "strict mode must fail with exit 1 on missing field\nstdout: {}\nstderr: {}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr),
    );
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("weights_digest_hex"),
        "strict error must name missing field: {stderr}"
    );
    assert!(
        stderr.contains("strict"),
        "error must mention strict mode: {stderr}"
    );
}

#[test]
fn cli_verify_strict_rejects_unknown_field() {
    let dir = tempfile::tempdir().unwrap();
    let att_path = write_envelope_with_typo_key(&dir);

    let output = Command::new(bin())
        .args(["--strict", "verify", "--attestation"])
        .arg(&att_path)
        .output()
        .unwrap();

    assert_eq!(
        output.status.code(),
        Some(1),
        "strict mode must fail with exit 1 on unknown field"
    );
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("weights_digets_hex"),
        "strict error must name the typo: {stderr}"
    );
    assert!(
        stderr.contains("unknown"),
        "strict error must say 'unknown': {stderr}"
    );
}

#[test]
fn cli_verify_permissive_silently_accepts_unknown_field() {
    // The whole point of --strict: WITHOUT it, an emitter that adds
    // `weights_digets_hex` (typo of the real field) is silently accepted
    // and only fails as an opaque "not valid hex" downstream. This test
    // pins that legacy behaviour so we don't accidentally tighten it in
    // permissive mode and break older auditors.
    let dir = tempfile::tempdir().unwrap();
    let att_path = write_envelope_with_typo_key(&dir);

    let output = Command::new(bin())
        .args(["verify", "--attestation"])
        .arg(&att_path)
        .output()
        .unwrap();

    // Without --strict, the typo is silently ignored and the real field
    // is still present, so the envelope verifies fine. This is legacy
    // permissive behaviour.
    assert_eq!(
        output.status.code(),
        Some(0),
        "permissive mode must silently accept unknown-field envelopes"
    );
}

#[test]
fn cli_verify_strict_accepts_clean_envelope() {
    let dir = tempfile::tempdir().unwrap();
    let att = attest(&kat_input()).unwrap();
    let att_path = dir.path().join("attestation.json");
    fs::write(&att_path, serde_json::to_string_pretty(&att).unwrap()).unwrap();

    let output = Command::new(bin())
        .args(["--strict", "verify", "--attestation"])
        .arg(&att_path)
        .output()
        .unwrap();

    assert!(
        output.status.success(),
        "strict mode must accept a clean envelope\nstderr: {}",
        String::from_utf8_lossy(&output.stderr),
    );
}

#[test]
fn cli_verify_strict_json_output_shape() {
    // The JSON error shape is scriptable — pin it so downstream auditors
    // can grep for `"strict": true` reliably.
    let dir = tempfile::tempdir().unwrap();
    let att_path = write_envelope_missing_weights_digest_hex(&dir);

    let output = Command::new(bin())
        .args(["--strict", "--json", "verify", "--attestation"])
        .arg(&att_path)
        .output()
        .unwrap();

    assert_eq!(output.status.code(), Some(1));
    let stdout = String::from_utf8(output.stdout).unwrap();
    let parsed: serde_json::Value = serde_json::from_str(stdout.trim()).unwrap();
    assert_eq!(parsed["ok"], serde_json::Value::Bool(false));
    assert_eq!(parsed["strict"], serde_json::Value::Bool(true));
    assert!(
        parsed["error"]
            .as_str()
            .unwrap()
            .contains("weights_digest_hex"),
        "JSON error field must name the missing field: {stdout}"
    );
}
