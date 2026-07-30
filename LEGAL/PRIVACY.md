# CasperProver — Privacy Policy

> **Status: DRAFT — self-authored, not reviewed by counsel or by a DPO.**
> This document is a good-faith draft of CasperProver's privacy posture
> prepared by the project maintainers. It has **not** been reviewed by
> qualified legal counsel and must **not** be relied upon as legal
> advice. It will be replaced with a counsel-reviewed version before any
> commercial launch. See `docs/MAINNET_LAUNCH_PLAN.md` (Pack AK) for the
> paid-legal-review milestone.

**Effective date (draft):** 2026-07-30
**Version:** 0.1-draft
**Data controller (draft):** CasperProver project maintainers
**Contact:** khrol.studio@gmail.com

Companion documents:

- `LEGAL/DATA_PROTECTION.md` — architectural and technical detail on the
  hash-only boundary and retention schedule; treat this Privacy Policy
  as the *user-facing* summary and that document as its *reference
  implementation*.
- `LEGAL/DPA.md` — the Data Processing Agreement template used when the
  Service is engaged as a data processor.
- `LEGAL/RETENTION.md` — canonical retention schedule (single source of
  truth; both this Policy and the Data Protection Notice reference it).

---

## 1. Scope

This Privacy Policy covers personal data processed by the CasperProver
maintainers in connection with:

- The hosted HTTP APIs, dashboards, and MCP endpoints operated at
  `casperprover-api-ylsh.onrender.com` and any successor domain.
- The Casper Network testnet smart contracts deployed by the project.
- The public documentation and marketing site.

The Service is designed to process *cryptographic commitments* to AI
decisions, not the underlying prompts, model weights, or personal
content. See §3.

## 2. What we collect and why

| Category | Examples | Purpose | Legal basis (draft) |
|---|---|---|---|
| Account / contact | Email of maintainer or API-key holder | Support, billing, security notices | Contract; legitimate interest |
| API key hashes | SHA-256 of raw API keys | Authenticate calls without storing the raw secret | Contract; legitimate interest |
| Operational telemetry | Timestamps, HTTP status, endpoint path, request-id, IP address | Rate-limiting, security, service quality | Legitimate interest |
| Merkle roots and receipt digests | 32-byte hashes | The Service's core function | Contract |
| Support correspondence | Emails you send us | Answer your question | Legitimate interest |

We do **not** collect:

- Raw AI prompts or completions.
- Model weights or intermediate activations.
- Personal identifiers embedded in prompts (they are hashed before they
  ever reach the anchor path).
- Payment card numbers (any future billing runs through a PCI-compliant
  processor; CasperProver never sees the PAN).

## 3. The hash-only architectural boundary

The Service's most important privacy property is that raw content stays
on the caller's side. What crosses the wire and what is anchored on
Casper Network are Merkle roots and receipt digests — one-way SHA-256
hashes of the caller's data.

If you never send us the raw content, we cannot lose it, subpoena it,
or leak it. This design is documented in
`LEGAL/DATA_PROTECTION.md` §3 and enforced by the engine's public API
surface (see `docs/API_REFERENCE.md`).

Caveat: metadata is *not* content. IP addresses, timestamps, endpoint
paths, and user-agent strings are legitimate operational telemetry and
are covered by §2 above and by the retention schedule in
`LEGAL/RETENTION.md`.

## 4. Cookies and similar

The Service does not set third-party marketing cookies. It sets
session and CSRF cookies where a dashboard is served; these are
strictly necessary and do not require consent under UK/EU rules.

## 5. Sharing and sub-processors

Sub-processors that could touch personal data:

| Sub-processor | Purpose | Data category | Location |
|---|---|---|---|
| Render (application hosting) | API runtime | Operational telemetry | US (default region) |
| Vercel (dashboard hosting) | UI runtime | Operational telemetry | Global CDN |
| GitHub (source hosting) | Development | Public issues / PRs only | US |
| NowNodes / Casper Node RPC | Anchor submission | Anchor transactions, no PII | Mixed |
| Casper Network validators | Consensus | On-chain anchors only | Global |

We do not sell personal data. We do not share it with advertisers.

## 6. International transfers

Where personal data of EU/UK residents is transferred outside the
EEA/UK (for example to a US sub-processor), we rely on the appropriate
transfer mechanism (Standard Contractual Clauses with the UK Addendum,
or the successor mechanism in force at the time). The current list of
sub-processors and their locations is in §5.

## 7. Retention

Full schedule in `LEGAL/RETENTION.md`. Summary:

- Support correspondence: 2 years after last contact.
- Operational telemetry (application logs): 90 days rolling.
- Audit trail (privileged actions): 400 days.
- API key hashes: for the lifetime of the contract + 30 days.
- Merkle roots and receipts (Postgres): 400 days.
- Merkle roots and receipts (cold storage): 7 years, immutable.
- Backups: 30 days rolling, cross-region.

At the end of retention we cryptographically shred by destroying the
wrapping DEK (see `docs/roadmap/KEY_MANAGEMENT.md`); ciphertext becomes
unrecoverable. Documented per NIST SP 800-88 rev 1.

## 8. Your rights

Where UK / EU GDPR applies to you, you have the right to:

- Access the personal data we hold about you.
- Rectify inaccuracies.
- Erase (subject to overriding legal obligations — anchor transactions
  on Casper Network are immutable by design and cannot be erased; we
  will delete off-chain metadata that references you).
- Restrict processing.
- Object to processing based on legitimate interest.
- Portability (structured, common, machine-readable export).
- Withdraw consent where processing is consent-based.
- Complain to a supervisory authority (in the UK: ICO; in the EU: your
  national DPA).

To exercise a right, email `khrol.studio@gmail.com`. We will respond
within 30 days. If we cannot meet a request (for example because
irreversible on-chain state is involved), we will explain why.

## 9. Security

Detailed in `LEGAL/DATA_PROTECTION.md` §9 and `docs/SECURITY_AUDIT.md`.
Summary: encryption in transit and at rest, hash-only boundary,
per-tenant API-key rotation on demand, m-of-n approval for privileged
operations, third-party audit planned (`docs/roadmap/LEGAL.md`
§Third-party cryptography / security audit).

## 10. Children

The Service is not directed to children under 16 and we do not
knowingly collect their personal data.

## 11. Changes

We keep prior versions of this Policy in git history. Material changes
will be notified via API-key contact email and posted at the top of
this file with a new *Effective date*.

## 12. Contact

- Email: `khrol.studio@gmail.com`
- Data controller (draft): CasperProver project maintainers
- Data subject requests: same email; response target 30 days.

---

**Draft label reminder.** This is a good-faith draft written by the
project maintainers, not by a solicitor. It exists so that anyone
evaluating the Service can see the intended privacy posture before the
counsel-reviewed version lands. See `docs/roadmap/LEGAL.md`.
