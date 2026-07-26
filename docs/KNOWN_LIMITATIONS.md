# Status & Roadmap

> Current state as of 2026-07-25. All four deployed contracts live on Casper testnet; three additional contracts (`proof-of-inference`, `model-registry`, `proof-aggregation`) are written and compile but currently exceed the 65 KiB wasm limit and remain undeployed — see `frontend/public/onchain.json` for the source of truth.

## ✅ What's Live

### Smart Contracts (4 on testnet)
- **proof-registry** — immutable proof store with on-chain anchoring
- **verifier-gate** — Merkle inclusion proof checker, cross-contract verification
- **defi-mock** — KYC-gated DeFi vault with admin-controlled whitelisting
- **stake-slashing** — economic penalty system for revoked/invalid proofs

### Proof Engine (32 API endpoints)
- SHA-256 Merkle proof generation & verification (<50ms)
- Real BN254 Groth16 ZK proofs via gnark (`/zk/groth16-real/*`)
- Conceptual Groth16 verification for rapid testing (`/zk/verify-groth16`)
- Post-quantum signing: hybrid Ed25519 + ML-DSA-65 (FIPS 204, cloudflare/circl); Lamport OTS occupies the hash-based ("SPHINCS+ family") slot until a Go SLH-DSA implementation ships
- Hash-chain aggregation with Postgres persistence
- Proof-chain DAG validation (Phase 2: cycle detection, input continuity)
- KYC whitelist with database persistence
- API key authentication on mutating endpoints
- Rate limiting (60 req/min) and input validation

### SDK & MCP
- Go SDK: 32 methods, 1:1 mapping to all API endpoints
- MCP server: 32 tools with full InputSchema definitions
- All tools backed by real API — no stubs

### Frontend (10 interactive tabs)
- Overview, Proofs, Models, Aggregation, ZK Proofs, PQ Crypto, Contracts, Agent Demo, Playground, KYC

## 🔜 Roadmap

### Near-term
- Real Groth16 circuit for full model-inference verification (current: MiMC preimage)
- Production trusted setup ceremony (current: per-start generation)
- Deploy `proof-of-inference`, `model-registry`, `proof-aggregation` contracts
- Demo/Real data toggle in lab

### Medium-term
- STARK recursive aggregation (pending mature Go library; winterfell is Rust-only)
- Multi-chain anchoring (EVM, Solana)
- Hardware attestation (TPM 2.0 / SGX / SEV — interfaces defined in Phase 2)
- Distributed prover network with MPC threshold support

### Long-term
- Full-circuit ZK proofs for arbitrary ML models
- Formal verification of proof pipeline (current TLC pass is *small-model* model-checking, not a full verification)
- Mainnet deployment with gas optimization

## ⚠️ Cryptographic honesty audit

Before claiming anything PQ / ZK / quantum related, read `docs/PQ_HONESTY.md`. In particular:

- SHA-256 → SHAKE-256 with the same output length is **not** an automatic PQ uplift. Grover gives at most a quadratic speedup against generic preimage/collision search; SHA-256 already meets ≥128-bit post-quantum resistance for its output length.
- "Quantum-inspired simulated annealing" is a classical heuristic. Never claim "quantum speedup" for QISA.
- Recursive ZK / folding / on-chain pairings / VRF sortition / range proofs are **roadmap**, not shipped.
- Citing a patent or academic publication is prior-art context, not evidence of patentability or freedom-to-operate. A professional FTO review is required before any commercialisation step.
