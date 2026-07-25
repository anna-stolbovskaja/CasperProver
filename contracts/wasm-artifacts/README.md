# WASM artifacts (shrunk)

These `.opt.wasm` blobs are the deploy-ready payloads for the three
previously-undeployed CP contracts, produced by

    cargo +nightly-2025-04-01 build --release --target wasm32-unknown-unknown \
        -p model-registry -p proof-aggregation -p proof-of-inference
    wasm-opt -Oz --strip-debug --strip-producers <foo>.wasm -o <foo>.opt.wasm

They were built and measured on 2026-07-25:

| File                            | Size   |
|---------------------------------|-------:|
| model-registry.opt.wasm         | 58 045 |
| proof-aggregation.opt.wasm      | 51 490 |
| proof-of-inference.opt.wasm     | 63 152 |

Cap: 65 536 bytes (casper-js-sdk 5.0.12 `installOrUpgrade`). All three fit.

Not committed in binary form to keep git history text-heavy; check them out
by rerunning `scripts/build-and-measure.sh`. This README is the audit trail.

## Reproducibility

Deterministic given `rustc 1.88.0-nightly (0b45675cf 2025-03-31)` and
`binaryen version_119`. Newer rustc nightlies emit different literals in
allocator/panic strings and the sizes will drift by a few hundred bytes.
