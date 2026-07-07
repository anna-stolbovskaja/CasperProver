# CasperProver SDK

`sdk/` is a standalone Go module (`github.com/anna-stolbovskaja/CasperProver/sdk`),
separate from `engine/`, so it can be published/imported independently. Every
`Client` method maps 1:1 to a real route in `engine/internal/api/server.go` -
see `docs/openapi.yaml` for the authoritative route list. Not every route has
a typed method yet; PRs welcome.

## Go client

```go
package main

import (
    "context"
    "fmt"

    "github.com/anna-stolbovskaja/CasperProver/sdk"
)

func main() {
    ctx := context.Background()
    c := sdk.NewClient(sdk.WithBaseURL("http://localhost:9090"))

    // generate proof
    proof, err := c.SubmitProof(ctx, sdk.SubmitProofRequest{
        Agent: "agent-1", Input: "input", Output: "output", Model: "model",
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(proof["id"], proof["proof_hash"])

    // verify
    result, _ := c.VerifyProof(ctx, proof["id"].(string))
    fmt.Println("valid:", result["valid"])

    // list all
    all, _ := c.ListProofs(ctx)
    fmt.Println(all)
}
```

Client responses are `map[string]any` rather than fixed structs, because
several endpoints (proofs, aggregation batches) return optional fields that
vary by state - see `sdk/types.go` for the rationale.

## python client

```python
from sdk.python_client import ProverClient

client = ProverClient("http://localhost:9090")
proof = client.submit("agent-1", b"input", b"output", b"model", "inference")
print(proof["id"], proof["proof_hash"])

ok = client.verify(proof["id"])
print("valid:", ok)
```

## mcp server

The MCP tool manifest and stdio JSON-RPC loop are defined in
`sdk/mcp_server.go`; the runnable entry point that wires a subset of those
tools to a real running API instance is `sdk/cmd/mcpserver`:

```bash
CASPERPROVER_API_URL=http://localhost:9090 go run ./sdk/cmd/mcpserver
```

### tools

| tool | description | backed by real API? |
|------|-------------|----------------------|
| `health_check` | API health check | yes |
| `generate_proof` | create proof of AI inference | yes |
| `verify_proof` | check proof validity | yes |
| `get_proof` | fetch proof details | yes |
| `list_proofs` | list all proofs | yes |
| `revoke_proof` | invalidate a proof | yes |
| `export_proof` | export proof + chain metadata | yes |
| `get_stats` | engine-wide proof stats | yes |
| `kyc_check` / `kyc_grant` / `kyc_whitelist` | KYC flow | yes |
| `get_model_info` / `register_model` | model registry | yes |
| `batch_proofs` | batch-verify proofs | not yet wired over MCP - use `POST /proofs/batch` directly |
| `get_merkle_root`, `list_models`, `get_model_registry`, `deprecate_model`, `estimate_complexity`, `get_complexity_report`, `submit_batch_task`, `get_task_status`, `list_worker_nodes` | aspirational manifest entries | **no backing API endpoint yet** - `cmd/mcpserver` returns a clear "not implemented" error rather than fabricating a response |

See `docs/KNOWN_LIMITATIONS.md` for details on the unimplemented tools.
