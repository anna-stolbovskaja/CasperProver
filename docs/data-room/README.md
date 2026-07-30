# CasperProver — Investor Data Room (index)

> **Status: SCAFFOLD.** This is the initial structure of the investor
> data room described in `docs/roadmap/DATA_ROOM.md`. The scaffold is
> committed so the shape is visible to reviewers; individual sections
> are filled progressively per the milestones in the roadmap doc. Each
> file below is either a real artefact, a stub with a documented next
> step, or a symlink-substitute pointer to the canonical source of
> truth elsewhere in the repo.

**Audience.** Seed / seed-extension investors evaluating CasperProver as
"AI accountability infrastructure" for a regulated buyer profile.

**Constraint from the roadmap.** Everything in the data room is *plan*
or *artefact*, not vapor. Every claim links to something a reviewer can
open. No hand-waving, no aspirational metrics.

---

## Structure

```
docs/data-room/
  README.md              — this index
  company/               — one-pager, team, timeline
  product/               — architecture, roadmap, demo, honesty labels
  market/                — thesis, ICP, competition, sizing
  traction/              — design partners, metrics, log
  financials/            — model, unit economics, hiring plan
  legal/                 — cap table (redacted), IP, audit reports
  ops/                   — SLO, key management, status page
```

---

## Company

- [`company/ONE_PAGER.md`](company/ONE_PAGER.md) — the 1-page summary.
- [`company/TEAM.md`](company/TEAM.md) — founders, advisors, open roles.
- [`company/TIMELINE.md`](company/TIMELINE.md) — product + company
  milestones.

## Product

- [`product/ARCHITECTURE.md`](product/ARCHITECTURE.md) — investor-language
  version of `docs/ARCHITECTURE.md`. See the canonical source there.
- [`product/ROADMAP.md`](product/ROADMAP.md) — pointer to
  `docs/roadmap/30-DAY.md` and `docs/roadmap/90-180-DAY.md`.
- [`product/DEMO.md`](product/DEMO.md) — how to reproduce the demo in
  under 10 minutes.
- [`product/HONESTY.md`](product/HONESTY.md) — pointer to
  `docs/PQ_HONESTY.md` and `docs/KNOWN_LIMITATIONS.md`. Overclaims are
  the fastest way to lose an evaluator's trust; we do not hide the
  labels.

## Market

- [`market/THESIS.md`](market/THESIS.md) — why regulated AI/DeFi needs
  a proof primitive.
- [`market/ICP.md`](market/ICP.md) — 2–3 named ideal customer archetypes.
- [`market/COMPETITION.md`](market/COMPETITION.md) — competitors and how
  we differ. No name-drop without a linked artefact.
- [`market/TAM_SAM_SOM.md`](market/TAM_SAM_SOM.md) — market sizing with
  methodology, not a headline.

## Traction

- [`traction/DESIGN_PARTNERS.md`](traction/DESIGN_PARTNERS.md) — signed
  partners, status, case studies.
- [`traction/METRICS.md`](traction/METRICS.md) — cohort retention,
  verifications/month, API uptime, pipeline; rolled up monthly.
- [`traction/LOG.md`](traction/LOG.md) — dated log of major events.

## Financials

- [`financials/MODEL.md`](financials/MODEL.md) — bottom-up model
  (pricing × usage × cost). Landed as a spreadsheet in the same folder
  when the first partner signs.
- [`financials/UNIT_ECONOMICS.md`](financials/UNIT_ECONOMICS.md) —
  per-verification COGS, gross-margin walk.
- [`financials/HIRING_PLAN.md`](financials/HIRING_PLAN.md) — 12-month
  hiring plan.

## Legal

- [`legal/CAP_TABLE.md`](legal/CAP_TABLE.md) — anonymised until a term
  sheet is in play.
- [`legal/IP.md`](legal/IP.md) — code and patents; points at
  `docs/PQ_HONESTY.md` §patents.
- [`legal/AUDIT_REPORTS/`](legal/AUDIT_REPORTS/) — third-party audits as
  they land. Empty until the RFP in `docs/roadmap/LEGAL.md` closes.

## Ops

- [`ops/SLO.md`](ops/SLO.md) — pointer to `docs/roadmap/SLO.md`.
- [`ops/KEY_MANAGEMENT.md`](ops/KEY_MANAGEMENT.md) — pointer to
  `docs/roadmap/KEY_MANAGEMENT.md`.
- [`ops/STATUS_PAGE_URL.txt`](ops/STATUS_PAGE_URL.txt) — public status
  page URL (placeholder until Pack AH's status page is live externally).

---

## Content standards (from the roadmap)

- **Numbers require a source.** Every metric line includes the query,
  dashboard link, or artefact hash.
- **Comparisons require artefacts.** "We are faster than X" needs a
  benchmark file committed to the repo.
- **No name-dropping.** External vendors (audit firms, chain analytics,
  RPC providers) must not appear here without a signed contract or a
  linked dashboard.
- **Redaction policy.** Design partners can request their name or
  metrics be redacted; the redacted version is what goes into the data
  room.

---

## Milestones (from `docs/roadmap/DATA_ROOM.md`)

1. Structure + one-pager (5 days) — **in progress with this scaffold**.
2. Market thesis + ICP + competition (10 days).
3. Design-partner section (10 days) — requires ≥ 1 signed partner.
4. Metrics roll-up cron (5 days).
5. Financial model (15 days).
6. Ops section symlinks + status-page URL (2 days) — pointers in place;
   awaiting external status page.
7. First investor share (soft — advisor review) (5 days).
8. First investor share (external — target VC list) (10 days).

---

## Acceptance criteria (from the roadmap)

- [x] `docs/data-room/README.md` renders as a valid index.
- [ ] Every claim in the data room has a linked artefact.
- [ ] Every roadmap doc that has a data-room counterpart is
      symlinked or explicitly cross-referenced.
- [ ] At least one external investor has read the data room and
      returned feedback captured in
      `docs/data-room/traction/LOG.md`.
