# roadmap

## shipped (hackathon)

- [x] 3 contracts on casper testnet (proof-registry, verifier-gate, defi-mock)
- [x] go api server with postgresql
- [x] merkle tree builder (sha-256, configurable depth)
- [x] proof registration and on-chain verification
- [x] kyc whitelisting via merkle inclusion
- [x] defi vault gated by verifier-gate
- [x] react dashboard at casperprover.xyz
- [x] sdk + mcp server
- [x] 83 tests (62 go + 21 rust)
- [x] ci via github actions

## next

- [ ] recursive proof aggregation (batch N proofs into 1 root)
- [ ] zk-snark adapter (groth16 verifier on casper)
- [ ] multi-model proof chains (model A output → model B input → combined proof)
- [ ] proof explorer with search and filter
- [ ] websocket real-time proof feed

## later

- [ ] mainnet deployment
- [ ] proof-of-inference standard proposal
- [ ] integration with model registries (huggingface, replicate)
- [ ] cross-chain proof bridging
- [ ] hardware attestation support (tpm/sgx)
