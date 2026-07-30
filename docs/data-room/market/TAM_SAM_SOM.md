# TAM / SAM / SOM (methodology-first)

Numbers require a source (per content standards). This page lays out
the sizing *methodology*; per-year numbers land here only when they
are backed by a public dataset or a peer-reviewed research report.

## TAM

- **Definition.** Global spend on evidence, audit-trail, and
  logging-and-verification tooling that AI decisions in regulated
  workflows will need over a 5-year horizon.
- **Methodology (draft).** Estimate the AI-decision volume in
  regulated workflows (KYC transaction monitoring, credit scoring,
  claims triage, DeFi risk) × the fraction that must produce
  independent evidence under 2026-onward regulation × a per-decision
  price CP could sustainably charge at scale.
- **Sources to cite when the number lands.** MiCA final texts, EU AI
  Act annexes, FinCEN AI risk statements, NIST AI RMF, industry
  reports from named research firms.

## SAM

- **Definition.** The subset of TAM CP can serve in the next 24
  months with today's chain, today's languages (TS / Python / Go),
  today's regulatory posture (see `docs/data-room/legal/`).
- **Methodology (draft).** Filter TAM by chain reach (today: Casper,
  roadmap: EVM / Solana / Cosmos), language reach (today: three
  SDKs), and regulatory posture (today: MiCA / EU AI Act / GDPR
  / FinCEN / NIST AI RMF aligned; not HIPAA).
- **Sources.** Same as TAM plus CP's own roadmap
  (`docs/roadmap/*`).

## SOM

- **Definition.** What CP realistically captures in the next 12–18
  months at design-partner scale.
- **Methodology (draft).** 2–3 signed partners under the first-partner
  motion (`docs/data-room/traction/DESIGN_PARTNERS.md`) × their
  observable AI-decision volume × the per-decision price we can
  extract at partner scale (usually below the sustainable long-run
  price because the partner is discounted for design work).
- **Sources.** Signed contracts (redacted). Never modelled.

## Rule against headline numbers

A headline sizing number without the methodology, sources, and
partner artefacts above is a marketing claim, not a data-room asset.
This page does not carry those until they exist.

## Cross-references

- `docs/data-room/market/THESIS.md` — the why-now.
- `docs/data-room/market/ICP.md` — who buys.
- `docs/data-room/traction/DESIGN_PARTNERS.md` — status.
