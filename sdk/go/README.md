# CasperProver Go SDK

> **Status:** `v0.1.0-scaffold`. Real code lives in the top-level `sdk/` module
> today (`github.com/anna-stolbovskaja/CasperProver/sdk`). This directory
> holds the packaging + release story for the next 30 days; see
> `docs/roadmap/30-DAY.md`.

## Install (once published)

```sh
go get github.com/anna-stolbovskaja/casperprover-go@v0.1.0
```

Until the first published tag, consumers import the in-repo module:

```sh
go get github.com/anna-stolbovskaja/CasperProver/sdk
```

## Quickstart

```go
package main

import (
    "context"
    "fmt"
    "log"

    cp "github.com/anna-stolbovskaja/CasperProver/sdk"
)

func main() {
    client, err := cp.NewClient(cp.Options{
        BaseURL: "https://api.casperprover.example",
        APIKey:  "sk_tenant_...",
    })
    if err != nil {
        log.Fatal(err)
    }

    receipt, err := client.SubmitDecision(context.Background(), cp.DecisionInput{
        AgentID: "agent-1",
        Input:   []byte("hello"),
        Output:  []byte("world"),
        ModelID: "modelhash-…",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("receipt: %s verdict=%s confidence=%.2f\n",
        receipt.ID, receipt.Verdict, receipt.Confidence)
}
```

## Version support table

| SDK version | Engine API version | Notes                     |
|-------------|--------------------|---------------------------|
| `v0.1.x`    | `v0` (implicit)    | Pre-`/v1/` routes         |
| `v0.2.x`    | `v1`               | After `docs/roadmap/API_LIFECYCLE.md` migration |

`v1.0.0` gates listed in `docs/roadmap/30-DAY.md#semver-policy-sdks`.

## Repo layout

- `sdk/*.go` — client, types, MCP wrapper (kept where the engine tests it).
- `sdk/go/README.md` — this file, the publish story.
- `sdk/go/go.mod` — will be added when the module is extracted for
  publication under `github.com/anna-stolbovskaja/casperprover-go`.

## Smoke test

```sh
cd /path/to/CasperProver
go test ./sdk/...
```

The suite exercises the client against an in-process fixture server. A
live-engine smoke test lives under `scripts/smoke-sdk-go.mjs` and is run
manually before each release tag.
