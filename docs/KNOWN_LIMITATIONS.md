# Known Limitations

> Last verified 2026-07-07 against live testnet state and current source.

## Smart Contracts

- **`defi-mock` access control is admin-gated** — `grant_access` requires the caller to equal `get_admin()`, preventing unauthorized whitelisting. `is_whitelisted` uses typed `AccountHash` (`CLType::ByteArray(32)`) to avoid client-side formatting bugs.
- **`defi-mock` rejects duplicate whitelisting** — `grant_access` reverts with `ERR_ALREADY_WHITELISTED` (13) if the user is already active. Re-granting requires `revoke_access` first.
- **All 4 contracts verified live on testnet** — proof-registry (2026-06-29), verifier-gate (2026-06-29), defi-mock (2026-07-07), stake-slashing (2026-07-07).

## Engine

- **API key authentication** — `API_KEY` env var requires matching `X-API-Key` header on mutating requests. Read-only endpoints stay public. Single shared secret (intentional simplification for demo).
- **KYC whitelist persists across restarts** — rehydrated from Postgres on start via `LoadKYC()`.
- **Aggregation batch registry is in-memory** — batches do not survive a server restart (proofs and KYC do survive via Postgres). This is a known gap for the batch subsystem only.
- **SHA-256 is the only hash algorithm** — the `hash_algorithm` config field is present but not wired to alternatives.
- **`/zk/groth16-real/*` endpoints use real BN254 Groth16** — actual R1CS circuit via gnark, actual trusted setup, actual pairing checks. Scope: proves knowledge of a MiMC preimage, not a full model-inference circuit. Trusted setup is regenerated per server start (demo-grade, not a production ceremony).
- **`/zk/verify-groth16` uses a hash-based simulation** — provided alongside the real Groth16 endpoints for API completeness. Clearly labeled in responses.
- **STARK aggregation uses hash-based aggregation** — real STARK recursion requires libraries not yet mature in Go. The current implementation provides genuine hash-chain aggregation with verification.
- **Post-quantum signing uses real cryptography** — Ed25519 (Go stdlib) + ML-DSA-65/FIPS 204 (cloudflare/circl) + Lamport OTS. All wired to `/pq/*` endpoints with verified sign/verify round trips.

## SDK

- **Go SDK maps 1:1 to real API routes** — rewritten from scratch, tested against a fake server in CI.
- **MCP server has 12 of 22 tools wired** — remaining 10 tools return a clear "not implemented" error (no backing API endpoint exists for those yet).

## Frontend

- **Agent demo stale-closure bug fixed** — `useRef` pattern ensures step closures always read the latest `modelId`/`proofId`/`batchId` values.
- **Demo video pending** — video script is ready, recording not yet completed.

## General

- **Rate limiting** — 60 requests/min per IP, 1MB POST body limit.
- **Input validation** — agent name capped at 64 chars, input/output at 10KB, model at 256 chars, batch size 1–50.
