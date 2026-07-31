# grpc-protoscope — Reproducing the Keploy gRPC Field-Ordering Bug

## Table of Contents

1. [The Issue Reported by the Client](#1-the-issue-reported-by-the-client)
2. [Root Cause Analysis](#2-root-cause-analysis)
3. [About This Sample Application](#3-about-this-sample-application)
4. [How to Run](#4-how-to-run)
5. [Reproducing the Bug with Keploy](#5-reproducing-the-bug-with-keploy)
6. [Files in This Repository](#6-files-in-this-repository)

---

## 1. The Issue Reported by the Client

A user reported that Keploy was **failing gRPC tests** even though the recorded and replayed responses had **identical structure and values**. The only difference was the **order of individual fields** inside nested protobuf sub-messages.

### Client's Exact Input

```yaml
expected: |-
  1: 67.0i32  # 0x42860000i32
  4: {"{\"hits\":[{\"_index\":\"pvid_search_products_v4\",\"_score\":15100000000000000000,\"_sou"
    "rce\":{\"_rankingInfo\":{\"typosPresent\":true,\"numberOfWordsMatched\":1}},\"match_type"
    "\":\"Other\",\"attributes\":{\"subThemes\":null},\"_id\":\"4f30407c-6a3c-4a4e-8a3d-652217d"
    "4b6cb_d67c25f8-3adb-40c1-9113-b46d54a6e8aa\",\"trimming_meta\":{\"trimming_type\":\"L3"
    "\"}}]}"}
  8: 0
  9: {  3: {    1: {      2: {2: 0.0}   # 0x0i64
  1: {"candidateCnt"}}
  1: {      2: {3: {"OVS"}}
  1: {"type"}}}
  2: {    1: {      2: {2: 1.0}   # 0x3ff0000000000000i64
  1: {"candidateCnt"}}
  1: {      2: {2: 1.0}   # 0x3ff0000000000000i64
  1: {"resultCnt"}}}}
actual: |-
  1: 67.0i32  # 0x42860000i32
  4: {"{\"hits\":[{\"_index\":\"pvid_search_products_v4\",\"_score\":15100000000000000000,\"_sou"
    "rce\":{\"_rankingInfo\":{\"typosPresent\":true,\"numberOfWordsMatched\":1}},\"match_type"
    "\":\"Other\",\"attributes\":{\"subThemes\":null},\"_id\":\"4f30407c-6a3c-4a4e-8a3d-652217d"
    "4b6cb_d67c25f8-3adb-40c1-9113-b46d54a6e8aa\",\"trimming_meta\":{\"trimming_type\":\"L3"
    "\"}}]}"}
  8: 0
  9: {  3: {    1: {      2: {3: {"OVS"}}
  1: {"type"}}
  1: {      2: {2: 0.0}   # 0x0i64
  1: {"candidateCnt"}}}
  2: {    1: {      2: {2: 1.0}   # 0x3ff0000000000000i64
  1: {"candidateCnt"}}
  1: {      2: {2: 1.0}   # 0x3ff0000000000000i64
  1: {"resultCnt"}}}}
```

### The Failure Classification

```yaml
failure_info:
  risk: HIGH
  category:
    - SCHEMA_BROKEN
```

### What's Actually Different?

If you look closely at field `9.3` (the availability facet bucket), the **same two sub-messages** appear but in **reversed order**:

**Expected** (recorded):
```
9: {  3: {    1: {      2: {2: 0.0}   # candidateCnt (numeric=0.0)
  1: {"candidateCnt"}}
  1: {      2: {3: {"OVS"}}            # type (text="OVS")
  1: {"type"}}}
```

**Actual** (replayed):
```
9: {  3: {    1: {      2: {3: {"OVS"}}  # type (text="OVS") — now first
  1: {"type"}}
  1: {      2: {2: 0.0}   # 0x0i64       # candidateCnt — now second
  1: {"candidateCnt"}}}
```

The values are **identical**: `candidateCnt = 0.0` and `type = "OVS"`. Only the wire serialization order changed — which is **perfectly valid** in protobuf, where `repeated` fields and map entries have no guaranteed order.

---

## 2. Root Cause Analysis

The bug lives in **three interacting layers** in Keploy's codebase.

### Layer 1: Protoscope Assigns Position-Dependent Indentation

Keploy uses the [`protoscope`](https://github.com/protocolbuffers/protoscope) library to convert raw protobuf wire bytes into human-readable text. The protoscope renderer assigns **indentation based on position**, not content.

When a sub-message is small enough, protoscope inlines it on the same line as the parent `{`:

```
9: {  3: {    1: {      2: {2: 0.0}   # 0x0i64     ← 6 spaces indent (inline)
  1: {"candidateCnt"}}                               ← 2 spaces indent (next line)
```

The **first** sub-message gets deeper inline indentation (it continues on the same line as `{`). The **second** sub-message starts on a new line with less indentation. So when the wire order flips, the same content gets **different leading whitespace**.

### Layer 2: Canonicalization Sorts With Indentation Included

The canonicalization function in `pkg/matcher/grpc/canonical.go` (`CanonicalizeTopLevelBlocks`) is designed to make protoscope text order-insensitive. It:

1. Splits text into "top-level field blocks" (lines starting with `\d+:`)
2. Recursively canonicalizes the content inside each `{...}` block
3. **Sorts blocks lexicographically**
4. Joins them back

The problem: `normalizeWhitespace()` only trims **trailing** whitespace and collapses blank lines. It does **not** strip or normalize **leading** indentation. So when `sort.Strings(blocks)` runs, the sort order is determined by the leading spaces, not the content:

```
"      2: {2: 0.0}"     sorts before    "  1: {\"candidateCnt\"}"
```

because `"      "` (6 spaces) sorts before `"  1"` (2 spaces then `1`) in ASCII. But when the wire order flips, the indentation flips too, producing a different sorted result — even though the actual protobuf data is identical.

### Layer 3: Non-JSON Mismatch Is Classified as SCHEMA_BROKEN

In `pkg/matcher/grpc/match.go`, when the two canonicalized strings don't match:

```go
if !decodedDataNormal {
    if json.Valid([]byte(expectedDecodedData)) && json.Valid([]byte(actualDecodedData)) {
        // JSON comparison with failure assessment
    } else {
        // non-JSON payload mismatch → Broken
        currentRisk = models.High
        currentCategories = append(currentCategories, models.SchemaBroken)
    }
}
```

Since protoscope text is **not valid JSON**, it falls into the `else` branch, which unconditionally classifies the failure as `HIGH` risk / `SCHEMA_BROKEN` — the most alarming category.

### Summary of the Chain

```
Wire bytes have different field order (valid in protobuf)
    → protoscope assigns different indentation
        → canonicalization sorts by indentation instead of content
            → canonicalized strings differ
                → classified as SCHEMA_BROKEN / HIGH risk
```

### The Fix

The fix needs to **strip leading whitespace from each block before sorting** in `canonicalizeRecursive`:

```go
// Before sorting, strip leading whitespace so that
// sort order depends on content, not position-dependent indentation.
for i := range blocks {
    blocks[i] = strings.TrimLeft(blocks[i], " \t")
}
sort.Strings(blocks)
```

---

## 3. About This Sample Application

This is a minimal Go gRPC client-server app that reproduces the exact conditions from the bug report.

### Why a Normal gRPC Server Isn't Enough

Go's standard `proto.Marshal()` serializes `repeated` fields and `map` entries in a **deterministic** (sorted) order. So a normal gRPC server would produce identical wire bytes on every call — the bug would never trigger.

### What This Server Does Differently

The server uses **raw wire encoding** via `google.golang.org/protobuf/encoding/protowire` to manually construct the protobuf response bytes with `rand.Shuffle()` on the repeated field entries:

```go
availEntries := [][]byte{
    buildFacetEntry("candidateCnt", &zero, nil),
    buildFacetEntry("type", nil, &ovs),
}
rand.Shuffle(len(availEntries), func(i, j int) {
    availEntries[i], availEntries[j] = availEntries[j], availEntries[i]
})
```

A `rawCodec` gRPC codec passes these pre-built bytes straight to the wire without re-marshaling, preserving the randomized field ordering.

### Proto Schema

```protobuf
message FacetValue {
  oneof value {
    double numeric = 2;
    string text    = 3;
  }
}

message FacetEntry {
  string     name = 1;
  FacetValue data = 2;
}

message FacetBucket {
  repeated FacetEntry entries = 1;
}

message FacetInfo {
  FacetBucket pricing      = 2;
  FacetBucket availability = 3;
}

message SearchResponse {
  float     score     = 1;
  string    hits_json = 4;
  int32     total     = 8;
  FacetInfo facets    = 9;
}
```

The field numbers (`1`, `4`, `8`, `9`) and nesting structure match the bug report exactly.

### Example: Recorded Test Case (Protoscope Format)

When Keploy records this server's response, the YAML test case looks like this:

```yaml
decoded_data: |
  1: 67.0i32  # 0x42860000i32
  4: {
    "{\"hits\":[{\"_index\":\"pvid_search_products_v4\",..."
  }
  8: 0
  9: {
    3: {
      1: {
        1: {"type"}
        2: {3: {"OVS"}}
      }
      1: {
        1: {"candidateCnt"}
        2: {2: 0.0}   # 0x0i64
      }
    }
    2: {
      1: {
        1: {"candidateCnt"}
        2: {2: 1.0}   # 0x3ff0000000000000i64
      }
      1: {
        1: {"resultCnt"}
        2: {2: 1.0}   # 0x3ff0000000000000i64
      }
    }
  }
```

On the next run (test mode), the `rand.Shuffle` may flip the inner field order, producing different protoscope indentation — triggering the `SCHEMA_BROKEN` false positive.

---

## 4. How to Run

### Prerequisites

- Go 1.24+
- `protoc` compiler (only needed if modifying the `.proto` file)

### Run without Keploy

```bash
# Terminal 1 — start the server
go run ./server/

# Terminal 2 — call it (run multiple times to see different field orderings)
go run ./client/
go run ./client/
go run ./client/
```

You'll see the facet entries printed in different orders across calls.

---

## 5. Reproducing the Bug with Keploy

### Step 1: Install Keploy

Install Keploy using the [official installation guide](https://keploy.io/docs/server/installation/), or build from source:

```bash
git clone https://github.com/keploy/keploy.git && cd keploy
go build -ldflags="-X main.apiServerURI=https://api.keploy.io" -o keploy
export PATH=$PWD:$PATH
```

### Step 2: Record a test case

```bash
# Start recording
keploy record -c "go run ./server/"
```

In another terminal, trigger the gRPC call:

```bash
go run ./client/
```

Then press `Ctrl+C` in the recording terminal. Keploy saves the test case in `keploy/test-set-0/tests/test-1.yaml`.

### Step 3: Replay (test mode)

```bash
keploy test -c "go run ./server/"
```

**Expected result:** Because `rand.Shuffle` randomizes field ordering each time, ~50% of test runs will produce a different wire order than the recording, triggering:

```
failure_info:
  risk: HIGH
  category:
    - SCHEMA_BROKEN
```

If the test passes (same random order happened to match), delete the `keploy/` folder and repeat steps 2–3.

---

## 6. Files in This Repository

```
grpc-protoscope/
├── README.md                      ← This file
├── go.mod
├── go.sum
├── proto/search.proto             ← Protobuf schema matching the bug report structure
├── searchpb/                      ← Generated Go protobuf/gRPC code
│   ├── search.pb.go
│   └── search_grpc.pb.go
├── server/main.go                 ← gRPC server with randomized wire field ordering
└── client/main.go                 ← gRPC client that calls the Search RPC
```

> **Note:** The `keploy/` directory (test artifacts) is generated at runtime when you run `keploy record` and is not checked into the repository.
