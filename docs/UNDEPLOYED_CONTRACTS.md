# Undeployed contracts — status & deploy path

> **Honest positioning:** these three Casper contracts are fully implemented
> and build cleanly against `nightly-2025-01-01` / `wasm32-unknown-unknown`, but
> they are **not yet deployed** to Casper testnet. They are archived from every
> CI run and available as build artifacts.

## Contracts

| Contract | Source | LOC | Purpose |
|---|---|---:|---|
| `proof-of-inference` | `contracts/proof-of-inference/` | 498 | Per-inference commitment + verifier binding |
| `model-registry` | `contracts/model-registry/` | 372 | Canonical model → owner / version lookup |
| `proof-aggregation` | `contracts/proof-aggregation/` | 179 | Batch verification receipt anchoring |

## Current WASM sizes (release, `--no-default-features`)

Sizes captured from local build on 2026-07-20:

| Contract | Bytes | KB |
|---|---:|---:|
| `proof-of-inference` | 90,398 | 88.3 |
| `model-registry` | 81,113 | 79.2 |
| `proof-aggregation` | 73,144 | 71.4 |

All three exceed the ~65 KB ceiling of the JS `installOrUpgrade` helper, so
deployment MUST go through the audited Casper CLI / RPC path
(`casper-client put-deploy` with `--session-path` pointing at the built
WASM), not the JS convenience wrapper. This is the same path we use for the
already-deployed contracts (`stake-slashing`, `defi-mock`, `verifier-gate`,
`proof-registry`).

## Why they are not live yet

- **Wallet gate.** Deploy MUST be signed by the CasperProver deployer key
  (Anna / CasperProver wallet only, per project isolation rules).
- **Judge boundary.** Publishing an unverified contract just to have a live
  address would put a claim ("on-chain") on something we cannot fully audit
  in the deadline window. We would rather ship compiled artifacts + a clear
  roadmap than break the "REAL vs ON-CHAIN vs SIMULATION" badge contract.
- **Time-box.** Per `CP_FINAL_TASKS_V2.md` Gate 2 spec: "if build+smoke green
  and deploy path is clear — deploy; if within 4 h there is no safe path — do
  not break the working four; document compiled artifact + roadmap and
  continue submission path."

## Reproducing the build

```bash
rustup toolchain install nightly-2025-01-01 --target wasm32-unknown-unknown
for c in proof-of-inference model-registry proof-aggregation; do
  (cd contracts/$c && cargo +nightly-2025-01-01 build --release \
     --target wasm32-unknown-unknown --no-default-features)
done
ls -la contracts/target/wasm32-unknown-unknown/release/*.wasm
```

CI runs this on every push; artifacts land under the `contract-wasms`
workflow artifact (7-day retention) and the size table is rendered into the
run summary.

## Roadmap to on-chain

1. Anna signs a deploy batch for the three contracts against Casper testnet.
2. Deploy transactions are captured in `deploy-out/onchain.json` alongside
   the existing four contracts, with `deployed_at` / `deploy_hash` /
   `contract_hash` / `contract_package_hash`.
3. `verify.sh` gains the three additional checks (existence + entry point
   smoke) automatically — it reads contracts from the manifest, no code
   change needed.
4. Frontend / SDK / docs pick up the new hashes with no rebuild since
   nothing hardcodes them (Gate 1.5 canonical manifest work).
