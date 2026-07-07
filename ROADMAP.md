# CasperProver — Roadmap

> Cryptographic proof engine for AI inference verification on Casper Network

---

## Shipped (Hackathon)

- [x] 4 contracts on Casper testnet (proof-registry, verifier-gate, defi-mock, stake-slashing)
- [x] Go API server with PostgreSQL persistence
- [x] Merkle tree builder (SHA-256, configurable depth)
- [x] Proof registration and on-chain verification
- [x] KYC whitelisting via Merkle inclusion (survives server restart)
- [x] DeFi vault gated by verifier-gate (admin-only `grant_access`, typed `AccountHash`)
- [x] React lab at casperprover.xyz with 9 interactive tabs
- [x] SDK (Go client + MCP server, 27/27 tools wired 1:1 to real API)
- [x] 83+ tests (62 Go + 21 Rust)
- [x] CI via GitHub Actions (engine test/vet/lint + SDK test/lint + contracts build/test)
- [x] API-key auth for mutating endpoints
- [x] Rate limiting (60 req/min per IP) + 1MB body limit
- [x] 200+ successful testnet transactions

## Phase 2 — Core Upgrades (completed items checked)

- [x] Post-quantum proof signing — real Ed25519 + ML-DSA-65 (FIPS 204) + Lamport OTS, wired to `/pq/*` endpoints
- [x] STARK aggregation — hash-based STARKPack simulation, wired to `/aggregation/*` with real batch registry
- [x] Groth16 zk-SNARK verifier — real BN254 pairing via `gnark`/`gnark-crypto` at `/zk/groth16-real/*` (MiMC preimage circuit)
- [x] Stake-slashing contract — real CSPR economic penalty for revoked proofs (20% slash, permissionless bounty)
- [x] SDK rewrite — Go client 1:1 mapped to real API routes, separate module with CI
- [x] MCP server — real entry point at `sdk/cmd/mcpserver`, all 27 tools wired
- [x] KYC whitelist persistence — rehydrates from Postgres on restart
- [x] Aggregation batch registry — real in-memory state (create/add/finalize/verify)
- [x] `defi-mock` hardening — typed `AccountHash`, duplicate whitelist prevention, admin gate
- [ ] Proof-of-Inference contract — model_hash + input_hash + output_hash on-chain
- [ ] Model hash commitment — SHA-256(architecture + weights + hyperparams) before inference
- [ ] ModelRegistry contract — on-chain model binding to prevent model-swap attacks
- [ ] Layerwise ZK (NANOZK pattern) — per-layer transformer proof
- [ ] ZK-KYC whitelisting factory (CEP-86, zk-SNARK upgrade)
- [x] MCP server expansion — all 27 tools wired (inference, aggregation, ZK, PQ crypto)
- [ ] Demo/Real data toggle in lab

## Phase 3 — Advanced

- [ ] Multi-model proof chains — DAG of proofs with topological validation
- [ ] Full on-chain ZK-SNARK verifier (Groth16/PLONK in Wasm contract)
- [ ] Hardware attestation support (TPM 2.0, Intel SGX, AMD SEV, ARM TrustZone)
- [ ] Distributed prover workers with MPC threshold proving
- [ ] Trusted setup ceremony manager
- [ ] EVM compatibility layer (Solidity verifier contract)
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
