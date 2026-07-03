# CasperProver — VIDEO SCRIPT
**Format:** Faceless tutorial | ~2 min | English  
**Style:** Terminal-first, KYC-demo first  
**No voiceover** — subtitles + tooltips + background music

---

## HOOK OPTIONS (pick one — first 5 seconds)

**Hook A:**
[SHOW: terminal with curl command generating a KYC proof]  
[SUBTITLE: "Every AI decision. Tamper-proof. On-chain in 1 second."]

**Hook B:**
[SHOW: testnet.cspr.live/deploy/ with a proof deploy hash loading live]  
[SUBTITLE: "This KYC approval just got permanently anchored to Casper blockchain."]

**Hook C:**
[SHOW: split-screen — bank lab left, Casper explorer right, deploy hash matching]  
[SUBTITLE: "Without proof, an AI said 'approved'. Now anyone can verify it forever."]

---

## SECTION 1 — KYC Demo (0:00–0:40)

[SHOW: Terminal / lab. Use case selector set to "KYC / Identity"]

[NARRATION / SUBTITLE]
> "Open the lab. Go to the Generate tab. Switch to **Anchored** mode — this writes the proof to Casper testnet."

[SHOW: Fill in Agent = "kyc-verifier-v2", Model = "kyc-model-v2.1", Use Case = KYC]

[SUBTITLE: "Input: passport data, country, user ID."]

[SHOW: Paste input JSON]
```json
{"user_id":"alice_0x3f","doc_type":"passport","country":"DE","issued":"2022-03-15"}
```

[SHOW: Paste output JSON]
```json
{"verified":true,"confidence":0.97,"risk_score":12,"flags":[]}
```

[SHOW: Click "Generate Proof" button — progress steps animate: Hashing inputs → Merkle tree → Generating proof → Anchoring on-chain → Complete]

[SUBTITLE: "5 steps. 4 fields. One cryptographic proof."]

[SHOW: Result panel — proof_hash, merkle_root, deploy_hash appear]

[SHOW: Click "View Deploy" → browser opens testnet.cspr.live with the deploy hash]

[TOOLTIP: "This is your proof. It lives on-chain. Forever."]

---

## RE-HOOK at ~0:40

[SHOW: Casper explorer deploy detail page — inputs/outputs visible]  
[SUBTITLE: "Regulators, auditors, users — anyone can verify this KYC without touching PII."]

---

## SECTION 2 — Proof Registry & Verification (0:40–1:20)

[SHOW: Switch to Proofs tab — list of attestations, each row with proof_hash, deploy_hash]

[SUBTITLE: "Every proof is indexed on your lab — filter by agent, use case, or mode."]

[SHOW: Click a proof row to expand detail — merkle_root, factors hash shown]

[SHOW: Click "Verify on-chain" link next to a proof → testnet.cspr.live/deploy/... opens]

[B-ROLL: Casper explorer showing deploy details, hash matching the lab]

[SHOW: Switch to Verify tab. Enter the proof ID]

[SUBTITLE: "Paste the original input and output to run full cryptographic verification."]

[SHOW: Click Verify — result shows `verified: true`, `inputs_match: true`, `output_match: true`]

[TOOLTIP: "Zero-knowledge integrity — the model can't deny what it decided."]

---

## SECTION 3 — Use Cases & Overview (1:20–1:55)

[SHOW: Demo tab — 3 use case cards: KYC, Loan Approval, Insurance Claim]

[SHOW: Click "Try it" on Loan Approval scenario — auto-fills Generate form]

[SUBTITLE: "Same flow: regulator-grade proof in seconds."]

[B-ROLL: Overview tab — stats: Total Proofs, Valid, Avg Generation time]

[SHOW: Deployed Contracts section — click Scoring Registry → testnet.cspr.live/contract/...]

[TOOLTIP: "Contract hash: 96e97c4d... — deployed and live on Casper testnet."]

---

## OUTRO (1:55–2:00)

[SHOW: Lab overview with stats + Casper explorer in background]  
[SUBTITLE: "CasperProver. AI accountability, on-chain."]  
[B-ROLL: GitHub repo URL]
