# SLO (pointer)

Canonical source: `docs/roadmap/SLO.md`.

## What an investor should know in one paragraph

CP publishes availability, latency, and correctness SLOs and ships a
rule-set that alerts on breach (`deploy/observability/alerts/`).
Alerts are `promtool`-clean with unit tests, but until commercial
launch they route to a null receiver (single-maintainer mode, no paid
pager rotation).

## Cross-references

- `docs/roadmap/SLO.md`
- `docs/OPS_RUNBOOKS.md`
- `docs/KNOWN_LIMITATIONS.md`
