# Keploy Mocks — End-to-End Flow

This document walks through every Keploy mock captured for the
`e2e-tls-run` sample app, in the exact order Keploy recorded them, and
maps them onto the HTTP tests in `keploy/test-set-0/tests/`. The
sample app is an Aerospike-Go client talking to Aerospike over TLS
(port 3001) via Keploy's TLS-terminating proxy, so every entry in
`mocks.yaml` is an Aerospike wire-protocol exchange that the proxy
intercepted in front of the parser.

* Source recording: `keploy/test-set-0/mocks.yaml` (4104 lines, 114
  mocks numbered `mock-0` … `mock-113`).
* Source tests:     `keploy/test-set-0/tests/*.yaml` (7 HTTP tests).

---

## 1. Mock anatomy

Every mock is one Aerospike request/response pair with the same
top-level shape:

```yaml
version: api.keploy.io/v1beta1
kind: Aerospike
name: mock-N
spec:
    metadata:
        protocol: aerospike
        reqType:  Info | Message
        respType: Info | Message
        type:     config | mocks
    requests:  [ { header, meta, message } ]
    responses: [ { header, meta, message } ]
    ReqTimestampMock: <client → server>
    ResTimestampMock: <server → client>
```

Two `metadata.type` buckets cover the whole file:

| `metadata.type` | Wire | `packet_type` | Protocol byte (header.type) | What it carries                                |
|-----------------|------|---------------|-----------------------------|------------------------------------------------|
| `config`        | Info | `Info`        | `1`                         | ASCII info commands: `build`, `node`, `peers-clear-std`, `partition-generation`, … |
| `mocks`         | Data | `Message`     | `3`                         | Binary Aerospike Message protocol: PUT, GET, BATCH, TOUCH, DELETE, … |

The `meta` block is parser-decoded sugar (the human-readable view of
`raw_body`, base64 of the original bytes). `message: null` means the
parser did not need a structured `message` field — the `meta` + raw
bytes are authoritative.

---

## 2. The two big phases

The recording splits cleanly into two phases:

1. **Bootstrap + tend chatter (`type: config`, packet `Info`).**
   The Aerospike Go client uses ASCII info commands to learn cluster
   topology at startup and to keep its view fresh via a ~250 ms tend
   loop. Most of `mocks.yaml` is this background traffic.

2. **Actual application traffic (`type: mocks`, packet `Message`).**
   Each HTTP test in `tests/` is backed by one or more binary
   Aerospike Message requests. There are only a handful of these
   compared to the tend chatter.

Counts in this run:

* `type: config` mocks: ~107
* `type: mocks`  mocks: 7 — one per data operation the app issued

---

## 3. Bootstrap (mocks 0–4) — initial cluster discovery

These fire once, before any HTTP test runs, while
`as.NewClientWithPolicyAndHost` is establishing the cluster view.

| Mock      | `info_command` (request) | Response highlights                          | Purpose                                |
|-----------|--------------------------|----------------------------------------------|----------------------------------------|
| `mock-0`  | `build`                  | `build\t7.2.0.1`                             | Server version probe                   |
| `mock-1`  | `node\npartition-generation\nfeatures` | `node\tBB9D8CBBBBACB36` + `partition-generation\t0` + long `features\t…` list (`batch-any;batch-index;cdt-list;cdt-map;udf;…`) | Identify node + read its capabilities |
| `mock-2`  | `node\npeers-generation\npartition-generation` | same node id + generations `0` / `0`     | First combined tend probe              |
| `mock-3`  | `peers-clear-std`        | `peers-clear-std\t0,3000,[]` (no peers, port 3000) | Peer list (empty — single-node)        |
| `mock-4`  | `partition-generation`   | `partition-generation\t0`                    | Partition-map version (still 0)        |

The empty `peers-clear-std` response is why `main.go` sets
`policy.SeedOnlyCluster = true` — Aerospike CE only advertises the
clear-text peer endpoint, which is unreachable behind the TLS
terminator, so the client must stay pinned to the seed.

---

## 4. Tend loop pattern (mocks 5–26, and sprinkled throughout)

After bootstrap, the Aerospike Go client's tend goroutine repeats a
two-mock pair roughly every 250–500 ms while the app is idle:

* **`node` + `peers-generation` + `partition-generation`** — node id +
  whether peers or partitions have changed. Always responds with the
  same id and generations `0` (no topology drift in this single-node
  recording).
* **`peers-clear-std`** — full peer list. Always empty.

These pairs (mocks 5/6, 7/8, 9/10, …) make up the bulk of
`mocks.yaml`. They are **only there to satisfy the client's
periodic refresh**; they do not correspond to any HTTP test. Replays
serve them on demand so the client stays "connected" between data
ops.

You will see exactly this `node` / `peers-clear-std` pair re-appear
between every pair of HTTP tests below.

---

## 5. The seven application operations (Message mocks)

The seven binary-protocol mocks line up one-to-one with the seven HTTP
tests, in capture order:

| Mock       | Test file                       | App endpoint            | Aerospike op (decoded)            | Notes                                                |
|------------|---------------------------------|-------------------------|------------------------------------|------------------------------------------------------|
| `mock-27`  | `post-put-1.yaml`               | `POST /put`             | PUT `test/demo/alice`              | Bins `{name:"Alice", age:30}`. Request 101 B → 22 B ack. |
| `mock-30`  | `get-get-alice-1.yaml`          | `GET /get/alice`        | GET `test/demo/alice`              | Request 65 B → 22 B + bin payload.                   |
| `mock-33`  | `post-batch-put-1.yaml` (1/2)   | `POST /batch/put`       | PUT `test/demo/a` `{n:1}`          | Implemented as a sequential PUT loop in `main.go`.   |
| `mock-34`  | `post-batch-put-1.yaml` (2/2)   | `POST /batch/put`       | PUT `test/demo/b` `{n:2}`          | Second PUT in the same handler.                      |
| `mock-37`  | `get-batch-get-1.yaml`          | `GET /batch/get?k=a&k=b`| BATCH_READ digests for `a`, `b`    | Request 105 B → 78 B with two records.               |
| `mock-42`  | `get-batch-get-1.yaml` (retry)  | (same)                  | Large BATCH_READ retry             | 8259 B request, fires after the test handler already returned `500 TIMEOUT` — see §6. |
| `mock-55`  | `post-touch-alice-1.yaml`       | `POST /touch/alice`     | TOUCH `test/demo/alice`            | `Operate(... 0x0B)` — touch generation/TTL.          |
| `mock-58`  | `delete-key-alice-1.yaml`       | `DELETE /key/alice`     | DELETE `test/demo/alice`           | Final cleanup. info3 flag `0x04` = delete.           |

Every Message mock carries `namespace: test`, `set: demo`, matching
the keys `main.go` constructs with `as.NewKey("test","demo",…)`.

### How to spot which is which

* Look at `requests[0].header.header.length` and the leading byte of
  the decoded `raw_body`:
  * info1/info2/info3 byte 1 of the body tells you the opcode flags
    (READ / WRITE / DELETE / TOUCH / BATCH).
* `meta.namespace` and `meta.set` confirm the target.
* Pair it with the nearest test by timestamp (`ReqTimestampMock` vs.
  the test's `req.timestamp`).

---

## 6. Special case — the BatchGet timeout (mocks 37 → 42)

`get-batch-get-1.yaml` is the only test that did **not** return 200.
Its recorded response is:

```
status_code: 500
body: |
  ResultCode: TIMEOUT, Iteration: 1, InDoubt: false, …
        read tcp 127.0.0.1:50236->127.0.0.1:3001: i/o timeout
```

That timeout is exactly what produces the unusual pair:

1. `mock-37` (`14:06:28.494`) — the first BATCH_READ attempt for
   `{a, b}`. Server returns successfully (78 B), but the client
   considers the call timed out.
2. The HTTP handler returns 500 to the curl at `14:06:29.490`.
3. `mock-42` (`14:06:30.501`) — the Aerospike client's retry path
   fires a much larger BATCH_READ (8259 B) on its own goroutine,
   well after the HTTP response was already sent. The response is
   164 B containing the two records' digests.

The same shape repeats much later in the recording as `mock-101`
(`14:07:00.622`, 8300 B), an even later orphaned retry — both are
artifacts of the timeout path, not new application traffic.

---

## 7. End-to-end flow diagram

```
T=0   App starts → Aerospike client constructor
      ├─ mock-0 .. mock-4    : bootstrap (build, node+features, peers, partition-gen)
      └─ (tend goroutine begins; mock-5..mock-26 fire every ~250 ms)

T+12s curl GET /health
      └─ no Aerospike traffic (handler only calls client.GetNodes() in-process)

      curl POST /put alice  ───────────────────►  mock-27  (PUT alice, ack)

      [tend pair: mock-28 / mock-29]

      curl GET /get/alice   ───────────────────►  mock-30  (GET alice)

      [tend pair: mock-31 / mock-32]

      curl POST /batch/put  ───────────────────►  mock-33  (PUT a {n:1})
                            ─────────────────────►  mock-34  (PUT b {n:2})

      [tend pairs: mock-35 / mock-36]

      curl GET /batch/get?k=a&k=b ──────────────►  mock-37  (BATCH_READ {a,b})
                                                    ↓ client-side timeout
                                                  HTTP 500 TIMEOUT to curl
      [tend pairs: mock-38 .. mock-41]
                                                  mock-42  (retry, 8259 B request)

      [long tend stretch: mock-43 .. mock-54]

      curl POST /touch/alice ──────────────────►  mock-55  (TOUCH alice)

      [tend pair: mock-56 / mock-57]

      curl DELETE /key/alice ──────────────────►  mock-58  (DELETE alice)

      [tail tend stretch: mock-59 .. mock-100]
                                                  mock-101 (late BATCH_READ retry)
      [tail tend stretch: mock-102 .. mock-113]
```

The numbered mocks not explicitly called out are all tend-loop
`node` / `peers-clear-std` pairs — the same two requests, with the
same two responses, replayed by Keploy to keep the Aerospike Go
client happy while it waits for the next HTTP test.

---

## 8. What the parser sees vs. what the proxy sees

Because the proxy terminates TLS upstream of the Aerospike parser,
every byte you see in `mocks.yaml` is **plain Aerospike wire
protocol** — there is no TLS framing in `raw_body`, even though the
app dials port 3001 with a real TLS ClientHello. That is the second
claim called out in `main.go`'s header comment:

> Proxy TLS detection is byte-pattern driven, not port driven.
> This client sends a real TLS ClientHello on a non-3306, non-443
> port; the proxy still recognises it via `Peek(5) → IsTLSHandshake`
> and MITMs the connection.

If you want to verify, base64-decode any `raw_body` and confirm the
leading byte is the Aerospike message header (`0x16` for Message
ops, ASCII text for Info ops) — never a TLS record (`0x16 0x03 …`
ClientHello).
