# postgres-wire-features

A small Go HTTP service that exercises a broad set of PostgreSQL wire-protocol features in one place — designed as a Keploy sample for recording and replaying Postgres interactions that go beyond plain `SELECT` / `INSERT`.

The service speaks the Postgres v3 frontend/backend protocol directly over a TCP socket (pure stdlib — no driver). That makes the recorded traffic a faithful representation of what Keploy sees on the wire, which matters for features like `COPY` whose payload is a stream of `CopyData` / `CopyDone` messages that driver libraries often abstract away.

## What it covers

A single call to `GET /run/all` walks through ten scenarios in order:

| Scenario | Statements |
| --- | --- |
| `setup` | `DROP TABLE`, `CREATE TABLE`, `INSERT`, `CREATE TABLE IF NOT EXISTS` |
| `dml` | `INSERT … ON CONFLICT DO UPDATE`, `UPDATE`, `DELETE`, `MERGE` |
| `catalog-cte` | `WITH … SELECT FROM pg_catalog.pg_class` |
| `catalog-sub` | `SELECT … FROM (SELECT …) AS s` |
| `catalog-setop` | `SELECT … UNION SELECT …` |
| `copy` | `TRUNCATE`, `COPY … FROM STDIN`, `COPY … TO STDOUT` |
| `prepare` | `PREPARE`, `EXECUTE`, `DEALLOCATE` |
| `cursor` | `BEGIN`, `DECLARE CURSOR`, `FETCH`×2, `CLOSE`, `COMMIT` |
| `admin` | `ANALYZE`, `REINDEX`, `LOCK TABLE`, `DO $$ … $$`, `CREATE PROCEDURE`, `CALL` |
| `validation-ping` | `SELECT 'ok'`, `SELECT true`, `SELECT NULL`, `SELECT 1` |

Each scenario is also reachable individually at `GET /case/<name>`. The response is JSON: per-statement `commands` (the Postgres command tag, e.g. `COPY 2`, `SELECT 1`), `rows`, and for `COPY TO STDOUT` a `copyData` array of the raw tab-separated rows.

## Endpoints

- `GET /health` → `200 ok`
- `GET /run/all` → run every scenario in order, return the combined result
- `GET /case/<name>` → run one scenario (see table above)

## Prerequisites

- Go `1.24+`
- Docker and Docker Compose
- Keploy CLI

Install Keploy:

```bash
curl --silent -O -L https://keploy.io/install.sh
source install.sh
```

## Setup

```bash
git clone https://github.com/keploy/samples-go.git
cd samples-go/postgres-wire-features
go mod download
```

## Run with Docker

```bash
docker compose up --build
```

The API will be available at `http://localhost:8080`.

## Record Testcases with Keploy

```bash
keploy record -c "docker compose up" --container-name api --delay 10
```

In another terminal:

```bash
./test.sh
```

Keploy writes the recorded HTTP tests and Postgres mocks under `keploy/`.

## Replay Recorded Tests

```bash
keploy test -c "docker compose up" --container-name api --delay 20
```

## Run Natively

Start Postgres yourself (e.g. `docker run -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16-alpine`) and then:

```bash
PGHOST=127.0.0.1 PGPORT=5432 PGUSER=postgres PGPASSWORD=postgres PGDATABASE=postgres go run .
```

## Environment variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `ADDR` | `:8080` | Listen address for the HTTP server |
| `PGHOST` | `127.0.0.1` | Postgres host |
| `PGPORT` | `5432` | Postgres port |
| `PGUSER` | `postgres` | Postgres user |
| `PGPASSWORD` | _(empty)_ | Postgres password |
| `PGDATABASE` | `postgres` | Postgres database |
