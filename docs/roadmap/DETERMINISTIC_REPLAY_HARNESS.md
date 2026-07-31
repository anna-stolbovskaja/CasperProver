# Deterministic Replay Harness — `cp-replay`

Status: **v0 shipped in branch `replay/deterministic-harness-v0` (not merged to `main`)**
Owner: engine team
Related: [`docs/ZKML_HONEST_VERDICT.md`](../ZKML_HONEST_VERDICT.md), [`docs/roadmap/ML_ATTESTATION_HARNESS.md`](./ML_ATTESTATION_HARNESS.md)

## 1. Problem this solves

CasperProver's `POST /v1/ml/attest` (shipped in `zkml/circuit-harness-v0`)
emits an `ml-attest-v0` envelope that hash-chains
`(model_id, weights_digest, inputs_digest, outputs_digest)` into a
`commit_hex`. An auditor who receives one of these envelopes today has
three questions they cannot answer without pulling in the full Go engine:

1. **Is `commit_hex` consistent with the four digest fields?** (i.e. was
   the envelope tampered with in flight?)
2. **Do the physical artefacts on disk actually match the digests the
   envelope commits to?**
3. **Does the CasperProver engine implement the commit function the way
   its documentation claims?**

`cp-replay` is a small, standalone Rust CLI that answers (1) and (2)
without a Go toolchain and gives (3) a KAT-pinned second implementation
that must byte-match the Go engine or the tests scream.

## 2. Non-goals — the honesty invariant, restated

This tool does **not**:

- Prove that a named model was actually executed on the named inputs
  (that would require real ZK-ML; the four gating conditions in
  [`docs/ZKML_HONEST_VERDICT.md`](../ZKML_HONEST_VERDICT.md) still stand).
- Emit or accept the reserved scheme label `zkml-fixed-v0` — the verifier
  refuses it up front, exactly as the Go verifier does.
- Introduce a new scheme label. `ml-attest-v0` remains the only scheme
  either implementation is willing to verify.

Every human-readable output ends with the same disclosure banner that is
embedded in the on-wire envelope's `disclosure` field. There is no way
to run this CLI, get a green tick, and reasonably believe you have
proven inference.

## 3. Wire-level contract (frozen)

The Rust `Attestation` struct in `tools/cp-replay/src/lib.rs` matches
the Go `mlattest.Attestation` struct field-for-field:

| JSON field           | Go type       | Rust type       | Notes                    |
|----------------------|---------------|-----------------|--------------------------|
| `scheme`             | `AttestationScheme` | `String`  | Must be `ml-attest-v0`   |
| `model_id`           | `string`      | `String`        | Non-empty, opaque        |
| `weights_digest_hex` | `string`      | `String`        | Hex SHA-256, 64 chars    |
| `inputs_digest_hex`  | `string`      | `String`        | Hex SHA-256, 64 chars    |
| `outputs_digest_hex` | `string`      | `String`        | Hex SHA-256, 64 chars    |
| `commit_hex`         | `string`      | `String`        | Hex SHA-256, 64 chars    |
| `disclosure`         | `string`      | `String`        | Optional in Rust, ignored on verify |

The commit function itself, re-stated in one place so both sides agree:

```
step_a = SHA256( model_id_utf8_bytes || weights_digest_bytes )
step_b = SHA256( inputs_digest_bytes || outputs_digest_bytes )
seed   = SHA256( "ml-attest-v0" utf8_bytes )
commit = SHA256( seed || step_a || step_b )
```

If either implementation ever changes this recipe, they must change it
together in the same PR, and the KAT below must be regenerated.

### Pinned known-answer vector

For inputs

- `model_id = "mnist-mlp-8x8-v0"`
- `weights_digest = SHA-256("weights")`
- `inputs_digest  = SHA-256("inputs")`
- `outputs_digest = SHA-256("outputs")`

both implementations (`HashMLAttestor.commit()` in Go, `cp_replay::commit()`
in Rust) MUST produce:

```
commit_hex = d384b504fb72a340c972b8ab3ceb15fa388dda59a5548ea411023ff204e0a24a
```

This is enforced by `commit_matches_pinned_go_reference_vector` in
`tools/cp-replay/src/lib.rs`. Cross-checked against the Go engine on
2026-07-31 (branch tip of `replay/deterministic-harness-v0`).

## 4. CLI reference

The binary lives at `tools/cp-replay/`. It builds on stable Rust
(currently 1.97), takes no network, and holds no state.

### `cp-replay verify --attestation <path>`

Verifies a single envelope offline:

- Rejects unknown or reserved scheme labels.
- Decodes and re-hashes the three digest fields.
- Recomputes `commit_hex` and compares to the envelope.

Exit codes:

- `0` — verified
- `1` — verification failed (tamper / wrong scheme / bad hex / length mismatch)
- `2` — I/O or JSON parse error

### `cp-replay replay-artefacts --attestation <path> --weights <path> --inputs <path> --outputs <path>`

Verifies the envelope, then SHA-256s the three physical files and
confirms they match the digests the envelope committed to. This is the
**strongest** integrity signal `cp-replay` produces: the envelope is
internally consistent AND the artefacts on disk are what the emitter
saw. Still not a ZK-ML proof.

### `cp-replay commit-only --model-id <s> --weights-digest-hex <hex> --inputs-digest-hex <hex> --outputs-digest-hex <hex>`

Computes a commit from raw fields and prints a fresh envelope. Useful
for cross-checking against a Go-emitted commit — used to regenerate the
pinned KAT above.

### `--json`

Every subcommand accepts `--json` to emit a machine-readable status
object on stdout. Auditors scripting bulk verification should prefer
this over the human report.

## 5. Tests

`cargo test --release` in `tools/cp-replay/` runs:

- 10 unit tests inside `src/lib.rs` — commit determinism, tamper
  detection on every field, hex/length rejection, reserved-scheme
  refusal, and the pinned KAT above.
- 8 end-to-end CLI tests in `tests/cli.rs` — happy path, tampered
  commit, reserved scheme, missing file, artefact swap detection,
  `--json` shape, `commit-only` cross-check against library `attest()`.

Total: **18 tests, all green on stable Rust 1.97**.

## 6. Deliberate scope limits

- **Not part of `contracts/` workspace.** `contracts/` targets
  `wasm32-unknown-unknown` under `nightly-2025-01-01` for Casper testnet
  compatibility. Mixing a host CLI in that workspace would force every
  contract crate to compile against `std`, breaking WASM. `cp-replay`
  gets its own `Cargo.toml`.
- **No release plumbing yet.** No `cargo publish`, no GitHub release
  binary, no `sdk-publish-*.yml`. When we're ready, add a
  `cp-replay-release.yml` workflow modelled on the existing SDK-publish
  ones and cross-link here.
- **No streaming / batch mode yet.** One envelope per invocation.
- **No signature check.** The envelope is unsigned by design in
  `ml-attest-v0`. If the future ZK-ML scheme adds a signature, extend
  `verify_attestation()` there, not here.

## 7. Roadmap follow-ups

Tracked here so they do not evaporate into daily-log churn:

- **cross-language KAT in Go** — mirror the pinned KAT as a Go test in
  `engine/internal/mlattest/` so a change to the Go commit function also
  breaks the Go build, not just the Rust build. Small, safe follow-up.
- **`MerkleMLAttestor` compatibility** — when the second `Attestor`
  impl lands (see [`ML_ATTESTATION_HARNESS.md`](./ML_ATTESTATION_HARNESS.md)),
  extend `cp-replay` with a `--scheme` selector and add per-scheme
  verify paths. Reject unknown schemes by default.
- **Bulk mode** — `cp-replay verify --stdin` reading NDJSON envelopes,
  emitting one status line per input, for auditors processing large
  batches from disk.
- **Trust-boundary doc** — a short piece under `docs/AUDITOR_TOOLING.md`
  that walks a third party through downloading a release binary,
  checksumming it, and running `cp-replay` end-to-end without ever
  touching the Go source. Draft after we have a signed release.
