# Go API with Postgres and Load Testing

This project spins up a Go API backed by PostgreSQL and includes a `k6` load test that mixes inserts, detail reads, filtered searches, and analytics queries.

## What it does

- `POST /customers` inserts customers.
- `POST /products` inserts products with inventory.
- `POST /orders` creates orders transactionally, inserts line items, and decrements inventory.
- `GET /orders/{id}` returns an order with joined customer and item details.
- `GET /orders` performs filtered order searches with aggregate item rollups.
- `GET /customers/{id}/summary` uses CTEs and a window function to compute lifetime spend and favorite category.
- `GET /analytics/top-products` ranks products by revenue over a recent time window.
- `POST /large-payloads` stores multi-megabyte request bodies in Postgres.
- `GET /large-payloads/{id}` retrieves the full large payload back from Postgres.
- `DELETE /large-payloads/{id}` deletes a large payload row and returns its metadata.

## Run the stack

1. Start Postgres and the API:

   ```bash
   docker compose up --build -d db api
   ```

2. Confirm the API is healthy:

   ```bash
   curl http://localhost:8080/healthz
   ```

3. Run the load test:

   ```bash
   docker compose --profile loadtest run --rm --no-deps k6 run /scripts/scenario.js
   ```

4. Or run the short validation profile first:

   ```bash
   docker compose --profile loadtest run --rm --no-deps -e TEST_PROFILE=smoke k6 run /scripts/scenario.js
   ```

The load test now includes a dedicated large-payload scenario that inserts, reads, and deletes 1 MB, 2 MB, and 4 MB payloads by default. Override that with `LARGE_PAYLOAD_SIZES_MB`, for example:

```bash
docker compose --profile loadtest run --rm \
  --no-deps \
  -e LARGE_PAYLOAD_SIZES_MB=2,4,6 \
  k6 run /scripts/scenario.js
```

## Local development

1. Copy the DSN from `.env.example` or export it directly:

   ```bash
   export POSTGRES_DSN='postgres://app_user:app_password@localhost:5432/orderdb?sslmode=disable'
   export APP_PORT=8080
   ```

2. Install Go dependencies and run the API:

   ```bash
   go mod tidy
   go run ./cmd/api
   ```

## Example requests

Create a customer:

```bash
curl -X POST http://localhost:8080/customers \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "alice@example.com",
    "full_name": "Alice Example",
    "segment": "startup"
  }'
```

Create a product:

```bash
curl -X POST http://localhost:8080/products \
  -H 'Content-Type: application/json' \
  -d '{
    "sku": "SKU-1001",
    "name": "GPU Cluster",
    "category": "compute",
    "price_cents": 250000,
    "inventory_count": 50
  }'
```

Create an order:

```bash
curl -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{
    "customer_id": 1,
    "status": "paid",
    "items": [
      { "product_id": 1, "quantity": 2 },
      { "product_id": 2, "quantity": 1 }
    ]
  }'
```

Search orders:

```bash
curl 'http://localhost:8080/orders?status=paid&min_total_cents=1000&limit=20'
```

Create a large payload:

```bash
curl -X POST http://localhost:8080/large-payloads \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Large Report",
    "content_type": "text/plain",
    "payload": "replace-this-with-a-large-string"
  }'
```

## For load testing run the script :
```bash
docker compose --profile loadtest run --rm --no-deps k6 run /scripts/scenario.js
```