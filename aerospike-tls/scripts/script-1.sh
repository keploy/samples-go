#!/usr/bin/env bash
# script-1.sh — record + replay test-set-0.
#
# Captures the single-endpoint CRUD coverage:
#   GET /health  POST /put  GET /get  POST /batch/put  GET /batch/get
#   POST /touch  DELETE /key
#
# Output: writes keploy/test-set-0/, then runs `keploy test`
# against it and exits non-zero if any case fails.

set -euo pipefail
source "$(dirname "$0")/common.sh"

curls_test_set_0() {
    curl -sf -o /dev/null "http://127.0.0.1:$PORT/health"
    sleep 1
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/put" \
        -H 'Content-Type: application/json' \
        -d '{"key":"alice","bins":{"age":30,"name":"Alice"}}'
    sleep 1
    curl -sf -o /dev/null "http://127.0.0.1:$PORT/get/alice"
    sleep 1
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/batch/put" \
        -H 'Content-Type: application/json' \
        -d '[{"key":"a","bins":{"n":1}},{"key":"b","bins":{"n":2}}]'
    sleep 1
    curl -s -o /dev/null "http://127.0.0.1:$PORT/batch/get?k=a&k=b" || true
    sleep 1
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/touch/alice"
    sleep 1
    curl -sf -o /dev/null -XDELETE "http://127.0.0.1:$PORT/key/alice"
}

run_test_set test-set-0 curls_test_set_0
