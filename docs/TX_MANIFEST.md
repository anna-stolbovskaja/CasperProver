# CasperProver — On-chain Transaction Manifest

> Every claim of "on-chain" the project makes is backed by a concrete
> testnet transaction. This document is the single source of truth.
> Update it whenever a contract is deployed or a new evidence tx is
> emitted; never invent a hash.

**Network**: `casper-test` (testnet).
**Explorer base**: https://testnet.cspr.live/

---

## 1. Deployed contracts (verified live)

| Contract | Package hash | First deploy | Last deploy | Status |
|----------|--------------|--------------|-------------|--------|
| **proof-registry** | [`96e97c4d...a10708`](https://testnet.cspr.live/contract/96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708) | 2026-06-29 | 2026-06-29 | ✅ Live |
| **verifier-gate**  | [`a37f9cde...9f77d3`](https://testnet.cspr.live/contract/a37f9cde9dbdc5bb8b9e92c663bdc59b83b42c89dc75ec73f7f7cde2619f77d3) | 2026-06-29 | 2026-06-29 | ✅ Live |
| **defi-mock**      | [`fe0c45f6...0b39ef`](https://testnet.cspr.live/contract/fe0c45f67c8cd99f0bda0047399a113588870ec0d79d9102f44107303f0b39ef) | 2026-07-07 | 2026-07-07 | ✅ Live |
| **stake-slashing** | [`1ad1b3d9...983d52`](https://testnet.cspr.live/contract/1ad1b3d94be631532d6daf3a195fafc9dfe8a16504e87d87784d51089b983d52) | (earlier) | 2026-07-19 | ✅ Live (hardened redeploy — arithmetic + self-verifying record_stake) |

## 2. Contracts written but not yet deployed

These are CI-built and test-covered but not yet on-chain. Deploy tx
hashes land here immediately after each `casper-client put-deploy`
succeeds. Do **not** claim these as "on-chain" until the hash is
present.

| Contract | Source path | CI build | Deploy status | Deploy tx |
|----------|-------------|----------|---------------|-----------|
| **proof-of-inference** | `contracts/proof_of_inference/` | ✅ | ⏳ pending | _pending Gate 2_ |
| **model-registry**     | `contracts/model_registry/`     | ✅ | ⏳ pending | _pending Gate 2_ |
| **proof-aggregation**  | `contracts/proof_aggregation/`  | ✅ (with create_batch guard suite — 28 tests) | ⏳ pending | _pending Gate 2_ |

**When Gate 2 completes**, each row above gains:

- `Package hash` column (with cspr.live link).
- `First deploy` date.
- Any subsequent hardening redeploys tracked as separate rows above
  the previous one, so the audit trail stays intact.

## 3. Evidence transactions (representative)

A handful of representative txs judges can inspect directly. This is
not exhaustive — 248+ total testnet transactions across the four live
contracts.

| Purpose | Tx hash | Notes |
|---------|---------|-------|
| proof-registry: submit proof | _add hash_ | Records an anchored SHA-256 root. |
| verifier-gate: verify Merkle inclusion | _add hash_ | Cross-contract read of proof-registry. |
| defi-mock: KYC whitelist add | _add hash_ | Admin-gated entry point. |
| stake-slashing: slash on revoked proof | _add hash_ | Economic penalty, hardened path (post-2026-07-19). |

**Adding an evidence tx**: open a deploy on cspr.live, copy the deploy
hash (starts with 64 hex chars, no `deploy-` prefix), and add a row
with `[hash-prefix...suffix](https://testnet.cspr.live/deploy/<full-hash>)`.

## 4. How to reproduce a hash

Every row in section 1 is reproducible without trust:

```bash
casper-client query-global-state \
  --node-address https://rpc.testnet.casperlabs.io \
  --state-root-hash "$(casper-client get-state-root-hash \
      --node-address https://rpc.testnet.casperlabs.io \
      | jq -r .result.state_root_hash)" \
  --key hash-<PACKAGE_HASH> \
  --path '' | jq
```

Replace `<PACKAGE_HASH>` with the hash from the manifest row. A
successful query returns the stored contract package with entry-point
list and named-key map — the same shape cspr.live renders.

## 5. Anti-linking pass

**CasperProver contracts are not shared with any other Casper hackathon
submission.** All four package hashes above resolve to accounts under
the CasperProver deployer key. No shared deploys with AgentEscrow402
or any other submission.

## 6. Change log

| Date | Change | Author |
|------|--------|--------|
| 2026-06-29 | Initial deploy: proof-registry, verifier-gate | anna-stolbovskaja |
| 2026-07-07 | defi-mock deployed | anna-stolbovskaja |
| 2026-07-18 | stake-slashing arithmetic + record_stake self-verifying fix (found in in-house audit) | anna-stolbovskaja |
| 2026-07-19 | stake-slashing hardened redeploy | anna-stolbovskaja |
| _pending Gate 2_ | Deploy proof-of-inference, model-registry, proof-aggregation | anna-stolbovskaja |

## 7. Non-goals of this document

- **No secrets.** Wallet keys, deployer secrets, RPC tokens are never
  in this file. Only public hashes.
- **No test fixture hashes.** Only real testnet deploys; local nctl
  hashes belong in `docs/DEV_ENVIRONMENT.md` if anywhere.
- **No promises.** A row without a hash is *pending*, not on-chain.
  Do not link to `docs/TX_MANIFEST.md#pending` as if it were evidence.
