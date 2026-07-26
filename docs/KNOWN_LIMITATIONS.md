# Status & Roadmap

> Current state as of 2026-07-07. All systems live on Casper testnet.

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
- Post-quantum signing: SPHINCS+ (NIST PQC), hybrid Ed25519 + ML-DSA-65 (FIPS 204)
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
- Formal verification of proof pipeline
- Mainnet deployment with gas optimization

## Hash primitive posture

`docs/HASH_ALGORITHM_ANALYSIS.md` catalogues every hash usage in the tree with
its security property, honesty label, post-quantum posture, and migration
urgency. Key open questions routed from that document (do not close without
reviewing them):

- **Q1 domain-sep prefixes** — confirm Merkle internal nodes and receipt hashes
  use distinct domain-separation prefixes. Missing prefix = real defect.
- **Q2 canonical serialisation** — confirm receipt canonical serialisation
  carries version + purpose tags before payload.
- **Q3 SHA-256 monoculture** — acceptable under current review; any diversity
  change deferred to G2.
- **Q4 SLH-DSA parameter set** — production target (`SHA2-192s` vs
  `SHA2-256s`) deferred to G2.
- **Q5 Poseidon-family immaturity** — any STARK-based ZK-ML prototype stays
  SIMULATION until independent cryptanalytic review.
- **Q6 HKDF info labels** — every KDF call site must be catalogued with its
  label. Missing labels = defect.
- **Q7 length-extension surface** — confirm no `H(key || message)` MAC
  substitute exists anywhere in the tree.
