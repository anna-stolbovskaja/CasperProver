#!/usr/bin/env bash
# verify.sh — single-command proof that CasperProver is real
#
# Usage:  ./verify.sh [--api URL]
# Default API: https://casperprover-api-ylsh.onrender.com
#
# Checks:
#   1. All 8 contracts exist on Casper testnet (via RPC)
#   2. API is live and returns health
#   3. Proof creation round-trip works
#   4. Frontend serves HTML
#   5. ZK proof verification works
#
# Requirements: curl, jq
set -euo pipefail

API="${1:-https://casperprover-api-ylsh.onrender.com}"
FRONTEND="https://casperprover.xyz"
PASS=0
FAIL=0
WARN=0

green()  { printf '\033[32m✅ %s\033[0m\n' "$*"; }
red()    { printf '\033[31m❌ %s\033[0m\n' "$*"; }
yellow() { printf '\033[33m⚠️  %s\033[0m\n' "$*"; }
bold()   { printf '\033[1m%s\033[0m\n' "$*"; }

check() {
  if "$@"; then
    PASS=$((PASS + 1))
  else
    FAIL=$((FAIL + 1))
  fi
}

# ── 1. On-chain contract verification ────────────────────────────────────

bold "═══ CasperProver Verification ═══"
echo ""
bold "1. On-chain contracts (Casper testnet)"

# Load contract manifest — root canonical is deploy-out/onchain.json
# (see docs/MANIFEST.md and scripts/generate_manifest.py). Never edit hashes
# in this script directly; regenerate the manifest instead.
MANIFEST="$(dirname "$0")/deploy-out/onchain.json"
if [ ! -f "$MANIFEST" ]; then
  red "Root manifest missing at $MANIFEST — run: python scripts/generate_manifest.py"
  exit 2
fi

# Extract contract entries as name:hash pairs from the root manifest.
# jq output format: "pretty_name<TAB>contract_hash"
readarray -t CONTRACTS < <(jq -r '
  .contracts | to_entries[]
  | (.key | gsub("_"; " ") | ascii_downcase | split(" ") | map(. as $w | (.[0:1] | ascii_upcase) + $w[1:]) | join(" ")) + ":" + .value.contract_hash
' "$MANIFEST")

if [ ${#CONTRACTS[@]} -eq 0 ]; then
  red "Root manifest contained zero contracts — regenerate: python scripts/generate_manifest.py"
  exit 2
fi
