# Changelog

All notable changes to CasperProver are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions follow [Semantic Versioning](https://semver.org/).

---

## [1.1.0] — 2026-07-02

*Major feature expansion — pre-submission hardening*

### Added

#### Smart Contracts (Rust/Odra)
- **ProofOfInference** (409 LOC) — individual proof anchoring with Merkle root on-chain verification
- **ProofAggregation** (346 LOC) — batch-aggregated proofs for gas-efficient on-chain anchoring
- **ModelRegistry** (278 LOC) — on-chain registry of approved AI models with version tracking

#### Backend Modules (Go)
- **Model Registry** (`engine/internal/model/registry.go`, 266 LOC) — model CRUD, versioning, metadata management, and framework validation
- **Complexity Analyzer** (`engine/internal/prover/complexity.go`, 162 LOC) — proof complexity estimation based on model parameters and input size
- **Distributed Worker** (`engine/internal/worker/distributed.go`, 431 LOC) — task distribution, worker pool management, and fault-tolerant scheduling

#### MCP Server
- Expanded from 15 to 23 tools — new tools for model registry, complexity analysis, and distributed task management

#### Dashboard
- 4 new tabs: Models, Complexity, Workers, Contracts
- Model registry browser with version and framework info
- Complexity metrics: average generation time, Merkle depth, gas estimates
- Worker node status with real-time load and region info

#### Tests
- 51 new business logic tests (3 files, 1,373 LOC)
- `registry_test.go`: 16 tests — model CRUD, versioning, metadata
- `complexity_test.go`: 15 tests — complexity estimation, boundary analysis
- `distributed_test.go`: 20 tests — task distribution, worker pool, fault tolerance

### Security
- Secure temp file creation for deployer keys (os.CreateTemp)
- X-Public-Key header validation for revoke operations
- Capped pagination limits
- Request body size limits
- Bounds checking in Merkle tree operations

---

## [1.0.0] — 2026-06-30

### Added
- **Proof Registry contract** deployed on Casper testnet ([96e97c4d...a10708](https://testnet.cspr.live/contract/96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708)) — stores Merkle roots, proof metadata, verification status
- **Verifier Gate contract** ([a37f9cde...9f77d3](https://testnet.cspr.live/contract/a37f9cde9dbdc5bb8b9e92c663bdc59b83b42c89dc75ec73f7f7cde2619f77d3)) — checks inclusion proofs, manages access control
- **DeFi Mock contract** ([b9b11a97...b81d3](https://testnet.cspr.live/contract/b9b11a976af20b4b5d128c44e5ee118b8830c26a79f4b603cdf0a00e537b81d3)) — sample vault gated by verifier-gate, demonstrating KYC-gated DeFi flow
- **Merkle tree builder** in Go engine — SHA-256 leaf hashing over `{H(input), H(output), H(model)}` triplets, binary tree construction, path serialization
- **Four proof types supported**: `merkle-inclusion`, `kyc-eligibility`, `balance-range`, `transaction-membership`
- **REST API** at `https://casperprover-api.onrender.com` — endpoints: `POST /api/v1/proof/submit`, `POST /api/v1/proof/verify`, `GET /api/v1/proofs`, `GET /api/v1/stats`
- **Dashboard** at `casperprover.xyz/dashboard` — 72 live proofs registered on testnet, proof type breakdown, verification status badges
- **Go SDK** (`sdk/`) — `client.go` with submit/verify helpers, Python client (`python_client.py`)
- **MCP server** (`sdk/mcp_server.go`) — Model Context Protocol adapter so AI frameworks (Claude, LangChain) can call CasperProver as a tool
- **Contract test suite** (`contracts/tests/`) — integration tests covering registry, gate, and mock interactions
- Configuration via `config.toml` — node URL, chain name, API port, rate limit, hash algorithm, precomputed proof directory

### Infrastructure
- Deployed on Casper testnet (chain: `casper-test`)
- API hosted on Render
- Frontend hosted on Vercel
- CI via GitHub Actions (`check.yml`)

---

## [Unreleased]

### Planned
- Mainnet deployment
- zk-SNARK proof type (Groth16)
- Batch proof submission
- Grafana/Prometheus metrics dashboard
- Proof expiry and rotation policies
