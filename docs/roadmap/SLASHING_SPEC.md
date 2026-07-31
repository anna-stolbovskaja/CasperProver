# SlashingSpec.tla — economic-slashing formal spec

**Status:** shipped and green in `specs/run-tlc.sh` on 2026-07-31.
**Complements:** `ProofSystemSpec.tla` (equivocation half — when a revoke is
allowed to happen). This spec covers the OTHER half — given that a revoke
happened, what the on-chain slash accounting must do.

## Scope

Formal model of `contracts/stake-slashing/src/main.rs` — the deployed
CasperProver economic-slashing contract:

- Stake bookkeeping (`record_stake`, `unstake`)
- 2026-07-18 record_stake hardening (no unbacked credit)
- Permissionless slash (`report_and_slash`) at 20% (2000 bps)
- One-shot tombstone per proof id
- Proof-registry mirror (author, revoked flag) as an external environment

## What is NOT modelled

- Purse mechanics, deploy fees, dictionary rekeying (Casper internals)
- Upstream revoke logic itself — `ProofSystemSpec.tla` already carries
  `SlashedProversHaveEvidence`; this spec treats `Revoke` as a non-
  deterministic environment action
- Byte-level dictionary encoding

## Invariants (12 total, all PASS)

| Invariant | What it proves against main.rs |
|---|---|
| `TypeOK` | All state variables stay in their declared universes |
| `StakesNonNegative` | No stake goes negative (checked_sub semantics) |
| `TotalRecordedMatchesSum` | Bookkeeping `total_recorded` = Σ per-agent stakes at every step |
| `TotalRecordedLeqBalance` | `record_stake` can never credit more than the purse actually holds — the 2026-07-18 anti-inflation hardening |
| `SlashedProofOncePerId` | Tombstone `slashed_proofs[pid]` is one-shot — no double-slash |
| `SlashRequiresRevoked` | Every slash event fired against a `revoked=1` proof |
| `SlashRequiresAgentMatch` | Every slash event named the agent that actually authored the proof on registry |
| `SlashBoundedByStake` | `slash_amount ≤ stake_before`, `stake_after = stake_before - slash_amount` |
| `SlashIsExactlyTwentyPercent` | `slash_amount = floor(stake_before * 2000 / 10000)` — matches SLASH_BPS in main.rs exactly |
| `SlashAmountPositive` | `slash_amount > 0` — contract rejects dust-stake slashes (ERR_NO_SLASHABLE_STAKE) |
| `UnstakeBoundedByStake` | Per-agent stake never goes negative under unstake |
| `SafetyInvariant` | Conjunction of everything above |

## Model dimensions

- 2 agents (`a1`, `a2`), 2 callers (`c1`, `c2`), 2 proof ids (`p1`, `p2`)
- Deposit range `1..4`
- Event budget: 5 mutating steps (`MaxEvents = 5`)
- 6 action families: `RecordStake`, `RecordStakeUnbacked`, `Unstake`,
  `PostProof`, `Revoke`, `ReportAndSlash`

## Numbers

- **State count:** 206,489 states generated, 14,599 distinct
- **Depth:** 6
- **Runtime:** ~2s
- **CI budget:** well under the 1500s ceiling of
  `.github/workflows/formal-verification.yml`

## Injection test (validates the check does something)

To confirm `SlashIsExactlyTwentyPercent` is not tautological, the
`ReportAndSlash` amount was intentionally weakened from
`SLASH_BPS = 2000` to `3000` and TLC re-run. It immediately produced
a counter-example:

```
State 5: <ReportAndSlash(p1,a1,c1)>
  stakes = (a1 :> 3 @@ a2 :> 0)
  slash_events = {[proof_id |-> p1, agent |-> a1, caller |-> c1,
                   stake_before |-> 4, stake_after |-> 3, slash_amount |-> 1]}
```

`slash_amount = 1` is `floor(4 * 3000/10000) = 1` under the buggy 30%
rule, whereas the spec's `SlashIsExactlyTwentyPercent` expects
`floor(4 * 2000/10000) = 0`. The mismatch was caught in 5 steps and
53,480 states, confirming the invariant really binds to the exact
`SLASH_BPS` constant. The buggy spec was reverted before commit.

## Cross-refs

- `contracts/stake-slashing/src/main.rs` — the deployed contract this
  spec models (5 entrypoints: `get_purse`, `record_stake`, `unstake`,
  `report_and_slash`, `get_stake`)
- `contracts/stake-slashing/SLASH_EQUIVOCATION_DRAFT.md` — draft of the
  future `slash_equivocation` entrypoint (a spec follow-up will extend
  `SlashingSpec` to cover it once that entrypoint lands in main.rs)
- `docs/roadmap/FORMAL_VERIFICATION.md` — Pack AV: the CI runner and
  the pattern this spec follows

## Reproducing locally

```
bash specs/run-tlc.sh SlashingSpec
```

## Follow-ups for future specs

- Extend `SlashingSpec` with `slash_equivocation` once the draft
  entrypoint from `SLASH_EQUIVOCATION_DRAFT.md` lands in `main.rs`
- Model the interaction between `report_and_slash` and a live
  `proof-registry` state machine — currently the registry is an
  external environment; a combined product spec would prove the joint
  invariant "no slash without a valid registry revoke"
- Model liveness (fairness constraints) — currently only safety is
  checked; a fairness-augmented spec could prove "every revoked proof
  eventually gets slashed if any caller keeps trying"
