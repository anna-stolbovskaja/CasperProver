# cp-replay `--strict` — envelope schema hardening

Status: **implemented** (branch `replay/strict-schema-v0`, not merged).
Sits on top of `replay/deterministic-harness-v0` (branch B).
Zero touch on engine / contracts / SDK / main.

## The gap

Baseline `cp-replay` uses stock `serde_json::from_str` to parse an
`Attestation` envelope. `serde` is deliberately permissive by default:

- **Unknown top-level keys are silently ignored.** An envelope that carries
  `weights_digets_hex` (typo — transposed letters) alongside the real
  `weights_digest_hex` deserialises without complaint. If the real field is
  *also* misspelled, `weights_digest_hex` lands as `String::default()`
  (empty string), which then fails downstream at `parse_digest_hex` with an
  opaque *"not valid hex"* error, indistinguishable from a genuine tamper.
- **Missing required keys are treated as empty strings.** Same failure
  mode, same opaque downstream error.

Both cases produce a legible-looking failure ("commit mismatch" or "not
valid hex") that hides the actual root cause: **the emitter produced a
malformed envelope**, and the auditor now has to reverse-engineer *which*
field is wrong from an error message that names none of them.

## The fix

`--strict` is a global CLI flag that reroutes envelope loading through
`load_attestation_strict()`, which:

1. Parses the JSON with `serde_json::Value` and confirms the root is a JSON
   object.
2. Confirms every key in `STRICT_REQUIRED_KEYS` is present, non-null, and
   not the empty string. All missing/empty fields are reported together
   (not one-at-a-time whack-a-mole).
3. Confirms every top-level key is either in `STRICT_REQUIRED_KEYS` or
   `STRICT_OPTIONAL_KEYS`. Any unknown key is a hard error naming the key.

Only then does `serde` deserialise into `Attestation`. Downstream code
(`verify_attestation`, `replay_artefacts`) is unchanged and unaware of
strict mode.

The trade-off is deliberately opt-in: baseline permissive behaviour is
preserved for auditors on old builds against newer emitters that add
backwards-compatible fields.

## Contract stability

- `STRICT_REQUIRED_KEYS` and `STRICT_OPTIONAL_KEYS` are `pub const &[&str]`.
- Unit test `strict_key_lists_match_struct` serialises a full `Attestation`
  and asserts every top-level key it produces is covered by one of the two
  lists. If a future patch adds a field to `Attestation` but forgets to
  update the lists, the test fires with a legible instruction to update
  `lib.rs`.

This closes the "silent drift between struct and schema lists" gap that
every hand-maintained allowlist eventually accumulates.

## Exit-code discipline

`cp-replay` inherits its exit-code scheme from `main.rs`:

- `0` — every requested check passed
- `1` — at least one check failed (mismatch, tamper, unknown scheme,
  **or strict-schema violation**)
- `2` — I/O or parse error (missing file, syntactically invalid JSON)

A strict-schema failure lives in class 1 (envelope rejected), not class 2
(I/O error), because auditors script exit codes to distinguish "envelope
is bad" from "I couldn't read the envelope at all". The distinction was
tightened during implementation — the CLI now formats the full anyhow
context chain (`{err:#}`) in both JSON and human output, since the outer
wrap `"strict-check <path>"` alone did not name the offending field. This
was caught by `cli_verify_strict_json_output_shape` on the first run.

## Design catch during implementation

The initial `--strict` implementation used the outer `err.to_string()` in
the JSON error branch, which produced only the outermost context
(`"strict-check /tmp/attestation.json"`) without the root cause naming the
missing field. The CLI test `cli_verify_strict_json_output_shape` failed
on first run with this error visible in stdout. Fixed by switching both
branches (JSON and human) to `format!("{err:#}")`, which preserves the
whole anyhow context chain.

This is why the CLI has an explicit test for the JSON error shape: the
error field is scriptable for downstream auditors, and its content is
part of the observable contract.

## Test coverage

**11 new unit tests** in `src/lib.rs`:

- `strict_accepts_valid_envelope` — clean happy path
- `strict_rejects_missing_required_field`
- `strict_rejects_null_required_field` — explicit `null` counts as missing
- `strict_rejects_empty_string_required_field`
- `strict_reports_all_missing_fields_together` — no whack-a-mole
- `strict_rejects_unknown_field` — the typo canary
- `strict_accepts_missing_optional_disclosure` — optional keys OK
- `strict_rejects_non_object_root`
- `strict_rejects_syntactic_garbage`
- `strict_key_lists_match_struct` — drift canary
- `load_strict_round_trip` — end-to-end file I/O

**5 new CLI tests** in `tests/cli.rs`:

- `cli_verify_strict_accepts_clean_envelope`
- `cli_verify_strict_rejects_missing_required_field`
- `cli_verify_strict_rejects_unknown_field`
- `cli_verify_permissive_silently_accepts_unknown_field` — **pins the
  legacy behaviour** so we don't accidentally tighten permissive mode
  and break older auditors
- `cli_verify_strict_json_output_shape` — pins the JSON error contract

Total: **34/34 PASS** (21 unit + 13 CLI), fmt clean, clippy zero warnings.

## Injection test

To confirm the strict check actually binds (not just an inert code path
that always passes), I weakened `strict_check_envelope` by removing the
unknown-key `bail!` branch. `cli_verify_strict_rejects_unknown_field`
immediately failed with `exit 0` instead of the expected `exit 1`. On
restoring the branch, the full suite went green again.

Evidence that the strict check is load-bearing, not decorative.

## Non-goals

- **Strict mode does NOT verify commit_hex any more strongly than default
  mode.** It's a *schema* check, not a *cryptographic* check. The
  cryptographic check (`verify_attestation`) is orthogonal and runs in
  both modes.
- **Strict mode does NOT constrain the shape of individual field values
  beyond "non-null, non-empty".** A malformed `weights_digest_hex` value
  (say, `"not-hex-at-all"`) still passes strict schema and gets caught by
  `parse_digest_hex` downstream. That's the right layer for the check —
  strict is a layer above.
- **Strict mode does NOT propagate to sub-envelopes.** If future
  attestation schemes carry nested structures, each layer needs its own
  strict check.

## Cross-links

- Baseline harness: `docs/roadmap/DETERMINISTIC_REPLAY_HARNESS.md`
- Honesty invariant: `docs/ZKML_HONEST_VERDICT.md`
- Reference commit implementation: `engine/internal/mlattest/harness.go`
