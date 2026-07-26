"""Unit tests for the CasperProver CLI.

These tests exercise the argument parser and command handlers with a fake
``ProverClient`` — no network calls.
"""

from __future__ import annotations

import io
import json
import sys
from contextlib import redirect_stdout
from typing import Any
from unittest.mock import patch

import pytest

from casperprover_sdk import cli


class FakeClient:
    def __init__(self, base_url: str = "http://x", timeout: float = 30.0):
        self.base_url = base_url
        self.timeout = timeout
        self.calls: list[tuple[str, tuple, dict]] = []
        self._verify_result = True

    def health(self) -> dict[str, Any]:
        self.calls.append(("health", (), {}))
        return {"status": "ok"}

    def list_proofs(self) -> list[dict]:
        self.calls.append(("list_proofs", (), {}))
        return [{"id": "p1"}, {"id": "p2"}]

    def get(self, proof_id: str) -> dict:
        self.calls.append(("get", (proof_id,), {}))
        return {"id": proof_id, "agent_id": "agent-1"}

    def verify(self, proof_id: str) -> bool:
        self.calls.append(("verify", (proof_id,), {}))
        return self._verify_result

    def submit(self, **kwargs) -> dict:
        self.calls.append(("submit", (), kwargs))
        return {"id": "new-proof", **kwargs}


@pytest.fixture
def fake_client(monkeypatch):
    inst = FakeClient()

    def _factory(base_url="http://x", timeout=30.0):
        inst.base_url = base_url
        inst.timeout = timeout
        return inst

    monkeypatch.setattr(cli, "ProverClient", _factory)
    return inst


def _run(argv: list[str]) -> tuple[int, str]:
    buf = io.StringIO()
    with redirect_stdout(buf):
        code = cli.main(argv)
    return code, buf.getvalue()


def test_version_returns_json(fake_client):
    code, out = _run(["version"])
    assert code == 0
    payload = json.loads(out)
    assert "casperprover_sdk" in payload


def test_health_hits_client(fake_client):
    code, out = _run(["health"])
    assert code == 0
    assert json.loads(out) == {"status": "ok"}
    assert fake_client.calls[0][0] == "health"


def test_proofs_list(fake_client):
    code, out = _run(["proofs", "list"])
    assert code == 0
    assert json.loads(out) == [{"id": "p1"}, {"id": "p2"}]


def test_proofs_get(fake_client):
    code, out = _run(["proofs", "get", "abc123"])
    assert code == 0
    payload = json.loads(out)
    assert payload["id"] == "abc123"
    assert fake_client.calls[-1] == ("get", ("abc123",), {})


def test_proofs_verify_valid(fake_client):
    fake_client._verify_result = True
    code, out = _run(["proofs", "verify", "abc123"])
    assert code == 0
    payload = json.loads(out)
    assert payload == {"proof_id": "abc123", "valid": True}


def test_proofs_verify_invalid(fake_client):
    fake_client._verify_result = False
    code, out = _run(["proofs", "verify", "abc123"])
    assert code == 2
    payload = json.loads(out)
    assert payload == {"proof_id": "abc123", "valid": False}


def test_proofs_submit_inline(fake_client):
    code, out = _run([
        "proofs", "submit",
        "--agent-id", "agent-1",
        "--input", "hello",
        "--output", "world",
        "--model", "model-v1",
        "--use-case", "inference",
    ])
    assert code == 0
    payload = json.loads(out)
    assert payload["id"] == "new-proof"
    kwargs = fake_client.calls[-1][2]
    assert kwargs["agent_id"] == "agent-1"
    assert kwargs["input_data"] == "hello"
    assert kwargs["output_data"] == "world"
    assert kwargs["model"] == "model-v1"
    assert kwargs["use_case"] == "inference"


def test_proofs_submit_missing_inputs(fake_client, capsys):
    code = cli.main([
        "proofs", "submit", "--agent-id", "agent-1",
    ])
    assert code == 1
    err = capsys.readouterr().err
    assert "provide" in err.lower()


def test_proofs_submit_from_files(fake_client, tmp_path):
    ip = tmp_path / "in.txt"; ip.write_text("in-body")
    op = tmp_path / "out.txt"; op.write_text("out-body")
    mp = tmp_path / "m.txt"; mp.write_text("mm")
    code, out = _run([
        "proofs", "submit",
        "--agent-id", "agent-1",
        "--input-file", str(ip),
        "--output-file", str(op),
        "--model-file", str(mp),
    ])
    assert code == 0
    kwargs = fake_client.calls[-1][2]
    assert kwargs["input_data"] == "in-body"
    assert kwargs["output_data"] == "out-body"
    assert kwargs["model"] == "mm"


def test_base_url_flag_and_env(monkeypatch, fake_client):
    monkeypatch.setenv("CP_BASE_URL", "http://envhost:9")
    # explicit --base-url wins
    code, _ = _run(["--base-url", "http://cli:1", "health"])
    assert code == 0
    assert fake_client.base_url == "http://cli:1"
