# CasperProver — Local Observability

## Status: REAL / LOCAL-STACK

**HONESTY BADGE.** The metrics and traces this repo emits are real and
serve a real Prometheus + Grafana stack you can bring up locally. The
in-process client library is a purpose-built, zero-dependency implementation
(exposition-format compatible with Prometheus 0.0.4, semantically aligned with
OpenTelemetry spans). It is not `prometheus/client_golang` and it does not
ship OTLP/gRPC transport. This is a deliberate trade-off — see
"Upgrade path" below.

No SaaS is used. No paid services are used. Only free/open-source components
(Prometheus, Grafana OSS).

## What lands on `/metrics`

The API server exposes `GET /metrics` (registered before authMiddleware
strips headers; see server.go). The exposition format is
`text/plain; version=0.0.4; charset=utf-8`.

Standard HTTP RED metrics:

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `cp_http_requests_total` | counter | `route`, `method`, `status` | Total requests handled. `route` is the mux pattern (bounded cardinality), never the raw URL. |
| `cp_http_errors_total` | counter | `route`, `method`, `status` | 5xx responses. |
| `cp_http_inflight` | gauge | `route` | Requests currently in-flight. |
| `cp_http_request_duration_seconds` | histogram | `route`, `method` | Latency in seconds. Buckets: 1 ms → 30 s exponential. |

Additional metrics can be registered by any subsystem via
`internal/obs.Registry`. See `internal/obs/metrics.go`.

## Traces

Traces are **opt-in** — set `CP_TRACES_ENABLED=1` in the engine environment
to enable them. When enabled:

- Every HTTP request emits an outer `http.request` span with OTel-flavoured
  attributes: `http.method`, `http.route`, `http.target`, `http.status_code`.
- Spans are written as NDJSON to stderr — one JSON record per line.
- Trace / span IDs follow W3C Trace Context (16-byte trace id, 8-byte span id,
  hex-encoded). Nested spans inherit their parent's trace id and record
  `parent_span_id`.
- Handlers may extend the ambient span via
  `span := obs.FromContext(r.Context()); span.SetAttr(...)`.

An external NDJSON→OTLP sidecar is trivial to write if you need to ship spans
to Tempo/Jaeger; it is not shipped in this repo yet (see upgrade path).

## Bringing up the local stack

```bash
# From the repo root:
cd deploy/observability
docker compose up -d

# Then start the engine locally so /metrics is reachable from the host:
#   API_KEY=devkey PORT=8081 CP_TRACES_ENABLED=1 ./cp-engine

# Prometheus  → http://localhost:9090
# Grafana     → http://localhost:3001   (anonymous viewer + admin/admin)
```

Grafana comes up with:

- Datasource `Prometheus` (auto-provisioned) pointing at `prometheus:9090`.
- Dashboard **CasperProver engine** with request rate, p50/p95 latency, 5xx
  rate, and in-flight requests.

## Tear-down / reset

```bash
cd deploy/observability
docker compose down -v
rm -rf _data
```

## Design notes

- **Zero external Go dependencies.** The `internal/obs` package uses only the
  standard library. No `prometheus/client_golang`, no OTel SDKs, no OTLP
  transport. This keeps the engine free of large trees during the hackathon
  window and avoids version drift.
- **Cardinality-safe labels.** The middleware labels every request by the
  matched mux pattern (`GET /proofs/{id}`), never the raw URL. A rogue caller
  cannot inflate label cardinality.
- **Concurrency-safe.** Counters and gauges use atomics; histograms use
  CAS on the float64 sum bits. Deterministic exposition order is enforced by
  sorted names and label keys.

## Upgrade path

When production hardening is warranted:

1. Swap `internal/obs/metrics.go` for `github.com/prometheus/client_golang`
   (same conceptual API: counters, gauges, histograms). The
   `Middleware`/`MiddlewareRoute` shapes stay identical.
2. Swap `internal/obs/tracing.go` for `go.opentelemetry.io/otel` with an OTLP
   exporter (Tempo, Honeycomb, DataDog, etc). The `Tracer`/`Span` shape maps
   1:1 onto the OTel API.
3. Add scrape auth (bearer token or mTLS) at the reverse-proxy layer — the
   `/metrics` endpoint itself is intentionally unauthenticated so Prometheus
   can scrape without secrets baked into the config.

Neither step requires changing any call site.

## What is NOT included

- No OTLP/gRPC transport. Traces are NDJSON on stderr; ship them with any
  sidecar you prefer.
- No log aggregation. Structured logs already go via `slog`; wire them into
  Loki/Vector separately if needed.
- No paid vendors. Repository is hackathon-clean.
