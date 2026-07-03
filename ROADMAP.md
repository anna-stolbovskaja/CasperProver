# CasperProver — Roadmap

> Cryptographic proof engine for AI inference verification on Casper Network

---

## Shipped (Hackathon)

- [x] 3 contracts on Casper testnet (proof-registry, verifier-gate, defi-mock)
- [x] Go API server with PostgreSQL
- [x] Merkle tree builder (SHA-256, configurable depth)
- [x] Proof registration and on-chain verification
- [x] KYC whitelisting via Merkle inclusion
- [x] DeFi vault gated by verifier-gate
- [x] React lab at casperprover.xyz
- [x] SDK + MCP server
- [x] 83 tests (62 Go + 21 Rust)
- [x] CI via GitHub Actions

## Phase 2 — Core Upgrades

- [ ] Proof-of-Inference contract — model_hash + input_hash + output_hash on-chain
- [ ] Model hash commitment — SHA-256(architecture + weights + hyperparams) before inference
- [ ] ModelRegistry contract — on-chain model binding to prevent model-swap attacks
- [ ] Proof aggregation registry — N proofs → 1 root hash with Merkle inclusion
- [ ] Post-quantum proof signing (ML-DSA FIPS 204 + SPHINCS+ FIPS 205 backup)
- [ ] Recursive STARK aggregation (STARKPack pattern, Winterfell)
- [ ] Groth16 zk-SNARK verifier (gnark, optimistic mode with fraud proofs)
- [ ] Layerwise ZK (NANOZK pattern) — per-layer transformer proof, 5.5KB/24ms
- [ ] ZK-KYC whitelisting factory (CEP-86, zk-SNARK upgrade)
- [ ] MCP server expansion to 15+ tools with ProofOfInference JSON-Schema
- [ ] Demo/Real data toggle in lab

## Phase 3 — Advanced

- [ ] Multi-model proof chains — DAG of proofs with topological validation
- [ ] Full on-chain ZK-SNARK verifier (Groth16/PLONK in Wasm contract)
- [ ] Hardware attestation support (TPM 2.0, Intel SGX, AMD SEV, ARM TrustZone)
- [ ] Distributed prover workers with MPC threshold proving
- [ ] Trusted setup ceremony manager
- [ ] EVM compatibility layer (Solidity verifier contract)
- [ ] Proof complexity classifier (ML-based)
- [ ] Property-based testing (Go rapid) with proof system invariants
- [ ] ZK circuit constraint testing + fuzz testing
- [ ] Gas benchmarking suite

## Phase 4 — Mainnet & Standard

- [ ] Security audit
- [ ] Mainnet deployment
- [ ] Proof-of-Inference standard proposal (CEP-style)
- [ ] ZK circuit design whitepaper
- [ ] Integration with model registries (HuggingFace, Replicate)
- [ ] Cross-chain proof bridging
- [ ] Formal verification (TLA+ specification for proof system)

## Planned Infrastructure (stubs in codebase)

- `VerifierConfig` — Optimistic/ZK/Hybrid verification modes
- `ProverConfig` — distributed proving parameters
- `AttestationType` — Software/TPM/SGX/SEV/TrustZone enum
- `HWQuote` struct — hardware attestation quote format
- `ProofChain` + `ProofDAG` — multi-model pipeline proofs
- `TargetVM` enum — CasperVM/EVM abstraction
