# Judge verification guide

This checklist is the shortest reproducible path through CasperProver. It separates **real cryptography**, **on-chain evidence**, and **simulation** so claims can be verified rather than trusted.

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

| Label | What CasperProver actually does | Endpoints |
|---|---|---|
| **REAL CRYPTO** — PRIMARY | gnark/BN254 Groth16 for a MiMC preimage circuit (real R1CS + trusted setup + pairing verification); ML-DSA-65 and Ed25519 hybrid signatures; Lamport OTS education path | `POST /zk/groth16-real/prove`, `POST /zk/groth16-real/verify`, `/pq/*` |
| **ON-CHAIN** | Casper testnet contracts store/validate proof metadata, hashes, access state and stake/slashing state | Casper testnet, see `docs/DEPLOYMENTS.md` |
| **`[sim]` SIMULATION** — DEPRECATED | Legacy conceptual Groth16 hash flow, kept for comparison. Responses carry `simulation:true, deprecated:true, use:"/zk/groth16-real/verify"` plus `Warning`/`Deprecation`/`Sunset` headers | `POST /zk/verify-groth16-sim` (alias `/zk/verify-groth16`), `POST /zk/batch-verify-sim` (alias `/zk/batch-verify`) |

CasperProver does **not** claim a Casper-native pairing verifier or a ZK proof of arbitrary ML inference. The current real Groth16 circuit proves knowledge of a MiMC preimage.

**Which endpoint to hit for a real check:** always `/zk/groth16-real/*`. The `/zk/verify-groth16{,-sim}` and `/zk/batch-verify{,-sim}` endpoints are hash-based simulations kept only for legacy demos and self-mark their responses as `simulation:true, deprecated:true`.

## 4. Failure behavior

A failed check exits non-zero and prints a bounded HTTP/network error without secrets. The script never retries mutating calls, never invents a transaction hash, and skips write-based crypto checks unless a judge key is explicitly supplied.
