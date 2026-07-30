# Honesty (pointer)

The single most-important artefact in this data room is the pair of
honesty documents in the main tree:

- `docs/PQ_HONESTY.md` — every post-quantum claim, labelled REAL or
  `[sim]` at endpoint granularity.
- `docs/KNOWN_LIMITATIONS.md` — the honest checklist of what is live,
  what is simulation, what is deferred to the roadmap.

Investors that value team credibility should read these first. A team
willing to publish its own limitations up front is a team that will
not hide the next one.

## What "honest labels" mean here

- **REAL** — real cryptography, real on-chain state, real values. If it
  says REAL and it is not, that is a bug — file it.
- **[sim] / SIMULATION** — the endpoint returns a shape-correct
  response for legacy comparison, but the underlying math is stubbed.
  Responses carry `simulation:true, deprecated:true,
  use:"<real-alt>"` plus `Warning` / `Deprecation` / `Sunset` headers.
- **PLANNED** — described in a roadmap doc, not in code today.
- **TESTNET-ONLY** — on-chain claims refer to Casper testnet during
  the hackathon and beta period.

## Related honesty surfaces

- `docs/JUDGE_GUIDE.md` — regulatory posture map.
- `docs/SECURITY_AUDIT.md` — static audit incl. two follow-ups
  deliberately deferred pre-submission with a documented reason.
- `docs/MERKLE_SCHEME.md` — domain-separation choices and known
  deviations from RFC 6962.
