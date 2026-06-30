# CasperProver — Roadmap to Production

> Verifiable cryptographic proofs for AI agent decisions — trust through verification, not faith.

---

## Current State (Hackathon Prototype)

**Done:**
- [x] Project scaffold (Go engine, Rust contracts, CI)
- [x] Smart contracts: ProofRegistry + VerifierGate + DeFi Mock (Rust/Wasm)
- [x] Go engine: hasher (SHA-256), Merkle prover, proof generator, verifier
- [x] Pre-computed proof cache (3 KYC proofs)
- [x] KYC demo flow: full KYC → DeFi access pipeline
- [x] CLI: prove, verify, revoke, agent commands
- [x] HTTP API: POST /prove, GET /verify/:id, GET /agent/:id
- [x] Tests: hasher, merkle, proof (Go unit tests)
- [x] CI/CD: GitHub Actions (go test + go vet)
- [x] README (minimalist style)

**Not done:**
- [ ] Smart contracts not compiled / not deployed to testnet
- [ ] No on-chain integration (local proof generation only)
- [ ] No landing page / demo video
- [ ] No MCP server
- [ ] Proof lifecycle (expiry, revocation) not wired end-to-end

---

## Phase 1 — Hackathon Submission (Deadline: July 1, 2026)

### 1.1 Testnet Deployment
- [ ] Set up Casper 2.0 local dev (NCTL Docker)
- [ ] Compile ProofRegistry contract → Wasm
- [ ] Compile VerifierGate contract → Wasm
- [ ] Compile DeFi Mock contract → Wasm
- [ ] Deploy all 3 to Casper Integration Testnet
- [ ] Smoke test: submit_proof → get_proof → verify → check_kyc → grant_access
- [ ] Record deploy hashes

### 1.2 On-Chain Integration
- [ ] Wire `submitter/casper.go` to Casper Rust SDK (via FFI) or Casper REST API
- [ ] Submit real proof hashes to ProofRegistry on testnet
- [ ] Verify proofs via VerifierGate on-chain
- [ ] DeFi Mock: demonstrate KYC → grant_access flow with real transactions

### 1.3 Landing Page
- [ ] Single-page site: black theme, monospace, hacker aesthetic
- [ ] Interactive proof generator: enter data → generate Merkle proof → verify
- [ ] Terminal-style animation for CLI demo
- [ ] Deploy to GitHub Pages

### 1.4 Demo Video
- [ ] Script: trust problem → CasperProver solution → CLI demo → DeFi flow
- [ ] Record 3-5 min video
- [ ] Upload to YouTube

### 1.5 Submission Package
- [ ] DoraHacks submission text
- [ ] Logo (generated externally)
- [ ] README with testnet links

---

## Phase 2 — Post-Hackathon MVP (Weeks 1–6)

### 2.1 Smart Contract Hardening
- [ ] Full Rust integration tests
- [ ] Proof revocation with signature verification
- [ ] Proof expiry: TTL-based auto-invalidation
- [ ] Rate limiting: max 100 verify() calls per block per caller
- [ ] Reputation system: verified/failed/total counts, score calculation
- [ ] Agent registration with model_hash commitment
- [ ] Contract upgrade path (CEP-86)

### 2.2 Cryptographic Improvements
- [ ] Replace pure SHA-256 with Poseidon hash (ZK-friendly)
- [ ] Sparse Merkle tree for efficient proof updates
- [ ] Batch proof aggregation: combine N proofs into 1 on-chain submission
- [ ] Proof compression: reduce on-chain storage per proof
- [ ] Integration with `litmus-zk` (Casper's ZK light client)
- [ ] Integration with `block-signature-prover` for block-level attestations

### 2.3 Use Case Expansion
- [ ] KYC verification (current, hardened)
- [ ] Credit scoring: AI model evaluates creditworthiness → verifiable proof
- [ ] Contract analysis: AI reviews legal contract → compliance proof
- [ ] Trade execution: AI trading decision → proof of reasoning
- [ ] Content authenticity: AI-generated content → provenance proof

### 2.4 SDK and MCP
- [ ] Go SDK: `import casperprover "github.com/anna-stolbovskaja/CasperProver/sdk"`
- [ ] Python SDK wrapper (via gRPC or REST)
- [ ] Node.js SDK wrapper
- [ ] MCP Server: `generate_proof`, `verify_proof`, `check_kyc`
- [ ] OpenAI function-calling schema

### 2.5 CLI Enhancements
- [ ] Interactive TUI mode (bubbletea or similar)
- [ ] `casperprover demo --kyc-defi` with live testnet
- [ ] `casperprover batch --dir proofs/` for bulk operations
- [ ] Config file support: `~/.casperprover/config.toml`
- [ ] Output formats: JSON, table, minimal

### 2.6 Testing
- [ ] Contract integration tests on NCTL
- [ ] End-to-end: CLI prove → on-chain submit → on-chain verify → DeFi access
- [ ] Benchmarks: proof generation time, Merkle tree depth limits
- [ ] Fuzz testing: random inputs to hasher and prover
- [ ] Coverage gate: ≥80%

---

## Phase 3 — Production Beta (Months 2–4)

### 3.1 Zero-Knowledge Migration
- [ ] Research: Groth16 vs PLONK vs STARK for Casper compatibility
- [ ] ZK circuit: prove AI model executed correctly without revealing inputs
- [ ] ZK proof generation: local prover (GPU-accelerated)
- [ ] ZK verification: on-chain verifier contract
- [ ] Hybrid mode: Merkle proofs for low-value, ZK proofs for high-value decisions
- [ ] Benchmark: proof generation time vs security level tradeoffs

### 3.2 Proof Marketplace
- [ ] Proof request system: DeFi protocols request KYC proofs
- [ ] Proof pricing: pay CSPR for proof verification
- [ ] Proof subscription: recurring verification for compliance
- [ ] Proof sharing: one proof valid across multiple DeFi protocols (with consent)

### 3.3 Agent Registry
- [ ] Public agent directory: verified AI agents with reputation scores
- [ ] Model commitment: prove agent uses specific model version
- [ ] Audit history: all proofs submitted by an agent
- [ ] Agent certification: threshold reputation for "verified" badge
- [ ] Revocation cascade: revoking agent revokes all their proofs

### 3.4 Security
- [ ] Cryptographic audit (external firm specializing in ZK/crypto)
- [ ] Side-channel analysis: timing attacks on proof generation
- [ ] Key management: secure key derivation for proof signing
- [ ] Formal verification of core Merkle proof logic (if feasible)

### 3.5 Infrastructure
- [ ] PostgreSQL: proof metadata, agent registry, verification logs
- [ ] Redis: proof caching, rate limiting
- [ ] gRPC API: high-performance alternative to REST
- [ ] Docker production stack
- [ ] Kubernetes deployment with horizontal scaling

---

## Phase 4 — Commercial Launch (Months 4–8)

### 4.1 Business Model
- [ ] Free tier: 100 proofs/month, local verification
- [ ] Pro ($79/mo): 10K proofs, on-chain verification, priority support
- [ ] Enterprise (custom): unlimited, SLA, dedicated infrastructure
- [ ] Pay-per-proof: $0.05 per on-chain proof submission
- [ ] Verification-as-a-Service: API for third parties to verify proofs

### 4.2 DeFi Integrations
- [ ] Lending protocols: proof-gated collateral access
- [ ] DEX: verified agent trading permissions
- [ ] Insurance: proof-based claim validation
- [ ] Staking: proof-of-compliance for institutional stakers
- [ ] Bridge: cross-chain proof relay

### 4.3 Mainnet
- [ ] Casper Mainnet deployment
- [ ] Contract versioning and migration
- [ ] Production nodes with redundancy
- [ ] Monitoring and alerting (Grafana, PagerDuty)

### 4.4 Compliance
- [ ] Privacy regulations: GDPR (proofs don't contain personal data by design)
- [ ] Financial regulations: MiCA compliance for EU
- [ ] Audit trail: immutable proof history for regulators
- [ ] Data handling policy: input data never stored, only hashes

### 4.5 Developer Portal
- [ ] Documentation site (GitBook or Docusaurus)
- [ ] API playground: interactive proof generation and verification
- [ ] Integration guides per use case (KYC, credit scoring, content auth)
- [ ] SDKs on package managers (go, npm, pip)

---

## Phase 5 — Scale & Ecosystem (Months 8–12+)

### 5.1 Advanced Cryptography
- [ ] Recursive proofs: prove a proof is valid without re-executing
- [ ] Proof aggregation across agents: combined attestation for multi-agent decisions
- [ ] Threshold signatures: N-of-M agents must sign for critical decisions
- [ ] Verifiable computation: prove arbitrary computation, not just AI decisions
- [ ] Post-quantum readiness: lattice-based proof schemes

### 5.2 Cross-Chain
- [ ] EVM verifier contracts (Solidity)
- [ ] Solana verifier program
- [ ] Cross-chain proof relay: prove on Casper, verify on Ethereum
- [ ] Universal proof format: chain-agnostic proof standard

### 5.3 AI Model Governance
- [ ] Model registry: on-chain record of all AI models used in proofs
- [ ] Model update proofs: prove new model is derivative of audited model
- [ ] Bias detection: verifiable fairness metrics for AI decisions
- [ ] Regulatory model cards: on-chain compliance metadata

### 5.4 Enterprise Features
- [ ] Private proof networks: permissioned deployment for banks
- [ ] Custom verification logic: pluggable verifier contracts
- [ ] Compliance dashboards: audit-ready reporting
- [ ] SLA: 99.99% uptime, <5s proof generation, <2s verification
- [ ] HSM integration: hardware-secured proof signing

### 5.5 Standards Contribution
- [ ] Propose Casper CEP for verifiable AI proofs standard
- [ ] Contribute to W3C Verifiable Credentials working group
- [ ] Open-source ZK circuits for common use cases
- [ ] Academic paper: verifiable AI decision proofs on blockchain

---

## Key Metrics

| Metric | Hackathon | MVP | Production | Scale |
|--------|-----------|-----|------------|-------|
| Proof types | Merkle + SHA-256 | + batch + sparse | + ZK (PLONK) | + recursive |
| Use cases | KYC only | + credit, contracts | + trading, content | All AI decisions |
| Chains | Casper Testnet | Casper Mainnet | + 1 EVM | 4+ chains |
| Proofs/month | demo | 10K | 500K | 10M+ |
| Verification time | <100ms local | <2s on-chain | <1s on-chain | <500ms |
| Test coverage | 65% | 80% | 90% | 95% |
