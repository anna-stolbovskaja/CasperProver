#!/usr/bin/env bash
# CI-runnable benchmark suite for CasperProver's hot paths (Merkle tree
# build/root/path, raw hashing, proof generate/verify).
#
# Usage:
#   scripts/run_benchmarks.sh                # run + print, don't persist
#   scripts/run_benchmarks.sh --baseline      # run + write baseline JSON
#
# Baseline metrics are written to benchmarks/baseline.json (ns/op + B/op +
# allocs/op per benchmark case), so regressions can be diffed by eye or by a
# future CI gate. This script never fails the build on drift by itself —
# it just produces the artifact.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${REPO_ROOT}/benchmarks"
RAW_OUT="${OUT_DIR}/last_run.txt"
BASELINE_OUT="${OUT_DIR}/baseline.json"

mkdir -p "${OUT_DIR}"

cd "${REPO_ROOT}/engine"
echo "Running Go benchmarks (prover + verifier)..."
go test -run '^$' -bench '.' -benchmem \
  ./internal/prover/... ./internal/verifier/... | tee "${RAW_OUT}"

if [[ "${1:-}" == "--baseline" ]]; then
  echo "Writing baseline metrics to ${BASELINE_OUT}"
  python3 "${REPO_ROOT}/scripts/bench_to_json.py" "${RAW_OUT}" "${BASELINE_OUT}"
fi

echo "Done. Raw output: ${RAW_OUT}"
