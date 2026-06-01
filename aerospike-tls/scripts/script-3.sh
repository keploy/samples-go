#!/usr/bin/env bash
# script-3.sh — record + replay test-set-2.
#
# Captures /multiclient (4 pre-built *as.Client, round-robined) and
# /freshclient (per-request *as.Client construction). Demonstrates
# that Keploy replay stays deterministic across more elaborate
# client-shape patterns.

set -euo pipefail
source "$(dirname "$0")/common.sh"

curls_test_set_2() {
    curl -sf -o /dev/null "http://127.0.0.1:$PORT/health"
    sleep 1
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/multiclient?n=4&prefix=mc1"
    sleep 2
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/multiclient?n=8&prefix=mc2"
    sleep 2
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/multiclient?n=12&prefix=mc3"
    sleep 2
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/multiclient?n=24&prefix=mc4"
    sleep 3
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/freshclient?n=4&prefix=fc1"
    sleep 3
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/freshclient?n=8&prefix=fc2"
}

run_test_set test-set-2 curls_test_set_2
