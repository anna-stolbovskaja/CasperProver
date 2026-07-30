# Break-Glass Runbook

> **Honesty label:** DRAFT — REAL for local + testnet operational
> posture. Not exercised in a real incident yet. The procedure below
> is derived from `docs/roadmap/KEY_MANAGEMENT.md`, `docs/HSM_PLAN.md`,
> `docs/OPS_RUNBOOKS.md`, and `docs/SECURITY_AUDIT.md`. Every
> break-glass use triggers a mandatory postmortem (§7).

## 1. What "break-glass" means here

Break-glass is the last-resort procedure for privileged actions that
the normal governance path cannot execute in time, or that the normal
path is itself compromised. It is:

- **Rare** — never invoked for a routine issue.
- **Audited** — every step is logged, signed, and shipped to cold
  storage.
- **Postmortemed** — a written postmortem always follows, no
  exceptions.

## 2. When to invoke

Break-glass is authorised for any of the following:

1. **Owner-key loss / compromise.** The single owner key of a
   contract with an owner (per `docs/SECURITY_AUDIT.md` §1) is lost or
   suspected compromised.
2. **Emergency pause.** A live exploit is being actively used against
   any deployed contract that carries a pause switch (`governance`,
   `zk-verifier`).
3. **HSM lockout.** The primary HSM is inaccessible and pending
   privileged operations block business continuity.
4. **Ceremony coordinator compromise.** The ceremony coordinator key
   is lost or suspected compromised mid-ceremony.

Any other scenario uses the normal governance path — 48h timelocked
propose → execute — described in `docs/roadmap/GOVERNANCE.md`.

## 3. Preconditions

- Two authorised humans present (m-of-n approver quorum, see §5).
- Offline paper vault accessible (Shamir 3-of-5 shares for the root
  break-glass credential per `docs/HSM_PLAN.md`).
- Audit log destination healthy (Postgres + cold-storage tail).
- Incident channel open (Slack / equivalent) with an incident
  identifier assigned.

## 4. Procedure — Owner-key recovery via guardian quorum

Path for scenario #1 above when `governance` is the affected contract.

1. **Declare incident.** Open the incident ticket, assign an
   identifier (`INC-YYYY-MM-DD-NNN`), post to the incident channel.
2. **Verify the loss.** Two humans independently confirm the owner
   key is lost or compromised. Record in the ticket.
3. **Assemble guardians.** Contact 2 of the 3 guardians listed in the
   governance install: `anna-stolbovskaja`, `defi_mock_owner`, and the
   reserved slot for the mainnet ceremony (per
   `docs/roadmap/GOVERNANCE_DEPLOY_2026-07-26.md`).
4. **Sign the recovery proposal.** Each guardian calls
   `sign_recovery(proposed_new_owner)` on the `governance` contract
   from their own account.
5. **Execute recovery.** After the second guardian signs, call
   `execute_recovery` — this rotates the owner to the proposed new
   account.
6. **Rotate downstream references.** Update Render `CONTRACT_*`
   environment variables that point at the compromised owner's
   deployment of any contract that was reinstalled (mirrors the
   `zk-verifier` 2026-07-28 procedure in `docs/SECURITY_AUDIT.md`
   §2.10). Re-run `/health` and confirm.
7. **Emergency pause if the compromise is active.** After the new
   owner is in place, call `emergency_pause` on the affected
   downstream contracts and validate with `is_executed(pid)`.
8. **Rotate the compromised private material.** Destroy the wrapping
   DEKs where the compromised key encrypted anything at rest;
   re-encrypt from clean backups.

## 5. Procedure — Emergency pause without owner-key loss

Path for scenario #2 when the owner key is still under control.

1. **Declare incident.**
2. **Owner calls `emergency_pause` on the affected contract.**
3. **Verify pause on-chain.**
4. **Coordinate with downstream integrations** — API `/health` will
   still respond; endpoint-level 503 with `Retry-After` returns until
   the pause lifts.
5. **Fix, then propose_unpause / execute_unpause** via the normal 48h
   timelock — never lift a pause faster than that, even in an
   incident, unless the guardian recovery has re-authorised.

## 6. Approver quorum (m-of-n)

Privileged break-glass operations require **m-of-n** human approvers.

- **m = 2** for owner recovery, emergency pause, HSM lockout release.
- **m = 3** for ceremony re-seal or cryptographic-shred of an entire
  data class.

Approver identities are recorded in the audit log with input digests
(never plaintext), timestamps (monotonic + wall clock), and their
signatures on the requested operation.

## 7. Postmortem — mandatory

Within 72 hours of a break-glass use, a written postmortem lands
under `docs/postmortems/INC-<incident-id>.md` covering:

- Timeline (declare / verify / execute / restore).
- Root cause.
- What worked, what did not.
- Follow-ups: procedural, code, key-management.
- Redactions: internal or public. Postmortems are internal by
  default; a public version is written when disclosure is
  appropriate.

## 8. Drill cadence

- Quarterly tabletop of scenario #1 (owner-key recovery via
  guardians).
- Semi-annual dry-run of scenario #2 (emergency pause + unpause via
  the timelock).
- The first real drill is a documented milestone under
  `docs/roadmap/KEY_MANAGEMENT.md` §Milestones (4). Result recorded in
  `docs/data-room/traction/LOG.md`.

## 9. Cross-references

- `docs/roadmap/KEY_MANAGEMENT.md`
- `docs/roadmap/GOVERNANCE.md`
- `docs/SECURITY_AUDIT.md` (esp. §2.9 governance guardian caveats,
  §2.10 zk-verifier redeploy)
- `docs/HSM_PLAN.md`
- `docs/OPS_RUNBOOKS.md`
- `LEGAL/DATA_PROTECTION.md` §10 (breach notification)
