"""CasperProver REST client (mirror of sdk/python_client.py, packaged)."""

from __future__ import annotations

import json
from typing import Any
from urllib.request import Request, urlopen
from urllib.error import HTTPError


class ProverClient:
    """Synchronous client for CasperProver API."""

    def __init__(self, base_url: str = "http://localhost:9090", timeout: float = 30.0) -> None:
        self._base = base_url.rstrip("/")
        self._timeout = timeout

    # ---------- read ----------
    def health(self) -> dict[str, Any]:
        return self._get("/health")

    def get(self, proof_id: str) -> dict[str, Any]:
        return self._get(f"/proofs/{proof_id}")

    def list_proofs(self) -> list[dict[str, Any]]:
        return self._get("/proofs")

    def verify(self, proof_id: str) -> bool:
        result = self._post(f"/proofs/{proof_id}/verify", {})
        return bool(result.get("valid", False))

    # ---------- write ----------
    def submit(
        self,
        agent_id: str,
        input_data: bytes | str,
        output_data: bytes | str,
        model: bytes | str,
        use_case: str = "inference",
    ) -> dict[str, Any]:
        return self._post(
            "/proofs",
            {
                "agent_id": agent_id,
                "input": input_data.decode() if isinstance(input_data, bytes) else input_data,
                "output": output_data.decode() if isinstance(output_data, bytes) else output_data,
                "model": model.decode() if isinstance(model, bytes) else model,
                "use_case": use_case,
            },
        )

    # ---------- internals ----------
    def _get(self, path: str) -> Any:
        req = Request(self._base + path, method="GET")
        return self._call(req)

    def _post(self, path: str, body: dict) -> Any:
        data = json.dumps(body).encode()
        req = Request(
            self._base + path,
            data=data,
            method="POST",
            headers={"Content-Type": "application/json"},
        )
        return self._call(req)

    def _call(self, req: Request) -> Any:
        try:
            with urlopen(req, timeout=self._timeout) as resp:
                raw = resp.read()
                if not raw:
                    return {}
                return json.loads(raw)
        except HTTPError as e:
            body = e.read().decode(errors="ignore") if hasattr(e, "read") else ""
            raise RuntimeError(f"HTTP {e.code}: {body or e.reason}") from e
