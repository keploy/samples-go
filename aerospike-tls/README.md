# aerospike-tls — Aerospike-Go sample with Keploy record/replay over TLS

A small Go HTTP service that talks to **Aerospike on a TLS-only
port (3001)** via the official `aerospike-client-go/v7` driver. It is
recorded and replayed end-to-end through Keploy, which transparently
MITMs the TLS connection and serves binary Aerospike traffic from
captured mocks at replay time.

The point of the sample is to demonstrate three things:

* **Keploy records Aerospike traffic over TLS** the same way it does
  over clear text — the proxy detects the TLS handshake by byte
  pattern (not by port) and terminates it upstream of the parser.
  What lands in `keploy/*/mocks.yaml` is plaintext Aerospike wire
  protocol, not ciphertext.
* **The recordings replay deterministically** at any concurrency the
  app exposes — single-client parallel ops, multiple-client
  round-robin, and per-request fresh-client construction all pass
  3× in a row.
* **Realistic Aerospike usage shapes** — connection pool sizing,
  retry policy, and warmup that survive the burst-load characteristics
  of mocked replay (which is much faster than real Aerospike and
  exposes pool-acquire races a regular load test would never hit).

## Layout

```
aerospike-tls/
├── main.go              # the HTTP service
├── go.mod / go.sum
├── gen-certs.sh         # self-signed CA + server + (optional) client certs
├── aerospike-conf/
│   └── aerospike.conf   # CE config: TLS-only service on 3001
├── docker-compose.yml   # Aerospike CE + stunnel TLS terminator
├── Dockerfile           # builds the sample binary
├── keploy.yml           # Keploy CLI config (command, ports)
├── keploy/              # captured test-sets + mocks
│   ├── test-set-0/      # single-endpoint CRUD: put/get/batch/touch/delete
│   ├── test-set-1/      # /parallel: shared client, n = 4..24
│   └── test-set-2/      # /multiclient + /freshclient
└── stunnel/             # (referenced by docker-compose for TLS termination)
```

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
| POST   | `/parallel?n=N&prefix=P`   | fans out N goroutines, each PUT+GET a unique key — **one shared client**     |
| POST   | `/multiclient?n=N&prefix=P`| same, but round-robins across **4 pre-built `*as.Client` instances**         |
| POST   | `/freshclient?n=N&prefix=P`| **each goroutine builds its own `*as.Client`** inside the request            |

## Run it

```bash
cd aerospike-tls

# 1) Self-signed PKI under ./certs (CN = aerospike.local).
./gen-certs.sh

# 2) Start the TLS-only Aerospike + stunnel.
docker compose up -d aerospike stunnel

# 3) Build + run the app.
go build -o aerospike-tls .
./aerospike-tls --aerospike-port=3001 --tls-name=aerospike.local --tls-ca=./certs/ca.pem --tls-insecure=true

# 4) Hit it.
curl -s localhost:8080/health
curl -s -XPOST localhost:8080/put -d '{"key":"alice","bins":{"age":30}}'
curl -s localhost:8080/get/alice
curl -s -XPOST 'localhost:8080/parallel?n=24&prefix=run4'
curl -s -XPOST 'localhost:8080/multiclient?n=24&prefix=mc4'
curl -s -XPOST 'localhost:8080/freshclient?n=8&prefix=fc'
```

## Record / replay with Keploy

`keploy.yml` is already wired with the command line above. To record:

```bash
sudo keploy record
# in another shell — fire the curls from "Run it"
# then Ctrl+C the recorder
```

To replay only one test-set:

```bash
sudo keploy test --test-sets test-set-2
```

The bundled `keploy/test-set-{0,1,2}/` directories were recorded with
the dev Aerospike parser; the replay path serves binary Aerospike
ops from `mocks.yaml` so the app never touches the real cluster.

## Concurrency notes — what makes replay deterministic

Mocked replay through Keploy is roughly 20× faster than real
Aerospike for the same op. A burst of N concurrent goroutines on
a cold client pool then races to open N fresh TLS-MITM'd sockets,
and the goroutine that loses the race surfaces as
`MAX_RETRIES_EXCEEDED` at the application — even though every peer
in the same burst succeeds.

`main.go` paints over this with four layered changes; together they
make N up to 24 (the largest burst in `test-set-1`) replay 5/5 on
every run:

1. **Sized pool** — `ClientPolicy.ConnectionQueueSize = 256`,
   `OpeningConnectionThreshold = 16`. The threshold is kept low so
   stunnel's `fork()` model on the upstream doesn't get hammered.
2. **Tolerant per-op policy** — `parallelWritePolicy` and
   `parallelReadPolicy` set `SocketTimeout 10s`, `TotalTimeout 30s`,
   `MaxRetries 10`, `SleepBetweenRetries 5ms`.
3. **Two-phase warmup** on the main client at startup: an 8-op
   sequential prelude (walks the proxy past cold-start TLS) followed
   by a 32-op parallel fill (puts 32 idle connections in the pool).
4. **App-level retry wrapper** (`parallelDo`) around each PUT and
   GET in `/parallel`, `/multiclient`, and `/freshclient`. Cooperative
   goroutines in the same burst return their connections during the
   10 ms backoff, so the retry hits a warm pool.

`/multiclient`'s extra clients are deliberately NOT warmed at
startup — five clients warming in parallel produces hundreds of
concurrent TLS dials and starves stunnel's fork rate. The retry
wrapper covers their first burst instead.

## A note on Aerospike CE vs EE for TLS discovery

The upstream `aerospike-client-go/v7` driver assumes the cluster
answers Enterprise-only info commands (`service-tls-std` /
`peers-tls-std`) during topology discovery on a TLS connection.
Aerospike Community Edition replies `ERROR:25:enterprise only`,
which fails node validation even though the TLS handshake itself
succeeded.

For replay (`keploy test`) that doesn't matter — Keploy serves the
recorded discovery responses from `mocks.yaml`. For live record
against CE, the cleanest options are to point at Aerospike
Enterprise, or to apply the two-line `serviceString` /
`peersString` override locally before recording. The bundled
test-sets in this repo were recorded with that override in place.
