# Submission

## Project

**CasperProver** — Verifiable proof layer for AI agent computations on Casper Network

## Links

| Item | URL |
|------|-----|
| GitHub | https://github.com/anna-stolbovskaja/CasperProver |
| Demo Video | _pending_ |
| Landing Page | _pending_ |
| Testnet Contract | _pending_ |
| Casper Explorer TX | _pending_ |

## Track

Agentic Infrastructure

## Team

Solo developer: anna-stolbovskaja

## Summary

CasperProver generates, stores, and verifies Merkle-anchored cryptographic proofs of AI agent computations. Given any computation f(x) = y with model M, it produces a proof that can be verified on-chain without replaying the computation. The system includes a KYC whitelisting demo where DeFi contracts gate access based on verified proofs.

## On-Chain Components

- `proof-registry.wasm` — Proof storage, retrieval, revocation, agent reputation
- `verifier-gate.wasm` — On-chain proof verification, batch checking
- `defi-mock.wasm` — KYC-gated DeFi access using verified proofs

## Tech Stack

- Go 1.22 (proof engine, API server, CLI)
- Rust (Casper smart contracts, 3 contracts)
