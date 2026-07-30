# Key Management (pointer)

Canonical source: `docs/roadmap/KEY_MANAGEMENT.md`.

## What an investor should know in one paragraph

Every signing / encryption / anchor key has a documented storage tier,
rotation cadence, and end-of-life mechanism. Root of trust is an HSM
(YubiHSM 2 on-prem, cloud KMS root for hosted); data-encryption keys
are wrapped and short-lived; privileged operations require m-of-n
human approval; every operation is audit-logged and shipped to cold
storage nightly. Full customer-managed keys (BYOK) and tenant
threshold signing are on the roadmap, not in production today.

## Cross-references

- `docs/roadmap/KEY_MANAGEMENT.md`
- `LEGAL/DATA_PROTECTION.md` §9 Security of processing
- `docs/runbooks/break-glass.md`
