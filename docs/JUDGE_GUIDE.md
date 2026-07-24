# Judge verification guide

This checklist is the shortest reproducible path through CasperProver. It separates **real cryptography**, **on-chain evidence**, and **simulation** so claims can be verified rather than trusted.

## 0. Landing entry points

The landing page (`/`) exposes three audience-scoped entry points, each linking to an existing route in this repo:

| Audience | Entry | Route |
|---|---|---|
| Users | **Try the product** — guided demo in the Playground | `/lab/playground` |
| Developers | **API, SDK & MCP** — docs hub | `/docs/api` (siblings: `/docs/sdk`, `/docs/mcp`) |
| Evaluators / investors | **Proof & architecture** — deployed contracts and source | `/lab/contracts` |

The secondary row also links to the full Lab (`/lab`) and the GitHub source. No entry point navigates outside routes that resolve in `App.tsx`.

## 1. One-command evidence

```bash
python3 scripts/judge_demo.py
```

Default mode is read-only: it queries all four Casper testnet contracts, API health/proofs, and the frontend. For the real Groth16 round-trip, obtain the temporary judge key from the submission notes and run:

```bash
CP_JUDGE_API_KEY='provided-out-of-band' python3 scripts/judge_demo.py
```

The key is never embedded in the script, URL, repository, or shell history examples. Revoke it after judging.

## 2. Functional walkthrough

| # | Action | Expected evidence | Boundary |
|---|---|---|---|
| 1 | Open `https://casperprover.xyz/lab/contracts` | Four deployed contracts, hashes and explorer links | **ON-CHAIN** |
| 2 | Run `python3 scripts/judge_demo.py` | Contract queries + API/frontend checks pass | **ON-CHAIN / LIVE SERVICE** |
| 3 | Open `/lab/zk-proofs`, prove preimage `42`, then verify | Valid gnark BN254/MiMC proof | **REAL CRYPTO, OFF-CHAIN** |
| 4 | Change one byte of `proof_hex` and verify | Verification fails | **NEGATIVE SECURITY TEST** |
| 5 | Open `/lab/pq-crypto`, hybrid-sign a message, then verify | Ed25519 + ML-DSA-65 both valid | **REAL CRYPTO, OFF-CHAIN** |
| 6 | Change the message and verify again | Signature verification fails | **NEGATIVE SECURITY TEST** |
| 7 | Open `/lab/proofs`, create and verify a proof | Deterministic hash/Merkle record; optional wallet anchoring | **ENGINE; ON-CHAIN ONLY WHEN TX HASH EXISTS** |
| 8 | Open `/lab/playground` | Every API operation shows request/response and errors | **DEVELOPER UX** |
| 9 | Run `./verify.sh` | Existing independent smoke suite passes | **REGRESSION EVIDENCE** |

## 3. Claim boundary

Every judge-facing badge on the site resolves through a single `TrustStatus` component with exactly four canonical labels. If you see a badge, its value is one of these:

| Label | Where it appears | What it means |
|---|---|---|
| **Real cryptography** | `/lab/zk-proofs` — Groth16 Real Prove, Groth16 Real Verify | Real primitive in the CasperProver engine (gnark BN254 Groth16, ML-DSA-65, Ed25519). Executes off-chain. |
| **On-chain** | `/lab/contracts` — the four deployed contract cards (`proof-registry`, `verifier-gate`, `defi-mock`, `stake-slashing`) | Deployed on Casper testnet with a real contract hash; click through to the CSPR.live explorer. |
| **Simulation** | `/lab/zk-proofs` — Groth16 Conceptual (Hash-Based) | Illustrative hash-based flow, not real pairing verification. |
| **Built, not deployed** | `/lab/contracts` — the three source-only cards (`proof-of-inference`, `model-registry`, `proof-aggregation`) | Rust source is in `contracts/` and reviewable on GitHub, but the contract has no on-chain address. |

Badges are strictly restricted to the four values above. "Verified", "live", "production-ready", "audited" are never used. The mapping between a UI element and its label is driven by an existing structured source — either the deployed contract table (`deployed: true|false`) or the presence of the real Groth16 endpoint — not by copy.

Underlying capability summary (unchanged):

| Category | What CasperProver actually does |
|---|---|
| **REAL CRYPTO** | gnark/BN254 Groth16 for a MiMC preimage circuit; ML-DSA-65 and Ed25519 hybrid signatures; Lamport OTS education path |
| **ON-CHAIN** | Casper testnet contracts store/validate proof metadata, hashes, access state and stake/slashing state |
| **SIMULATION** | Legacy conceptual Groth16/STARK-style hash flows, kept for comparison and explicitly labeled |

CasperProver does **not** claim a Casper-native pairing verifier or a ZK proof of arbitrary ML inference. The current real Groth16 circuit proves knowledge of a MiMC preimage.

## 4. Failure behavior

A failed check exits non-zero and prints a bounded HTTP/network error without secrets. The script never retries mutating calls, never invents a transaction hash, and skips write-based crypto checks unless a judge key is explicitly supplied.
