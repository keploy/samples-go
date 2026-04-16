# PS-Cache Postgres — Connection Affinity Mock Mismatch Reproduction

This sample demonstrates a bug in Keploy's Postgres mock matcher where **connection pool eviction causes the replay to return the wrong person's data**.

## The Bug

When a connection pool evicts and creates a new database connection mid-session, the same parameterized query (`SELECT ... WHERE member_id = $1`) produces mocks on **two different recording connections** with identical query structure but different bind parameter values. During replay, the matcher picks mocks from the wrong recording connection — returning Alice's data (member=19) when Charlie's data (member=31) was requested.

## Architecture

```
                    ┌─────────────────┐
  HTTP requests ──> │   Go App (pgx)  │
                    │  pool_size = 1  │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
         /account       /evict         /account
        member=19    (pool reset)      member=31
              │              │              │
         Connection A   destroyed     Connection B
         (cold cache)                 (cold cache)
              │                            │
         Parse+Bind+                  Parse+Bind+
         Describe+Exec               Describe+Exec
              │                            │
         mock connID=0               mock connID=2
         (Alice, 1000)              (Charlie, 500)
```

During replay, both connections start fresh. The matcher sees identical query text on both and — without the affinity fix — grabs Connection A's mocks for Connection B's request.

## How to Reproduce the Bug

### Prerequisites
```bash
docker run -d --name pg-demo -e POSTGRES_PASSWORD=testpass -e POSTGRES_DB=demodb -p 5433:5432 postgres:16
```

### With the OLD keploy binary (demonstrates failure)
```bash
# Build from keploy/keploy + keploy/integrations origin/main (before the fix)
go build -o server .

# Record
sudo keploy record -c ./server
# In another terminal:
bash test.sh
# Stop recording (Ctrl+C)

# Replay
sudo keploy test -c ./server --skip-coverage
```

**Expected failure:**
```
test-5 (/account?member=31):
  EXPECTED: {"member_id":31, "name":"Charlie", "balance":500}
  ACTUAL:   {"member_id":19, "name":"Alice",   "balance":1000}  ← WRONG PERSON
```

### With the FIXED keploy binary (demonstrates fix)
```bash
# Build from keploy/keploy + keploy/integrations with fix (PR #121)
# Same record + replay steps → all 6 tests pass
```

## What the Fix Does

The fix adds **recording-connection affinity** to the Postgres mock matcher:

1. When the first `Bind` mock is consumed on a replay connection, its recording `connID` is stored
2. Subsequent scoring applies +50 bonus for same-connID mocks and -50 penalty for different-connID
3. Only activates when the mock set has 2+ distinct recording connections (zero impact on single-connection apps)

See: [keploy/integrations#121](https://github.com/keploy/integrations/pull/121)

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable` | PostgreSQL connection string |
| `POOL_MAX_CONNS` | `1` | Max connections in pgx pool |
| `INIT_DB` | _(unset)_ | Set to `true` to auto-create schema and seed data on startup |
| `PORT` | `8080` | HTTP server port |

## Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check |
| `GET /account?member=N` | Query travel_account by member_id (inside BEGIN/COMMIT transaction) |
| `GET /evict` | Close and recreate the connection pool (simulates HikariCP eviction) |
