# CasperProver — One-Pager

**What it is.** CasperProver ("CP") is a proof primitive for AI
accountability: a Merkle-anchored, post-quantum-signed, on-chain-attested
record of every AI decision that matters.

**The gap it closes.** Regulated buyers (compliance, audit, DeFi risk)
increasingly consume AI decisions but cannot verify, after the fact, that
the log they were shown is the log that actually happened. CP closes the
**audit-trail** gap. It does not prove the model's internal computation
was correct — that is a research-grade zkML problem the roadmap addresses
separately — it commits inputs, outputs, and the model fingerprint to
Casper Network so any later party can verify the record is unaltered.

**How it works, in one line.** SHA-256 Merkle over decision receipts →
BN254 Groth16 proof over the Merkle root → post-quantum hybrid signature
(Ed25519 + ML-DSA-65 or SPHINCS+) → anchor on Casper Network.

**Live surface (2026-07-27).**

- **9 smart contracts on Casper testnet** (proof-registry, verifier-gate,
  defi-mock, stake-slashing, proof-of-inference, model-registry,
  proof-aggregation, governance, zk-verifier). See `docs/TX_MANIFEST.md`
  for hashes.
- **32-endpoint HTTP API** on Render (`casperprover-api-ylsh.onrender.com`).
- **TS + Python + Go SDKs** published on npm, PyPI, and Go module proxy.
- **MCP server** with 32 tools, so any Model Context Protocol client can
  request and verify proofs.
- **11-tab interactive frontend** for direct exploration.

**Post-quantum honesty.** The PQ signature layer is real (ML-DSA-65 and
SPHINCS+ from NIST PQC), not marketing. Where a claim is *simulation*
we label it `[sim]` at the endpoint level. See `docs/PQ_HONESTY.md` and
`docs/KNOWN_LIMITATIONS.md`.

**Regulatory posture.** MiCA / EU AI Act / GDPR / FinCEN / NIST AI RMF
alignment is mapped in `docs/JUDGE_GUIDE.md`. CP does not offer
financial services directly and does not custody customer keys.

**Team.** See `docs/data-room/company/TEAM.md`.

**Milestones.** See `docs/data-room/company/TIMELINE.md`.

**Contact.** `khrol.studio@gmail.com`.
