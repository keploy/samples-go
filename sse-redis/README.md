# sse-redis-app

A minimal Go application that exposes SSE, NDJSON, multipart, and plain-text
streaming endpoints backed by Redis.  Designed as a test fixture for **keploy's
streaming test-case support** — it produces responses with a mix of stable
fields (served from Redis mocks during replay) and flaky runtime fields
(timestamps, UUIDs) that exercise noise configuration.

## Prerequisites

| Tool | Purpose |
|------|---------|
| Go 1.21+ | Build the app |
| Redis 6+ | Message storage |
| Docker / Docker Compose | Easiest way to run Redis |
| keploy CLI | Record and replay test cases |

## Quick start

```bash
cd sse-redis-app

# 1. Start Redis
docker compose up -d

# 2. Build the app
go build -o sse-redis-app .
```

## Endpoints

| Method | Path | Content-Type | Description |
|--------|------|--------------|-------------|
| `POST` | `/messages` | `application/json` | Create a message (stored in Redis). Response includes generated `id`, `created_at`, `server_time`. |
| `DELETE` | `/messages` | `application/json` | Clear all stored messages. |
| `GET` | `/events/sse` | `text/event-stream` | Stream all messages as SSE. Emits a heartbeat comment, then JSON-object events, JSON-array events, a count (primitive number), and a status (plain string). |
| `GET` | `/events/ndjson` | `application/x-ndjson` | Stream all messages as newline-delimited JSON. |
| `GET` | `/events/multipart` | `multipart/x-mixed-replace` | Stream all messages as multipart parts with JSON bodies. |
| `GET` | `/events/plain` | `text/plain` | Stream all messages as plain-text log lines. |
| `GET` | `/health` | `application/json` | Health check with `server_time`, `uptime_seconds`, `instance_id`. |

### SSE data types exercised

The `/events/sse` endpoint emits multiple SSE event types to fully exercise the
matching logic:

| SSE field | Data type | Example |
|-----------|-----------|---------|
| `:` (comment) | Heartbeat with timestamp | `:heartbeat 2026-02-23T10:00:00Z` |
| `data` | JSON object | `{"id":"...","text":"hello","delivered_at":"..."}` |
| `data` | JSON array | `[{"tag":"greeting","ts":"..."},{"tag":"hello","ts":"..."}]` |
| `data` | Primitive number | `3` |
| `data` | Plain string | `stream-complete` |
| `event` | String (varied) | `message`, `tags`, `count`, `status` |
| `id` | Sequential integer | `1`, `2`, `3`, ... |

### Streaming payload shape (JSON events)

```json
{
  "id":           "a1b2c3d4-...",
  "text":         "hello world",
  "category":     "greeting",
  "created_at":   "2026-02-23T10:00:00.000Z",
  "delivered_at": "2026-02-23T10:00:01.500Z",
  "request_id":   "e5f6a7b8-..."
}
```

**Stable fields** (same during record and replay — they come from mocked Redis
data): `text`, `category`.

**Flaky fields** (generated at request time — change every run):
`id`, `created_at`, `delivered_at`, `request_id`, `server_time`,
`uptime_seconds`, `instance_id`, `ts`.

## Recording test cases

```bash
# Terminal 1 — start the app under keploy record
keploy record -c "./sse-redis-app"

# Terminal 2 — fire all requests
./request.sh

# Back in terminal 1 — press Ctrl+C to stop recording
```

This generates test cases in `keploy/test-set-0/` covering:

- Regular JSON `POST` / `DELETE` / `GET` responses
- SSE streaming (`text/event-stream`) with multiple data types
- NDJSON streaming (`application/x-ndjson`)
- Multipart streaming (`multipart/x-mixed-replace`)
- Plain-text streaming (`text/plain`)
- Redis outgoing call mocks in `keploy/test-set-0/mocks.yaml`

### One-liner

```bash
keploy record -c "./sse-redis-app" & sleep 5 && ./request.sh && sleep 2 && kill %1
```

## Running tests (replay with mocks)

```bash
keploy test -c "./sse-redis-app" --delay 7
```

During replay keploy:

1. Starts the app.
2. Mocks all outgoing Redis calls (no live Redis needed).
3. Replays each recorded HTTP request.
4. For streaming responses, validates the body incrementally:
   - **SSE**: compares field keys, event names, and JSON `data` payloads per
     frame. Comments are presence-checked only. Non-JSON `data` is type- and
     size-matched.
   - **NDJSON**: compares each JSON line after stripping noise keys.
   - **Multipart**: compares each part's Content-Type and JSON body (or size
     for non-JSON parts).
   - **Plain text**: compares each line by size.
5. Reports pass/fail per test case.

## Flaky fields and noise configuration

The included `keploy.yml` configures global noise for all runtime-generated
fields:

```yaml
test:
  globalNoise:
    global:
      body:
        delivered_at: []
        request_id: []
        server_time: []
        uptime_seconds: []
        instance_id: []
        ts: []
        id: []
        created_at: []
      header:
        Date: []
```

### How noise keys work

**Regular JSON responses** — keploy's matcher strips top-level keys listed
under `body` from both expected and actual JSON before comparing with
`reflect.DeepEqual`.

**Streaming JSON responses** (SSE `data`, NDJSON lines, multipart JSON parts) —
the same noise keys are passed to the stream comparator via
`collectStreamingGlobalNoiseKeys`. Each JSON chunk is stripped of these keys
before comparison.

**Non-JSON streaming** (plain text, binary) — noise keys don't apply. Matching
is by line count and line size (plain text) or total byte size (binary).

### Key naming rules

| Pattern | Scope |
|---------|-------|
| `fieldname` (no dots) | Global — stripped at any depth in the JSON tree |
| `body.nested.field` (with dots) | Path-specific — matched against a JSON path |

### Adding more noise keys

If your application introduces additional flaky fields, add them to the `body`
(or `header`) section of `keploy.yml`. An empty array `[]` means "ignore this
field regardless of its value."

## Cleaning up

```bash
rm -rf keploy/          # remove recorded test cases
docker compose down      # stop Redis
rm -f sse-redis-app      # remove binary
```

## Project structure

```
sse-redis-app/
├── main.go              # Application source
├── go.mod / go.sum      # Go module
├── docker-compose.yml   # Redis service
├── keploy.yml           # Keploy config with noise settings
├── request.sh           # Script to generate all test traffic
└── README.md            # This file
```
