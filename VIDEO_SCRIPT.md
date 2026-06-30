# VIDEO_SCRIPT.md — CasperProver
## Format: Faceless Tutorial · 2 min · English
## Topic: Zero-knowledge AI verification — Merkle proof registry for AI agent outputs on Casper Network

---

## HOOK VARIANTS (choose one · first 15 seconds)

### Hook A — The Audit Problem
[NARRATION: Every AI agent running code today is a black box. Nobody can prove what it computed — without replaying the whole model.]
[SHOW: Plain dark screen with text "AI ran. Nobody knows what it did."]
[B-ROLL: Terminal scrolling output rapidly]

### Hook B — The Trust Question
[NARRATION: What if you could prove an AI made a decision — without revealing the model or re-running it? That's what CasperProver does.]
[SHOW: Split screen — AI output on left, blockchain explorer on right]

### Hook C — The On-Chain Moment ← Recommended
[NARRATION: Seventy-two proofs. On-chain. Verifiable. Permanent. This is what cryptographic accountability for AI agents looks like.]
[SHOW: casperprover.xyz/dashboard — proof counter ticking, green verification badges]

---

## SEGMENT 1: PROBLEM (0:00–0:25)

[NARRATION: AI agents are running critical workflows — KYC checks, financial decisions, compliance rules. But there's no audit trail. You can't prove what an agent computed without re-running the entire model.]
[SHOW: Diagram — "Agent computes → output → ?" with a question mark on-chain]
[B-ROLL: Abstract neural network animation]

[NARRATION: That's the accountability gap. CasperProver closes it with Merkle proofs on the Casper Network.]
[SHOW: Text overlay "Merkle-anchored · On-chain · No replay needed"]

---

## SEGMENT 2: SOLUTION OVERVIEW (0:25–0:50)

[NARRATION: Here's how it works. An agent submits its input hash, output hash, and model hash. CasperProver builds a Merkle tree from those three leaves.]
[SHOW: Mermaid-style diagram animating: Agent → Input/Output Hash → Merkle Tree Builder → Root On-Chain]

[NARRATION: The Merkle root gets committed on-chain. The inclusion proof is stored. Anyone can verify — anytime — without re-running the model.]
[SHOW: Arrow from "Verifier queries proof" back to "Inclusion proof stored"]

---

## RE-HOOK at 60 seconds

[NARRATION: So far we've seen the theory. Now let's see 72 live proofs on testnet — and verify one in real time.]
[SHOW: Browser navigating to casperprover.xyz]

---

## SEGMENT 3: LIVE DEMO (0:50–1:35)

[SHOW: Land on casperprover.xyz — hero section "Cryptographic proof registry for AI agent computations"]

[NARRATION: This is CasperProver's live dashboard. We're on the Casper testnet with 72 submitted proofs.]
[SHOW: Navigate to /dashboard — counter "72 proofs registered", status badges green]

[NARRATION: Each proof belongs to one of four types: Merkle inclusion, KYC eligibility, balance range, and transaction membership.]
[SHOW: Proof type breakdown chart/table — 4 categories with counts]

[NARRATION: Let's verify a proof via the API. One curl call — proof ID, and we get back the Merkle path and on-chain root.]
[SHOW: Terminal — curl command]
```bash
curl https://casperprover-api.onrender.com/api/v1/proof/verify \
  -H "Content-Type: application/json" \
  -d '{"proof_id":"proof_001","input_hash":"abc123","output_hash":"def456"}'
```
[SHOW: JSON response with `"valid": true`, merkle_root, inclusion_path]

[NARRATION: Verified. Now let's find that transaction in the Casper explorer.]
[SHOW: Open testnet.cspr.live — paste deploy hash — show contract call to proof-registry]
[SHOW: Contract hash 96e97c4d...a10708 with state entry for proof root]

---

## SEGMENT 4: PROOF TYPES (1:35–1:50)

[NARRATION: Four proof types ship out of the box:]
[SHOW: Animated list appearing one by one]

- **merkle-inclusion** — prove a value was in a computation  
- **kyc-eligibility** — prove a wallet passed KYC without revealing PII  
- **balance-range** — prove a balance was in a range without the exact number  
- **transaction-membership** — prove a tx was processed by an agent  

[B-ROLL: DeFi vault UI showing "Access granted — KYC proof verified"]

---

## SEGMENT 5: CLOSE + CTA (1:50–2:00)

[NARRATION: CasperProver — cryptographic accountability for AI agents. Open source. Live on testnet. Fork it, build on it.]
[SHOW: casperprover.xyz with GitHub link visible]
[SHOW: Text overlay "github.com/anna-stolbovskaja/CasperProver"]

[NARRATION: Link in the description. Star the repo if this solves a problem you've been thinking about.]
[B-ROLL: Dashboard with proofs scrolling, fade out]

---

## PRODUCTION NOTES
- Total runtime: ~115 seconds at normal narration pace
- Screen recordings needed: dashboard, terminal curl, cspr.live explorer
- No face cam required — pure screen + narration
- Background music: lo-fi ambient, -18dB under narration
