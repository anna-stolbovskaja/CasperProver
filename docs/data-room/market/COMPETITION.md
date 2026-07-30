# Competition

Ground rules from the data-room content standards
(`docs/roadmap/DATA_ROOM.md` §Content standards):

- No name-drop without a linked artefact.
- Comparisons require an artefact (benchmark file, published API,
  regulator statement).

For that reason, competitors are listed by category rather than by
name until a specific comparison artefact is committed. Named
comparisons will appear here only when they cite a public source or
an artefact in the repo.

## Category map

| Category | What they solve | How CP differs |
|---|---|---|
| Log-signing SaaS | Digitally sign the operator's log with a private key | CP anchors on-chain, so verifying does not require trusting a private-key holder; adds a PQ signature; runs a verifier ecosystem |
| DIY on-chain audit trails | Team builds its own Merkle + anchor + verifier pipeline | CP is a productised primitive with SDKs, verifier gate, governance, and honesty labels |
| ZK-ML startups | Prove the *correctness* of the model's internal computation | CP solves the adjacent problem — accountability of the decision, not correctness of the computation — where regulators are ready to buy now |
| Attestation/TEE providers (SGX/SEV/TDX) | Hardware-rooted attestation of the runtime environment | CP is chain-anchored and PQ-signed; hardware attestation is *complementary* input, not a substitute |
| Chain analytics vendors | Post-hoc analysis of on-chain flows | Different problem (they analyse the chain; CP writes to the chain the receipts of a caller's AI decisions) |

## Positioning statement

CasperProver is the **audit-trail** layer. It commits inputs,
outputs, and model fingerprints; a caller can plug in a ZK-ML circuit
(when the market catches up) or a TEE attestation as an input. CP does
not compete with those layers; it composes.

## Where a name-drop will land here

When any of the following are true, this document is updated with the
specific competitor and a linked artefact:

- A regulator or public buyer publishes a comparison.
- CP publishes a benchmark against a named vendor with the benchmark
  file committed to the repo.
- A design partner signs and permits attribution.

Until then, positioning is category-based, per the content standard.
