#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "=== postgres-wire-features traffic ==="

echo -e "\n--- /health (sanity) ---"
curl -s -w "  HTTP %{http_code}\n" "$BASE_URL/health"

# /run/all exercises every scenario (setup, dml, catalog-{cte,sub,setop},
# copy, prepare, cursor, admin, validation-ping) inside a single HTTP
# request. That keeps the recorded HTTP testcase count small (one per
# scenario batch) while still driving the full Postgres wire surface
# Keploy needs to see.
echo -e "\n--- /run/all (one call exercises every scenario) ---"
curl -s -w "\n  HTTP %{http_code}\n" "$BASE_URL/run/all" -o /tmp/run-all.json
echo "  response bytes: $(wc -c </tmp/run-all.json)"

echo -e "\n=== done ==="
