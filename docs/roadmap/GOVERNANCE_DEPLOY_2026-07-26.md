# Governance contract deployment — 2026-07-26

Deployment record for the 8th CasperProver contract: `governance` — timelock, emergency pause, owner recovery. Closes BACKLOG items 1.6, 1.9, 1.10, 1.11.

## On-chain fingerprint

| field | value |
|---|---|
| **network** | casper-test |
| **deploy_hash** | `1d7c9e8fa8a249977003fc1b36e443cf35c021b20c82e14513725b028ae77440` |
| **contract_hash** | `03189ea1721b517c64073c319e93d6a8cd0e53191925fcf530ebc3b897f9548e` |
| **contract_package_hash** | `0cf2674d2ff1f1b78e90cd1ce6c07099850aa3a58076ecb448806e2c865b4dbb` |
| **block_height** | 8634452 |
| **block_hash** | `d2655f01afd17b8aebe2eb25d72263ee85d1f5766df700da2140a014d43544a4` |
| **timestamp** | 2026-07-26T20:14:00Z |
| **status** | processed / error=none ✅ |
| **cost** | 90 CSPR (payment budget) |
| **consumed_gas** | 72_415_206_192 |
| **deployer / owner (bootstrap)** | `defi_mock_owner` — public_key `0202da6cfba1c1e595fdcd6539146611326e5506479d89dab735b252dc200b80f7e7`, acct_hash `84ff0a4692ddeaa2c2991b0a9b084e48772968c242ff9434106571e77d23be10` |

## Guardians (3-of-3 slots, 2-of-3 threshold for recovery)

Configured at install-time via runtime args `guardian_1` / `guardian_2` / `guardian_3`. Values are Casper account-hash strings (32-byte hex, no `account-hash-` prefix).

| slot | account_hash | identity |
|---|---|---|
| guardian_1 | `cac1862d745c06860c6acfa275d6afec3be692dedf6c46253ce813dba74ad298` | anna-stolbovskaja |
| guardian_2 | `74c96cd0073c4c973b70e7925adca8a4ba58ffcb9737304631381b82695007a8` | alexbelij |
| guardian_3 | `535a3d80c9c1cba0190b036f8edd5a9623551a5eca4ab01a0d24dda537895467` | triumphkrug |

## Entry points

`propose`, `execute`, `cancel`, `get_proposal`, `is_executed`, `emergency_pause`, `propose_unpause`, `execute_unpause`, `propose_owner_transfer`, `execute_owner_transfer`, `sign_recovery`, `execute_recovery`.

Default `timelock_secs = 48 * 60 * 60` (48 h).

## MVP-clean WASM build recipe

`casper-js-sdk 5.0.12` installOrUpgrade caps modules at ~65_536 bytes AND rejects any wasm carrying non-MVP opcodes (`bulk-memory`, `sign-ext`, `reference-types`, `mutable-globals`). A stock `cargo +nightly build --release` emits `memory.copy` / `memory.fill` from `core` even when RUSTFLAGS forbid them, because `core` is prebuilt for the target. Fix: rebuild `core` with `-Z build-std`.

Exact reproducer used for this deploy (from `contracts/`):

```bash
rustup toolchain install nightly-2025-04-01 --profile minimal --target wasm32-unknown-unknown
rustup component add rust-src --toolchain nightly-2025-04-01

export RUSTFLAGS='-C target-cpu=mvp -C target-feature=-bulk-memory,-sign-ext,-mutable-globals,-reference-types'

cargo +nightly-2025-04-01 build --release --target wasm32-unknown-unknown \
    -Z build-std=core,alloc,compiler_builtins,panic_abort \
    -Z build-std-features=panic_immediate_abort \
    -p governance
```

Post-build shrink + strip:

```bash
wasm-opt -Oz --strip-debug --strip-producers --strip-target-features \
    --signext-lowering --disable-bulk-memory --disable-sign-ext --disable-reference-types \
    --converge \
    contracts/target/wasm32-unknown-unknown/release/governance.wasm \
    -o /tmp/governance.opt.wasm

node scripts/wasm-strip-features.mjs /tmp/governance.opt.wasm /tmp/governance.final.wasm
```

Verification:

```
$ stat -c%s /tmp/governance.final.wasm
53762                             # under 65536 cap ✅

$ python3 -c "d=open('/tmp/governance.final.wasm','rb').read(); \
    print('memory.copy', d.count(b'\\xfc\\x0a')); \
    print('memory.fill', d.count(b'\\xfc\\x0b'))"
memory.copy 0                     # MVP-clean ✅
memory.fill 0                     # MVP-clean ✅
```

## Deploy submission

`scripts/deploy-wasm.mjs` was extended in-flight for this contract because governance's `call()` reads three named args. A dedicated helper `deploy-governance.mjs` (private to the deployer, not committed) wired the args:

```js
const args = Args.fromMap({
  guardian_1: CLValue.newCLString(G1_ACCT_HASH_HEX),
  guardian_2: CLValue.newCLString(G2_ACCT_HASH_HEX),
  guardian_3: CLValue.newCLString(G3_ACCT_HASH_HEX),
});
```

Submitted via `SessionBuilder.installOrUpgrade()` against `https://node.testnet.cspr.cloud/rpc` with `Authorization` header set to the cspr.cloud API key. Payment budget 90 CSPR (52 % headroom over consumed 72.4 CSPR of gas).

## Two failed pre-fix attempts (full honesty)

Before switching to `-Z build-std`, two deploy attempts hit the Casper wasm preprocessor:

| # | deploy_hash | payment | wasm bytes | outcome |
|---|---|---|---|---|
| 1 | `4b3b6c3ba253c86d8a31806aa5ad4fcbacbad151bab6ca62d0bd386c7b33b003` | 80 CSPR | 65379 (had `memory.copy`) | `Wasm preprocessing error: Bulk memory operations are not supported` |
| 2 | `09341260efca179538faf653b8d0ac62cadbd44e72593bdb7d88a00716b8eb7e` | 75 CSPR | 65427 (still had `memory.copy` — wasm-opt does not rewrite existing opcodes) | Same error |

These 155 CSPR of payment budget were consumed by the network (no refund on preprocessing failure). Deployer wallet was topped up by two 100 CSPR transfers from anna → defi_mock_owner (`tx transfers verifiable on-chain`) so the third attempt could land.

## Post-install verification

```
$ curl -sS -H "Authorization: $CSPR_CLOUD_API_KEY" \
    "https://api.testnet.cspr.cloud/deploys/1d7c9e8fa8a249977003fc1b36e443cf35c021b20c82e14513725b028ae77440" \
    | jq '.data | {status, error_message, contract_hash, contract_package_hash, block_height}'
{
  "status": "processed",
  "error_message": null,
  "contract_hash": null,
  "contract_package_hash": null,
  "block_height": 8634452
}
```

The `contract_hash` isn't populated in the cspr.cloud `/deploys/` view because install-time contract discovery lands under the deployer's named-keys, not the deploy itself. Direct read:

```
$ curl -sS -H "Authorization: $CSPR_CLOUD_API_KEY" \
    "https://api.testnet.cspr.cloud/accounts/0202da6cfba1c1e595fdcd6539146611326e5506479d89dab735b252dc200b80f7e7" \
    | jq -r '.data.named_keys[]? | select(.name=="governance" or .name=="governance_pkg") | "\(.name) = \(.key)"'
governance = 03189ea1721b517c64073c319e93d6a8cd0e53191925fcf530ebc3b897f9548e
governance_pkg = 0cf2674d2ff1f1b78e90cd1ce6c07099850aa3a58076ecb448806e2c865b4dbb
```

Cross-referenced via `queryGlobalStateByStateHash("account-hash-84ff…", state_root)` at the latest block — both hashes present.

## Downstream integration

`verify.sh` now checks all 8 contracts (verified 12/12 pass). Frontend `onchain.json` promoted from 4 → 8 contracts. Judge guide references 8 contracts.

Existing deployed contracts (proof-registry, verifier-gate, defi-mock, stake-slashing, proof-aggregation, model-registry, proof-of-inference) opt into governance via the `is_executed(pid)` read entry-point on their next upgrade. No live redeploy is required — the governance record travels in whichever contract calls it, keyed by proposal id.
