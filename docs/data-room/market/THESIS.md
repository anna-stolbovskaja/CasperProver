# Market Thesis

## The gap in one sentence

Regulated buyers increasingly consume AI decisions but cannot verify,
after the fact, that the log they were shown is the log that actually
happened.

## Why now

- **AI decisions are entering regulated workflows** — credit scoring,
  KYC, DeFi risk, trading, compliance. Each of these has a legal
  duty-of-record obligation that "trust the operator's log" no longer
  discharges.
- **Regulators are catching up in 2025–2026** — MiCA (EU), the EU AI
  Act, FinCEN AI risk statements, NIST AI RMF, UK FCA AI feedback
  statement. All converge on evidence, retention, and independent
  verifiability.
- **Cryptographic infrastructure caught up too** — PQ-secure signature
  schemes (ML-DSA, SLH-DSA) are now NIST-standardised (2024). Real
  Groth16 tooling (gnark, arkworks) is production-mature. On-chain
  anchoring is cheap. The primitive is finally practical.

## Why an on-chain audit trail (not a log DB)

Any log DB, however encrypted, can be replaced by whoever holds the
credentials. An on-chain anchor cannot be silently rewritten. The
buyer's downside from "operator faked the log" collapses from "trust
the counterparty" to "verify the Merkle root against the chain," and
the vendor's downside from "we lose the DB" collapses too.

## Why Casper Network

- Native WASM contracts — the crypto surface is straightforward to
  audit.
- Predictable gas via `payment_amount` — the anchor cost is a business
  input, not a probability distribution.
- Post-quantum roadmap alignment — the chain's cryptographic direction
  matches CP's PQ-honest positioning.

Cross-chain adapters (EVM, Solana, Cosmos) are on the roadmap. The
proof primitive itself is chain-agnostic.

## Why CP wins vs. the obvious alternatives

- **DIY on-chain log.** Every team that has tried this has learned it
  is 6–9 months of engineering they did not want to own. CP sells that
  primitive as infrastructure.
- **Log-signing SaaS.** Signs the operator's log, does not commit it
  on-chain, does not have a PQ story, does not have a verifier
  ecosystem. CP does all three.
- **ZK-ML startups.** Solve a research-hard problem (prove the
  model's internal computation was correct). CP does not compete
  there today; it solves the adjacent problem — accountability of the
  decision, not correctness of the computation — where the market is
  ready to buy right now. The two layers compose.

## What has to be true for the thesis to succeed

- Regulators keep tightening the evidence duty (high confidence — the
  trajectory is already public in 2025–2026 rulemaking).
- Buyers accept an infrastructure primitive rather than building it
  in-house (moderate confidence — the DIY tax is well-known).
- CP ships a design-partner motion with 2–3 named partners by end of
  the 90-day roadmap (execution risk — the direct thing to prove
  quarter-over-quarter).

## Where the thesis is thin (and how we plan to close it)

- We have not yet booked a signed design partner. First proof-point.
- We have not yet published the counsel-reviewed legal surface. Pack
  AK milestone under `docs/MAINNET_LAUNCH_PLAN.md`.
- We have not yet completed a third-party crypto audit. RFP under
  `docs/roadmap/LEGAL.md`.

## Cross-references

- `docs/data-room/market/ICP.md` — who the ideal customer is.
- `docs/data-room/market/COMPETITION.md` — how we differ from
  adjacent players.
- `docs/data-room/market/TAM_SAM_SOM.md` — sizing with methodology.
