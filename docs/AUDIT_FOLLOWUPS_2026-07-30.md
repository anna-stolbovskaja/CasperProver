# Audit Follow-ups — 2026-07-30

Documentation-only follow-up branch. Everything below lands as
`docs/`-scope changes: no contract redeploys, no CI wiring changes, no
API changes, no publish steps. The branch is
`docs/audit-followups`; ordinary PR + merge review discipline
applies.

## What this branch delivers

### 1. LEGAL/ surface completion (per `docs/roadmap/LEGAL.md`)

Adds the three draft policy documents named in the roadmap's
acceptance criteria that were not yet in-tree:

- `LEGAL/PRIVACY.md` — user-facing Privacy Policy (v0.1-draft).
- `LEGAL/DPA.md` — Data Processing Agreement template (v0.1-draft).
- `LEGAL/RETENTION.md` — canonical retention schedule (single
  source of truth referenced by PRIVACY, DATA_PROTECTION, and
  DPA).

All three carry the standard **DRAFT — not counsel-reviewed** label
used across the LEGAL/ surface. Fixes a path drift in
`docs/roadmap/LEGAL.md` (roadmap said `docs/legal/`; canonical path
is the pre-existing `LEGAL/`) and updates its acceptance criteria to
reflect what is in-tree.

### 2. Investor data room scaffold (per `docs/roadmap/DATA_ROOM.md`)

Adds `docs/data-room/` with the full section layout from the roadmap:

- Company: `ONE_PAGER`, `TEAM`, `TIMELINE`.
- Product: `ARCHITECTURE`, `ROADMAP`, `DEMO`, `HONESTY` — pointers to
  canonical sources in `docs/` with investor-language framing.
- Market: `THESIS`, `ICP`, `COMPETITION`, `TAM_SAM_SOM` —
  methodology-first (no headline TAM number without a source, no
  competitor name-drop without an artefact, per the roadmap's
  content standards).
- Traction: `DESIGN_PARTNERS`, `METRICS`, `LOG` — scaffold; empty
  until a partner signs and traffic is live.
- Financials: `MODEL`, `UNIT_ECONOMICS`, `HIRING_PLAN` — shape
  documented, numbers land with the first partner.
- Legal: `CAP_TABLE`, `IP`, `AUDIT_REPORTS/` — anonymised until a
  term sheet is in play; audit reports empty until the RFP in
  `docs/roadmap/LEGAL.md` closes.
- Ops: `SLO`, `KEY_MANAGEMENT`, `STATUS_PAGE_URL.txt` — pointers to
  canonical sources; external status page pending Pack AH.

### 3. Break-glass runbook

Adds `docs/runbooks/break-glass.md`, the file referenced by
`docs/roadmap/KEY_MANAGEMENT.md` §Access control that did not
previously exist. Covers owner-key recovery via governance guardian
quorum (mirrors the `zk-verifier` 2026-07-28 procedure in
`docs/SECURITY_AUDIT.md` §2.10), emergency pause without owner-key
loss (48h timelock lift), m-of-n approver quorum, mandatory 72h
postmortem, and drill cadence.

## What this branch does *not* touch

Deliberately excluded so nothing goes live at judging time:

- No contract source, ABI, or deploy.
- No CI workflow definitions.
- No engine code, no API contract.
- No SDK package versions, no `publish` steps.
- No frontend code.
- No third-party integrations.
- No Merkle / ZK / PQ crypto surfaces.
- No governance transactions, on- or off-chain.
- No merges of existing open PR branches.

## Rollback

Every change in this branch is a text file. Revert is a `git revert`
of the merge commit (when merged) or of the individual commit (when
reviewed pre-merge). Nothing in this branch has a deployment surface,
so revert does not touch running systems.

## Validation

- Markdown lint: pass (all files well-formed, no unclosed code blocks,
  no unmatched headings).
- Broken-link check: every relative link in the new files resolves to
  an existing file in-tree at branch tip, or is explicitly labelled
  as a planned artefact (empty audit-reports directory, placeholder
  status-page URL).
- Self-review: every claim about live contracts, endpoints, SDKs, or
  historical events is cross-checked against `docs/TX_MANIFEST.md`,
  `docs/openapi.yaml`, `docs/SECURITY_AUDIT.md`,
  `docs/roadmap/*.md`, and `sdk/PUBLISHING.md`.
- No secrets, no PII, no credentials in any commit.

## Commits (in order)

1. `docs(legal): add PRIVACY, DPA, RETENTION drafts + cross-link
   roadmap`
2. `docs(data-room): scaffold investor data room per DATA_ROOM.md
   plan`
3. `docs(runbooks): add break-glass runbook referenced by
   KEY_MANAGEMENT`

## Cross-references

- `docs/roadmap/LEGAL.md`
- `docs/roadmap/DATA_ROOM.md`
- `docs/roadmap/KEY_MANAGEMENT.md`
- `LEGAL/README.md`
- `docs/data-room/README.md`
- `docs/runbooks/break-glass.md`
