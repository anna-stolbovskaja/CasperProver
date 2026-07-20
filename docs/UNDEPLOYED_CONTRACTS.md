# Undeployed contracts — status & deploy path

> **Honest positioning:** these three Casper contract prototypes build cleanly
> against `nightly-2025-01-01` / `wasm32-unknown-unknown`, but they are **not
> deploy-ready** and are not deployed to Casper testnet. CI archives the WASM
> files as build evidence only; a successful build is not a security approval.

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

- **Security gate.** Source review found pre-deploy blockers: verifier/state
  ordering and unbounded caller-controlled fields in `proof-of-inference`;
  public verification and unbounded pricing configuration in `model-registry`;
  and missing duplicate/existence/finalized/cap checks in `proof-aggregation`.
- **Execution-test gate.** The current 22-test harness reimplements selected
  logic; it does not execute these WASM contracts in a Casper/Odra engine.
  Each blocker needs a failing execution test before its fix is accepted.
- **Wallet isolation.** Any future deploy must use Anna's CasperProver wallet
  only. Wallet availability does not override the security and test gates.
- **Judge boundary.** A compiled artifact is labelled `built / undeployed`,
  never `audited`, `live`, or `on-chain`.

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

1. Add failing contract-execution tests for every blocker listed above.
2. Apply the minimum contract fixes and rerun WASM builds plus the execution
   harness; review installer/admin and runtime arguments.
3. Only then may Anna sign a Casper testnet deploy batch.
4. Record deploy and contract/package hashes in `deploy-out/onchain.json` and
   run on-chain existence plus entry-point smoke checks.
5. Regenerate frontend config from the canonical manifest and update the
   real/off-chain/simulation labels only after evidence is green.
