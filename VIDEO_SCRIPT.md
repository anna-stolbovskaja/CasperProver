# walkthrough script

target: 2 min

## setup

- casperprover.xyz loaded
- terminal with curl
- testnet explorer ready

## hook (0:00-0:08)

[screen: landing page]
"when an ai model runs inference, how do you prove what it computed? CasperProver stores merkle proofs of agent outputs on casper."

## problem (0:08-0:20)

[screen: scroll to features]
"right now if agent A says it ran model X and got output Y, there's no verification. you either re-run it (expensive) or trust it (risky). CasperProver fixes that."

## how it works (0:20-0:45)

[screen: pipeline section]
"the agent submits input hash and output hash. CasperProver builds a merkle tree, registers the root on-chain, and stores the inclusion proof. anyone can verify later — no re-execution needed."

## dashboard (0:45-1:15)

[screen: casperprover.xyz/dashboard]
"72 proofs registered. types include merkle-inclusion, kyc-eligibility, balance-range, transaction-membership. each one is verified on-chain."

[screen: expand proof details]
"proof metadata: submitter key, root hash, leaf hash, tree depth, verification time in milliseconds. every field auditable."

[screen: show kyc section]
"kyc whitelisting built in. only accounts with valid proofs can interact with gated defi contracts."

## contracts (1:15-1:35)

[screen: click contract link → explorer]
"three contracts. proof-registry stores roots and metadata. verifier-gate checks inclusion proofs and manages access. defi-mock is a sample vault gated by the verifier."

## api demo (1:35-1:50)

[screen: terminal]
```bash
curl -X POST https://casperprover-api.onrender.com/proofs \
  -H 'Content-Type: application/json' \
  -d '{"agent":"merkle-prover-v1","input_hash":"abc...","output_hash":"def..."}'
```
"one POST and the proof is registered. GET /proofs/{id} returns full merkle path and on-chain tx hash."

## close (1:50-2:00)

[screen: landing page footer]
"CasperProver. verifiable proofs for agent compute, stored on casper. repo and docs linked below."

[screen: github url + casperprover.xyz]
