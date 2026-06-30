# Can You Trust Your AI Agent? CasperProver Brings Cryptographic Accountability to On-Chain AI

*Published: 2026-06-30 · 7 min read*

---

AI agents are running consequential code right now — and almost nobody can audit them.

They approve loans. They run KYC. They decide which wallets get access to DeFi protocols. They execute compliance rules on behalf of institutions. And when something goes wrong, the standard response is: *"We'll check the logs."*

The logs are mutable. The model is a black box. The operator controls the audit trail. That is not accountability — it is theater.

---

## The Accountability Gap in AI-Driven Systems

Here is the concrete problem: given an AI agent that computed `f(x) = y` using model `M`, how do you prove that output `y` was the actual result — without re-running the model, without revealing the inputs, without trusting the operator?

Today, you can't. You either:

1. **Re-run the model** — expensive, often impossible (proprietary weights, inference costs, non-determinism)
2. **Trust the operator's logs** — centralized, mutable, and the operator has every incentive to alter them
3. **Reveal PII** — if the proof requires showing the raw input (e.g. passport scan for KYC), you've violated privacy to achieve "verification"

None of these work for regulated industries. None of them work for high-stakes DeFi. And as autonomous AI agents get deployed in more critical systems, the gap between "agent said so" and "cryptographically proven" becomes a systemic risk.

---

## The Solution: Merkle Proofs as an AI Audit Layer

CasperProver is a Merkle proof registry deployed on Casper Network. The core idea is simple:

**Hash the inputs, outputs, and model. Build a Merkle tree. Commit the root on-chain. Store the inclusion proof. Anyone can verify — forever — without re-running anything.**

Concretely:

```
π = MerkleProof(H(x), H(y), H(M))
```

where `H = SHA-256`. The leaf set `{H(x), H(y), H(M)}` forms a binary Merkle tree. The root `r` is submitted as a deploy to the Proof Registry contract on Casper testnet. The inclusion proof `π` is stored and queryable via API.

Verification is a single API call:

```bash
curl https://casperprover-api.onrender.com/api/v1/proof/verify \
  -d '{"proof_id": "proof_001"}'
# → {"valid": true, "merkle_root": "abc...", "on_chain_tx": "0x..."}
```

That's it. No model re-run. No operator trust. The on-chain root is immutable — if someone tampers with the stored proof, the Merkle path won't reconstruct the registered root.

---

## Demo Walkthrough

CasperProver has 72 proofs registered on Casper testnet right now. Here's the live flow:

**1. Land on [casperprover.xyz/dashboard](https://casperprover.xyz/dashboard)**

The dashboard shows all registered proofs — type, timestamp, verification status. Green badges mean the on-chain root is present and the inclusion path is valid.

**2. Submit a proof via the API**

An AI agent (or any HTTP client) submits input + output + model hashes:

```bash
curl https://casperprover-api.onrender.com/api/v1/proof/submit \
  -H "Content-Type: application/json" \
  -d '{
    "proof_type": "kyc-eligibility",
    "input_hash": "sha256:...",
    "output_hash": "sha256:...",
    "model_id": "compliance-v2"
  }'
```

The Go engine builds the Merkle tree, submits the root to the Proof Registry contract, and returns a `proof_id`.

**3. Find the on-chain transaction**

Every proof submission is a real Casper deploy. You can look up the Proof Registry contract ([96e97c4d...a10708](https://testnet.cspr.live/contract/96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708)) in testnet.cspr.live and see the registered Merkle roots directly in contract state.

**4. Gate downstream access**

The Verifier Gate contract checks inclusion proofs. The DeFi Mock contract only allows deposits from wallets with a valid `kyc-eligibility` proof. This is the full end-to-end: agent computes → proof registered → gate verifies → vault unlocks.

---

## Four Proof Types Explained

CasperProver ships with four proof types covering the most common AI accountability scenarios:

**`merkle-inclusion`** — General purpose. Proves a specific value was part of an agent's computation. Use when you need a tamper-evident audit log of any AI output.

**`kyc-eligibility`** — KYC result without PII. The agent proves a wallet passed KYC without revealing the raw identity documents. The hash of the eligibility decision is committed; the underlying data stays private.

**`balance-range`** — Range proofs for financial data. An agent can prove a user's balance was between $10,000 and $100,000 without revealing the exact number. Critical for creditworthiness checks in privacy-preserving DeFi.

**`transaction-membership`** — Proves a specific transaction was processed by a specific agent. Useful for compliance audits, dispute resolution, and agent liability tracking.

---

## Building on CasperProver

The SDK ships with Go and Python clients. If you're using an AI framework that supports Model Context Protocol (MCP), there's a native MCP server — your agent can call CasperProver as a tool with zero custom integration:

```go
// Go SDK
client := casperprover.NewClient("https://casperprover-api.onrender.com")
proofID, err := client.Submit(ctx, casperprover.SubmitRequest{
    ProofType:  "merkle-inclusion",
    InputHash:  sha256sum(input),
    OutputHash: sha256sum(output),
    ModelID:    "my-model-v1",
})
```

The MCP server means frameworks like LangChain, Claude Desktop, or any MCP-compatible runtime can register proofs at inference time — before the output even hits downstream systems.

---

## Why Casper?

Casper's account model and contract-level storage make it straightforward to build a proof registry where each entry is queryable by key, and where access control logic (the Verifier Gate) is a separate contract that downstream protocols call trustlessly. The deterministic execution model matters here: a verification call has a known outcome with no oracle dependency.

---

## What's Next

CasperProver is a hackathon submission — but the problem it solves is real and growing. The roadmap includes:

- **Mainnet deployment**
- **zk-SNARK proof type** (Groth16) for full zero-knowledge verification
- **Batch proof submission** for high-throughput agent pipelines
- **SDK integrations** for more AI frameworks

---

## Try It

- **Dashboard:** [casperprover.xyz/dashboard](https://casperprover.xyz/dashboard)
- **API:** [casperprover-api.onrender.com](https://casperprover-api.onrender.com)
- **GitHub:** [github.com/anna-stolbovskaja/CasperProver](https://github.com/anna-stolbovskaja/CasperProver)

If you're building AI agents that need an audit trail — or DeFi protocols that need to gate on AI outputs — CasperProver gives you the cryptographic primitives to do it without trusting anyone.

Star the repo if this solves a problem you've been thinking about.
