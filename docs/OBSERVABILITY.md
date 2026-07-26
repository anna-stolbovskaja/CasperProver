# Observability

The engine ships a first-party observability layer: a Prometheus `/metrics`
endpoint and W3C Trace Context propagation. Both are always on — they
cost effectively nothing when no scraper or tracing peer is connected.

Package: [`engine/internal/observability`](../engine/internal/observability/).

## What is honestly shipped

- **Prometheus text exposition (v0.0.4).** Fully self-hosted — we do not
  depend on `prometheus/client_golang`. Counter, gauge, and histogram
  metric kinds are supported. Any Prometheus-compatible scraper (a real
  `prometheus-server`, Grafana Agent, VictoriaMetrics, etc.) can scrape
  `GET /metrics` and store the results.
- **W3C Trace Context, level 1.** We parse and generate `traceparent`
  headers per the [W3C REC](https://www.w3.org/TR/trace-context-1/).
  Version `00` only. `tracestate` is passed through verbatim (opaque).
  Every request that enters the API gets a `SpanContext` attached to
  its `context.Context` — inherited from the upstream `traceparent`
  header when present, or created as a root span otherwise. The
  outgoing response echoes a new `traceparent` with a fresh span-id.

## What is NOT shipped

- **No native OTLP export.** We do not push spans to an OpenTelemetry
  collector. Trace propagation is inbound-and-outbound *headers*, so a
  downstream OTel collector that scrapes the engine can still stitch
  our spans into a distributed trace — but the engine itself does not
  export spans over gRPC/OTLP. That is out of scope for this pack.
- **No exemplars, summaries, or native histograms.** These are
  Prometheus 2.x+ extensions; when we need them we will pull in
  `prometheus/client_golang`. For now the classic bucket histogram
  covers everything we sample.
- **No auto-instrumentation of upstream deps.** Only the HTTP surface
  is measured. Postgres, submitter, and provers can be added later
  with the same Registry.

## Metrics currently emitted

The HTTP layer registers three metrics under the `cp_http` prefix:

| Metric | Kind | Labels | Meaning |
|---|---|---|---|
| `cp_http_requests_total` | counter | `method`, `route`, `status` | Total requests processed. `status` is bucketed as `2xx`/`4xx`/`5xx` etc. so label cardinality stays bounded. |
| `cp_http_request_duration_seconds` | histogram | `method`, `route` | Request latency in seconds. Buckets: `0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10`. |
| `cp_http_requests_in_flight` | gauge | (none) | Requests currently being served. |

The root-chain wrapper labels every request as `route="all"` — this
keeps cardinality at O(number-of-methods) rather than
O(number-of-distinct-URLs) and avoids the classic /:id explosion.
When you need per-route detail, add a dedicated app-level counter
with a bounded label (e.g. `proofs_generated_total{mode="anchored"}`).

## Enabling scraping

```
$ curl http://localhost:8080/metrics
# HELP cp_http_requests_total total HTTP requests processed
# TYPE cp_http_requests_total counter
cp_http_requests_total{method="GET",route="all",status="2xx"} 42
...
```

The alias `GET /v1/metrics` returns the same content — versioned URL
for scrape configs that pin to a specific API version.

Sample Prometheus scrape config:

```yaml
scrape_configs:
  - job_name: casperprover
    static_configs:
      - targets: ["cp-engine:8080"]
    metrics_path: /metrics
    scrape_interval: 15s
```

## Trace propagation contract

When a client sends a `traceparent` header, the engine:

1. Parses it against the W3C level 1 grammar. Malformed headers do
   NOT reject the request — the engine falls back to a fresh root
   span (the spec's forgiveness posture).
2. Reuses the caller's `trace-id`, generates a fresh `span-id` for
   this request, and records the caller's span-id as `parent-id`.
3. Attaches a `SpanContext` to `context.Context` for downstream code
   (`observability.SpanContextFromContext(ctx)`).
4. Echoes a `traceparent` header on the response so ANY downstream
   observer (client-side APM, sidecar, mesh) sees a linked span.
5. Passes through `tracestate` verbatim.

Example:

```
$ curl -H 'traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01' \
       http://localhost:8080/health -i
HTTP/1.1 200 OK
traceparent: 00-0af7651916cd43dd8448eb211c80319c-<new-span-id>-01
...
```

## Where the code lives

- `engine/internal/observability/metrics.go` — Registry, Counter,
  Gauge, Histogram, text exposition writer.
- `engine/internal/observability/trace.go` — `traceparent` parse +
  generate, context helpers.
- `engine/internal/observability/middleware.go` — `HTTPMetrics` +
  the `Instrument` / `InstrumentAll` wrappers, `MetricsHandler` for
  `/metrics`.

All three are covered by unit tests
(`engine/internal/observability/*_test.go`) plus an integration test
that runs the full HTTP chain and validates the exposition
(`engine/internal/api/metrics_endpoint_test.go`).

## Roadmap items closed

Backlog item **10.3 (remaining)** — Prometheus `/metrics`, OTel spans,
sampled tracing — is now closed for the HTTP surface. Follow-ups:

- **10.4** SLO error budgets + burn-rate alerts (needs the metrics
  above scraped for ≥1 week to derive baseline SLOs).
- **10.5** Blue/green deploy runbook + Grafana dashboards (once metrics
  are wired into a stack).
- Optional: swap our in-tree Prometheus writer for
  `prometheus/client_golang` if we later need exemplars, summaries,
  or native histograms. The `Registry` API surface is compatible
  enough to make the swap mechanical.
