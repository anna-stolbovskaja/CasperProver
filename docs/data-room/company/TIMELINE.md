# Timeline

Dated log of major product + company milestones. Kept append-only.

## Product milestones

| Date | Milestone | Evidence |
|---|---|---|
| 2026-07-20 | Static security audit complete, ownership cheat-sheet published | `docs/SECURITY_AUDIT.md` (Gate 1) |
| 2026-07-25 | `proof-of-inference`, `model-registry`, `proof-aggregation` deployed to Casper testnet | `docs/TX_MANIFEST.md` |
| 2026-07-25 | Deployment verification 8/8 pass locally without secrets | `verify.sh`, `docs/roadmap/DEPLOYMENT_VERIFY_2026-07-25.md` |
| 2026-07-26 | `governance` deployed (48h timelock + 2-of-3 guardian recovery) | `docs/roadmap/GOVERNANCE_DEPLOY_2026-07-26.md` |
| 2026-07-27 | `zk-verifier` deployed, P0 `governance_approved` bypass fixed and live-regressed on-chain | `docs/SECURITY_AUDIT.md` §2.10 |
| 2026-07-28 | Ownership cheat-sheet embedded in `docs/ARCHITECTURE.md`; ops runbook `docs/OPS_RUNBOOKS.md` §4.3 for guardian unpause landed | `docs/SECURITY_AUDIT.md` §4 |
| 2026-07-27 | TS SDK `0.1.2`, Python SDK `0.1.2`, Go SDK `0.1.2` published (npm / PyPI / Go proxy) | `sdk/PUBLISHING.md`, tags `sdk-ts-v0.1.2`, `sdk-py-v0.1.2`, `sdk/v0.1.2` |
| 2026-07-30 | LEGAL/ surface expanded (PRIVACY, DPA, RETENTION drafts) and data-room scaffold committed | this branch, `docs/AUDIT_FOLLOWUPS_2026-07-30.md` |

## Company milestones

| Date | Milestone | Evidence |
|---|---|---|
| (populate as they land) | | |

## Cadence

- Product milestones append to this table on landing.
- Company milestones (fundraise, hires, design-partner signings, audit
  contracts) append here on execution, not on intent.
