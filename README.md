# CasperProver

Verifiable proof layer for AI agent computations on Casper Network.

![MPL-2.0](https://img.shields.io/badge/license-MPL--2.0-orange.svg) ![Go 1.22+](https://img.shields.io/badge/Go-1.22+-00ADD8.svg) ![Casper 2.x](https://img.shields.io/badge/Casper-2.x-red.svg)

## Specification

CasperProver generates, stores, and verifies cryptographic proofs of AI agent outputs. Given an agent computation `f(x) = y` with model `M`, the system produces a Merkle-anchored proof `π` such that:

```
π = MerkleProof(H(x), H(y), H(M))
```

where `H` is SHA-256. The proof can be verified on-chain without replaying the computation.

### Proof Construction

Let `L = [H(x), H(y), H(M)]` be the leaf set. The Merkle tree is built bottom-up:

```
        root
       /    \
    h01      H(M)
   /   \
H(x)  H(y)
```

The commit hash `c = root` is submitted to the `proof-registry` contract. Verification requires presenting `(leaf, path, root)` to the `verifier-gate` contract.

### Use Cases

- KYC verification proofs for DeFi whitelisting
- Audit trails for AI-generated decisions
- Cross-agent computation verification

## Build

```
cd engine
go build ./cmd/casperprover
```

## Usage

```bash
# generate a proof from input/output/model data
./casperprover prove

# verify an existing proof
./casperprover verify

# run the full KYC demo flow
./casperprover demo

# start the API server
API_PORT=8080 ./casperprover serve
```

## API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | `{"status":"ok","version":"0.1.0"}` |
| `GET` | `/proofs` | List all stored proofs |
| `GET` | `/proofs/{id}` | Retrieve proof by ID |
| `POST` | `/proofs` | Submit a new proof |

POST body:

```json
{
  "agent": "agent-id",
  "input": "raw input data",
  "output": "computation result",
  "model": "model-v1",
  "use_case": "kyc"
}
```

Response:

```json
{
  "id": "proof-00a1b2c3",
  "root": "e3b0c44298fc...",
  "leaves": ["a1b2...", "c3d4...", "e5f6..."],
  "path": [0, 1],
  "verified": true
}
```

## Contracts

### proof-registry (`contracts/proof-registry/`)

| Entry Point | Description |
|-------------|-------------|
| `submit_proof` | Store proof root + metadata on-chain |
| `get_proof` | Retrieve proof by ID |
| `revoke_proof` | Mark a proof as revoked |
| `register_agent` | Register an agent with reputation tracking |
| `update_reputation` | Update agent trust score after verification |
| `get_agent` | Query agent registration data |

### verifier-gate (`contracts/verifier-gate/`)

| Entry Point | Description |
|-------------|-------------|
| `verify_proof` | Check proof against registry, emit result |
| `batch_verify` | Verify multiple proofs in one call |
| `is_verified` | Query verification status by proof ID |

### defi-mock (`contracts/defi-mock/`)

| Entry Point | Description |
|-------------|-------------|
| `check_kyc` | Call verifier-gate for a given proof |
| `grant_access` | Whitelist user if proof is valid |
| `is_whitelisted` | Query whitelist status |

## Testing

```bash
cd engine
go test ./...

# contract tests
cd contracts/tests
cargo test
```

## Configuration

`config.toml`:

```toml
[node]
url = "https://node.integration.cspr.cloud/"
chain = "casper-test"

[api]
port = 8080
read_timeout = "30s"

[prover]
hash_algorithm = "sha256"
max_leaves = 16
```

## Structure

```
CasperProver/
├── engine/
│   ├── cmd/casperprover/   # CLI entry
│   └── internal/
│       ├── hasher/         # SHA-256 utils
│       ├── prover/         # Merkle tree + proof engine
│       ├── verifier/       # Local verification
│       ├── submitter/      # Casper RPC client
│       ├── kyc/            # KYC demo flow
│       └── api/            # HTTP server
├── contracts/
│   ├── proof-registry/     # Proof storage
│   ├── verifier-gate/      # On-chain verification
│   └── defi-mock/          # DeFi whitelisting demo
├── config.toml
├── Makefile
└── Dockerfile
```

## Buildathon

Casper Agentic Buildathon 2026 submission.

## License

This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
See [LICENSE](LICENSE) for details.
