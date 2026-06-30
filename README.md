<a id="readme-top"></a>

<div align="center">

# CasperProver

**Cryptographic proof registry for AI agent computations — merkle-verified, on-chain immutable**

*Prove what an agent computed. Verify it on-chain. No replay needed.*

[![CI](https://github.com/anna-stolbovskaja/CasperProver/actions/workflows/check.yml/badge.svg)](https://github.com/anna-stolbovskaja/CasperProver/actions/workflows/check.yml)
[![Go 1.22+](https://img.shields.io/badge/Go-1.22+-00ADD8.svg?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Casper 2.x](https://img.shields.io/badge/Casper-2.x-FF0000.svg?style=flat-square)](https://casper.network)
[![MPL-2.0](https://img.shields.io/badge/license-MPL--2.0-orange.svg?style=flat-square)](LICENSE)
[![Live Demo](https://img.shields.io/badge/demo-casperprover.xyz-6366f1.svg?style=flat-square)](https://casperprover.xyz/dashboard)

[**Live Demo →**](https://casperprover.xyz/dashboard) · [Architecture](docs/ARCHITECTURE.md) · [SDK](docs/SDK.md) · [API](https://casperprover-api.onrender.com)

</div>

---

> [!NOTE]
> Three contracts live on Casper testnet. 72 proofs registered. The Go engine generates Merkle-anchored proofs of agent outputs and submits commit hashes on-chain. A downstream DeFi mock contract gates access based on proof validity — working KYC demo included.

---

## Why this matters

AI agents are executing critical workflows — KYC checks, financial decisions, compliance rules. But there is **no audit trail**. You cannot prove what an agent computed without re-running the entire model.

CasperProver closes that gap:

| Without CasperProver | With CasperProver |
|---|---|
| Re-run the model to verify | Verify inclusion proof in milliseconds |
| Trust the agent operator | Verify cryptographically on-chain |
| Black-box outputs | Merkle-anchored, tamper-evident record |
| Centralized log (mutable) | Immutable on-chain commitment |

---

## Architecture

```mermaid
flowchart LR
    A[AI Agent] -->|input + output + model hash| B[Merkle Tree Builder]
    B -->|SHA-256 leaf set| C[Merkle Root]
    C -->|deploy tx| D[Proof Registry\nCasper Contract]
    D -->|proof stored| E[Inclusion Proof]
    E -->|query| F[Verifier Gate\nCasper Contract]
    F -->|access decision| G[DeFi Mock / Downstream App]
```

**How it works:**  
Given `f(x) = y` with model `M`, CasperProver produces `π = MerkleProof(H(x), H(y), H(M))` where `H = SHA-256`. The root is committed on-chain; the inclusion proof is stored and queryable forever without re-running the model.

---

## Quickstart

**Prerequisites:** Go 1.22+, access to Casper testnet node

```bash
git clone https://github.com/anna-stolbovskaja/CasperProver
cd CasperProver
cp config.toml.example config.toml   # edit node_url if needed
go mod download
go run ./engine/cmd/...
```

**Submit a proof:**
```bash
curl https://casperprover-api.onrender.com/api/v1/proof/submit \
  -H "Content-Type: application/json" \
  -d '{
    "proof_type": "merkle-inclusion",
    "input_hash": "sha256:abc123...",
    "output_hash": "sha256:def456...",
    "model_id": "gpt-4o"
  }'
```

**Verify a proof:**
```bash
curl https://casperprover-api.onrender.com/api/v1/proof/verify \
  -H "Content-Type: application/json" \
  -d '{"proof_id": "proof_001"}'
# → {"valid": true, "merkle_root": "...", "on_chain_tx": "..."}
```

**Success check:** `{"valid": true}` with a `merkle_root` field means the proof is live on-chain.

---

## Proof Types

| Type | What it proves | Use case |
|---|---|---|
| `merkle-inclusion` | A value was part of a computation | General AI output audit |
| `kyc-eligibility` | Wallet passed KYC (no PII revealed) | DeFi access control |
| `balance-range` | Balance was in a range (no exact value) | Creditworthiness without exposure |
| `transaction-membership` | A tx was processed by a specific agent | Compliance, dispute resolution |

---

## Live Demo

**Dashboard:** [casperprover.xyz/dashboard](https://casperprover.xyz/dashboard)  
72 proofs live on Casper testnet. Filter by proof type, click any row to see the Merkle path and on-chain tx.

**Deployed Contracts (testnet):**

| Contract | Address |
|---|---|
| Proof Registry | [96e97c4d...a10708](https://testnet.cspr.live/contract/96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708) |
| Verifier Gate | [a37f9cde...9f77d3](https://testnet.cspr.live/contract/a37f9cde9dbdc5bb8b9e92c663bdc59b83b42c89dc75ec73f7f7cde2619f77d3) |
| DeFi Mock | [b9b11a97...b81d3](https://testnet.cspr.live/contract/b9b11a976af20b4b5d128c44e5ee118b8830c26a79f4b603cdf0a00e537b81d3) |

---

## Tech Stack

| Layer | Technology |
|---|---|
| Smart contracts | Rust / Casper 2.x |
| Proof engine | Go 1.22 |
| API | Go HTTP server |
| Frontend | Vite + TypeScript + Tailwind |
| SDK | Go + Python client |
| MCP adapter | Model Context Protocol server |
| Hosting | Vercel (frontend) · Render (API) |
| Chain | Casper testnet (`casper-test`) |

---

## Project Structure

```
CasperProver/
├── contracts/          # Rust smart contracts
│   ├── proof-registry/ # Main proof store
│   ├── verifier-gate/  # Inclusion proof checker
│   └── defi-mock/      # KYC-gated vault demo
├── engine/             # Go proof generation engine
│   ├── cmd/            # Entry point
│   └── internal/       # Merkle builder, Casper client
├── sdk/                # Go + Python SDK
│   ├── client.go
│   ├── mcp_server.go   # MCP adapter for AI frameworks
│   └── python_client.py
├── frontend/           # Vite/TS dashboard
└── docs/               # Architecture, SDK docs
```

---

## Team

**anna-stolbovskaja** — smart contracts, proof engine, API, frontend  
Track: Agentic Infrastructure / Verifiable Compute

---

## License

[MPL-2.0](LICENSE) · Last verified: 2026-06-30

<div align="right"><a href="#readme-top">↑ back to top</a></div>
