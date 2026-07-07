# CasperProver — Cryptographic Accountability for AI Agents

## Project Name & Tagline

**CasperProver**  
*Cryptographic accountability for AI agents — Merkle proof registry on Casper Network*

---

## The Problem

AI agents are making consequential decisions — approving loans, running KYC checks, executing compliance rules — and nobody can audit them.

There is no standard way to prove what an AI agent computed without:
1. Re-running the entire model (expensive, often impossible)
2. Trusting the operator's logs (centralized, mutable)
3. Revealing the inputs or model weights (privacy violation)

The result: AI outputs are **unverifiable black boxes** deployed in critical infrastructure. As agents become more autonomous, this accountability gap becomes a systemic risk.

---

## The Solution

CasperProver is a **Merkle proof registry on Casper Network**. Agents submit a cryptographic fingerprint of their computation; anyone can verify inclusion on-chain — instantly, permanently, without replaying the model.

### How it works

```
Agent computes f(x) = y with model M
  → Submit: H(x), H(y), H(M)
  → Engine builds Merkle tree over {H(x), H(y), H(M)}
  → Root committed on-chain (Proof Registry contract)
  → Inclusion proof stored + queryable
  → Verifier Gate checks any proof in milliseconds
```

**Four contracts deployed on Casper testnet:**
- **Proof Registry** — stores Merkle roots, proof metadata, verification status
- **Verifier Gate** — checks inclusion proofs, manages downstream access
- **DeFi Mock** — vault gated by verifier-gate (live KYC-gated DeFi demo)
- **Stake Slashing** — real CSPR economic penalty for revoked proofs (20% slash + permissionless bounty)

---

## What's Live

| Artifact | Link |
|---|---|
| Lab (72+ proofs) | https://casperprover.xyz/lab |
| API | https://casperprover-api.onrender.com |
| GitHub | https://github.com/anna-stolbovskaja/CasperProver |
| Proof Registry | [96e97c4d...a10708](https://testnet.cspr.live/contract/96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708) |
| Verifier Gate | [a37f9cde...9f77d3](https://testnet.cspr.live/contract/a37f9cde9dbdc5bb8b9e92c663bdc59b83b42c89dc75ec73f7f7cde2619f77d3) |
| DeFi Mock | [fe0c45f6...0b39ef](https://testnet.cspr.live/contract/fe0c45f67c8cd99f0bda0047399a113588870ec0d79d9102f44107303f0b39ef) |
| Stake Slashing | [cf70e1fe...d9bd1](https://testnet.cspr.live/contract/cf70e1fedf52f250a807e2bece5eccaa3ae12c58115e40393f3d3f77246d9bd1) |

---

## Video
[Video](TBD)

---

## Tech Stack

| Layer | Technology | Why |
|---|---|---|
| Smart contracts | Rust / Casper 2.x | Native Casper contract model, deterministic execution |
| Proof engine | Go 1.22 | Fast Merkle tree construction, low latency (~99ms avg) |
| API | Go HTTP | Lightweight, same language as engine |
| Frontend | Vite + TypeScript + Tailwind | Fast lab, zero framework lock-in |
| SDK | Go + MCP adapter | Meets agents where they live |
| ZK Backend | gnark (BN254 Groth16) | Real pairing-based zk-SNARK verification |
| PQ Crypto | circl (ML-DSA-65) + Ed25519 + Lamport | Post-quantum readiness |

---

## Key Features

### Real Cryptography (not stubs)
- **Groth16 zk-SNARK** — actual R1CS circuit, actual trusted setup, actual BN254 pairing checks via gnark. Proves knowledge of a MiMC preimage without revealing it. Rejects tampered proofs.
- **Post-Quantum Signatures** — real Ed25519 + ML-DSA-65 (FIPS 204) hybrid signing. Lamport one-time signatures for quantum-resistant fallback. All wired to live API endpoints.
- **STARK Aggregation** — batch N proofs into a single verifiable aggregate with merkle-based STARKPack.

### Economic Security (Stake & Slash)
An agent stakes CSPR before submitting proofs. If a proof is revoked, anyone can permissionlessly call `report_and_slash` — 20% of the stake goes to the reporter as a bounty. Each proof can only be slashed once (tombstoned). This creates real economic skin-in-the-game for honest agent behavior. Verified with real CSPR transfers on testnet.

### KYC-Gated DeFi Demo
End-to-end flow: agent proves KYC eligibility → proof anchored on-chain → verifier gate checks inclusion → DeFi vault access granted — all without exposing PII. Admin-gated access control with typed AccountHash (no string-formatting bugs).

### MCP Integration
SDK includes an MCP (Model Context Protocol) server — any AI agent using MCP can call CasperProver as a native tool. Submit proofs, verify them, check KYC status, all through standard tool calls.

---

## Proof Types

| Type | What it proves | Use case |
|---|---|---|
| `merkle-inclusion` | A value was part of a computation | General AI output audit |
| `kyc-eligibility` | Wallet passed KYC (no PII revealed) | DeFi access control |
| `balance-range` | Balance was in a range (no exact value) | Creditworthiness without exposure |
| `transaction-membership` | A tx was processed by a specific agent | Compliance, dispute resolution |
| `state-transition` | State change was valid | On-chain state verification |
| `merkle-exclusion` | A value was NOT part of a set | Blacklist/sanctions screening |

---

## Team

**anna-stolbovskaja** — Solo builder focused on verifiable compute infrastructure. Architected the Merkle proof registry, wrote all four Casper smart contracts (Proof Registry, Verifier Gate, DeFi Mock, Stake Slashing), built the Go proof engine with real Groth16 and post-quantum cryptography, developed the REST API, created the Vite lab for live proof inspection, and shipped SDK clients plus an MCP adapter for AI frameworks.

GitHub: [github.com/anna-stolbovskaja](https://github.com/anna-stolbovskaja)
