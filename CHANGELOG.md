# changelog

## v0.3.0 (2026-06-29)

feat:
- 22 rust contract tests + 58 go engine tests (80 total)
- security: batch size limit, rate limiting on is_valid, whitelist revocation
- `revoke_access()` with tombstone pattern on defi-mock
- ci: go test/vet/lint + contract build/test pipeline

fix:
- `is_valid()` now checks rate limit before verification
- `batch_check()` capped at 50 entries
- input validation on verifier-gate

security:
- risk score reduced from 4/10 to 2/10 (avg across 3 contracts)

## v0.2.0 (2026-06-28)

feat:
- contracts deployed to casper testnet (proof-registry, verifier-gate, defi-mock)
- go prover engine with merkle tree, proof generation, local verification
- kyc/defi flow integration
- docs and landing page

## v0.1.0 (2026-06-28)

- initial project structure
