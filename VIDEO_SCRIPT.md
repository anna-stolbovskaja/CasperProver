# Demo Video Script

Target: 2:00-2:30

| Timestamp | Content |
|-----------|---------|
| 0:00-0:08 | Title: "CasperProver — Verifiable Proofs for AI Agent Computations" |
| 0:08-0:25 | Problem: AI outputs are opaque. DeFi protocols cannot trust agent decisions without proof. |
| 0:25-0:45 | Solution: Merkle-anchored proofs. Show the formula: `pi = MerkleProof(H(x), H(y), H(M))`. |
| 0:45-1:15 | Demo: Run `./casperprover prove` and `./casperprover verify`. Show the Merkle tree construction and verification output. |
| 1:15-1:35 | KYC flow: Run `./casperprover demo`. Show proof submission to `proof-registry`, verification by `verifier-gate`, and whitelisting by `defi-mock`. |
| 1:35-1:50 | On-chain: Open Casper Explorer. Show the `submit_proof` and `verify_proof` transactions. |
| 1:50-2:10 | API: Show `POST /proofs` request and response. Demonstrate `GET /proofs/{id}` for retrieval. |
| 2:10-2:25 | Architecture: System diagram with engine, 3 contracts, Casper RPC. |
| 2:25-2:30 | Close: "Verifiable computation, on-chain, for any AI agent." |

## Commands

```
cd engine
go build ./cmd/casperprover

./casperprover prove
./casperprover verify
./casperprover demo
API_PORT=8080 ./casperprover serve
```
