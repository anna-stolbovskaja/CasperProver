# Demo (< 10 minutes)

For the mechanics an investor can reproduce locally, follow the judge
guide: `docs/JUDGE_GUIDE.md`. It runs the same 8-criteria vertical
slice that a hackathon judge would.

## TL;DR

```bash
git clone https://github.com/anna-stolbovskaja/casperprover.git
cd casperprover
./verify.sh                     # 8/8 pass locally without secrets
python3 scripts/judge_demo.py   # end-to-end vertical slice
```

## What the demo shows

1. **Merkle proof** — a decision receipt is committed and independently
   verified.
2. **On-chain anchor** — the receipt's Merkle root is anchored on
   Casper testnet; the transaction hash is in
   `deploy-out/onchain.json`.
3. **ZK verification** — `zk/groth16-real/verify` returns a real
   BN254-Groth16 verdict, not a simulation.
4. **PQ signature** — the receipt is signed with a hybrid Ed25519 +
   ML-DSA-65 key. The signature verifies both halves.
5. **Cross-contract** — `verifier-gate` reads `proof-registry` to check
   inclusion; `stake-slashing` reads a revoked proof to slash.

## What the demo does *not* show

- ZK-ML over full model inference. Real Groth16 circuit is roadmap
  (see `docs/roadmap/90-180-DAY.md`); today's Groth16 covers a
  MiMC-preimage as the primitive.
- Multi-party ceremony transcript. Today's setup uses a real gnark
  Phase 1 + Phase 2 with a single-coordinator N-contribution chain;
  see `zk/ceremony/README.md` for the honesty label.
- Mainnet anchoring. Testnet only until the mainnet-launch checklist
  in `docs/MAINNET_LAUNCH_PLAN.md` completes.

## Where to look next

- `docs/JUDGE_GUIDE.md` — 8-criteria judge map.
- `docs/API_REFERENCE.md` — endpoint reference.
- `docs/KNOWN_LIMITATIONS.md` — honest checklist of gaps.
