# Financial Model (draft)

The bottom-up financial model lands here as a spreadsheet
(`MODEL.xlsx` or an equivalent open format) when the first partner
signs and there is real per-verification pricing to model against.

Until then, this page documents the *shape* of the model so a
reviewer can see the intended structure.

## Shape

- **Revenue drivers**
  - Verifications per month, per partner.
  - Blended price per verification (varies by SLA tier and PQ
    surface used).
  - Overage pricing above committed tier.
- **Cost drivers**
  - Casper anchor gas per aggregation batch.
  - Compute for gnark ZK proving (cores × seconds × cloud rate).
  - Storage for receipt + evidence blobs (hot Postgres +
    object-storage cold).
  - Sub-processor costs (Render, Vercel, RPC providers).
  - People (see `HIRING_PLAN.md`).
- **Contribution margin per partner**
  - Revenue − direct-attributable compute + storage + anchor gas.
- **Gross margin walk**
  - Contribution margin − allocated infra + observability + audit
    amortisation.

## Cross-references

- `docs/data-room/financials/UNIT_ECONOMICS.md` — per-verification
  COGS breakdown that feeds into revenue and cost drivers here.
- `docs/data-room/financials/HIRING_PLAN.md` — the 12-month personnel
  cost path.
- `docs/data-room/traction/DESIGN_PARTNERS.md` — the signed-contract
  inputs.
