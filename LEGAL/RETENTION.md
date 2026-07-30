# CasperProver — Retention Policy (canonical schedule)

> **Status: DRAFT — self-authored, not reviewed by counsel.**
> This document is the canonical retention schedule referenced by
> `LEGAL/PRIVACY.md`, `LEGAL/DATA_PROTECTION.md`, and `LEGAL/DPA.md`. It
> has **not** been reviewed by qualified legal counsel and must **not**
> be relied upon as legal advice. It will be replaced with a
> counsel-reviewed version before any commercial launch. See
> `docs/MAINNET_LAUNCH_PLAN.md` (Pack AK) for the paid-legal-review
> milestone.

**Effective date (draft):** 2026-07-30
**Version:** 0.1-draft
**Contact:** khrol.studio@gmail.com

---

## 1. Purpose

Every data class the Service touches has a retention window. This
document is the single source of truth for those windows so the
Privacy Policy, Data Protection Notice, DPA, and the engine's
data-lifecycle cron cannot drift apart.

The schedule mirrors `docs/roadmap/LEGAL.md` §Retention, plus the more
granular support-and-billing rows a signed customer would expect to
see.

## 2. Canonical schedule

| Data class | Retention | Storage | End-of-life |
|---|---|---|---|
| Decision receipts (Postgres) | 400 days | Hot; encrypted at rest with per-blob DEK | Cryptographic shred (destroy wrapping DEK) |
| Decision receipts (object storage) | 7 years | Cold; immutable; per financial-audit norms | Cryptographic shred |
| Merkle roots on Casper Network | Forever | On-chain | Immutable by design; cannot be shredded |
| Audit log (privileged actions) | 400 days | Append-only, KMS-signed | Cryptographic shred |
| API access log | 90 days | Hot | Roll-off after quota rollup completes |
| Webhook delivery log | 90 days | Hot | Roll-off after retry cap + investigation window |
| HITL evidence blobs | 7 years | Cold; content-addressed | Cryptographic shred |
| Tenant PII | Lifetime of contract + 30 days | Hot | Cryptographic shred |
| API-key hashes (SHA-256) | Lifetime of contract + 30 days | Hot | Row deletion + DEK shred |
| Support correspondence | 2 years from last contact | Email/ticket system | Purge from ticket system + DEK shred |
| Billing records | 7 years (tax record norms) | Cold; encrypted at rest | Cryptographic shred |
| Application backups | 30 days | Cross-region | Automatic expiry, DEK shred |
| Operational telemetry (metrics) | 90 days rolling | Prometheus / equivalent | Roll-off |
| Third-party audit reports | Forever (public) | Repo | Superseded by newer version, not deleted |
| Ceremony transcripts | Forever (public) | Repo + object storage | Immutable |
| Break-glass incident postmortems | Forever (internal) | Repo | Redacted for public, never deleted |

Retention windows are **maxima**; the Controller (in the DPA sense)
may configure shorter retention on request, subject to overriding
legal obligations (tax records, ongoing regulatory holds).

## 3. Enforcement

- **On-chain (Casper Network):** anchors are immutable by design and
  are not subject to this policy. The Service does not attempt to erase
  on-chain state; §7 of the Privacy Policy explains this to data
  subjects.
- **Off-chain (Postgres):** a data-lifecycle cron reads this schedule
  from a machine-readable copy (planned; see §4) and expires rows past
  their retention window. The cron logs every deletion into the audit
  log.
- **Object storage:** lifecycle rules configured per bucket, mirroring
  the table above.
- **Cryptographic shred:** the wrapping DEK is destroyed. Ciphertext
  remains but is unrecoverable. Documented per NIST SP 800-88 rev 1.

## 4. Machine-readable source

A YAML mirror of §2 lives at `deploy/retention/schedule.yaml`
(planned — see `docs/roadmap/LEGAL.md` acceptance criteria) so the
data-lifecycle cron and the Privacy Policy rendering both consume the
same source. Any change here must land in both files in the same
commit.

## 5. Legal holds

A legal hold suspends the retention timer for the affected data
class(es) until the hold is lifted. Holds are recorded in the audit
log with a case identifier and the requesting authority.

## 6. Backups

Backups run through the same retention pipeline: the wrapping DEK for
a backup expires with the backup. Restoring from a backup restores the
wrapping DEK from a live backup, not from an already-shredded one.

## 7. Cross-references

- User-facing summary: `LEGAL/PRIVACY.md` §7.
- Architectural detail: `LEGAL/DATA_PROTECTION.md` §7.
- Roadmap milestone: `docs/roadmap/LEGAL.md` §Retention, acceptance
  criteria "Retention enforced automatically by the engine's
  data-lifecycle cron".
- Key management (for DEK shredding): `docs/roadmap/KEY_MANAGEMENT.md`.

---

**Draft label reminder.** This is the maintainers' good-faith
schedule, not a counsel-reviewed policy. Do not treat it as legal
advice or as a compliance certification.
