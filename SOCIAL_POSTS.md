# SOCIAL_POSTS.md — CasperProver

---

## Twitter/X Thread (6 tweets)

**Tweet 1 — Hook**
AI agents are running critical workflows right now — KYC, compliance, DeFi access control.

Nobody can audit them.

You can't prove what an agent computed without re-running the whole model. That's a systemic accountability gap.

Here's how CasperProver fixes it 🧵

---

**Tweet 2 — The Problem**
The verification options today:
• Re-run the model (expensive, often impossible)
• Trust operator logs (mutable, centralized)
• Reveal raw inputs (privacy violation)

For regulated industries and high-stakes DeFi, none of these work.

We need cryptographic proof. Not promises.

---

**Tweet 3 — The Solution**
CasperProver: a Merkle proof registry on Casper Network.

Agent submits H(input) + H(output) + H(model)
→ Merkle tree built
→ Root committed on-chain
→ Inclusion proof stored
→ Anyone can verify, anytime, without replaying the model

One curl call to verify. Immutable. Permanent.

---

**Tweet 4 — Live Demo**
72 proofs registered on Casper testnet right now.

Dashboard: casperprover.xyz/dashboard
API: casperprover-api.onrender.com

Try it:
```
curl .../proof/verify -d '{"proof_id":"proof_001"}'
→ {"valid": true, "merkle_root": "..."}
```

On-chain tx visible in testnet.cspr.live. Real deploys. Real state.

---

**Tweet 5 — Four Proof Types**
Ships with 4 proof types:

🔐 merkle-inclusion — general AI output audit
✅ kyc-eligibility — prove KYC passed, no PII revealed  
💰 balance-range — prove balance in range, no exact value
📋 transaction-membership — prove a tx was processed by a specific agent

KYC-gated DeFi demo included — vault gates on proof validity, not operator trust.

---

**Tweet 6 — CTA**
Open source. MCP adapter included (Claude/LangChain-compatible).

If you're building AI agents that need an audit trail — or DeFi protocols that need to gate on AI outputs — this is the cryptographic layer.

⭐ github.com/anna-stolbovskaja/CasperProver

Built for @CasperNetwork Casper Agentic Buildathon.

---

## LinkedIn Post (150 words)

**AI accountability is the next compliance frontier.**

Right now, AI agents are making consequential decisions — KYC approvals, compliance checks, DeFi access control — and there is no cryptographic audit trail. Logs are mutable. Models are black boxes. Operators control the evidence.

CasperProver changes that. It's a Merkle proof registry on Casper Network: an AI agent submits hashes of its inputs, outputs, and model at inference time. The Merkle root is committed on-chain. Anyone can verify the inclusion proof forever — without re-running the model, without revealing PII.

72 proofs live on testnet. Four proof types: general audit, KYC eligibility, balance range, and transaction membership. A KYC-gated DeFi demo shows the end-to-end flow — verifier gate contract unlocks vault access based on on-chain proof validity, not operator trust.

For teams building regulated AI pipelines or compliance infrastructure, this is the primitive you've been missing.

→ casperprover.xyz | github.com/anna-stolbovskaja/CasperProver

---

## Telegram Announcement (3 sentences)

🔐 **CasperProver is live on Casper testnet** — a Merkle proof registry for AI agent computations. Submit hashes of any agent's inputs, outputs, and model; get an on-chain inclusion proof you can verify anytime without replaying the model. 72 proofs registered, 4 proof types, KYC-gated DeFi demo included → casperprover.xyz/dashboard
