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

## Off-repo design plans (`SIMULATION` / DRAFT)

These plans ship no code and change no runtime behaviour. They exist so the
honesty ladder is discoverable from the front door.

- `docs/ZKML_RESEARCH_SPIKE.md` — full landscape survey and feasibility matrix
  for ZK-ML proving approaches (Groth16/PLONK-family, STARK/FRI, zkVM,
  lookup+PLONK, recursion). Non-endorsement, non-procurement.
- `docs/ZKML_HONEST_VERDICT.md` — single-page decision record explaining why
  every ML-inference claim in the tree is currently labelled `SIMULATION` and
  what four conditions would all have to hold before any of them could
  honestly be relabelled `REAL (ZK-ML)`. Bound to G2 in the mainnet launch
  plan.

### `SIMULATION` label — not negotiable until conditions met

Any claim that implies a cryptographic proof of a model's inference (as
opposed to an attestation of inputs, outputs, and a model identifier) is
labelled `SIMULATION`. Relabelling to `REAL (ZK-ML)` requires ALL of:

1. A named model with a compiled circuit and both **weights hash** and
   **circuit hash** published and anchored.
2. Independent third-party audit sign-off on both the circuit and the
   underlying IOP/lookup argument (reserved for mainnet-plan gate G2).
3. Per-inference proving cost that fits under the Challenger dispute cost
   ceiling from the reputation economics draft.
4. Receipt-format extension carrying circuit hash, verifying-key hash,
   weights hash, and toolchain version (breaking schema change; must be
   scheduled as such).

Skipping any one turns `REAL` into laundered `SIMULATION`. Explicitly rejected.
