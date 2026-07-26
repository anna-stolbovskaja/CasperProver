"""CasperProver CLI.

Entry points registered in ``pyproject.toml``:
  - ``cprover`` (short)
  - ``casperprover`` (long alias)

Commands
--------
- ``cprover health``                     — GET /health
- ``cprover proofs list``                — list all proofs
- ``cprover proofs get <id>``            — fetch a proof
- ``cprover proofs verify <id>``         — verify a proof
- ``cprover proofs submit ...``          — generate a new proof
- ``cprover version``                    — print SDK version

Environment
-----------
- ``CP_BASE_URL`` — default API base URL (fallback: ``http://localhost:9090``).

Exit codes: 0 success, 1 user error, 2 API/HTTP error.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from typing import Any

from . import __version__
from .client import ProverClient


DEFAULT_BASE = os.environ.get("CP_BASE_URL", "http://localhost:9090")


def _print(payload: Any) -> None:
    if isinstance(payload, (dict, list)):
        print(json.dumps(payload, indent=2, ensure_ascii=False))
    else:
        print(payload)


def _mk_client(args: argparse.Namespace) -> ProverClient:
    return ProverClient(base_url=args.base_url, timeout=args.timeout)


# ---------- command handlers ----------
def cmd_health(args: argparse.Namespace) -> int:
    _print(_mk_client(args).health())
    return 0


def cmd_version(_: argparse.Namespace) -> int:
    _print({"casperprover_sdk": __version__})
    return 0


def cmd_proofs_list(args: argparse.Namespace) -> int:
    _print(_mk_client(args).list_proofs())
    return 0


def cmd_proofs_get(args: argparse.Namespace) -> int:
    _print(_mk_client(args).get(args.id))
    return 0


def cmd_proofs_verify(args: argparse.Namespace) -> int:
    ok = _mk_client(args).verify(args.id)
    _print({"proof_id": args.id, "valid": ok})
    return 0 if ok else 2


def cmd_proofs_submit(args: argparse.Namespace) -> int:
    input_data = _read_source(args.input, args.input_file)
    output_data = _read_source(args.output, args.output_file)
    model_data = _read_source(args.model, args.model_file)
    if input_data is None or output_data is None or model_data is None:
        print("error: provide --input/--output/--model (inline) or -file variants", file=sys.stderr)
        return 1
    result = _mk_client(args).submit(
        agent_id=args.agent_id,
        input_data=input_data,
        output_data=output_data,
        model=model_data,
        use_case=args.use_case,
    )
    _print(result)
    return 0


def _read_source(inline: str | None, path: str | None) -> str | None:
    if inline is not None:
        return inline
    if path is not None:
        with open(path, "r", encoding="utf-8") as f:
            return f.read()
    return None


# ---------- parser ----------
def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="cprover",
        description="CasperProver CLI — submit and verify agent proofs.",
    )
    p.add_argument(
        "--base-url",
        default=DEFAULT_BASE,
        help=f"API base URL (default: env CP_BASE_URL or {DEFAULT_BASE!r}).",
    )
    p.add_argument("--timeout", type=float, default=30.0, help="HTTP timeout seconds (default 30).")

    sub = p.add_subparsers(dest="cmd", required=True)

    # health
    sp = sub.add_parser("health", help="GET /health")
    sp.set_defaults(func=cmd_health)

    # version
    sp = sub.add_parser("version", help="Print SDK version.")
    sp.set_defaults(func=cmd_version)

    # proofs
    proofs = sub.add_parser("proofs", help="Proof operations.")
    proofs_sub = proofs.add_subparsers(dest="proofs_cmd", required=True)

    sp = proofs_sub.add_parser("list", help="List proofs.")
    sp.set_defaults(func=cmd_proofs_list)

    sp = proofs_sub.add_parser("get", help="Fetch a proof by id.")
    sp.add_argument("id")
    sp.set_defaults(func=cmd_proofs_get)

    sp = proofs_sub.add_parser("verify", help="Verify a proof by id.")
    sp.add_argument("id")
    sp.set_defaults(func=cmd_proofs_verify)

    sp = proofs_sub.add_parser("submit", help="Generate a new proof.")
    sp.add_argument("--agent-id", required=True)
    sp.add_argument("--use-case", default="inference")
    sp.add_argument("--input", help="Inline input string.")
    sp.add_argument("--input-file", help="Path to file containing input.")
    sp.add_argument("--output", help="Inline output string.")
    sp.add_argument("--output-file", help="Path to file containing output.")
    sp.add_argument("--model", help="Inline model identifier/bytes.")
    sp.add_argument("--model-file", help="Path to file containing model bytes.")
    sp.set_defaults(func=cmd_proofs_submit)

    return p


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    try:
        return args.func(args)
    except RuntimeError as e:
        print(f"error: {e}", file=sys.stderr)
        return 2
    except KeyboardInterrupt:
        return 130


if __name__ == "__main__":
    sys.exit(main())
