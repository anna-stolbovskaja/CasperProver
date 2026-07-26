# CasperProver TypeScript SDK

> **Status:** `v0.1.0-scaffold`. The 30-day plan is to extract a proper
> `@casperprover/sdk` npm package from the current frontend's API client
> code; see `docs/roadmap/30-DAY.md`.

## Install (once published)

```sh
npm install @casperprover/sdk
# or
pnpm add @casperprover/sdk
```

## Quickstart

```ts
import { CasperProverClient } from "@casperprover/sdk";

const client = new CasperProverClient({
  baseUrl: "https://api.casperprover.example",
  apiKey: process.env.CP_API_KEY!,
});

const receipt = await client.submitDecision({
  agentId: "agent-1",
  input: new TextEncoder().encode("hello"),
  output: new TextEncoder().encode("world"),
  modelId: "modelhash-...",
});

console.log(receipt.id, receipt.verdict, receipt.confidence);
```

## Version support table

| SDK version | Engine API version | Notes                     |
|-------------|--------------------|---------------------------|
| `v0.1.x`    | `v0` (implicit)    | Pre-`/v1/` routes         |
| `v0.2.x`    | `v1`               | After `docs/roadmap/API_LIFECYCLE.md` migration |

## Publish plan (30-day)

1. Extract the frontend's API client (`frontend/src/lib/api/*`) into a
   framework-agnostic package under `sdk/typescript/`.
2. Build with `tsup` → ESM + CJS + `.d.ts`.
3. `vitest` smoke against an in-memory fetch mock.
4. First tag `v0.1.0` after a green smoke against a live testnet-facing
   engine.

## `verify.sh` helper

A small Node CLI wrapper (`sdk/typescript/bin/verify.mjs`) will front
the existing `verify.sh` scenario for CI use:

```sh
npx @casperprover/sdk verify --receipt <receipt-id>
```
