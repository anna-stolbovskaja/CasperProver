# CasperProver SDK

## Go client

```go
package main

import (
    "fmt"
    "github.com/anna-stolbovskaja/CasperProver/sdk"
)

func main() {
    c := sdk.NewClient("http://localhost:9090")

    // generate proof
    proof, _ := c.Submit("agent-1", []byte("input"), []byte("output"), []byte("model"), "inference")
    fmt.Println(proof.ID, proof.PH)

    // verify
    ok, _ := c.Verify(proof.ID)
    fmt.Println("valid:", ok)

    // list all
    proofs, _ := c.List()
    fmt.Println("total:", len(proofs))
}
```

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

```bash
go run sdk/mcp_server.go           # stdio
casperprover serve --mcp           # via cli
```

### tools

| tool | description |
|------|-------------|
| `generate_proof` | create proof of AI inference |
| `verify_proof` | check proof validity |
| `get_proof` | fetch proof details |
| `list_proofs` | list all proofs |
| `revoke_proof` | invalidate a proof |
