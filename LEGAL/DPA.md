# CasperProver — Data Processing Agreement (DPA) — Template

> **Status: DRAFT — self-authored template, not reviewed by counsel.**
> This template exists so that a prospective customer can see the
> intended shape of a CasperProver DPA before the counsel-reviewed
> version lands. It has **not** been reviewed by qualified legal counsel
> and must **not** be relied upon as legal advice or executed as-is. It
> will be replaced with a counsel-reviewed version before any commercial
> engagement. See `docs/MAINNET_LAUNCH_PLAN.md` (Pack AK) for the
> paid-legal-review milestone.

**Effective date (draft):** 2026-07-30
**Version:** 0.1-draft
**Contact:** khrol.studio@gmail.com

Companion documents:

- `LEGAL/PRIVACY.md` — Privacy Policy (user-facing).
- `LEGAL/DATA_PROTECTION.md` — architectural detail.
- `LEGAL/RETENTION.md` — canonical retention schedule.

---

## 1. Parties

- **Controller** — the customer entity engaging CasperProver as a data
  processor.
- **Processor** — the CasperProver maintainers, referred to below as
  "CP".

## 2. Subject matter and duration

CP processes personal data on behalf of the Controller solely to
provide the Service as defined in `LEGAL/TOS.md`. Processing continues
for the term of the underlying agreement plus the retention windows in
`LEGAL/RETENTION.md`.

## 3. Nature and purpose of processing

- Nature: authenticated API access to CP's anchoring, verification, and
  receipt-storage endpoints.
- Purpose: producing cryptographically verifiable AI-decision receipts
  on behalf of the Controller.

## 4. Categories of data subjects

- The Controller's end-users, insofar as any personal data appears in
  data the Controller chooses to submit.
- The Controller's own personnel (API-key holders, admins).

## 5. Categories of personal data

By design, CP does not process raw prompts or completions. It processes
Merkle roots and receipt digests, plus the operational telemetry listed
in `LEGAL/PRIVACY.md` §2. If the Controller submits any personal data
in a category listed in Article 9 GDPR (special-category data),
processing of that data is out of scope for the standard Service and
requires a bespoke amendment.

## 6. Obligations of the Processor

CP shall:

- Process personal data only on documented instructions from the
  Controller.
- Ensure persons authorised to process the data are under confidentiality.
- Implement the security measures described in
  `LEGAL/DATA_PROTECTION.md` §9 (Security of processing).
- Assist the Controller with data-subject requests to the extent
  reasonably possible given the architecture (see §8).
- Notify the Controller of a personal data breach without undue delay
  and no later than 48 hours after becoming aware, providing the
  information required by GDPR Article 33(3).
- Delete or return personal data at the end of the Service, subject to
  the retention windows in `LEGAL/RETENTION.md` and the immutable
  nature of on-chain anchors.
- Make available all information necessary to demonstrate compliance
  and allow audits under §10.

## 7. Sub-processors

CP relies on the sub-processors listed in `LEGAL/PRIVACY.md` §5. CP
will:

- Impose data-protection obligations on each sub-processor equivalent
  to those in this DPA.
- Give the Controller ≥ 30 days prior notice of any intended change
  in sub-processors and honour a reasonable objection.

## 8. Data-subject rights

The Controller is the primary interlocutor for its end-users. CP will
assist by providing:

- Extracts of the personal data CP holds about a data subject on the
  Controller's request (30-day SLA).
- Correction, deletion, or restriction of processing on the
  Controller's instruction, subject to two constraints:
  1. On-chain anchors are immutable by design and cannot be erased; CP
     will delete the off-chain material referencing the data subject.
  2. Where a request conflicts with an overriding legal obligation (for
     example an ongoing regulatory hold), CP will explain the conflict
     and hold processing to the minimum lawful scope.

## 9. International transfers

Where personal data leaves the UK/EEA, CP relies on Standard
Contractual Clauses with the UK Addendum, or the successor transfer
mechanism in force at the time. The current sub-processor locations
are listed in `LEGAL/PRIVACY.md` §5.

## 10. Audits

The Controller may audit CP's compliance with this DPA:

- Annually, by written request with ≥ 30 days notice.
- At any time following a personal data breach affecting the
  Controller.

CP will provide reasonable co-operation and — where the third-party
security audit under `docs/roadmap/LEGAL.md` has landed — the report
in lieu of a full on-site audit, at the Controller's option.

## 11. Retention and deletion

Per `LEGAL/RETENTION.md`. At the end of the Service, off-chain personal
data is either returned to the Controller in a structured export or
cryptographically shredded (see `docs/roadmap/KEY_MANAGEMENT.md`) at
the Controller's option.

## 12. Confidentiality and precedence

This DPA is confidential between the parties. In the event of conflict
between this DPA and the Master Services Agreement, this DPA controls
with respect to personal data.

## 13. Contact

- Data protection contact: `khrol.studio@gmail.com`
- Sub-processor change notifications: same channel.

---

**Draft label reminder.** This is a good-faith template, not a signed
agreement. Do not execute this document as-is; wait for the
counsel-reviewed replacement.
