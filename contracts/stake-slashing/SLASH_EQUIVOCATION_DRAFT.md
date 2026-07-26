# `slash_equivocation` — Entrypoint Draft

**Status**: `DRAFT — contract change proposal`. Non-deployed. Distills
the minimum-viable equivocation-slashing extension to the existing
`stake-slashing` contract without breaking the deployed contract's
storage layout or ABI. **Does not add code to `main.rs` in this
commit.** Per the invariant "не редеплоим до аудита" in
`docs/MAINNET_LAUNCH_PLAN.md`, a live contract change ships only after
G2 audit sign-off.

Cross-refs:
- `contracts/stake-slashing/src/main.rs` — existing 5 entrypoints
  (`get_purse`, `record_stake`, `unstake`, `report_and_slash`,
  `get_stake`).
- `docs/REPUTATION_ECONOMICS.md` (AL) §5-6 — Challenger lifecycle
  and payoff matrix that this entrypoint implements a specific
  slice of.
- `docs/MAINNET_LAUNCH_PLAN.md` (AK) §3 G2 audit — gate that any
  live redeploy must pass.
- `docs/HASH_ALGORITHM_ANALYSIS.md` (AN) §2.2 — the receipt-hash
  domain separation this proposal relies on to make equivocation
  evidence unambiguous.

---

## 1. Framing — what "equivocation" means here

Equivocation in CasperProver's model: one agent produces two *distinct*
signed receipts for *the same input hash and model id* but with
*different output hashes*. Structurally this is the on-chain analogue
of a validator producing two conflicting blocks at the same height.

Detection today is **ledger-only** in the sense that anyone can look at
proof-registry state and observe two receipts under the same agent for
the same `(input_hash, model_id)` with distinct `output_hash` values.
Enforcement is missing: the deployed `stake-slashing` contract only
slashes via `report_and_slash`, which requires the proof to be
`revoked = 1` in the registry, and revocation is a *self-service* flag
today. That means an equivocating agent has no on-chain enforcement
lever unless *someone else* also holds the revoke key for their proof.

`slash_equivocation` closes that gap: **evidence of two conflicting
receipts is itself sufficient** to authorise a slash, without waiting
for a manual revoke.

## 2. Entrypoint signature (proposed)

```rust
pub extern "C" fn slash_equivocation() {
    let agent: AccountHash        = runtime::get_named_arg("agent");
    let proof_id_a: String        = runtime::get_named_arg("proof_id_a");
    let proof_id_b: String        = runtime::get_named_arg("proof_id_b");
    // Optional external oracle vouch: reserved for a future dispute
    // module. If non-empty, the caller must be authorised (see §6).
    let evidence_hash: Option<[u8; 32]> = runtime::get_named_arg_opt("evidence_hash");
    // ...
}
```

Every argument is content-bound:

- `agent` — the account being penalised.
- `proof_id_a`, `proof_id_b` — two distinct proof ids in the deployed
  proof-registry. Order does not matter and duplicates are rejected.
- `evidence_hash` — optional; ties the slash to an off-chain evidence
  bundle whose hash is anchored elsewhere. See §6.

## 3. Preconditions (all must hold; any violation reverts)

1. `proof_id_a ≠ proof_id_b` (revert `ERR_SAME_PROOF`).
2. `validate_proof_id(a)` and `validate_proof_id(b)` (reuse existing
   `validate_proof_id`; revert on length overflow).
3. Both proofs exist in proof-registry (existing `get_proof` call
   pattern).
4. Both proofs have the **same `agent`** (matches the `agent`
   argument; revert `ERR_AGENT_MISMATCH`).
5. Both proofs have the **same `input_hash`** and the **same
   `model_id`** (revert `ERR_INPUTS_DIFFER`).
6. Their **`output_hash` values differ** (revert `ERR_NOT_EQUIVOCATION`
   if equal — two identical receipts are duplicates, not equivocation).
7. Neither proof has been used in a prior equivocation slash
   (tombstone check against `SLASHED_DICT` under a new
   `equivocation_slashed_pairs` dictionary keyed by
   `min(a, b) || "|" || max(a, b)`).
8. The agent has a non-zero recorded stake in `STAKES_DICT`
   (revert `ERR_NO_SLASHABLE_STAKE`).

Cross-references to registry storage MUST use `get_proof_from_registry`
already defined in `main.rs` — no new cross-contract call surface.

## 4. Effects

On preconditions met:

1. Compute `slash_amount = current_stake * SLASH_BPS_EQUIVOCATION / 10000`
   using the same `checked_mul` guard already present in
   `report_and_slash`.
2. `SLASH_BPS_EQUIVOCATION` **must be strictly larger than the existing
   `SLASH_BPS`** for `report_and_slash` (proposal: `SLASH_BPS_EQUIVOCATION
   = 5000`, i.e. 50%, vs the existing 20%). Rationale: equivocation is a
   double-signing offence — the strictest failure mode — so it must
   carry the strictest penalty. Anything softer inverts the incentive.
3. Debit `slash_amount` from the agent's `STAKES_DICT` entry.
4. Debit `slash_amount` from `total_recorded` via
   `decrease_total_recorded`.
5. Store the tombstone under the composite key in
   `equivocation_slashed_pairs` (new dictionary; see §7 storage).
6. Pay `slash_amount` to `runtime::get_caller()` (the Challenger, in
   `REPUTATION_ECONOMICS.md` §4 terminology).
7. Emit no on-chain event beyond the state writes (the deployed
   contract does not emit events either; consistency is more
   important than adding a bespoke event here — event emission is
   itself a G2-audit item).

Explicit non-effects:
- **Does not** call `revoke_proof` on the registry. Revocation and
  equivocation-slashing are decoupled — a revoked-and-slashed proof
  under `report_and_slash` and an equivocation-slashed pair under
  `slash_equivocation` are independent state transitions.
- **Does not** modify the existing `SLASHED_DICT`.
- **Does not** affect the agent's ability to receive stake in the
  future via `record_stake`.

## 5. New constants

```rust
const ERR_SAME_PROOF: u16                 = 10;
const ERR_INPUTS_DIFFER: u16              = 11;
const ERR_NOT_EQUIVOCATION: u16           = 12;
const ERR_EQUIVOCATION_ALREADY_SLASHED: u16 = 13;

const SLASH_BPS_EQUIVOCATION: u64         = 5000; // 50%
const EQUIVOCATION_SLASHED_PAIRS: &str    = "equivocation_slashed_pairs";
```

## 6. Optional `evidence_hash` — future extension, NOT MVP

If `evidence_hash: Some(h)` is provided, the entrypoint additionally
requires that `runtime::get_caller()` be an authorised **dispute
module** — a designated `AccountHash` stored in a new `NamedKeys`
entry `dispute_module_hash`. Setting that named key is an admin
operation via the deploy `call` fn; if it is unset, `evidence_hash`
must be `None` or the entrypoint reverts.

Rationale: for the MVP path (§2 signature with `evidence_hash =
None`), anyone can call `slash_equivocation` because the equivocation
is *self-witnessing* — the two conflicting registry entries are the
whole evidence and are publicly readable. The optional `evidence_hash`
lane is reserved for future dispute-resolver integrations that anchor
richer off-chain evidence bundles.

For hackathon-scope: **ship without `evidence_hash`**. Its schema is
documented here so a future release can bolt it on without a MAJOR
API bump.

## 7. Storage impact (backward-compatible)

- One new dictionary: `equivocation_slashed_pairs`.
- No changes to existing `STAKES_DICT`, `SLASHED_DICT`,
  `CONTRACT_PURSE`, `KEY_TOTAL_RECORDED`.
- One new `EntityEntryPoint` registration inside the existing `call`
  fn's `EntryPoints` builder.
- No changes to the deployed contract's existing entrypoints'
  signatures, semantics, or return types.

This means the redeploy is a **new contract hash**, not an in-place
mutation. The Go SDK and MCP wrappers must be aware of both contract
hashes for the compatibility window per
`docs/SDK_VERSIONING.md`.

## 8. Test plan (before any deploy)

Unit tests (Rust, `cargo casper test`):
1. Happy path — two conflicting proofs, distinct output_hash → agent
   loses 50% stake, Challenger receives 50%.
2. Same proof twice → `ERR_SAME_PROOF`.
3. Different agents on the two proofs → `ERR_AGENT_MISMATCH`.
4. Different `input_hash` → `ERR_INPUTS_DIFFER`.
5. Different `model_id` → `ERR_INPUTS_DIFFER`.
6. Same `input_hash`, same `model_id`, same `output_hash` (duplicate
   receipts, not equivocation) → `ERR_NOT_EQUIVOCATION`.
7. Zero stake for the agent → `ERR_NO_SLASHABLE_STAKE`.
8. Replay of the same pair → `ERR_EQUIVOCATION_ALREADY_SLASHED`.
9. Ordering: `(a, b)` and `(b, a)` are stored under the same
   composite key (min/max canonicalisation) — replay-through-reorder
   fails.
10. Interaction with `report_and_slash`: an equivocation-slashed pair
    that later gets revoked still allows `report_and_slash` to run
    (independent state transitions).
11. Arithmetic guard: a max-U512 stake still produces a valid
    `checked_mul(SLASH_BPS_EQUIVOCATION)` without overflow.
12. Cross-contract failure: registry returns a proof missing an
    output_hash — revert cleanly (no half-state).

Integration tests (session code):
- End-to-end: two conflicting proofs anchored via ordinary flow, then
  `slash_equivocation` called in the same deploy → expected balances
  observed.

## 9. Security review checklist (must pass at G2)

- Cross-contract call is a **read** only (no state mutation on
  registry). ✅ by construction.
- Composite key canonicalisation prevents order-swap replay. ✅ §7.
- Permissionless entrypoint cannot force revocation elsewhere. ✅ §4.
- Slash amount is bounded to current stake via `checked_sub`. ✅
  (reuse of existing pattern).
- Payout to `get_caller()` is via `system::transfer_from_purse_to_account`
  as in the existing `report_and_slash`. ✅
- Tombstone is checked BEFORE any state mutation. ✅ §3-7.
- No admin-only functions added in the MVP path. ✅ §6.

## 10. What this document does NOT do

- It does not modify `contracts/stake-slashing/src/main.rs`.
- It does not compile or deploy a new WASM.
- It does not authorise a redeploy.
- It does not name a specific auditor.
- It does not commit to a schedule.
- It does not add or change any Go SDK method.
- It does not weaken the invariant "не редеплоим до аудита".

The single deliverable is a spec that a future contract change can be
audited against without any live-contract risk today.

## 11. Reference to reputation model

`docs/REPUTATION_ECONOMICS.md` §5-6 (AL) describes a full Challenger
lifecycle with bond, response window, adjudicator quorum, and appeal.
`slash_equivocation` is the *narrowest* slice of that lifecycle:
equivocation is self-witnessing, so it needs no adjudicator and no
Challenger bond. Broader challenge types (correctness disputes,
availability disputes) still need the full lifecycle from AL and are
NOT covered here.

If and when the full lifecycle is implemented, this entrypoint stays
as-is (the self-witnessing case) and the general case gets its own
entrypoint `slash_via_adjudication` or similar. Do not overload
`slash_equivocation` to carry the general case.

---

*This is a contract change proposal. It ships no code and authorises
no redeploy. Its only purpose is to make the equivocation-slashing
extension auditable against a written spec before any live contract
change.*
