# CasperProver Python SDK

> **Status:** `v0.1.0-scaffold`. The client code lives in
> `sdk/python_client.py` today; this directory is the packaging story for
> the next 30 days (see `docs/roadmap/30-DAY.md`).

## Install (once published)

```sh
pip install casperprover
```

Until the first release, consumers add the repo checkout to `PYTHONPATH`
and import directly:

```python
import sys
sys.path.insert(0, "/path/to/CasperProver")
from sdk.python_client import ProverClient
```

## Quickstart

```python
from sdk.python_client import ProverClient

client = ProverClient(
    base_url="https://api.casperprover.example",
    api_key="sk_tenant_...",
)

proof = client.submit(
    agent_id="agent-1",
    input_bytes=b"hello",
    output_bytes=b"world",
    model_hash=b"modelhash-...",
    kind="inference",
)
print(proof["id"], proof["verdict"], proof["confidence"])

assert client.verify(proof["id"])
```

## Version support table

| SDK version | Engine API version | Notes                     |
|-------------|--------------------|---------------------------|
| `v0.1.x`    | `v0` (implicit)    | Pre-`/v1/` routes         |
| `v0.2.x`    | `v1`               | After `docs/roadmap/API_LIFECYCLE.md` migration |

## Publish plan (30-day)

1. Extract `sdk/python_client.py` into a real package layout:
   ```
   sdk/python/
     pyproject.toml
     src/casperprover/__init__.py
     src/casperprover/client.py
     tests/test_smoke.py
   ```
2. Add `pytest` + `mypy --strict` in CI.
3. First tag `v0.1.0` after a green smoke against a live testnet-facing
   engine.

## Smoke test (today)

```sh
python3 -m pytest sdk/python_client_test.py
```

_(The test file will be added alongside the extraction step.)_
