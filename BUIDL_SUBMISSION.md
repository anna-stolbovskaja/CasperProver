# CasperProver — BUIDL submission

## form

| field | value |
|-------|-------|
| **name** | CasperProver |
| **logo** | red/black terminal icon |
| **category** | crypto/web3 |
| **repo** | https://github.com/anna-stolbovskaja/CasperProver |
| **site** | https://casperprover.xyz |
| **video** | _(pending)_ |
| **social** | github.com/anna-stolbovskaja |

---

## one-liner

merkle proofs for ai agent computations, stored on casper. verify what was computed without re-running it.

## details

### the problem

an ai agent runs inference. produces output. claims it used model X on input Y and got result Z.

how do you verify that?

option 1: re-run the computation. expensive. sometimes impossible (non-deterministic models, proprietary APIs, hardware-specific results).

option 2: trust the agent. fine until money is involved.

option 3: store a cryptographic proof of the computation on-chain. anyone can verify inclusion without re-execution. that's CasperProver.

### how it works

1. agent submits input hash + output hash to CasperProver API
2. server builds a merkle tree from all registered computations
3. merkle root gets written to casper via proof-registry contract
4. inclusion proof (merkle path) is stored and queryable
5. any third party calls verifier-gate contract with the proof — gets a boolean

the proof is compact. a tree of depth 12 has a 12-hash path regardless of how many leaves exist. verification is O(log n) hashes, not O(n) re-computation.

### why it matters

defi protocols need to trust agent output before acting on it. a lending protocol shouldn't liquidate a position because an agent said the price dropped — it should verify the agent's computation is in the registered merkle root.

CasperProver adds that verification layer.

### kyc gating

the verifier-gate contract also manages a kyc whitelist. accounts proven via merkle inclusion proofs get whitelisted for interaction with gated protocols.

defi-mock is a sample vault that only accepts deposits from whitelisted accounts. the whitelist check happens on-chain through verifier-gate. no off-chain dependency.

this pattern generalizes: any contract can call verifier-gate to check if a user has a valid proof before proceeding.

### contracts

three on casper testnet:

**proof-registry** (`96e97c4d...a10708`)
- stores merkle roots, proof metadata (agent, input/output hashes, model hash)
- verification status and timestamps
- supports batch registration

**verifier-gate** (`a37f9cde...9f77d3`)
- verifies inclusion proofs against registered roots
- manages kyc whitelist
- access control for downstream contracts

**defi-mock** (`b9b11a97...b81d3`)
- sample defi vault gated by verifier-gate
- deposit/withdraw restricted to whitelisted accounts
- demonstrates the gating pattern

### server

go 1.22, standard library only (net/http, crypto/sha256, encoding/json). no frameworks. ~2200 lines.

endpoints:
- POST /proofs — register new proof
- GET /proofs/:id — get proof with merkle path
- GET /proofs — list with pagination
- POST /verify — check inclusion
- GET /health — status + contract info
- GET /kyc/:account — check whitelist status

postgresql on neon. two tables: proofs (17 columns), kyc_whitelist (4 columns).

### frontend

react + vite + tailwind. black/red theme. dashboard shows proof list with status badges, kyc section, contract links. navbar links to landing page sections from both landing and dashboard views.

### numbers

- 72 proofs registered in db
- 22 kyc entries
- 83 tests (62 go + 21 rust)
- 3 contracts deployed
- api response time: <50ms for proof lookup
- merkle verification: 5-180ms depending on tree depth

### what's different

existing proof-of-computation projects focus on zk-snarks (heavy, slow to generate) or optimistic verification (requires challenge periods). CasperProver uses merkle trees — simple, fast, and sufficient for most agent verification use cases.

a merkle proof takes milliseconds to generate and verify. a zk-snark takes seconds to minutes. for the common case of "prove this agent actually computed this output," merkle inclusion is the right tool.

zk-snark support is on the roadmap for cases that need zero-knowledge properties (proving computation without revealing inputs). but the merkle layer works today.

### next

recursive proof aggregation, zk-snark adapter, multi-model proof chains, cross-chain bridging.
