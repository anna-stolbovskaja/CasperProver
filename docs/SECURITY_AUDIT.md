# CasperProver — Security Audit: Owner/Admin Lifecycle & Cross-Contract Invariants

**Scope:** all 8 contract crates in `contracts/` (deployed + undeployed).  
**Date:** 2026-07-20 (Gate 1, item 4 of `CP_FINAL_TASKS_V2_new.md`).  
**Reviewer:** in-repo static audit (no on-chain formal verification tool).  
**Purpose:** document the actual ownership model per contract, flag *irreversible* privilege paths, and record cross-contract reentrancy/invariant reasoning **before** any redeploy of Gate 2.

Nothing in this document changes contract source. It is the reference security review that must be re-read whenever a new privileged entry point is added.

---

## 1. Ownership model — cheat sheet

| Contract | Admin key | Set at | Mutable? | Renounce? | Entry points gated | Verdict |
|---|---|---|---|---|---|---|
| `defi-mock` | `ADMIN_KEY` → `Key::Account(deployer)` | `call()` (install) | No | No | `grant_access`, `revoke_access` | **Safe — no privilege escalation, no lost-key risk beyond deployer wallet.** |
| `verifier-gate` | none | — | — | — | none (rate-limit is per-caller, not admin-gated) | **Safe — read-only proxy over proof-registry.** |
| `stake-slashing` | none | — | — | — | none (all state gated by caller identity or proof-registry state) | **Safe — permissionless-by-design.** |
| `proof-registry` | none (agent = record.owner) | per-record | Per-record `agent` string | No | `revoke_proof` (agent-only) | **Safe — no global admin.** |
| `proof-of-inference` | `INSTALLER_KEY` → `Key::Account(installer)` | `call()` | No | No | `resolve_challenge`, `register_verifier`, `revoke_verifier` | **Advisory: hot deployer key. Add timelock roadmap.** |
| `model-registry` | `INSTALLER_KEY` (for `set_price_bps` only) + per-model `owner` string | `call()` / `register_model` | Per-model transferable | **No renounce** (transfer only) | `set_price_bps` (installer), `deprecate_model` / `update_model_metadata` / `transfer_ownership` (per-model owner) | **Safe — dual-tier, no burn address.** |
| `proof-aggregation` | `INSTALLER_KEY` → `Key::Account(installer)` | `call()` | No | No | `finalize_batch` | **Advisory: hot deployer key. Same roadmap as proof-of-inference.** |
| `stake-slashing-session` | n/a (session code, not a contract) | — | — | — | — | Session runs under caller's own account context. |

**Global observation.** No contract exposes a `renounce_ownership()` entry point today. This is intentional: an irreversible renounce on a testnet demo with no recovery/timelock would be a footgun. Whenever renounce is added later (roadmap item), it must ship together with:

1. A timelocked propose → confirm → activate flow (min 48h).
2. An emergency-pause switch that survives renounce (multi-sig or on-chain governance module).
3. A documented recovery path if the successor key is lost.

Without those three, **do not merge a renounce entry point in any contract**.

---

## 2. Per-contract audit notes

### 2.1 `defi-mock` (deployed)

- **Admin:** `Key::Account(deployer)` captured in `call()` as `ADMIN_KEY`.
- **Gated entry points:**
  - `grant_access(user, proof_id)` — refuses to overwrite an active whitelist entry (post-2026-07-18 hardening — must go through `revoke_access` first, keeps on-chain history explicit).
  - `revoke_access(user)` — writes tombstone `(user, "", 0)` (post-fix: `is_whitelisted` returns `false` on `ts == 0`).
- **Cross-contract:** calls `verifier-gate::is_valid(proof_id)` **before** whitelist mutation (checks-effects-interactions ordering respected — external call cannot re-enter to force a spurious `true`).
- **Reentrancy analysis:** `call_is_valid` is a *read-only* proxy that ends in `runtime::ret`. It cannot mutate this contract's state, and if it did (via a malicious swapped verifier), the current implementation still checks the returned `bool` and reverts on `false`. Only worry vector: admin replaces `VERIFIER_HASH` — but there is **no entry point to update `VERIFIER_HASH` post-install**, so the trust anchor is immutable-at-install.
- **Renounce risk:** admin key lost = whitelist is frozen (no new grants/revokes). Read paths (`check_kyc`, `is_whitelisted`) continue to work. **Not catastrophic.**
- **Verdict:** ✅ Safe as-is. No changes recommended before redeploy.

### 2.2 `verifier-gate` (deployed)

- **Admin:** none. Contract has no privileged operations.
- **Gated entry points:** none. `is_valid`, `is_valid_batch`, and `get_verify_count` are all public reads.
- **Cross-contract:** calls `proof-registry::get_proof`. Read-only, cannot cause state-drift.
- **Rate limit:** per-caller `VERIFY_COUNTS` dictionary + `ERR_RATE_LIMIT` revert. This is bookkeeping, not a privilege check.
- **Reentrancy analysis:** `runtime::call_contract` to proof-registry returns typed data; the contract does not modify state after the call (only `runtime::ret`). No CEI concern.
- **Renounce risk:** none — no admin exists.
- **Verdict:** ✅ Safe as-is.

### 2.3 `stake-slashing` (deployed, hardened 2026-07-18/19)

- **Admin:** none. Deliberately permissionless — anyone can call `report_and_slash` because the only trigger is that proof-registry already recorded a revocation.
- **Gated by caller identity (not admin):**
  - `record_stake` — credits *the caller* only. Cannot be used to inflate a third party's stake.
  - `unstake` — withdraws from *the caller's* recorded stake only, using `checked_sub` (no underflow).
- **Cross-contract:** calls `proof-registry::get_proof` (read-only) before slashing.
- **Reentrancy analysis:** the only external interaction after state mutation is `system::transfer_from_purse_to_account` in `unstake` and the `report_and_slash` payout. Both use Casper's native transfer primitives, not contract calls back into this crate, so there is no callback surface to re-enter. State (`stakes`, `slashed_proofs`, `total_recorded`) is updated *before* the transfer in both paths (checks-effects-interactions respected).
- **Bookkeeping invariant:** `sum(stakes) + slash_reserve ≤ contract_purse_balance` — enforced by:
  - `record_stake` capping credit to `actual_balance − total_recorded` (post-2026-07-18 hardening — prevents unbacked credit).
  - `decrease_total_recorded()` on every `unstake` / `report_and_slash` payout.
- **Zero-slash invariant:** `ERR_NO_SLASHABLE_STAKE` (post-2026-07-19 fix) prevents burning the one-shot slash tombstone with a zero payout — closes the "consume tombstone, pay nothing" attack.
- **Renounce risk:** none — no admin exists.
- **Verdict:** ✅ Safe as-is. Prior hardening (record_stake self-verification, zero-slash guard, checked arithmetic) is preserved.

### 2.4 `proof-registry` (deployed)

- **Admin:** none globally. Per-record: `agent` field stored at registration (`(agent_id, owner, model_hash)` tuple).
- **Gated entry points:** `revoke_proof` — only the recorded `agent` may revoke their own proof.
- **Cross-contract:** none — `proof-registry` is a leaf contract.
- **Reentrancy analysis:** no external calls; only local dictionary updates.
- **Renounce risk:** none globally. A specific agent losing their key freezes only that agent's own proofs (they cannot self-revoke), which is a per-user operational issue, not a systemic one.
- **Verdict:** ✅ Safe as-is.

### 2.5 `proof-of-inference` (undeployed)

- **Admin:** `INSTALLER_KEY` → `Key::Account(installer)`, set in `call()`.
- **Gated entry points:**
  - `resolve_challenge(proof_id, valid)` — installer-only; can decide rejected/resolved verdict.
  - `register_verifier(verifier_id, pub_key)` — installer-only.
  - `revoke_verifier(verifier_id)` — installer-only.
- **Cross-contract:** none.
- **Reentrancy analysis:** no external calls. All state transitions are local dictionary updates with prior status checks (`ERR_INVALID_STATUS`, `ERR_ALREADY_VERIFIED`, `ERR_NOT_CHALLENGED`).
- **Renounce risk:** if the installer key is lost, the verifier roster is frozen (no add/revoke) and **challenges cannot be resolved** (permanently stuck at status 2). This is a **hot-key concentration risk**. Advisory below.
- **Recommended before deploy (non-blocking for Gate 2):**
  - Add a `pending_installer` + `accept_installer` two-step transfer entry point, gated by the current installer, so the key can rotate before it is lost.
  - Do **not** add `renounce_installer` — irreversible on a demo with no timelock.
- **Verdict:** ⚠️ Deploy as-is is acceptable for the hackathon (single-owner, testnet). Timelocked two-step transfer is a 30-day roadmap item.

### 2.6 `model-registry` (undeployed)

- **Admin (dual-tier):**
  - `INSTALLER_KEY` → global, for `set_price_bps` only.
  - Per-model `owner` string (stored inside each `ModelRecord`) → for `deprecate_model`, `update_model_metadata`, `transfer_ownership`.
- **Ownership transfer:** `transfer_ownership(model_hash, new_owner)` — current owner only. **No renounce entry point** (cannot transfer to `AccountHash::default()` because the record still requires *someone* to own it for downstream deprecate/update calls). This is the correct posture.
- **Cross-contract:** none.
- **Reentrancy analysis:** no external calls. Status transitions (`ERR_ALREADY_DEPRECATED`, `ERR_ALREADY_REGISTERED`) prevent double-updates.
- **Renounce risk:** installer-key loss freezes `set_price_bps` (pricing knob). Per-model owner-key loss freezes that model's lifecycle. Both are per-record blast radius, not systemic.
- **Owner string format caveat:** `owner` is stored as `format!("{:?}", caller)` (Rust `Debug` output of `AccountHash`). This is *stable within a single Rust toolchain build* but has bitten other Casper projects on version bumps. **Follow-up:** normalize to `caller.to_string()` (hex, no prefix) in the next non-breaking migration; currently a comparison bug not a security bug because writer and comparator both use the same `Debug` format.
- **Verdict:** ⚠️ Deploy as-is is acceptable. The `owner` string format normalization is a P2 hygiene item — file as follow-up.

### 2.7 `proof-aggregation` (undeployed)

- **Admin:** `INSTALLER_KEY` → `Key::Account(installer)`, set in `call()`.
- **Gated entry points:** `finalize_batch(batch_id)` — installer-only via `ApiError::PermissionDenied`.
- **Ungated writes:** `create_batch`, `add_proof` — public. **This is a design choice** (permissionless Merkle-batch construction; only finalization is trusted), but there is no per-batch access control:
  - Anyone can call `add_proof` against an existing `batch_id`.
  - `add_proof` writes to key `format!("{}:{}", batch_id, proof_hash)` — duplicates by different callers overwrite each other **only if `proof_hash` is identical**, which is fine.
  - **Attention:** a malicious caller cannot forge someone else's `proof_hash`, but they *can* pre-empt a `batch_id` by calling `create_batch` first with a low `max_proofs` and different `merkle_root`. Since the batch value written by `create_batch` is `format!("{}|{}|{}|0|open", …)`, the second `create_batch` for the same `batch_id` silently overwrites the first.
  - **Recommended fix (non-blocking for Gate 2):** in `create_batch`, refuse to overwrite an existing batch — check `storage::dictionary_get(batches_uref, &batch_id)`; if `Some(_)`, revert with a new `ERR_BATCH_EXISTS`. Same pattern already used in `defi-mock::grant_access` and `model-registry::register_model`.
- **Cross-contract:** none.
- **Reentrancy analysis:** no external calls.
- **Renounce risk:** installer-key loss freezes finalization → batches never close. Per-batch blast radius.
- **Verdict:** ⚠️ Deploy as-is only after fixing `create_batch` overwrite (10 LOC change). File as blocking follow-up before Gate 2 deploy of this crate.

### 2.8 `stake-slashing-session`

- **Session code, not a contract** — runs with the caller's own account context, no installer, no cross-account privilege.
- **Purpose:** atomically transfer CSPR from caller's main purse to `stake-slashing`'s contract purse **and** call `record_stake` in the same deploy so they cannot be split/front-run.
- **Verdict:** ✅ Safe by construction — session code has no persistent state to attack.

---

## 3. Cross-contract invariants (system-level)

### 3.1 Call graph

```
defi-mock.grant_access
  └─ verifier-gate.is_valid
       └─ proof-registry.get_proof                   (read)

stake-slashing.report_and_slash
  └─ proof-registry.get_proof                        (read)

stake-slashing.record_stake
  └─ system::get_purse_balance(contract_purse)       (read, native)

stake-slashing.unstake / .report_and_slash payout
  └─ system::transfer_from_purse_to_account          (native transfer)
```

Every cross-contract edge is either **read-only** (returns via `runtime::ret`) or a **native system transfer** (not a contract-to-contract call that could re-enter). There is no callback path that lets a downstream contract re-enter an upstream contract in mid-write. **Casper's `call_contract` is synchronous but the contract being called cannot re-enter the caller unless the caller passes its own contract hash as an argument** — and none of our contracts do this.

### 3.2 CEI (checks-effects-interactions) posture

| Contract & entry point | Order | Notes |
|---|---|---|
| `defi-mock.grant_access` | Check (admin), Check (`ALREADY_WHITELISTED`), Effect-less external call (`is_valid`), Check (result), Effect (write whitelist) | External call is read-only; even if reordered, no state-drift. |
| `stake-slashing.record_stake` | Check (`available` vs claimed), Effect (write stakes + total_recorded) | No external call after state write. |
| `stake-slashing.unstake` | Check (`checked_sub`), Effect (dict write), Interaction (`transfer_from_purse_to_account`), Effect (`decrease_total_recorded`) | Slight CEI drift: `total_recorded` decreases *after* transfer. Native transfer cannot reenter, so acceptable. Documented here so future reviewers don't refactor blind. |
| `stake-slashing.report_and_slash` | Check (`ALREADY_SLASHED`), Interaction (`get_proof` — read), Check (`AGENT_MISMATCH`, `NOT_REVOKED`, `NO_SLASHABLE_STAKE`), Effect (stakes + slashed dict), Interaction (transfer), Effect (`decrease_total_recorded`) | Same "native transfer at end" pattern. Safe. |
| `proof-of-inference.*` | Check → Effect only | No external calls. |
| `model-registry.*` | Check → Effect only | No external calls. |
| `proof-aggregation.create_batch/add_proof/finalize_batch` | Check → Effect only | No external calls. |

**No entry point in the system performs an external contract call after writing state and before finishing.** The two `transfer_from_purse_to_account` cases in `stake-slashing` are the only interactions after state effects, and they are native system transfers with no callback surface. Reentrancy risk is minimal.

### 3.3 Trust boundaries at install time

Every cross-contract reference is written to `NamedKeys` inside `call()` (install-time only). There is **no entry point in any contract to update `VERIFIER_HASH` / `PROOF_REGISTRY_HASH` / etc. post-install.** This is intentional: it removes an entire class of "swap a trusted dependency" attacks. Cost: to upgrade, we redeploy and re-wire — which is what we already do.

---

## 4. Follow-up items (opened as GitHub issues after this audit lands)

1. **[P1, pre-Gate 2 blocking for `proof-aggregation`]** Fix `create_batch` silent overwrite — 10 LOC, add `ERR_BATCH_EXISTS` guard.
2. **[P2, roadmap 30-day]** Two-step installer transfer (`pending_installer` + `accept_installer`) in `proof-of-inference`, `proof-aggregation`, `model-registry`. **No** `renounce_installer` without a timelock module.
3. **[P2, hygiene]** Normalize `model-registry`'s per-model `owner` from `format!("{:?}", caller)` to `caller.to_string()` — non-breaking behavior today but toolchain-fragile.
4. **[P3, doc-only]** Add this file's summary table to `docs/ARCHITECTURE.md`'s security section so it's linked from the top-level architecture doc, not only referenced here.

---

## 5. Sign-off

- Static audit complete on the pinned tree.
- No P0 issues; one P1 blocker for the `proof-aggregation` redeploy path (file the fix before Gate 2 rebuilds that crate).
- Reentrancy: analyzed, low risk — every cross-contract call is read-only or native transfer, and no callback surface exists.
- Ownership: single-owner model per contract with clear renounce policy = **no irreversible renounce anywhere**.
