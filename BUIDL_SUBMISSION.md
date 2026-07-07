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

**Three contracts deployed on Casper testnet:**
- **Proof Registry** — stores Merkle roots, proof metadata, verification status
- **Verifier Gate** — checks inclusion proofs, manages downstream access
- **DeFi Mock** — sample vault gated by verifier-gate (live KYC-gated DeFi demo)

---

## What's Live

| Artifact | Link |
|---|---|
| Lab (72 proofs) | https://casperprover.xyz/lab |
| API | https://casperprover-api.onrender.com |
| GitHub | https://github.com/anna-stolbovskaja/CasperProver |
| Proof Registry contract | [96e97c4d...a10708](https://testnet.cspr.live/contract/96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708) |
| Verifier Gate contract | [a37f9cde...9f77d3](https://testnet.cspr.live/contract/a37f9cde9dbdc5bb8b9e92c663bdc59b83b42c89dc75ec73f7f7cde2619f77d3) |
| DeFi Mock contract | [fe0c45f6...0b39ef](https://testnet.cspr.live/contract/fe0c45f67c8cd99f0bda0047399a113588870ec0d79d9102f44107303f0b39ef) |

---

## Video
[Video](TBD)

---

## Tech Stack

| Layer | Technology | Why |
|---|---|---|
| Smart contracts | Rust / Casper 2.x | Native Casper contract model, deterministic execution |
| Proof engine | Go 1.22 | Fast Merkle tree construction, low latency |
| API | Go HTTP | Lightweight, same language as engine |
| Frontend | Vite + TypeScript + Tailwind | Fast lab, zero framework lock-in |
| SDK | Go + Python + MCP adapter | Meets agents where they live |

---

## Unique Angle

**AI accountability, not just blockchain storage.** CasperProver is not a generic data registry. It is specifically designed so AI frameworks (LangChain, Claude via MCP, any REST client) can register proof commitments at inference time. The MCP server (`sdk/mcp_server.go`) means any AI agent using Model Context Protocol can call CasperProver as a native tool — submit a proof without custom integration.

**Trustless verification** — the verifier-gate contract means a downstream DeFi protocol can gate access based on proof validity without trusting the agent operator. The KYC demo shows this end-to-end: agent proves KYC eligibility → on-chain gate checks proof → vault unlocks.

---

## Four Proof Types

| Type | What it proves |
|---|---|
| `merkle-inclusion` | A value was part of a computation |
| `kyc-eligibility` | Wallet passed KYC (no PII revealed) |
| `balance-range` | Balance was in a range (no exact value) |
| `transaction-membership` | A tx was processed by a specific agent |

---

## Team

**anna-stolbovskaja** — Solo builder focused on verifiable compute infrastructure. Architected the Merkle proof registry, wrote all three Casper smart contracts (Proof Registry, Verifier Gate, DeFi Mock), built the Go proof engine with SHA-256 Merkle trees, developed the REST API, created the Vite lab for live proof inspection, and shipped SDK clients in Go and Python plus an MCP adapter for AI frameworks. Background in cryptographic systems and blockchain infrastructure; track is Agentic Infrastructure / Verifiable Compute.

GitHub: [github.com/anna-stolbovskaja](https://github.com/anna-stolbovskaja)
