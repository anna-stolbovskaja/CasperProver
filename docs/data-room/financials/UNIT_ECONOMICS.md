# Unit Economics (draft)

Per-verification COGS decomposition. Numbers land here when they are
measurable against real traffic; today the page documents the
components and the current known ranges.

## Cost components per verification

| Component | Order-of-magnitude | Comment |
|---|---|---|
| ZK prove (BN254 Groth16 over the current MiMC-preimage circuit) | milliseconds of CPU per proof | Real gnark, measured locally |
| PQ signature (Ed25519 + ML-DSA-65) | sub-millisecond | Real NIST-standardised implementation |
| Merkle tree construction | sub-millisecond for typical batch | SHA-256, size proportional to receipt count |
| Postgres write (receipt row) | few hundred µs | Hot storage |
| Object-storage write (evidence blob) | few ms + network | Cold storage |
| Casper anchor gas | flat per aggregation batch | Amortised across the batch |

## Gross-margin thesis

At batch amortisation (many receipts per anchor), the marginal cost
per verification is dominated by compute + storage, and the anchor
gas becomes a small line. Batch size is the primary lever.

## Where the numbers land

Once a partner has ≥ 30 days of live traffic, this page swaps the
"order of magnitude" column for a measured p50 / p95 and links to the
observability dashboard that regenerates it. See
`docs/roadmap/SLO.md` for the metric taxonomy.

## Cross-references

- `docs/data-room/financials/MODEL.md`
- `docs/roadmap/SLO.md`
- `docs/data-room/product/ARCHITECTURE.md`
