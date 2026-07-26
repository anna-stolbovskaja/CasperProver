/**
 * Quickstart example for the CasperProver TypeScript SDK.
 *
 * Usage: CP_API_URL=http://localhost:9090 CP_API_KEY=... \
 *        node --experimental-strip-types sdk/examples/typescript/quickstart.ts
 */

import { Client } from "../../typescript/src/client.ts";

async function main(): Promise<void> {
  const baseUrl = process.env.CP_API_URL ?? "http://localhost:9090";
  const apiKey = process.env.CP_API_KEY;

  const c = new Client({ baseUrl, apiKey });

  const h = await c.health();
  console.log("health:", h);

  const proof = await c.prove(
    {
      agent: "example-agent",
      model: "gpt-toy-v1",
      input: "hello world",
      output: "42",
      use_case: "quickstart",
    },
    { idempotencyKey: "quickstart-1" },
  );
  console.log(`proof: id=${proof.id} vk_hash=${proof.vk_hash ?? ""}`);

  const v = await c.verify(proof.id);
  console.log(`verify: valid=${v.valid}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
