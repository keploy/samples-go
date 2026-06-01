#!/usr/bin/env bash
# script-2.sh — record + replay test-set-1.
#
# Captures /parallel at four concurrency levels (n = 4, 8, 12, 24) so
# the replay path exercises bursts of concurrent goroutines sharing a
# single *as.Client.

set -euo pipefail
source "$(dirname "$0")/common.sh"

curls_test_set_1() {
    curl -sf -o /dev/null "http://127.0.0.1:$PORT/health"
    sleep 1
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/parallel?n=4&prefix=run1"
    sleep 2
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/parallel?n=8&prefix=run2"
    sleep 2
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/parallel?n=12&prefix=run3"
    sleep 2
    curl -sf -o /dev/null -XPOST "http://127.0.0.1:$PORT/parallel?n=24&prefix=run4"
}

run_test_set test-set-1 curls_test_set_1
