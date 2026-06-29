# CasperProver

Verifiable proof layer for AI agent computations on Casper Network.

## Overview

CasperProver generates, stores, and verifies cryptographic proofs of AI agent outputs. Any computation — KYC checks, document processing, model inference — gets a Merkle-anchored proof that can be verified on-chain without replaying the computation.

Use cases:
- KYC verification proofs for DeFi whitelisting
- Audit trails for AI-generated decisions
- Cross-agent computation verification

## How it works

```
input + output + model → commit hash → Merkle tree → on-chain proof registry
                                              ↓
                              verifier-gate → DeFi access / whitelisting
```

1. Agent submits input/output/model data
2. Engine hashes each component and builds a Merkle tree
3. Proof + Merkle path submitted to `proof-registry` contract
4. `verifier-gate` contract checks proof validity for downstream consumers
5. `defi-mock` contract demonstrates KYC-gated access

## Build

```
cd engine
go build ./cmd/casperprover
```

## Usage

```
# generate a demo proof
./casperprover prove

# verify a demo proof
./casperprover verify

# run full KYC demo flow
./casperprover demo

# start API server
API_PORT=8080 ./casperprover serve
```

## API

```
GET  /health         → {"status":"ok","version":"0.1.0"}
GET  /proofs         → list all proofs
GET  /proofs/{id}    → get proof by ID
POST /proofs         → submit new proof
```

POST body:
```json
{
  "agent": "agent-id",
  "input": "raw input data",
  "output": "raw output data",
  "model": "model-v1",
  "use_case": "kyc"
}
```

## Contracts

`contracts/proof-registry/` — submit, get, revoke proofs, agent reputation tracking.

`contracts/verifier-gate/` — verify proof validity, batch checking.

`contracts/defi-mock/` — demo contract using verified proofs for KYC whitelisting.

## Tests

```
cd engine
go test ./...
```

## Structure

```
CasperProver/
├── engine/
│   ├── cmd/casperprover/    # CLI entry point
│   └── internal/
│       ├── hasher/          # SHA-256 utilities
│       ├── prover/          # Merkle tree + proof engine
│       ├── verifier/        # Local proof verification
│       ├── submitter/       # Casper RPC submission
│       ├── kyc/             # KYC demo flow
│       └── api/             # HTTP server
├── contracts/
│   ├── proof-registry/      # Proof storage contract
│   ├── verifier-gate/       # Verification contract
│   └── defi-mock/           # DeFi demo contract
└── config.toml
```

## Casper Agentic Buildathon 2026

Track: Agentic Infrastructure.

## License

MPL-2.0
