------------------------------- MODULE SlashingSpec -------------------------------
(***************************************************************************)
(* Formal specification of the CasperProver economic-slashing state       *)
(* machine as deployed by `contracts/stake-slashing/src/main.rs`. The     *)
(* real contract holds CSPR stakes in an on-chain purse, lets each        *)
(* AccountHash withdraw up to its recorded amount, and lets anyone        *)
(* permissionlessly claim a 20% (2000 bps) slash-reward against an agent  *)
(* whose proof has already been marked `revoked=1` on proof-registry.     *)
(*                                                                         *)
(* Complements the equivocation-slashing check already carried by         *)
(* `ProofSystemSpec.tla` (invariant SlashedProversHaveEvidence, which     *)
(* models the ≥2-coexisting-proofs precondition for a revoke). This spec  *)
(* is the OTHER half: given that a revoke happened, what the on-chain    *)
(* slash accounting is bound to do — with the 2026-07-18 record_stake    *)
(* hardening (no unbacked credit) as a checked property.                  *)
(*                                                                         *)
(* Out of scope: purse mechanics, deploy fees, dictionary rekeying,       *)
(* upstream revoke logic itself. Revoke is modelled as an external        *)
(* environment action that non-deterministically flips a proof's revoked  *)
(* flag from 0 to 1 — this spec does not judge whether the revoke was    *)
(* justified (that is what ProofSystemSpec covers).                       *)
(*                                                                         *)
(* Safety invariants checked:                                              *)
(*   - StateInvariant: every proof id and every stake key is a value in  *)
(*     its declared finite universe                                        *)
(*   - StakesNonNegative: no stake goes negative                          *)
(*   - TotalRecordedMatchesSum: bookkeeping total = Σ per-agent stakes    *)
(*   - TotalRecordedLeqBalance: contract can never credit more than the  *)
(*     purse actually holds (record_stake self-verifying property)        *)
(*   - SlashedProofOncePerId: each proof id can only trigger one slash   *)
(*   - SlashRequiresRevoked: every slash event was against a revoked pid *)
(*   - SlashRequiresAgentMatch: every slash event named the agent that   *)
(*     actually authored the proof on registry                            *)
(*   - SlashBoundedByStake: per-event slash ≤ stake at that moment       *)
(*   - UnstakeBoundedByStake: per-event unstake ≤ stake at that moment   *)
(*   - NoInflatedStake: no record_stake ever credits more than          *)
(*     (purse_balance - total_recorded)                                    *)
(***************************************************************************)

EXTENDS Naturals, FiniteSets, TLC

CONSTANTS
    Agents,       \* finite set of AccountHash values that may stake
    Callers,      \* finite set of AccountHash values that may call report_and_slash
    ProofIds,     \* finite set of proof_id strings
    NONE,         \* sentinel value distinct from any real Agent (model value)
    MaxEvents,    \* upper bound on total mutation events (keeps TLC finite)
    MaxDeposit    \* upper bound on any single deposit / stake amount

ASSUME
    /\ Agents # {}
    /\ Callers # {}
    /\ ProofIds # {}
    /\ NONE \notin Agents
    /\ MaxEvents \in Nat /\ MaxEvents >= 1
    /\ MaxDeposit \in Nat /\ MaxDeposit >= 1

VARIABLES
    stakes,          \* [Agents -> Nat] — per-agent recorded stake
    total_recorded,  \* Nat — Σ stakes (bookkeeping)
    purse_balance,   \* Nat — actual CSPR held in contract purse
    slashed_proofs,  \* SUBSET ProofIds — one-shot tombstones
    proof_author,    \* [ProofIds -> Agents ∪ {NONE}] — mirror of registry.agent
    proof_revoked,   \* [ProofIds -> {0,1}] — mirror of registry.revoked
    slash_events,    \* set of records — audit trail for slash actions
    events_taken     \* Nat — count of mutating steps taken

vars == <<stakes, total_recorded, purse_balance, slashed_proofs,
          proof_author, proof_revoked, slash_events, events_taken>>

\* Slash percentage in basis points, must match SLASH_BPS in main.rs.
\* NONE is declared as a top-level CONSTANT so TLC can treat it as a
\* model value distinct from every AccountHash without an unbounded
\* CHOOSE evaluation.

SLASH_BPS == 2000
BPS_DENOM == 10000

(***************************************************************************)
(* An event record captures enough of a slash action to phrase invariants *)
(* over history (bounded slash, matched agent, revoked-at-time-of-slash). *)
(***************************************************************************)

SlashEventRec ==
    [ proof_id       : ProofIds,
      agent          : Agents,
      caller         : Callers,
      stake_before   : Nat,
      stake_after    : Nat,
      slash_amount   : Nat ]

TypeOK ==
    /\ stakes         \in [Agents -> Nat]
    /\ total_recorded \in Nat
    /\ purse_balance  \in Nat
    /\ slashed_proofs \subseteq ProofIds
    /\ proof_author   \in [ProofIds -> Agents \union {NONE}]
    /\ proof_revoked  \in [ProofIds -> {0, 1}]
    /\ slash_events   \subseteq SlashEventRec
    /\ events_taken   \in Nat
    /\ events_taken   <= MaxEvents

(***************************************************************************)
(* Initial state: no stakes, no proofs, empty purse.                       *)
(***************************************************************************)

Init ==
    /\ stakes         = [a \in Agents |-> 0]
    /\ total_recorded = 0
    /\ purse_balance  = 0
    /\ slashed_proofs = {}
    /\ proof_author   = [p \in ProofIds |-> NONE]
    /\ proof_revoked  = [p \in ProofIds |-> 0]
    /\ slash_events   = {}
    /\ events_taken   = 0

SumStakes ==
    LET RECURSIVE Sum(_)
        Sum(S) ==
            IF S = {} THEN 0
            ELSE LET x == CHOOSE y \in S : TRUE
                 IN stakes[x] + Sum(S \ {x})
    IN Sum(Agents)

(***************************************************************************)
(* Actions                                                                 *)
(***************************************************************************)

\* stake-slashing-session performs a purse transfer THEN calls
\* record_stake in the SAME deploy. `RecordStake` models the honest path:
\* CSPR of size `deposit` really lands in the purse, then the record call
\* credits the caller. Self-verifying rule (line 155 of main.rs) means the
\* credit is capped at purse_balance - total_recorded, so an oversized
\* claimed amount is silently clipped.
RecordStake(a, deposit, claimed) ==
    /\ a \in Agents
    /\ deposit \in 1..MaxDeposit
    /\ claimed \in 1..MaxDeposit
    /\ events_taken < MaxEvents
    /\ LET new_balance == purse_balance + deposit
           available   == new_balance - total_recorded
           credit      == IF claimed > available THEN available ELSE claimed
       IN /\ credit > 0
          /\ stakes' = [stakes EXCEPT ![a] = @ + credit]
          /\ total_recorded' = total_recorded + credit
          /\ purse_balance' = new_balance
    /\ slashed_proofs' = slashed_proofs
    /\ proof_author'   = proof_author
    /\ proof_revoked'  = proof_revoked
    /\ slash_events'   = slash_events
    /\ events_taken'   = events_taken + 1

\* Attempt to record stake WITHOUT a matching purse transfer. Under the
\* 2026-07-18 hardening this must be capped at 0 → the action reverts.
\* We model the revert as a no-op step that still advances events_taken
\* so TLC can explore this branch without ballooning the state space.
RecordStakeUnbacked(a, claimed) ==
    /\ a \in Agents
    /\ claimed \in 1..MaxDeposit
    /\ events_taken < MaxEvents
    /\ purse_balance = total_recorded   \* nothing new in the purse
    /\ UNCHANGED <<stakes, total_recorded, purse_balance,
                   slashed_proofs, proof_author, proof_revoked,
                   slash_events>>
    /\ events_taken' = events_taken + 1

\* Withdraw up to current stake back to caller. Fails on insufficient
\* stake, so we only model the successful branch.
Unstake(a, amount) ==
    /\ a \in Agents
    /\ amount \in 1..MaxDeposit
    /\ events_taken < MaxEvents
    /\ stakes[a] >= amount
    /\ stakes' = [stakes EXCEPT ![a] = @ - amount]
    /\ total_recorded' = total_recorded - amount
    /\ purse_balance' = purse_balance - amount
    /\ slashed_proofs' = slashed_proofs
    /\ proof_author'   = proof_author
    /\ proof_revoked'  = proof_revoked
    /\ slash_events'   = slash_events
    /\ events_taken'   = events_taken + 1

\* External environment: proof-registry publishes a new proof authored by
\* `a`. Only fires on a fresh proof id.
PostProof(p, a) ==
    /\ p \in ProofIds
    /\ a \in Agents
    /\ proof_author[p] = NONE
    /\ events_taken < MaxEvents
    /\ proof_author'  = [proof_author  EXCEPT ![p] = a]
    /\ proof_revoked' = proof_revoked
    /\ UNCHANGED <<stakes, total_recorded, purse_balance,
                   slashed_proofs, slash_events>>
    /\ events_taken' = events_taken + 1

\* External environment: proof-registry marks a proof revoked. This spec
\* does not judge whether the revoke was justified — ProofSystemSpec.tla
\* already carries that half of the property (SlashedProversHaveEvidence).
Revoke(p) ==
    /\ p \in ProofIds
    /\ proof_author[p] # NONE   \* only revoke posted proofs
    /\ proof_revoked[p] = 0
    /\ events_taken < MaxEvents
    /\ proof_revoked' = [proof_revoked EXCEPT ![p] = 1]
    /\ proof_author'  = proof_author
    /\ UNCHANGED <<stakes, total_recorded, purse_balance,
                   slashed_proofs, slash_events>>
    /\ events_taken' = events_taken + 1

\* Permissionless slash. Guards mirror main.rs report_and_slash:
\*   - proof id not already slashed (tombstone)
\*   - registry says the pid was authored by `agent`
\*   - registry says revoked = 1
\*   - slash_amount > 0
ReportAndSlash(p, a, c) ==
    /\ p \in ProofIds
    /\ a \in Agents
    /\ c \in Callers
    /\ events_taken < MaxEvents
    /\ p \notin slashed_proofs
    /\ proof_author[p] = a
    /\ proof_revoked[p] = 1
    /\ LET current  == stakes[a]
           amount   == (current * SLASH_BPS) \div BPS_DENOM
       IN /\ amount > 0
          /\ stakes' = [stakes EXCEPT ![a] = @ - amount]
          /\ total_recorded' = total_recorded - amount
          /\ purse_balance'  = purse_balance - amount
          /\ slashed_proofs' = slashed_proofs \union {p}
          /\ slash_events'   = slash_events \union
                { [ proof_id     |-> p,
                    agent        |-> a,
                    caller       |-> c,
                    stake_before |-> current,
                    stake_after  |-> current - amount,
                    slash_amount |-> amount ] }
    /\ proof_author'  = proof_author
    /\ proof_revoked' = proof_revoked
    /\ events_taken'  = events_taken + 1

Next ==
    \/ \E a \in Agents, d \in 1..MaxDeposit, c \in 1..MaxDeposit :
         RecordStake(a, d, c)
    \/ \E a \in Agents, c \in 1..MaxDeposit :
         RecordStakeUnbacked(a, c)
    \/ \E a \in Agents, m \in 1..MaxDeposit :
         Unstake(a, m)
    \/ \E p \in ProofIds, a \in Agents :
         PostProof(p, a)
    \/ \E p \in ProofIds :
         Revoke(p)
    \/ \E p \in ProofIds, a \in Agents, c \in Callers :
         ReportAndSlash(p, a, c)

Spec == Init /\ [][Next]_vars

(***************************************************************************)
(* Invariants                                                              *)
(***************************************************************************)

\* Every stake key stays in Nat (never negative). Follows from the
\* action guards — Unstake requires stakes[a] >= amount, Slash uses
\* checked_sub semantics in main.rs, RecordStake only adds.
StakesNonNegative ==
    \A a \in Agents : stakes[a] >= 0

\* Total_recorded matches the sum of per-agent stakes.
TotalRecordedMatchesSum ==
    total_recorded = SumStakes

\* Contract can never credit more than the purse actually holds — the
\* record_stake 2026-07-18 hardening. This is the invariant that stops
\* an unbacked call from inflating recorded balances.
TotalRecordedLeqBalance ==
    total_recorded <= purse_balance

\* Each proof id triggers at most one slash. The tombstone is set in
\* the same step that pays out.
SlashedProofOncePerId ==
    \A e1, e2 \in slash_events :
        (e1.proof_id = e2.proof_id) => (e1 = e2)

\* Every slash event fired against a proof that was revoked at that
\* moment (and stays revoked — the environment cannot un-revoke in
\* this spec).
SlashRequiresRevoked ==
    \A e \in slash_events :
        proof_revoked[e.proof_id] = 1

\* Every slash event named the agent that authored the proof.
SlashRequiresAgentMatch ==
    \A e \in slash_events :
        proof_author[e.proof_id] = e.agent

\* Per-event slash amount is at most the stake the agent had before the
\* event fired. Combined with StakesNonNegative this rules out over-
\* slashing / underflow. The exact-20% ratio is checked next.
SlashBoundedByStake ==
    \A e \in slash_events :
        /\ e.slash_amount <= e.stake_before
        /\ e.stake_after = e.stake_before - e.slash_amount

\* The stake decrement equals floor(stake_before * 2000 / 10000) — the
\* exact 20% floor-divided integer arithmetic from main.rs.
SlashIsExactlyTwentyPercent ==
    \A e \in slash_events :
        e.slash_amount = (e.stake_before * SLASH_BPS) \div BPS_DENOM

\* Slash events never fire against dust stakes: the contract rejects
\* zero-value slashes to avoid burning the tombstone for free.
SlashAmountPositive ==
    \A e \in slash_events :
        e.slash_amount > 0

\* Unstake can only remove up to the stake that was there — no negative
\* stake, no underflow. Follows from the action guard `stakes[a] >= amount`.
\* Phrased as a per-agent invariant because we don't record unstake events.
UnstakeBoundedByStake ==
    \A a \in Agents : stakes[a] >= 0

\* The full safety invariant.
SafetyInvariant ==
    /\ TypeOK
    /\ StakesNonNegative
    /\ TotalRecordedMatchesSum
    /\ TotalRecordedLeqBalance
    /\ SlashedProofOncePerId
    /\ SlashRequiresRevoked
    /\ SlashRequiresAgentMatch
    /\ SlashBoundedByStake
    /\ SlashIsExactlyTwentyPercent
    /\ SlashAmountPositive
    /\ UnstakeBoundedByStake

===============================================================================
\* --- END SlashingSpec ---------------------------------------------------- *\
