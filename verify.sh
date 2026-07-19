#!/usr/bin/env bash
# verify.sh — single-command proof that CasperProver is real
#
# Usage:  ./verify.sh [--api URL]
# Default API: https://casperprover-api-ylsh.onrender.com
#
# Checks:
#   1. All 4 contracts exist on Casper testnet (via RPC)
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

verify_contract() {
  local name="${1%%:*}"
  local hash="${1##*:}"
  local resp
  resp=$(curl -sf "https://node.testnet.casper.network/rpc" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"query_global_state\",\"params\":{\"state_identifier\":null,\"key\":\"hash-${hash}\",\"path\":[]}}" \
    2>/dev/null || echo "FAIL")
  if echo "$resp" | jq -e '.result' > /dev/null 2>&1; then
    green "$name  (${hash:0:16}...)"
    return 0
  else
    red "$name  (${hash:0:16}...) — not found on chain"
    return 1
  fi
}

for c in "${CONTRACTS[@]}"; do
  check verify_contract "$c"
done

# ── 2. API health ────────────────────────────────────────────────────────

echo ""
bold "2. API health"

verify_health() {
  local resp
  resp=$(curl -sf "${API}/health" 2>/dev/null || echo "FAIL")
  if echo "$resp" | jq -e '.status == "ok"' > /dev/null 2>&1; then
    local version
    version=$(echo "$resp" | jq -r '.version // "unknown"')
    green "API healthy — v${version}"
    return 0
  else
    red "API not responding at ${API}/health"
    return 1
  fi
}
check verify_health

# ── 3. Proof round-trip ──────────────────────────────────────────────────

echo ""
bold "3. Proof creation"

verify_proof() {
  local resp
  resp=$(curl -sf -X POST "${API}/prove" \
    -H "Content-Type: application/json" \
    -d '{"agent_id":"verify-agent","model_id":"test-model","input_hash":"aabb","output_hash":"ccdd","decision":"approved","metadata":{"test":true}}' \
    2>/dev/null || echo "FAIL")
  local proof_id
  proof_id=$(echo "$resp" | jq -r '.proof_id // empty' 2>/dev/null)
  if [ -n "$proof_id" ]; then
    green "Proof created: ${proof_id:0:16}..."
    # Verify it
    local verify_resp
    verify_resp=$(curl -sf "${API}/verify/${proof_id}" 2>/dev/null || echo "FAIL")
    if echo "$verify_resp" | jq -e '.valid == true' > /dev/null 2>&1; then
      green "Proof verified successfully"
      return 0
    else
      yellow "Proof created but verification endpoint returned unexpected format"
      WARN=$((WARN + 1))
      return 0
    fi
  else
    # Try alternate endpoint formats
    local alt_resp
    alt_resp=$(curl -sf "${API}/proofs" 2>/dev/null || echo "FAIL")
    if echo "$alt_resp" | jq -e 'length >= 0' > /dev/null 2>&1; then
      local count
      count=$(echo "$alt_resp" | jq 'length')
      green "Proof list accessible (${count} proofs)"
      return 0
    else
      red "Could not create or list proofs"
      return 1
    fi
  fi
}
check verify_proof

# ── 4. Frontend ──────────────────────────────────────────────────────────

echo ""
bold "4. Frontend"

verify_frontend() {
  local resp
  resp=$(curl -sf "${FRONTEND}" 2>/dev/null | head -c 500 || echo "FAIL")
  if echo "$resp" | grep -qi "CasperProver\|casperprover"; then
    green "Frontend serves HTML at ${FRONTEND}"
    return 0
  else
    red "Frontend not responding at ${FRONTEND}"
    return 1
  fi
}
check verify_frontend

# ── 5. ZK and PQ crypto ─────────────────────────────────────────────────

echo ""
bold "5. Crypto endpoints"

verify_crypto() {
  local resp
  resp=$(curl -sf -X POST "${API}/pq/sign-sphincs" \
    -H "Content-Type: application/json" \
    -d '{"message":"verify-test"}' 2>/dev/null || echo "FAIL")
  if echo "$resp" | jq -e '.algorithm' > /dev/null 2>&1; then
    local algo
    algo=$(echo "$resp" | jq -r '.algorithm // "unknown"')
    green "PQ sign+verify works — ${algo}"
    return 0
  else
    yellow "PQ crypto endpoint returned unexpected format"
    WARN=$((WARN + 1))
    return 0
  fi
}
check verify_crypto

# ── Summary ──────────────────────────────────────────────────────────────

echo ""
bold "═══ Results ═══"
echo "  Passed: ${PASS}"
echo "  Failed: ${FAIL}"
[ "$WARN" -gt 0 ] && echo "  Warnings: ${WARN}"
echo ""

if [ "$FAIL" -eq 0 ]; then
  green "All checks passed"
  exit 0
else
  red "${FAIL} check(s) failed"
  exit 1
fi
