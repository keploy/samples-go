# sse-preflight (Keploy repro)

This is a tiny Go app meant to reproduce the Keploy replay failure where an `OPTIONS` CORS preflight to an **SSE endpoint** (recorded on port `8047`) is replayed against the **normal HTTP port** (`8000`) and returns `404`.

## What the app does

- `:8000` (normal HTTP): only serves `GET /health` (everything else is `404`)
- `:8047` (SSE): serves:
  - `OPTIONS /subscribe/student/events` → `200` + CORS headers (no `text/event-stream` Content-Type)
  - `GET /subscribe/student/events` → `200` + `Content-Type: text/event-stream`

## Run the server (without Keploy)

```bash
cd sse-preflight
go run ./cmd/server
```

In another terminal:

```bash
go run ./cmd/client --url "http://localhost:8047/subscribe/student/events?doubtId=repro" --host "doubt-service.example.com"
```

You should see `status=200 OK`.

## Record + replay with Keploy (to reproduce the failure)

Prereq: make sure Keploy Enterprise auth is configured (for example `KEPLOY_API_KEY`), otherwise `keploy record/test` will prompt for an API key.

### 1) Record

Terminal A:

```bash
keploy record -c "go run ./cmd/server"
```

Terminal B (during the 20s window):

```bash
go run ./cmd/client --url "http://localhost:8047/subscribe/student/events?doubtId=repro" --host "doubt-service.example.com"
```

### 2) Replay

```bash
keploy test -c "go run ./cmd/server"
```

Expected: the preflight testcase fails with `EXPECT 200` vs `ACTUAL 404`, and in debug logs you should see the key part:

- `Overriding port with app_port` (8047)
- `Config port overrides recorded app_port` (8000 overrides 8047)
- `Final resolved target` uses `localhost:8000/...`
