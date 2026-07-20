#!/usr/bin/env python3
"""Read-only CasperProver judge demo; live writes require explicit opt-in."""
from __future__ import annotations
import argparse, json, os, sys, urllib.error, urllib.request
from dataclasses import dataclass

DEFAULT_API = "https://casperprover-api-ylsh.onrender.com"
DEFAULT_SITE = "https://casperprover.xyz"
RPC = "https://node.testnet.casper.network/rpc"

# Canonical on-chain contract manifest lives at deploy-out/onchain.json in repo
# root (Gate 1). Load it dynamically so hashes are never duplicated in scripts.
# Falls back to a pinned map only if the manifest is missing (e.g. a stripped
# archive) — logged loudly so nobody ships against a stale fallback silently.
_MANIFEST_LABELS = {
    "proof_registry": "Proof Registry",
    "verifier_gate": "Verifier Gate",
    "defi_mock": "DeFi Mock",
    "stake_slashing": "Stake Slashing",
}


def _load_contracts() -> dict[str, str]:
    here = os.path.dirname(os.path.abspath(__file__))
    manifest = os.path.join(here, "..", "deploy-out", "onchain.json")
    try:
        with open(manifest, "r", encoding="utf-8") as fh:
            data = json.load(fh)
        contracts = data.get("contracts", {})
        out = {}
        for key, entry in contracts.items():
            label = _MANIFEST_LABELS.get(key, key.replace("_", " ").title())
            h = entry.get("contract_hash")
            if h:
                out[label] = h
        if out:
            return out
    except (OSError, json.JSONDecodeError) as exc:
        print(f"[judge_demo] WARNING: manifest load failed ({exc}); using pinned fallback", file=sys.stderr)
    return {
        "Proof Registry": "96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708",
        "Verifier Gate": "a37f9cde9dbdc5bb8b9e92c663bdc59b83b42c89dc75ec73f7f7cde2619f77d3",
        "DeFi Mock": "fe0c45f67c8cd99f0bda0047399a113588870ec0d79d9102f44107303f0b39ef",
        "Stake Slashing": "1ad1b3d94be631532d6daf3a195fafc9dfe8a16504e87d87784d51089b983d52",
    }


CONTRACTS = _load_contracts()

@dataclass
class Result:
    name: str
    ok: bool
    detail: str

class Client:
    def __init__(self, timeout: float = 20): self.timeout = timeout
    def request(self, url: str, *, method="GET", payload=None, api_key=""):
        data = json.dumps(payload).encode() if payload is not None else None
        headers = {"Accept": "application/json", "User-Agent": "CasperProver-Judge-Demo/1.0"}
        if data: headers["Content-Type"] = "application/json"
        if api_key: headers["X-API-Key"] = api_key
        req = urllib.request.Request(url, data=data, headers=headers, method=method)
        with urllib.request.urlopen(req, timeout=self.timeout) as response:
            body = response.read().decode("utf-8", errors="replace")
            return response.status, json.loads(body) if body else {}

def contract_checks(client: Client) -> list[Result]:
    out = []
    for name, contract_hash in CONTRACTS.items():
        payload = {"jsonrpc":"2.0","id":1,"method":"query_global_state","params":{"state_identifier":None,"key":f"hash-{contract_hash}","path":[]}}
        try:
            _, body = client.request(RPC, method="POST", payload=payload)
            out.append(Result(name, "result" in body, f"{contract_hash[:16]}… on Casper testnet" if "result" in body else str(body.get("error", "not found"))))
        except Exception as exc: out.append(Result(name, False, friendly_error(exc)))
    return out

def read_checks(client: Client, api: str, site: str) -> list[Result]:
    checks = []
    try:
        _, body = client.request(f"{api}/health")
        checks.append(Result("API health", body.get("status") == "ok", f"version {body.get('version', 'unknown')}"))
    except Exception as exc: checks.append(Result("API health", False, friendly_error(exc)))
    try:
        _, body = client.request(f"{api}/proofs")
        count = len(body) if isinstance(body, list) else len(body.get("proofs", []))
        checks.append(Result("Proof registry API", True, f"{count} proof record(s) readable"))
    except Exception as exc: checks.append(Result("Proof registry API", False, friendly_error(exc)))
    try:
        req = urllib.request.Request(site, headers={"User-Agent":"CasperProver-Judge-Demo/1.0"})
        with urllib.request.urlopen(req, timeout=client.timeout) as response:
            text = response.read(200_000).decode(errors="replace").lower()
        checks.append(Result("Frontend", "casperprover" in text, site))
    except Exception as exc: checks.append(Result("Frontend", False, friendly_error(exc)))
    return checks

def crypto_checks(client: Client, api: str, api_key: str) -> list[Result]:
    if not api_key:
        return [Result("Real Groth16 round-trip", True, "SKIP — set CP_JUDGE_API_KEY; no hidden/default key")]
    try:
        _, proof = client.request(f"{api}/zk/groth16-real/prove", method="POST", payload={"preimage":"42"}, api_key=api_key)
        request = {"hash": proof["hash"], "proof_hex": proof["proof_hex"]}
        _, verified = client.request(f"{api}/zk/groth16-real/verify", method="POST", payload=request, api_key=api_key)
        ok = bool(verified.get("valid"))
        return [Result("Real Groth16 round-trip", ok, "real off-chain gnark/BN254 MiMC prove + verify; not on-chain pairing verification")]
    except Exception as exc: return [Result("Real Groth16 round-trip", False, friendly_error(exc))]

def friendly_error(exc: Exception) -> str:
    if isinstance(exc, urllib.error.HTTPError): return f"HTTP {exc.code} {exc.reason}"
    if isinstance(exc, urllib.error.URLError): return f"network error: {exc.reason}"
    return f"{type(exc).__name__}: {exc}"

def print_result(r: Result):
    skipped = r.detail.startswith("SKIP")
    status = "SKIP" if skipped else ("PASS" if r.ok else "FAIL")
    color = "\033[33m" if skipped else ("\033[32m" if r.ok else "\033[31m")
    print(f"{color}[{status}]\033[0m {r.name}: {r.detail}")

def main() -> int:
    p = argparse.ArgumentParser(description="Reproducible, read-only-by-default CasperProver judge demo")
    p.add_argument("--api", default=DEFAULT_API); p.add_argument("--site", default=DEFAULT_SITE)
    p.add_argument("--api-key", default=os.getenv("CP_JUDGE_API_KEY", ""), help="or set CP_JUDGE_API_KEY; never committed")
    p.add_argument("--timeout", type=float, default=20); p.add_argument("--json", action="store_true")
    args = p.parse_args(); client = Client(args.timeout)
    print(r"""[36m
   ____                          ____
  / ___|__ _ ___ _ __   ___ _ _|  _ \ _ __ _____   _____ _ __
 | |   / _` / __| '_ \ / _ \ '__| |_) | '__/ _ \ \ / / _ \ '__|
 | |__| (_| \__ \ |_) |  __/ |  |  __/| | | (_) \ V /  __/ |
  \____\__,_|___/ .__/ \___|_|  |_|   |_|  \___/ \_/ \___|_|
                |_|       JUDGE VERIFICATION[0m""")
    results = contract_checks(client) + read_checks(client, args.api.rstrip('/'), args.site.rstrip('/')) + crypto_checks(client, args.api.rstrip('/'), args.api_key)
    if args.json: print(json.dumps([r.__dict__ for r in results], indent=2))
    else:
        for result in results: print_result(result)
        print("\nProof boundary: REAL CRYPTO = off-chain gnark/BN254 MiMC; ON-CHAIN = Casper hashes/registries; SIMULATION = legacy conceptual endpoints.")
        print("Docs: https://github.com/anna-stolbovskaja/CasperProver#readme")
    failures = sum(not r.ok for r in results)
    skipped = sum(r.detail.startswith("SKIP") for r in results)
    passed = len(results) - failures - skipped
    print(f"\nSummary: {passed} passed, {skipped} skipped, {failures} failed")
    return 1 if failures else 0

if __name__ == "__main__": raise SystemExit(main())
