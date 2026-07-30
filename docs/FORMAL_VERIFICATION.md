# Formal Verification

Scope statement. **What has actually been done** in this repository and
**what a full formal verification effort would require**. Everything in the
second half is out of scope for the hackathon submission and is documented
here so a reviewer never has to guess whether "formal verification" is a
shipped feature or a roadmap item.

## TL;DR

- **Shipped (engine-side state machines):** four TLA+ specs under
  `specs/` — `ProofSystemSpec`, `QuorumSpec`, `ReceiptLineageSpec`,
  `CanonicalOrderSpec` — model-checked by TLC on every push/PR that
  touches `specs/` via `.github/workflows/formal-verification.yml`.
  Full disclosure: `docs/roadmap/FORMAL_VERIFICATION.md`.
- **NOT shipped (contract-side invariants):** no machine-checked proof
  or model-check of the Odra Rust contracts (I-*, X-*, F-* in
  `docs/CONTRACT_INVARIANTS.md`) exists. The four shipped specs cover
  engine-side state machines (proof registry, quorum registry, receipt
  lineage, canonical-hash sort-normalisation), not the on-chain state.
- **What holds instead for the contracts:** informal invariants
  documented in `docs/CONTRACT_INVARIANTS.md` (I-*, X-*, F-*), Odra
  unit tests, and a `verify.sh` smoke pipeline. That is the current
  level of assurance for the contract layer — nothing more, nothing
  less.

If a reviewer sees the phrase "formal verification" applied to the
contract layer anywhere else in the repo (README, marketing site,
pitch), and the phrasing implies more than "we wrote down the invariants
and unit-tested them", that phrasing is a bug — please file it as an
INVARIANT BREAK issue. The engine-side TLA+ specs, in contrast, are
real machine-checked artefacts.

## What is shipped today (engine-side)

Four TLA+ specs, all TLC-checked on every push/PR that touches
`specs/**` via `.github/workflows/formal-verification.yml`. Full
detail per-spec (invariants, model bounds, state counts, runtimes) is
in `docs/roadmap/FORMAL_VERIFICATION.md`. Summary:

| Spec                     | Models                                              | Distinct states | Wall time |
|--------------------------|-----------------------------------------------------|-----------------|-----------|
| `ProofSystemSpec.tla`    | proof-registry / decision-attestation state machine | ~6.15M          | ~2m40s    |
| `QuorumSpec.tla`         | BLS12-381 threshold-quorum registry                 | 1,576           | ~1s       |
| `ReceiptLineageSpec.tla` | receipt-lineage DAG (Ancestors walk)                | 68              | <1s       |
| `CanonicalOrderSpec.tla` | canonical-hash sort-normalisation invariance        | 41              | ~1s       |

All four run under `bash specs/run-tlc.sh` locally (pod ~3 min total).
CI budget: 25 min per spec, 30 min per job. Failure uploads counter-
example traces as an artefact for 7 days.

None of these cover the contract layer.

## What "shipped for the contracts" would mean

The engine-side specs above do not touch Odra Rust invariants. To
claim a small-model TLC / Apalache / Kani pass over the contracts, we
would need:

1. A `specs/ContractCore.tla` module (or Quint / Apalache / Kani
   equivalent) covering the reachable contract state.
2. An `MC.tla` (or Kani harness) config bounding state (typically at
   most 3 accounts, 4 stake amounts, 2 slash amounts).
3. The existing `.github/workflows/formal-verification.yml` picks up
   any new `specs/*.tla` file automatically — no CI change required.
   A Kani-based approach would add a parallel job.
4. A short write-up in this file mapping each modelled invariant to
   its source-code enforcement point (`contracts/*/src/main.rs`, line
   numbers).

This section is the checklist for the change that would flip
"contract layer: not shipped" to "contract layer: shipped: small-model
TLC / Kani pass".

## What would ever be in scope

The invariants worth modelling first (in this order):

- **I-1 · Owner isolation.** Small state space (one owner slot, three
  candidate callers). Model checks that no reachable state has a non-owner
  successfully mutating configuration.
- **I-3 · Monotonic reputation.** Reputation floor never decreases across
  `revoke_proof` / `record_stake` / `report_and_slash` sequences.
- **I-4 · Checked arithmetic.** Small U64 domain, all pairs of
  `checked_add` / `checked_sub` cover boundary cases. Doable in Kani or a
  Rust proof harness rather than TLA+.
- **X-3 · Cross-contract writeback on slash.** Requires modelling two
  contracts as separate processes with a shared oracle. Non-trivial.

## What would never be in scope (for this project)

- End-to-end proof of the Groth16 verifier. The `arkworks` / `gnark`
  crates it depends on are large enough that "proving them correct"
  would be a multi-year research project. We rely on the audits and
  test suites those crates ship.
- Proof of the Casper Condor 1.5 host functions. Not our surface.
- Proof of the frontend or the SDK code paths. Not the kind of code
  that benefits from formal methods.

## Related

- `docs/roadmap/FORMAL_VERIFICATION.md` — Pack AV: per-spec invariants,
  model bounds, state counts, CI wiring, injection-test evidence.
- `specs/` — the four TLA+ specs + configs + `run-tlc.sh` runner.
- `.github/workflows/formal-verification.yml` — the CI job that
  model-checks every spec on every push/PR touching `specs/`.
- `docs/CONTRACT_INVARIANTS.md` — the informal statement of the
  contract-layer invariants that would eventually be modelled.
- `docs/JUDGE_GUIDE.md` — the assurance level actually shipped for
  the hackathon submission.
- `contracts/stake-slashing/src/main.rs` — the hardened redeploy
  that hand-audited checked-arithmetic invariants (I-4). Precedent
  for the kind of hand review a formal proof would replace.
