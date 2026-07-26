"""Quickstart example for the CasperProver Python SDK.

Usage::

    CP_API_URL=http://localhost:9090 CP_API_KEY=... \\
        python -m casperprover.examples.quickstart

(or just run this file directly from `sdk/python/`).
"""

from __future__ import annotations

import os
import sys

# Add ../../python to sys.path when run standalone so `casperprover` resolves.
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "..", "python"))

from casperprover import Client, ProveRequest  # noqa: E402


def main() -> None:
    base_url = os.environ.get("CP_API_URL", "http://localhost:9090")
    api_key = os.environ.get("CP_API_KEY")

    c = Client(base_url=base_url, api_key=api_key)

    # 1. Health
    print("health:", c.health())

    # 2. Prove
    proof = c.prove(
        ProveRequest(
            agent="example-agent",
            model="gpt-toy-v1",
            input="hello world",
            output="42",
            use_case="quickstart",
        ),
        idempotency_key="quickstart-1",
    )
    print(f"proof: id={proof.id} vk_hash={proof.vk_hash}")

    # 3. Verify
    v = c.verify(proof.id)
    print(f"verify: valid={v.valid}")


if __name__ == "__main__":  # pragma: no cover
    main()
