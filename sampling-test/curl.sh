#!/usr/bin/env bash
# Fire $TOTAL_REQUESTS concurrent GET /work calls against the sample app.
# Each curl uses its own TCP connection (no -K, no session reuse), so the
# Keploy proxy sees TOTAL_REQUESTS connection-acquire attempts inside the
# server-side handler delay window — far more than --enable-sampling slots,
# forcing the bypass path on most of them.
#
# Writes one status-code line per request to $RESULTS_DIR/req-<i>.status.
# Caller (workflow / verify script) inspects those files to assert that
# every client got a 2xx response (no 502s, no timeouts) regardless of
# whether the request was captured.

set -euo pipefail

TOTAL_REQUESTS="${TOTAL_REQUESTS:-20}"
TARGET_URL="${TARGET_URL:-http://localhost:8080/work}"
RESULTS_DIR="${RESULTS_DIR:-./curl-results}"
CONNECT_TIMEOUT="${CONNECT_TIMEOUT:-5}"
MAX_TIME="${MAX_TIME:-15}"

mkdir -p "$RESULTS_DIR"
rm -f "$RESULTS_DIR"/req-*.status "$RESULTS_DIR"/req-*.body

echo "Firing $TOTAL_REQUESTS concurrent requests to $TARGET_URL"

pids=()
for i in $(seq 1 "$TOTAL_REQUESTS"); do
    (
        # -o body file, -w status code, -s silent, --no-keepalive so each
        # curl invocation opens its own TCP connection.
        status=$(curl -s -o "$RESULTS_DIR/req-${i}.body" \
            -w "%{http_code}" \
            --no-keepalive \
            --connect-timeout "$CONNECT_TIMEOUT" \
            --max-time "$MAX_TIME" \
            "$TARGET_URL" || echo "000")
        echo "$status" > "$RESULTS_DIR/req-${i}.status"
    ) &
    pids+=("$!")
done

# Wait for every concurrent curl to finish.
fail=0
for pid in "${pids[@]}"; do
    if ! wait "$pid"; then
        fail=$((fail + 1))
    fi
done

ok=0
nonok=0
for f in "$RESULTS_DIR"/req-*.status; do
    code=$(cat "$f")
    if [[ "$code" =~ ^2[0-9][0-9]$ ]]; then
        ok=$((ok + 1))
    else
        nonok=$((nonok + 1))
        echo "non-2xx response from $(basename "$f"): $code"
    fi
done

echo "curl summary: total=$TOTAL_REQUESTS ok=$ok non2xx=$nonok wait_failures=$fail"

# Any non-2xx is a client-visible failure (proxy must never break clients
# just because the request was bypassed for sampling). Fail loudly.
if [[ "$nonok" -ne 0 || "$fail" -ne 0 ]]; then
    exit 1
fi
