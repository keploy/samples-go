# aerospike-tls — Aerospike-Go sample with Keploy record/replay

A small Go HTTP service that talks to Aerospike CE over the clear-text
service port (3000) using `aerospike-client-go/v7`. The sample is
recorded and replayed end-to-end with Keploy: bundled scripts spin
up the dependency, drive the API with `curl`, record the resulting
Aerospike traffic, and replay it deterministically against captured
mocks.

What the sample demonstrates:

* **Keploy records binary Aerospike protocol traffic** — Info,
  AS_MSG (single-record PUT/GET/TOUCH/DELETE), BATCH_READ/WRITE,
  SCAN, QUERY, UDF, CDT — and replays them from `mocks.yaml`
  without needing the real cluster.
* **Replay stays deterministic at any concurrency the app exposes** —
  single-client `/parallel`, multi-client round-robin, and per-request
  fresh-client construction all pass cleanly.
* **A pipeline-friendly shape**. Three `scripts/script-{1,2,3}.sh`
  entry points each record and replay one test-set independently,
  so CI can call them as separate jobs (or as one matrix).

## Layout

```
aerospike-tls/
├── main.go              # the HTTP service
├── go.mod / go.sum
├── aerospike-conf/
│   └── aerospike.conf   # CE config: clear-text on 3000
├── docker-compose.yml   # Aerospike CE + the sample
├── Dockerfile           # builds the sample binary for compose
├── keploy.yml           # Keploy CLI config (command, ports)
└── scripts/
    ├── common.sh        # shared helpers (boot, build, record, replay)
    ├── script-1.sh      # records + replays test-set-0 (CRUD)
    ├── script-2.sh      # records + replays test-set-1 (/parallel)
    └── script-3.sh      # records + replays test-set-2 (/multiclient + /freshclient)
```

There is no committed `keploy/` directory — the scripts produce it
from scratch every run. That keeps the repo lean and means every CI
run validates the full record-then-replay loop.

## Endpoints

| Method | Path                       | What it does                                                                 |
| ------ | -------------------------- | ---------------------------------------------------------------------------- |
| GET    | `/health`                  | `info "build" + "namespaces"`                                                |
| POST   | `/put`                     | single-record PUT                                                            |
| GET    | `/get/{key}`               | single-record GET                                                            |
| POST   | `/batch/put`               | sequential write loop                                                        |
| GET    | `/batch/get?k=a&k=b`       | BATCH_READ                                                                   |
| POST   | `/scan`                    | full namespace scan                                                          |
| POST   | `/query`                   | secondary-index range query                                                  |
| POST   | `/udf`                     | UDF_EXECUTE                                                                  |
| POST   | `/cdt/list/append`         | CDT list append                                                              |
| POST   | `/cdt/map/put`             | CDT map put                                                                  |
| POST   | `/touch/{key}`             | TOUCH                                                                        |
| DELETE | `/key/{key}`               | DELETE                                                                       |
| POST   | `/parallel?n=N&prefix=P`   | fans out N goroutines, each PUT+GET on a unique key — **one shared client**  |
| POST   | `/multiclient?n=N&prefix=P`| same, but round-robins across **4 pre-built `*as.Client` instances**         |
| POST   | `/freshclient?n=N&prefix=P`| **each goroutine builds its own `*as.Client`** inside the request            |

## Run it manually

```bash
# 1) Boot Aerospike CE on clear-text 3000.
docker compose up -d aerospike

# 2) Build + run the sample.
go build -o aerospike-tls .
./aerospike-tls

# 3) Hit it.
curl -s localhost:8080/health
curl -s -XPOST localhost:8080/put -d '{"key":"alice","bins":{"age":30}}'
curl -s localhost:8080/get/alice
curl -s -XPOST 'localhost:8080/parallel?n=24&prefix=run1'
curl -s -XPOST 'localhost:8080/multiclient?n=24&prefix=mc1'
curl -s -XPOST 'localhost:8080/freshclient?n=8&prefix=fc1'
```

## Record + replay with the scripts

```bash
# Each script is self-contained: brings up Aerospike, builds, records,
# replays. Exit code is non-zero if any case fails on replay.
sudo ./scripts/script-1.sh    # test-set-0: single-endpoint CRUD
sudo ./scripts/script-2.sh    # test-set-1: /parallel n = 4..24
sudo ./scripts/script-3.sh    # test-set-2: /multiclient + /freshclient
```

Pipeline-friendly knobs (env vars):

| Var          | Default       | What it does                                                  |
|--------------|---------------|---------------------------------------------------------------|
| `KEPLOY`     | `sudo keploy` | binary + auth invocation. Override to `keploy` if root        |
| `PORT`       | `8090`        | HTTP port the recorded sample listens on                      |
| `LOG_DIR`    | `/tmp`        | where to drop the keploy record log                           |
| `SKIP_DOCKER`| (unset)       | `=1` skips `docker compose up -d aerospike` (already running) |
| `SKIP_BUILD` | (unset)       | `=1` skips `go build` (binary already in place)               |

A typical CI job looks like:

```yaml
- run: docker compose up -d aerospike
- run: go build -o aerospike-tls .
- run: SKIP_DOCKER=1 SKIP_BUILD=1 ./scripts/script-1.sh
- run: SKIP_DOCKER=1 SKIP_BUILD=1 ./scripts/script-2.sh
- run: SKIP_DOCKER=1 SKIP_BUILD=1 ./scripts/script-3.sh
```

## Concurrency notes — what makes replay deterministic

Mocked replay through Keploy is roughly 20× faster than real
Aerospike for the same op. A burst of N concurrent goroutines on a
cold client pool then races to open N fresh sockets, and the
goroutine that loses the race surfaces as `MAX_RETRIES_EXCEEDED` at
the application — even though every peer in the same burst succeeds.

`main.go` paints over this with four layered changes; together they
make `/parallel?n=24`, `/multiclient?n=24`, and `/freshclient?n=8`
replay clean on every run:

1. **Sized pool** — `ClientPolicy.ConnectionQueueSize = 256`,
   `OpeningConnectionThreshold = 16`.
2. **Tolerant per-op policy** — `parallelWritePolicy` and
   `parallelReadPolicy` set `SocketTimeout 10s`, `TotalTimeout 30s`,
   `MaxRetries 10`, `SleepBetweenRetries 5ms`.
3. **Two-phase warmup** on the main client at startup: a sequential
   prelude that walks the cluster past cold-start latencies,
   followed by a parallel fill that puts idle connections in the
   pool before the HTTP server accepts the first request.
4. **App-level retry wrapper** (`parallelDo`) around each PUT and
   GET in `/parallel`, `/multiclient`, and `/freshclient`.

`/multiclient`'s extra clients are deliberately NOT warmed at
startup — a hundred concurrent dials at boot can stall a record-time
proxy. The retry wrapper covers their first burst instead.
