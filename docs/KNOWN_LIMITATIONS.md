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

## `slash_equivocation` entrypoint (spec, non-deployed)

`contracts/stake-slashing/SLASH_EQUIVOCATION_DRAFT.md` specifies the
minimum-viable equivocation-slashing extension to the existing `stake-slashing`
contract. Ships no code and authorises no redeploy — per invariant "не
редеплоим до аудита" a live contract change requires G2 sign-off.

Highlights: two conflicting proofs (same input_hash + model_id, different
output_hash) are self-witnessing evidence; entrypoint is permissionless
(anyone can call); slash is 50% of current stake (stricter than the 20% of
`report_and_slash`, because equivocation is the strictest failure mode);
composite key `min(a,b)|max(a,b)` prevents order-swap replay; optional
`evidence_hash` lane is reserved for future dispute-resolver, NOT in MVP.
