---- MODULE ProofSystemSpec ----
\* Formal specification of CasperProver proof system invariants.
\* Status: initial stub — full verification planned for Phase 4.

EXTENDS Naturals, Sequences

CONSTANTS MaxProofs, MaxChainDepth

VARIABLES proofs, models, chains, aggregations

TypeOK ==
    /\ \A p \in proofs: p.model_hash \in DOMAIN models
    /\ \A c \in chains: c.depth >= 1
    /\ \A c \in chains: c.depth <= MaxChainDepth

\* Invariant: verified proof is immutable
ProofImmutability == \A p \in proofs:
    p.status = "verified" => UNCHANGED p

\* Invariant: every proof references a registered model
ModelBinding == \A p \in proofs:
    p.model_hash \in DOMAIN models

\* Invariant: aggregation root = merkle root of children
AggregationCorrectness == \A a \in aggregations:
    a.root_hash = MerkleRoot(a.child_hashes)

\* Invariant: proof chain continuity (no gaps)
ChainContinuity == \A c \in chains:
    \A i \in 1..(Len(c.steps) - 1):
        c.steps[i + 1].input_hash = c.steps[i].output_hash

\* Invariant: PQ signatures always verify
PQSignatureValidity == \A p \in proofs:
    p.pq_signature /= <<>> =>
        PQVerify(p.pq_signature, p.hash, p.prover_pubkey) = TRUE

\* Invariant: KYC whitelist modifications require admin
KYCAdminOnly == \A op \in whitelist_ops:
    op.type = "remove" => op.sender = admin

SafetyInvariant ==
    ProofImmutability /\ ModelBinding /\ ChainContinuity

====
