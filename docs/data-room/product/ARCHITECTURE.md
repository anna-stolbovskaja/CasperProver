# Architecture (investor pointer)

The engineering source of truth for the CasperProver architecture is
`docs/ARCHITECTURE.md`. This page exists so an investor can navigate
without needing to open a code-oriented doc.

## In one paragraph

CasperProver takes a decision — an AI agent's `(input, output,
model_id)` tuple — hashes it into a Merkle tree, wraps the Merkle root
in a BN254 Groth16 proof, signs the whole thing with a hybrid
(Ed25519 + ML-DSA-65 or SPHINCS+) key, and anchors the commitment to
Casper Network. Any later party — auditor, regulator, counter-party —
can verify the anchor in milliseconds without re-running the model.

## Layers

1. **Content** — the raw prompt / output / model, kept on the caller's
   side. Never enters CP.
2. **Merkle layer** — SHA-256 over receipt leaves. Domain-separated per
   `docs/MERKLE_SCHEME.md`.
3. **ZK layer** — BN254 Groth16 over the Merkle root. Real gnark
   circuit. Simulation paths are labelled `[sim]` at the endpoint and
   deprecated per `docs/PQ_HONESTY.md`.
4. **PQ signature layer** — Ed25519 + ML-DSA-65 or SPHINCS+, both from
   the NIST PQC standard. Not marketing PQ.
5. **Anchor layer** — Merkle root committed to Casper Network via
   `proof-registry` and, for aggregated batches, `proof-aggregation`.
6. **Governance layer** — `governance` contract with 48h timelock and
   2-of-3 guardian recovery. See `docs/roadmap/GOVERNANCE.md`.
7. **Verification layer** — `verifier-gate` for Merkle inclusion,
   `zk-verifier` for on-chain Groth16 verdicts, `stake-slashing` for
   economic penalties.

## Where investors should look first

- One-line pitch: `README.md` §tagline.
- 8-criteria judge map: `docs/JUDGE_GUIDE.md`.
- Deployed contracts index: `docs/TX_MANIFEST.md`.
- Honesty checklist: `docs/PQ_HONESTY.md`, `docs/KNOWN_LIMITATIONS.md`.
- Security audit: `docs/SECURITY_AUDIT.md`.

## What the architecture does *not* do (today)

- Prove the model's internal computation. Roadmap under
  `docs/roadmap/90-180-DAY.md` (real ZK-ML circuit).
- Anchor on chains other than Casper. Roadmap: multi-chain adapters.
- BYOK / tenant threshold signing. Roadmap:
  `docs/roadmap/KEY_MANAGEMENT.md` non-goals.

Honesty about non-goals is a feature; see `docs/PQ_HONESTY.md`.
