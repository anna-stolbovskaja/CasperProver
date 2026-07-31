# Merkle ML Attestor — Selective-Disclosure Harness

**Status:** SIMULATION / attestation-only.
**Scheme label:** `ml-attest-merkle-v0` (distinct from `ml-attest-v0`).
**Related:**
- `docs/ZKML_HONEST_VERDICT.md` — the durable decision record; the four gating
  conditions before ANY relabel to REAL.
- `docs/roadmap/ML_ATTESTATION_HARNESS.md` — sibling harness (`ml-attest-v0`,
  hash-only over whole weights blob).
- `docs/roadmap/DETERMINISTIC_REPLAY_HARNESS.md` — auditor-side replay.

## What this adds

`HashMLAttestor` commits to a **single** SHA-256 of the whole weights blob:
the auditor either learns every weight (prover discloses the blob), or learns
nothing (prover discloses only the hex digest). There is no middle ground.

`MerkleMLAttestor` replaces the flat digest with a **Merkle root** over an
ordered list of weight chunks. Given the envelope, the prover can then answer
an auditor's opening request for **one** chunk index by disclosing:

- the raw chunk bytes,
- the sibling hash at each tree level,
- the path bit at each level.

The auditor recomputes the root from that opening alone and compares against
the envelope root. **No other chunk is disclosed.** This is the property a
flat-hash scheme fundamentally cannot provide.

## What this is NOT

Same discipline as `ml-attest-v0`. This is **not** a cryptographic proof of ML
inference. The envelope attests to `(model_id, weights_root, inputs_digest,
outputs_digest)` — it does not attest that the named model was executed on the
named inputs. All four gating conditions in `ZKML_HONEST_VERDICT.md` remain in
force. The `zkml-fixed-v0` label is still reserved and refused by verify.

## Scheme

```
LEAF_TAG = "ml-attest-merkle-v0:leaf"
NODE_TAG = "ml-attest-merkle-v0:node"
SEED_TAG = "ml-attest-merkle-v0:commit-seed"

leaf_i    = SHA256( LEAF_TAG || chunk_i )
node(L,R) = SHA256( NODE_TAG || L || R )
root      = fold nodes over levels; odd levels pad by duplicating last node

step_a  = SHA256( model_id || root )
step_b  = SHA256( inputs_digest || outputs_digest )
seed    = SHA256( SEED_TAG )
commit  = SHA256( seed || step_a || step_b )
```

Domain tags (`LEAF_TAG` distinct from `NODE_TAG` distinct from `SEED_TAG`)
prevent second-preimage attacks between the three layers and prevent
cross-scheme collision with `ml-attest-v0` even when the caller supplies
identical logical inputs.

## Opening protocol

The **opening** for leaf `i` carries `(leaf_index, leaf_count, chunk_bytes,
siblings[], is_right_path[])`. The verifier:

1. Confirms the envelope's scheme label is `ml-attest-merkle-v0` (else refuse).
2. Confirms `leaf_count` matches the envelope and `leaf_index ∈ [0, leaf_count)`.
3. Confirms `is_right_path` is consistent with the low bits of `leaf_index` —
   this **cryptographically binds the claimed index to the tree path**;
   without it, a valid opening for leaf `i` would silently verify at any
   claimed index that shares the same tree path.
4. Recomputes `leaf = SHA256(LEAF_TAG || chunk_bytes)` and climbs the tree,
   folding with each sibling under `NODE_TAG` on the appropriate side.
5. Compares against `envelope.weights_root_hex`.

## Envelope fields

| Field                | Meaning                                              |
| -------------------- | ---------------------------------------------------- |
| `scheme`             | Always `ml-attest-merkle-v0`.                        |
| `model_id`           | Opaque caller-supplied identifier.                   |
| `weights_root_hex`   | Merkle root over ordered weight chunks.              |
| `leaf_count`         | Number of leaves (for range-checking openings).      |
| `inputs_digest_hex`  | SHA-256 of input tensor.                             |
| `outputs_digest_hex` | SHA-256 of output tensor.                            |
| `commit_hex`         | Envelope commitment binding all fields together.     |
| `disclosure`         | Constant honesty clause (same text as `ml-attest-v0`). |

## Test surface

`engine/internal/mlattest/merkle_test.go` (13 tests) covers:

- Determinism of attest (same input → same commit and root).
- Distinct scheme label from `ml-attest-v0`; commits do not collide even with
  matching logical inputs.
- Cross-scheme rejection: `HashMLAttestor.Verify` refuses a merkle-scheme
  envelope; `MerkleMLAttestor.VerifyEnvelope` refuses `zkml-fixed-v0`.
- Every leaf opens and verifies against the envelope root (odd-leaf-count
  padding path exercised).
- Tampered chunk → root mismatch, refuse.
- Tampered sibling → root mismatch, refuse.
- **Wrong claimed index → refuse.** This is the injection-tested invariant
  that binds `leaf_index` to the tree path (see next section).
- Out-of-range index → refuse.
- Envelope round-trip verifies; tampered commit → refuse.
- Empty chunk or empty chunk list → refuse.
- Pinned KAT — round-trips every leaf to lock the scheme against silent drift.

## Injection test — proof the index-binding invariant is not tautological

While writing the tests, `TestMerkleOpen_WrongIndexRejected` initially failed:
a valid opening for leaf 1 verified under `LeafIndex = 2` because the sibling
path happened to match. That was a real design gap — `LeafIndex` was
metadata, not cryptographically bound. Fixed by deriving the expected
`is_right_path` bit sequence from `LeafIndex` and refusing a mismatch.

Verified the fix with an injection run: comment out the new binding-check
loop, re-run `TestMerkleOpen_WrongIndexRejected` → FAIL. Restore → PASS. The
invariant binds.

## What a real ZK-ML implementation would still need

Same as `ml-attest-v0`: the four gating conditions in `ZKML_HONEST_VERDICT.md`
must all hold before any relabel to REAL. Merkle openings give the auditor a
selective-disclosure channel over the model weights — they say nothing about
whether the named model was actually executed on the named inputs.

## What the Merkle harness is good for

- **Auditor sampling.** Auditor requests `k` random chunk indices, prover
  discloses `k` chunks + their openings. Auditor confirms the sampled chunks
  are consistent with the committed model root, without seeing the full
  weights blob. Not a proof of inference — a stronger check on model
  identity than a flat digest.
- **Layer-level disclosure.** Chunks can align with layer boundaries; the
  prover can then disclose (e.g.) the attention weights but not the MLP
  weights, or vice versa.
- **Regression fence.** The pinned scheme means any change to leaf/node tags,
  seed derivation, or padding rule breaks the KAT test — no silent scheme
  drift.

## Files

- `engine/internal/mlattest/merkle.go` — the attestor.
- `engine/internal/mlattest/merkle_test.go` — 13-test suite incl. pinned KAT.
- `docs/roadmap/MERKLE_ATTESTOR.md` — this document.
