# CasperProver Python SDK

> **Status:** `v0.1.0`. Full-feature client with `prove`, `verify`, `batch`,
> `anchor`, and a shared receipt validator. Feature parity with the Go and
> TypeScript SDKs.

## Install (once published)

```sh
pip install casperprover
```

Until the first release, consumers add the repo checkout to `PYTHONPATH`:

```python
import sys
sys.path.insert(0, "/path/to/CasperProver/sdk/python")
from casperprover import Client, ProveRequest
```

## Quickstart

```python
from casperprover import Client, ProveRequest, verify_receipt_bytes

c = Client(base_url="https://casperprover-api-ylsh.onrender.com",
           api_key="pk_...")

proof = c.prove(
    ProveRequest(agent="a", model="gpt-toy-v1",
                 input="hello", output="42"),
    idempotency_key="run-1",
)
print(proof.id, proof.vk_hash)

check = c.verify(proof.id)
assert check.valid
```

Every write primitive accepts `idempotency_key=` — safe retries against the
server-side dedup cache (24h TTL).

Legacy unversioned routes: `Client(..., api_version="")`.

## Receipt validator

`verify_receipt_bytes(payload)` and `verify_receipt(dict)` re-derive
`input_hash`, `output_hash`, and `model_hash` locally (SHA-256 of the UTF-8
plaintext) and raise `ReceiptValidationError` on any mismatch. Bit-identical
output to `sdk/receipt.go` and `sdk/typescript/src/receipt.ts`.

## Test

```sh
cd sdk/python
python3 -m unittest casperprover.tests.test_client -v
```
