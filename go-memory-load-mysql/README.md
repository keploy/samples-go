# go-memory-load-mysql

A MySQL-backed order-management HTTP API used as a **memory-load sample** for
Keploy. It mirrors `go-memory-load-mongo` (same domain and endpoints) but over
MySQL, so Keploy's record/replay is stressed against a high volume of SQL mocks
— exercising the agent's mock buffer, batched disk flush, and memory-pressure
handling end to end.

## What it does

A small e-commerce backend over MySQL (`database/sql`):

- **customers / products / orders** — create + read
- **analytics** — customer summary and top-products queries
- **large payloads** — insert/get/delete up to 8 MiB rows

IDs and timestamps are **content-derived** (SHA-256 of the inputs), so the same
request always produces the same row — which keeps Keploy mocks deterministic
across record and replay.

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
| GET    | `/analytics/top-products` | top-products query |
| POST   | `/large-payloads` | store a large row |
| GET    | `/large-payloads/{id}` | fetch a large row |
| DELETE | `/large-payloads/{id}` | delete a large row |

## Configuration

| Env | Default | Description |
|-----|---------|-------------|
| `APP_PORT` | `8080` | HTTP listen port |
| `MYSQL_DSN` | `app_user:app_password@tcp(db:3306)/orderdb?parseTime=true` | MySQL DSN |

## Run it

```bash
docker compose up
# API on http://localhost:8080, MySQL on :3306
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
# record: capture HTTP test cases + the MySQL mocks behind them
keploy record -c "docker compose up"

# (drive traffic — curl the endpoints or run the k6 load test)

# replay: re-run the recorded test cases against served mocks
keploy test -c "docker compose up"
```

## Note on connection pooling

The pool uses Go's `database/sql`, which opens connections **lazily**
(`SetMaxIdleConns` only caps idle connections, it does not pre-open them). So
there is no startup burst of un-intercepted connections — this app records
correctly in both Docker and the Keploy **k8s sidecar**.
