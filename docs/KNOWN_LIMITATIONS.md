# Known Limitations

> Last verified 2026-07-07 against live testnet state (via CSPR.cloud) and current source, not just re-copied from a prior version of this doc.

## Smart Contracts

- **`grant_access` (defi-mock) is admin-gated in the deployed source** — verified in `contracts/defi-mock/src/main.rs`: the caller must equal `get_admin()` or the call reverts with `ERR_UNAUTHORIZED`. A previous version of this doc claimed there was no access control here; that was stale/incorrect.
- **String-based key matching** — Dictionary keys use `AccountHash::to_string()` which may differ from client-supplied strings. Raw byte comparison would be more reliable.
- **No input length validation** — String parameters (proof IDs, hashes) have no maximum length check. Very large inputs could increase gas costs.
- **`defi-mock` does not prevent duplicate whitelisting** — Calling `grant_access` twice overwrites the previous entry without error.

## Engine

- **No authentication on API** — The HTTP server accepts requests from any client without API keys or JWT verification.
- **No persistent proof storage** — Proofs are stored in memory. Server restart loses all data. A database backend (SQLite or PostgreSQL) is needed for production. (Model registry, KYC whitelist, and proof storage schemas are now auto-created by `store.Open()` if `DATABASE_URL` is set — see `internal/store/pg.go` — but the in-memory engine is still the primary source of truth at runtime.)
- **Hardcoded hash algorithm** — SHA-256 is the only supported hash. The config field `hash_algorithm` is present but not wired to alternative implementations.
- **Go engine cannot be cross-compiled to WASM** — The engine runs as a native binary only.
- **`/zk/verify-groth16` uses a conceptual simulation, not real BN254 pairing math** — see `internal/zkverifier/groth16.go`'s own doc comment. Real Groth16 verification needs a library like `gnark`.
- **Post-quantum signing (`internal/crypto`) and STARK aggregation (`internal/aggregator`) are conceptual simulations, not wired into any API endpoint or the main binary** — scaffolding for the roadmap items, not a live/demoable feature yet.
- **Model registry (`internal/model`, `internal/inference.RegisterModel`) works end-to-end against a real Postgres and the on-chain `model-registry` contract's `register_model` entry point**, but as of 2026-07-07 nothing in the frontend/API surface calls `RegisterModel` yet — the engine-side wiring is real, but it's not reachable from the deployed product.

## General

- **Demo video pending** — Video script is ready but recording is not completed.
- **All 3 contracts (proof-registry, verifier-gate, defi-mock) are live on Casper testnet** — verified directly against `https://api.testnet.cspr.cloud/contracts/{hash}` for the hashes in `README.md` (deployed 2026-06-29, block heights 8338239/8338279/8338296). A previous version of this doc claimed testnet deployment was still pending; that was stale/incorrect — always re-verify this kind of claim against on-chain state before trusting either this file or the README.
- **CI was red for 4+ days (since commit `3a829f4`, 2026-07-03) before 2026-07-07** — several packages added since then (`internal/model`, `internal/worker`, `internal/crypto`, `internal/aggregator`, `internal/inference`) failed to compile, and their test files referenced functions that never existed in the corresponding source files (evidence the tests were never actually run before being committed). Fixed 2026-07-07: build, vet, and all engine tests are green again; see git history for the specific fixes (real bugs found and fixed: a key-generation panic in `internal/crypto`, a hash-length overflow, and several fabricated/incorrect test expectations replaced with values verified against the real implementation).
