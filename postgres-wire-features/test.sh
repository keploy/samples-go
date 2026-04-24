#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "=== postgres-wire-features traffic ==="

# --fail-with-body makes curl exit non-zero on HTTP >= 400 while still
# printing the response body, so a failing endpoint surfaces as a real
# shell failure instead of silently producing a bad recording.
echo -e "\n--- /health (sanity) ---"
curl -s --fail-with-body -w "  HTTP %{http_code}\n" "$BASE_URL/health"

# /run/all exercises every scenario (setup, dml, catalog-{cte,sub,setop},
# copy, prepare, cursor, admin, validation-ping, multistatement,
# empty-query) inside a single HTTP request. That keeps the recorded HTTP
# testcase count small while still driving the full Postgres wire surface
# Keploy needs to see.
echo -e "\n--- /run/all (one call exercises every scenario) ---"
curl -s --fail-with-body -w "\n  HTTP %{http_code}\n" "$BASE_URL/run/all" -o /tmp/run-all.json
echo "  response bytes: $(wc -c </tmp/run-all.json)"

# Standalone hits for scenarios that target specific v3 protocol edges.
# Running them on fresh connections (the /case endpoints use withClient)
# keeps them isolated from the long-lived /run/all connection so the
# recorder sees a clean connection boundary around each.
echo -e "\n--- /case/multistatement (BEGIN; SELECT 1; SELECT 2; COMMIT as ONE Q packet) ---"
curl -s --fail-with-body -w "  HTTP %{http_code}\n" "$BASE_URL/case/multistatement" -o /tmp/case-multistatement.json
echo "  response bytes: $(wc -c </tmp/case-multistatement.json)"

echo -e "\n--- /case/empty-query (Q packet with empty SQL body) ---"
curl -s --fail-with-body -w "  HTTP %{http_code}\n" "$BASE_URL/case/empty-query" -o /tmp/case-empty-query.json
echo "  response bytes: $(wc -c </tmp/case-empty-query.json)"

echo -e "\n=== done ==="
