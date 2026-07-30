# Metrics

> Cohort retention, verifications per month, API uptime, pipeline —
> rolled up monthly once traffic is live. Until then, this page tracks
> the *infrastructure metrics* that a partner can independently verify
> today.

## Live infrastructure metrics (verifiable today)

| Metric | Value | How to verify |
|---|---|---|
| Deployed contracts | 9 | `docs/TX_MANIFEST.md` |
| API endpoints | 32 | `docs/API_REFERENCE.md` |
| SDKs published | 3 (TS / Python / Go) | npm `@casperprover/sdk`, PyPI `casperprover-sdk`, Go module `github.com/anna-stolbovskaja/casperprover/sdk` |
| MCP tools | 32 | `mcp/README.md` |
| Interactive frontend tabs | 11 | `frontend/README.md` |
| Static security audits | 1 (all 9 contracts) | `docs/SECURITY_AUDIT.md` |

## Monthly roll-up (populated when traffic is live)

| Month | Verifications | Anchor tx | API uptime | Notes |
|---|---|---|---|---|
| (pending first partner) | | | | |

## Cohort retention (populated when ≥ 1 partner has ≥ 30 days of data)

Empty until then.

## Pipeline (populated when it is more than a Slack thread)

Empty until then.

## Content standard

Every number here must come with either (a) a repo path where the
artefact lives, (b) a dashboard link, or (c) the exact query that
regenerates it. No headline metric without a source.
