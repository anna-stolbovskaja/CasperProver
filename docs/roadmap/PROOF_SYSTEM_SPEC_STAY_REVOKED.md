# ProofSystemSpec — SlashedProversStayRevoked (v0)

*Extension to `specs/ProofSystemSpec.tla` — one new invariant plus history
variable to close a monotonicity gap.*

## What & why

`ProofSystemSpec` already tracks `slashedSet` and guards `SubmitProof`,
`OpenChain`, `ExtendChain` on `pr \notin slashedSet` — so a slashed prover
cannot submit new work. The existing invariant `SlashedProversHaveEvidence`
asserts that every currently-slashed prover has an equivocating pair on
record.

Gap: **nothing in the spec forbids a future refactor from introducing an
`Unslash` / rehabilitation action.** TLC evaluates invariants on states, not
on transitions — the "slashedSet only grows" property was implicit,
enforced only by "no existing action decreases it". A refactor that adds
`Unslash` would silently pass every existing invariant (the state where
`slashedSet` shrank is a legal state) and only fail if some downstream
liveness property caught it.

Fix: add a **history variable** `everSlashed` that accumulates every
prover ever slashed, and an invariant that binds it to the live set:

```
SlashedProversStayRevoked == slashedSet = everSlashed
```

Every action that keeps `slashedSet` unchanged also keeps `everSlashed`
unchanged. Only `Slash` appends to both. Any future action that removes
from `slashedSet` without also removing from `everSlashed` (impossible —
history is monotone) fails the invariant immediately.

## Contract mapping

Mirrors the on-chain slashing contract (`contracts/proofs-of-stake/src/main.rs`):

- `slashing_report` is one-shot per (agent, pid) — tombstoned in
  `slashed_proofs` map, cannot be replayed. That map is append-only; there
  is no `unslash_report` endpoint.
- `PoS::report_and_slash` decreases `stakes[agent]` and marks the pid
  tombstone but does not maintain a per-agent "slashed" flag. The
  spec-level `slashedSet` is the abstract lift.
- `SlashedProversStayRevoked` formalises "the tombstone map is
  append-only" at the spec level.

## Model dimensions

Reuses existing config (`specs/ProofSystemSpec.cfg`):

| Constant        | Value |
|-----------------|-------|
| Provers         | {p1, p2} |
| Models          | {m1, m2} |
| MaxProofs       | 3 |
| MaxChainDepth   | 2 |
| ChallengeWindow | 2 |

## Validation

- `bash specs/run-tlc.sh ProofSystemSpec` → PASS.
- **State count:** 12,642,985 generated, 6,153,405 distinct, depth 12.
- **Runtime:** ~2 min 35 s single-run (well under the 900 s `TLC_TIMEOUT`
  and 1500 s CI ceiling).
- **All 12 invariants hold** — the 11 pre-existing plus the new
  `SlashedProversStayRevoked`.

## Injection test

Confirms the invariant actually binds, in the same style as SlashingSpec.

Weakened `Slash` action:

```tla
Slash ==
    /\ \E p \in proofs, q \in proofs : Equivocates(p, q)
       /\ p.prover \notin slashedSet
       /\ slashedSet' = slashedSet \union {p.prover}
       /\ everSlashed' = everSlashed  \* INJECT: don't record in history
    /\ ...
```

TLC caught it in **4 steps / 924 states / 1 s**:

```
State 4: <Slash>
/\ slashedSet = {p1}
/\ everSlashed = {}
```

`SlashedProversStayRevoked` failed as expected. Restored, PASS again.

## Non-goals

- Not a rehabilitation model. The spec never claims a slashed prover may
  ever be un-slashed — this invariant enforces exactly that.
- Not a new action. This adds a state predicate + one history variable;
  no observable behavioural change to the spec's actions.
- Not a temporal / liveness property. This is a safety invariant checked
  on every state.

## Related

- `docs/roadmap/FORMAL_VERIFICATION.md` — pack AV runbook.
- `docs/roadmap/SLASHING_SPEC.md` — companion economic slashing spec.
- `contracts/proofs-of-stake/src/main.rs` — the on-chain contract this
  spec is the abstract lift of.
