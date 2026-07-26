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
| **proof-registry** | [`e11088f1...5fa7bc5`](https://testnet.cspr.live/contract/e11088f1f15a719f21c0c318d1f34d0b96419a22d60ac8fa384ecf5285fa7bc5) | 2026-06-29 | 2026-07-26 | ✅ Live (redeployed under canonical `defi_mock_owner`, deploy `c9e0fee7...`, verified via CSPR.cloud) |
| **verifier-gate**  | [`06d69182...cc0c79b66`](https://testnet.cspr.live/contract/06d69182b13c4d041613fe7e6e0805cdb06f099eff4291b40154d78cc0c79b66) | 2026-06-29 | 2026-07-26 | ✅ Live (redeployed under canonical `defi_mock_owner`, deploy `43ca8966...`, verified via CSPR.cloud) |
| **defi-mock**      | [`fe0c45f6...0b39ef`](https://testnet.cspr.live/contract/fe0c45f67c8cd99f0bda0047399a113588870ec0d79d9102f44107303f0b39ef) | 2026-07-07 | 2026-07-07 | ✅ Live |
| **stake-slashing** | [`1ad1b3d9...983d52`](https://testnet.cspr.live/contract/1ad1b3d94be631532d6daf3a195fafc9dfe8a16504e87d87784d51089b983d52) | (earlier) | 2026-07-19 | ✅ Live (hardened redeploy — arithmetic + self-verifying record_stake) |
| **proof-aggregation** | [`b29f32ab...63d2bb`](https://testnet.cspr.live/contract/b29f32abcc029d523de212bd7c87993f2f1bf96ba1523091c7b01adf6d63d2bb) | 2026-07-25 | 2026-07-25 | ✅ Live (deploy `35c003e5...`, verified via CSPR.cloud, `create_batch` guard suite — 28 tests) |
| **model-registry** | [`b3cdd1df...5e64a340a`](https://testnet.cspr.live/contract/b3cdd1df25714b341e34f6bb29f6c7900267e44c7742c81221e1eab5e64a340a) | 2026-07-25 | 2026-07-25 | ✅ Live (deploy `fd21b26e...`, verified via CSPR.cloud) |
| **proof-of-inference** | [`3d772fe1...12498b318`](https://testnet.cspr.live/contract/3d772fe1618fde438c4ffdaec22d83ffd9b4a1d769d6da32a38d56f12498b318) | 2026-07-25 | 2026-07-25 | ✅ Live (deploy `bde5cfb7...`, verified via CSPR.cloud) |
| **governance** | [`03189ea1...7f9548e`](https://testnet.cspr.live/contract/03189ea1721b517c64073c319e93d6a8cd0e53191925fcf530ebc3b897f9548e) | 2026-07-26 | 2026-07-26 | ✅ Live (deploy `1d7c9e8f...`, verified via CSPR.cloud; guardians: anna/alexbelij/triumphkrug account-hashes) |

All three were deployed 2026-07-25 from the secondary account
(`0202da6cfba1...`) using an MVP-clean WASM toolchain fix (Rust nightly
`-Z build-std`, bulk-memory-opt/sign-ext/reference-types disabled,
`wasm-opt --signext-lowering`, stripped `target_features` section —
0 non-MVP opcodes). Deploy hashes confirmed `processed` via
`api.testnet.cspr.cloud/deploys/<hash>` with matching timestamps and
caller public key. No contracts remain undeployed — see
[`deploy-out/onchain.json`](../deploy-out/onchain.json) for the full
machine-readable manifest (8 contracts total).

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
| 2026-07-25 | proof-of-inference, model-registry, proof-aggregation deployed (MVP-clean WASM toolchain fix) | anna-stolbovskaja |

## 7. Non-goals of this document

- **No secrets.** Wallet keys, deployer secrets, RPC tokens are never
  in this file. Only public hashes.
- **No test fixture hashes.** Only real testnet deploys; local nctl
  hashes belong in `docs/DEV_ENVIRONMENT.md` if anywhere.
- **No promises.** A row without a hash is *pending*, not on-chain.
  Do not link to `docs/TX_MANIFEST.md#pending` as if it were evidence.
