# Changelog

All notable changes to CasperProver are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions follow [Semantic Versioning](https://semver.org/).

---

## [Unreleased]

*Frontend polish, DoraHacks submission prep, docs hardening.*

### Added
- Skeleton loader components (`Skeleton`, `CardSkeleton`, `TableSkeleton`) wired into Overview + Proofs.
- `EmptyState` component for lab list views (Proofs / Models / Aggregation / KYC).
- `CopyButton` with checkmark feedback and legacy-browser fallback.
- Full favicon set: `favicon.ico` (multi-size) + `favicon-16x16.png` + `favicon-32x32.png` + `apple-touch-icon.png`.
- Sitemap.xml expanded to every public Lab and Docs route.
- Verify / Secret Scan / Contracts / Proofs / PQ status badges in README.
- Telegram + X + GitHub social links row in footer.
- `docs/SUBMISSION_CHECKLIST.tmp` — DoraHacks submission tracker.
- `docs/RED_TEAM.tmp` — 18-vector red-team self-audit.
- `docs/API.tmp` — curl-first API reference for all 32 endpoints.

### Changed
- All 20 `console.error` calls in Lab components guarded by `import.meta.env.DEV`; only `ErrorBoundary` (line 18) and `toast` fallback (line 122) keep unconditional logging by design.
- Main marketing navbar auto-hides on `/lab/*` routes (Lab has its own sticky top bar).
- `ErrorBoundary` auto-resets on route change via `key={location.pathname}` — no stale error screens across nav.
- Mobile Lab sidebar auto-closes on route change.
- `README.md` badges regrouped: CI status row + capability row.

### Fixed
- Vite env types wired via `src/vite-env.d.ts` so `import.meta.env.DEV` type-checks under `strict`.

### Planned
- Deploy `proof-of-inference`, `model-registry`, `proof-aggregation` contracts to mainnet
- Full model-inference ZK circuit (beyond MiMC preimage)
- STARK recursive aggregation
- Grafana/Prometheus metrics
- Proof expiry and rotation policies

---

## [1.2.0] — 2026-07-08

*Wallet integration, UX overhaul, stake-slashing contract, CI fix*

### Added
- **stake-slashing** contract deployed on testnet — 20% CSPR slash + permissionless bounty
- **defi-mock** contract redeployed with hardened `is_whitelisted(user: ByteArray32)` signature
- CSPR.click wallet integration (Casper Wallet, Ledger, MetaMask Snap, Google SSO)
- On-chain proof anchoring via wallet signing (`casper-js-sdk ^5.0.12`)
- Real Groth16 ZK-SNARK proofs via gnark (BN254 pairing-based cryptography)
- ML-DSA-65 post-quantum signing (FIPS 204, cloudflare/circl) + Lamport OTS
- 10 interactive Lab pages with sticky header, toast system, demo/live mode
- SectionIntro blocks, click-to-copy, search/filter, confirm modals
- Mobile-responsive menu
- Proof Pipeline merged into Playground

### Changed
- API expanded to 32 endpoints, SDK to 34 methods, MCP to 32 tools
- Agent field max length increased to 128 for wallet public keys

### Fixed
- SDK build: `Model` → `ModelID` in MCP server (CI was broken)
- Table refresh before wallet signing
- Revoke without X-Public-Key header
- Toast ordering: wallet popup before success toast

---

## [1.1.0] — 2026-07-02

*Major feature expansion — pre-submission hardening*

### Added

#### Smart Contracts (Rust/Wasm)
- **ProofOfInference** (498 LOC) — individual proof anchoring with Merkle root on-chain verification
- **ProofAggregation** (179 LOC) — batch-aggregated proofs for gas-efficient on-chain anchoring
- **ModelRegistry** (372 LOC) — on-chain registry of approved AI models with version tracking

#### Backend Modules (Go)
- **Model Registry** (`engine/internal/model/registry.go`, 283 LOC) — model CRUD, versioning, metadata management, and framework validation
- **Complexity Analyzer** (`engine/internal/prover/complexity.go`, 186 LOC) — proof complexity estimation based on model parameters and input size
- **Distributed Worker** (`engine/internal/worker/distributed.go`, 438 LOC) — task distribution, worker pool management, and fault-tolerant scheduling

#### MCP Server
- Expanded from 15 to 23 tools — new tools for model registry, complexity analysis, and distributed task management

#### Lab
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
- **Proof Registry contract** deployed on Casper testnet ([96e97c4d...a10708](https://testnet.cspr.live/contract/e11088f1f15a719f21c0c318d1f34d0b96419a22d60ac8fa384ecf5285fa7bc5)) — stores Merkle roots, proof metadata, verification status
- **Verifier Gate contract** ([a37f9cde...9f77d3](https://testnet.cspr.live/contract/06d69182b13c4d041613fe7e6e0805cdb06f099eff4291b40154d78cc0c79b66)) — checks inclusion proofs, manages access control
- **DeFi Mock contract** ([b9b11a97...b81d3](https://testnet.cspr.live/contract/b9b11a976af20b4b5d128c44e5ee118b8830c26a79f4b603cdf0a00e537b81d3)) — sample vault gated by verifier-gate, demonstrating KYC-gated DeFi flow
- **Merkle tree builder** in Go engine — SHA-256 leaf hashing over `{H(input), H(output), H(model)}` triplets, binary tree construction, path serialization
- **Four proof types supported**: `merkle-inclusion`, `kyc-eligibility`, `balance-range`, `transaction-membership`
- **REST API** at `https://casperprover-api-ylsh.onrender.com` — endpoints: `POST /api/v1/proof/submit`, `POST /api/v1/proof/verify`, `GET /api/v1/proofs`, `GET /api/v1/stats`
- **Lab** at `casperprover.xyz/lab` — 72 live proofs registered on testnet, proof type breakdown, verification status badges
- **Go SDK** (`sdk/`) — `client.go` with submit/verify helpers, Python client (`python_client.py`)
- **MCP server** (`sdk/mcp_server.go`) — Model Context Protocol adapter so AI frameworks (Claude, LangChain) can call CasperProver as a tool
- **Contract test suite** (`contracts/tests/`) — integration tests covering registry, gate, and mock interactions
- Configuration via `config.toml` — node URL, chain name, API port, rate limit, hash algorithm, precomputed proof directory

### Infrastructure
- Deployed on Casper testnet (chain: `casper-test`)
- API hosted on Render
- Frontend hosted on Vercel
- CI via GitHub Actions (`check.yml`)


