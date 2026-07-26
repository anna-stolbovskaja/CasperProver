# CasperProver — Roadmap

> Cryptographic proof engine for AI inference verification on Casper Network

---

## ✅ Shipped

- [x] 4 smart contracts on Casper testnet (proof-registry, verifier-gate, defi-mock, stake-slashing)
- [x] 4 additional contracts written (proof-of-inference, model-registry, proof-aggregation, stake-slashing-session)
- [x] Go API server — 32 endpoints, PostgreSQL persistence, rate limiting
- [x] Merkle tree builder (SHA-256, configurable depth, <50ms)
- [x] Real BN254 Groth16 ZK proofs via gnark (R1CS circuit, trusted setup, pairing verification)
- [x] Post-quantum signing: ML-DSA-65 (FIPS 204), hybrid Ed25519 + ML-DSA-65, Lamport OTS
- [x] Hash-chain batch aggregation with Postgres persistence
- [x] Proof-chain DAG validation (cycle detection, input continuity, single root)
- [x] KYC whitelisting via Merkle inclusion (cross-contract, persistent)
- [x] DeFi vault gated by verifier-gate (admin-only `grant_access`, typed `AccountHash`)
- [x] Stake-slashing — real CSPR economic penalty (20% slash, permissionless bounty)
- [x] Go SDK — 32 methods, 1:1 API mapping
- [x] MCP server — 32 tools with full InputSchema definitions
- [x] React lab at casperprover.xyz — 10 interactive tabs, all calling real API
- [x] Casper Wallet integration with demo fallback
- [x] 194 tests (172 Go + 22 Rust)
- [x] CI via GitHub Actions
- [x] API-key auth, rate limiting (60 req/min), input validation
- [x] 248+ testnet transactions

## 🔜 Next

- [ ] Deploy `proof-of-inference`, `model-registry`, `proof-aggregation` contracts
- [ ] Full model-inference ZK circuit (beyond MiMC preimage)
- [ ] Production trusted setup ceremony
- [ ] STARK recursive aggregation (pending mature Go STARK library)
- [x] Demo/Real data toggle in lab

## 🔮 Future

- [ ] Multi-chain anchoring (EVM, Solana, Cosmos)
- [ ] Hardware attestation (TPM 2.0, Intel SGX, AMD SEV)
- [ ] Distributed prover network with MPC threshold
- [ ] Proof marketplace for verified computation
- [ ] Mainnet deployment with gas optimization
- [ ] Formal verification (TLA+ specification)
- [ ] Cross-chain proof bridging
- [ ] Integration with model registries (HuggingFace, Replicate)
