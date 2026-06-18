# go-memory-load-mongo

A MongoDB-backed order-management HTTP API used as a **memory-load sample** for
Keploy. It exercises Keploy's record/replay under a high volume of mocks
(connection pooling, aggregations, large payloads) so the agent's mock buffer,
batched disk flush, and memory-pressure handling are stressed end to end.

## What it does

A small e-commerce backend over MongoDB (driver v2):

- **customers / products / orders** — create + read
- **analytics** — customer summary and top-products aggregations
- **large payloads** — insert/get/delete up to 8 MiB documents

IDs and timestamps are **content-derived** (SHA-256 of the inputs), so the same
request always produces the same document — which keeps Keploy mocks
deterministic across record and replay.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET    | `/healthz` | liveness |
| POST   | `/customers` | create a customer |
| POST   | `/products` | create a product |
| POST   | `/orders` | create an order (decrements inventory) |
| GET    | `/orders/{id}` | fetch an order |
| GET    | `/orders` | search orders (`status`, `customer_id`, paging) |
| GET    | `/customers/{id}/summary` | per-customer aggregation |
| GET    | `/analytics/top-products` | top-products aggregation |
| POST   | `/large-payloads` | store a large document |
| GET    | `/large-payloads/{id}` | fetch a large document |
| DELETE | `/large-payloads/{id}` | delete a large document |

## Configuration

| Env | Default | Description |
|-----|---------|-------------|
| `APP_PORT` | `8080` | HTTP listen port |
| `MONGO_URI` | `mongodb://db:27017/orderdb` | MongoDB connection string |

## Run it

```bash
docker compose up
# API on http://localhost:8080, MongoDB on :27017
```

## Load test (k6)

The k6 scenario ramps virtual users across the mixed API and a large-payload
ramp. Run it via the compose `loadtest` profile (it targets the `api` service):

```bash
docker compose --profile loadtest up k6
```

Useful knobs (env on the `k6` service): `TEST_PROFILE=smoke` for a quick run,
`MIXED_API_VU_STAGE_TARGETS`, `LARGE_PAYLOAD_SIZES_MB`.

## Record & replay with Keploy

```bash
# record: capture HTTP test cases + the MongoDB mocks behind them
keploy record -c "docker compose up"

# (drive traffic — curl the endpoints or run the k6 load test)

# replay: re-run the recorded test cases against served mocks
keploy test -c "docker compose up"
```

## Note on connection pooling (Kubernetes)

The Mongo client is configured **without** `SetMinPoolSize`, so connections open
**lazily** on the first query. This matters for the Keploy **k8s sidecar**: its
eBPF interception only tracks connections opened *after* the agent attaches. An
eager pool (`SetMinPoolSize(N)`) would open connections at process startup —
before the agent is ready — and that traffic would never be captured. Lazy
pooling lets every connection be recorded. (In Docker this is a non-issue, since
Keploy starts the agent before the app.)
