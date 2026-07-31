# ML attestation harness — status & disclosure

**TL;DR.** The repo now ships an ML-attestation **harness** — an `Attestor` interface and an HTTP surface (`POST /v1/ml/attest`, `POST /v1/ml/verify-attest`) whose contract matches what a real ZK-ML implementation would eventually expose. The default implementation is a SHA-256 chain stand-in labelled `ml-attest-v0`. **It is not a cryptographic proof of ML inference.**

The label `zkml-fixed-v0` is reserved for a future named-circuit ZK-ML implementation. The harness deliberately refuses to emit or verify that label until every gating condition in [`docs/ZKML_HONEST_VERDICT.md`](../ZKML_HONEST_VERDICT.md) is met.

This document is the durable disclosure. It sits alongside — and follows the exact discipline of — [`NOVA_HARNESS.md`](./NOVA_HARNESS.md).

## Why this file exists

`docs/ZKML_HONEST_VERDICT.md` is the decision record that says: *every ML-inference claim in CasperProver is `SIMULATION` until four conditions all hold*. Those conditions cover named circuit + published hashes, third-party audit sign-off (G2), per-inference cost under Challenger economics, and a receipt schema extension.

The verdict permits one thing today: an **attestation** of `(model_id, weights_digest, inputs_digest, outputs_digest)` — a signed statement about what was named and observed, *not* a proof that the named model was executed. This harness ships exactly that surface, under an unambiguous label, so SDKs and downstream callers can stabilise the wire contract without laundering `SIMULATION` into `REAL`.

## What ships

- `engine/internal/mlattest/harness.go` — `Attestor` interface with `Attest(in)` and `Verify(in, att)`. Default implementation `HashMLAttestor`.
- `engine/internal/mlattest/harness_test.go` — deterministic attest, round-trip verify, tamper detection (commit / inputs / model_id), reserved-scheme refusal, short-digest and empty-`model_id` rejection.
- `engine/internal/api/ml_attest_handlers.go` — `POST /v1/ml/attest`, `POST /v1/ml/verify-attest`.
- `engine/internal/api/ml_attest_test.go` — HTTP tests including the **laundering guard**: an attestation with a well-formed commit but relabelled as `zkml-fixed-v0` MUST NOT verify.
- Scoped as `ml:write` / `ml:read` (see `engine/internal/api/scopes.go`).
- Response envelope carries an explicit `scheme` field and a `disclosure` string.

## What the harness actually does

For each attestation:

```
step_a  = SHA256( model_id || weights_digest )
step_b  = SHA256( inputs_digest || outputs_digest )
commit  = SHA256( domain_seed || step_a || step_b )
```

with the initial `domain_seed = SHA256("ml-attest-v0")` so future schemes do not collide with this one on identical inputs. The envelope carries the four hex digests plus the final commit; verification recomputes the chain from the envelope's own fields.

This is a genuine, deterministic hash commitment. Tampering with any of the four digests, with the model id, or with the commit itself gets caught by `Verify`.

## What the harness does NOT do

A cryptographic proof of ML inference (Groth16/PLONK circuit of a compiled model, STARK/FRI, zkVM, lookup+PLONK, recursion) proves that a specific named circuit `C_model` — deterministically derived from the weights — was satisfied on the given inputs. The harness in this repo does none of that. Specifically:

- **No circuit.** There is no compiled arithmetic circuit for any model.
- **No witness satisfiability claim.** The output digest is taken as given; nothing binds it to the actual execution of the model.
- **No zero-knowledge property.** The inputs and outputs are committed as public digests; a caller who wants privacy over inputs or outputs must produce those digests themselves and manage disclosure separately.
- **No soundness reduction to a hardness assumption over ML inference.** The commit's soundness reduces to SHA-256 collision resistance — which says nothing about whether the named model actually produced the output.

## Scheme labels

The response envelope carries `scheme` deliberately. Downstream code checking for a real ZK-ML proof MUST match on a specific label — not on the presence of the `Attestation` type.

| Label            | Semantics                                                                                          | Ships? |
|------------------|-----------------------------------------------------------------------------------------------------|--------|
| `ml-attest-v0`   | Hash-chain attestation of `(model_id, weights, inputs, outputs)`. Not a proof of inference.         | ✅ this commit |
| `zkml-fixed-v0`  | Reserved for a future named-circuit ZK-ML implementation. `Verify()` refuses this label today.     | ❌ not implemented |

`VerifyAll` refuses to verify an attestation labelled with an unimplemented scheme (`TestVerify_RejectsReservedScheme` / `TestMLAttest_VerifyRejectsRelabelledAsZKML` both cover the refusal path). Sending back an `ml-attest-v0` commit under a `zkml-fixed-v0` label MUST fail loudly. This is the laundering gate.

## What a real implementation would need

To honestly emit `zkml-fixed-v0` we must ship all four conditions from `docs/ZKML_HONEST_VERDICT.md`:

1. Named model, named circuit, published hashes (circuit hash, verifying-key hash, weights hash, toolchain version) — extended into the receipt schema.
2. Third-party audit sign-off on both the circuit and the underlying IOP/lookup argument. Reserved for G2 in `docs/MAINNET_LAUNCH_PLAN.md`.
3. Per-inference proving cost inside the Challenger economics ceiling.
4. Receipt schema extension (breaking) to carry (1). Scheduled at G2.

Skipping any one of these turns a relabel into laundered `SIMULATION`. That outcome is explicitly rejected by the honest verdict.

## What the harness IS good for

- **Contract stabilisation.** SDKs and downstream callers can be built against the final HTTP shape today without waiting on the ZK-ML decision at G2.
- **Model-identity bookkeeping.** The commit gives a compact fingerprint over `(model_id, weights, inputs, outputs)` that a receipt or an anchor contract can already accept.
- **Regression fence.** Tampering with any envelope field or with the commit breaks verification — enough to catch integrity bugs in the model-serving pipeline, distinct from correctness of the underlying inference.

Anything beyond that requires the real scheme, and the real scheme requires G2.

## Cross-references

- [`docs/ZKML_HONEST_VERDICT.md`](../ZKML_HONEST_VERDICT.md) — durable decision record.
- [`docs/ZKML_RESEARCH_SPIKE.md`](../ZKML_RESEARCH_SPIKE.md) — the survey of prover families that landed on the honest verdict.
- [`docs/roadmap/NOVA_HARNESS.md`](./NOVA_HARNESS.md) — parallel disclosure discipline for folding-scheme aggregation.
- [`docs/roadmap/MERKLE_ATTESTOR.md`](./MERKLE_ATTESTOR.md) — sibling harness that replaces the flat weights digest with a Merkle root over ordered chunks, adding selective disclosure for auditor sampling. Distinct scheme label `ml-attest-merkle-v0`; the same four gating conditions still apply before any relabel to REAL.
- [`docs/MAINNET_LAUNCH_PLAN.md`](../MAINNET_LAUNCH_PLAN.md) — G2 is the third-party audit gate that must clear before any relabel to `REAL`.
