# submission

## project

**CasperProver** — verifiable proof layer for AI agent computations on Casper Network

## links

| what | where |
|------|-------|
| repo | https://github.com/anna-stolbovskaja/CasperProver |
| site | https://casperprover.xyz |
| lab | https://casperprover.xyz/lab |
| api | https://casperprover-api.onrender.com |
| proof-registry | [96e97c4d...a10708](https://testnet.cspr.live/contract/96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708) |
| verifier-gate | [a37f9cde...9f77d3](https://testnet.cspr.live/contract/a37f9cde9dbdc5bb8b9e92c663bdc59b83b42c89dc75ec73f7f7cde2619f77d3) |
| defi-mock | [fe0c45f6...0b39ef](https://testnet.cspr.live/contract/fe0c45f67c8cd99f0bda0047399a113588870ec0d79d9102f44107303f0b39ef) |
| video | _TBD_ |

## track

agentic infrastructure / verifiable compute

## who

anna-stolbovskaja

## what it does

CasperProver records merkle proofs of AI computations on-chain. an agent submits input + output hashes, the prover builds a merkle tree, registers the root on-chain, and anyone can verify inclusion without re-running the model. supports kyc-gated defi where only whitelisted accounts (proven via merkle inclusion) can interact with protocols.

three contracts:
- proof-registry: stores merkle roots, proof metadata, verification status
- verifier-gate: checks inclusion proofs, manages access control
- defi-mock: sample defi vault gated by verifier-gate

## stack

| part | tech |
|------|------|
| server | go 1.22, net/http, stdlib |
| contracts | rust, casper-contract |
| frontend | react, vite, tailwind |
| db | postgresql (neon) |
| hosting | vercel + render |
| tests | go test (62) + cargo test (21) |
| ci | github actions |

## checklist

- [x] original work
- [x] open source
- [x] working prototype on casper testnet
- [x] on-chain txs (proof registration, verification, kyc whitelist)
- [x] public github + readme
- [x] demo video — _TBD_
