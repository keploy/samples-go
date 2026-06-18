# go-memory-load-grpc

A gRPC order-management service used as a **memory-load sample** for Keploy. It
covers the same domain as `go-memory-load-mongo`/`-mysql` but over **gRPC**, with
an **in-memory** store (no database). It stresses Keploy's record/replay of gRPC
traffic at high volume — the agent's mock buffer, batched disk flush, and
memory-pressure handling.

## What it does

`LoadTestService` over gRPC, backed by in-memory maps:

- **customers / products / orders** — create + read
- **analytics** — customer summary and top-products
- **large payloads** — insert/get up to multi-MiB messages

IDs and timestamps are **content-derived** (SHA-256 of the inputs), so the same
request always produces the same record — which keeps Keploy mocks deterministic
across record and replay. Because the store is in-memory, this sample isolates
the gRPC transport path (no DB mocks involved).

## Service (gRPC)

`LoadTestService` (`api/proto/loadtest.proto`):

| RPC | Description |
|-----|-------------|
| `CreateCustomer` / `GetCustomerSummary` | customers + per-customer aggregation |
| `CreateProduct` | products |
| `CreateOrder` / `GetOrder` / `SearchOrders` | orders |
| `TopProducts` | top-products aggregation |
| `CreateLargePayload` / `GetLargePayload` | large messages |

A plain HTTP **`GET /healthz`** is also served for liveness/readiness probes.

## Configuration

| Env | Default | Description |
|-----|---------|-------------|
| `APP_HTTP_PORT` | `8080` | HTTP port (health check) |
| `APP_GRPC_PORT` | `50051` | gRPC port (`LoadTestService`) |

## Run it

```bash
docker compose up
# gRPC on :50051, health on http://localhost:8080/healthz
```

## Load test (k6)

The k6 scenario drives the gRPC service. Run it via the compose `loadtest`
profile (it targets the `api` service):

```bash
docker compose --profile loadtest up k6
```

Useful knobs (env on the `k6` service): `TEST_PROFILE=smoke` for a quick run.

## Record & replay with Keploy

```bash
# record: capture gRPC test cases + mocks
keploy record -c "docker compose up"

# (drive traffic — run the k6 load test or a gRPC client)

# replay: re-run the recorded test cases against served mocks
keploy test -c "docker compose up"
```
