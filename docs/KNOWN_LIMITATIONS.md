# Known Limitations

## Smart Contracts

- **No access control on `grant_access`** — Any caller can whitelist any user if they know a valid proof ID. Should restrict to the user themselves or an admin.
- **String-based key matching** — Dictionary keys use `AccountHash::to_string()` which may differ from client-supplied strings. Raw byte comparison would be more reliable.
- **No input length validation** — String parameters (proof IDs, hashes) have no maximum length check. Very large inputs could increase gas costs.
- **`defi-mock` does not prevent duplicate whitelisting** — Calling `grant_access` twice overwrites the previous entry without error.

## Engine

- **No authentication on API** — The HTTP server accepts requests from any client without API keys or JWT verification.
- **No persistent proof storage** — Proofs are stored in memory. Server restart loses all data. A database backend (SQLite or PostgreSQL) is needed for production.
- **Hardcoded hash algorithm** — SHA-256 is the only supported hash. The config field `hash_algorithm` is present but not wired to alternative implementations.
- **Go engine cannot be cross-compiled to WASM** — The engine runs as a native binary only.

## General

- **Demo video pending** — Video script is ready but recording is not completed.
- **Testnet deployment pending** — Contracts are compiled but not yet deployed to Casper testnet.
