# Ideal Customer Profile

CasperProver serves teams that (a) run AI decisions in a workflow that
has a legal duty-of-record obligation and (b) will fail a regulator's
audit if the log is not independently verifiable. Three archetypes.

## ICP #1 — Regulated DeFi risk / compliance platform

- **Shape.** 20–100 people, Series A to B. Sells risk scoring,
  transaction monitoring, or on-chain compliance tooling to banks,
  exchanges, or custodians.
- **Trigger.** MiCA / FinCEN / EU AI Act requires them to produce a
  tamper-evident record of AI-driven decisions on demand. Their
  in-house log is not currently verifiable to a third party.
- **Value.** Buys CP because "our AI decisions are anchored on Casper
  Network and independently verifiable" moves a regulatory-response
  cycle from weeks to a link.
- **Willingness to pay.** Priced against internal engineering: 6–9
  months of a dedicated 2-eng team to build the primitive in-house.

## ICP #2 — AI-native SaaS with an audited buyer

- **Shape.** 10–50 people, seed to Series A. Sells AI decisions
  (credit, hiring, insurance triage) into a regulated buyer.
- **Trigger.** Their buyer's audit team asks for a duty-of-record
  demonstration. They cannot produce one that does not require
  trusting them.
- **Value.** Buys CP because the audit team accepts a link to a
  Merkle-anchored receipt where they would not accept a signed
  log-file.
- **Willingness to pay.** Priced against the deal they lose if the
  audit team says no. Usually more than they are paid per audit.

## ICP #3 — Enterprise AI risk / governance team

- **Shape.** Internal team at a bank / carrier / auditor / regulator.
- **Trigger.** Board-level or regulator-level requirement to
  independently verify AI decisions used in the enterprise.
- **Value.** Buys CP as a horizontal primitive across the enterprise's
  own AI vendors — "every vendor must anchor their decisions here."
- **Willingness to pay.** Enterprise pricing; usually a shorter list
  of deals but larger. Not the near-term wedge.

## Where CP is *not* the right fit

- Teams whose AI decisions have no downstream party that will ever
  audit them. There is no forcing function.
- Teams that need to prove the *correctness* of the model's internal
  computation. That is the zkML problem; CP is complementary but does
  not solve it today.
- Teams that require on-chain state on a chain CP has not yet
  adapted to. The chain-adapter roadmap exists; today's live chain is
  Casper.

## First-partner motion

The 30-day and 90-180-day roadmaps target signing 2–3 partners in
ICP #1 first (highest urgency + fastest willingness-to-pay signal). See
`docs/data-room/traction/DESIGN_PARTNERS.md` for status.
