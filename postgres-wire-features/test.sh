#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "=== postgres-wire-features traffic ==="

echo -e "\n--- /health (sanity) ---"
curl -s -w "  HTTP %{http_code}\n" "$BASE_URL/health"

echo -e "\n--- /run/all (one call exercises every scenario) ---"
curl -s -w "\n  HTTP %{http_code}\n" "$BASE_URL/run/all" -o /tmp/run-all.json
echo "  response bytes: $(wc -c </tmp/run-all.json)"

echo -e "\n--- individual scenarios ---"
for name in setup dml catalog-cte catalog-sub catalog-setop copy prepare cursor admin validation-ping; do
  curl -s -w "  /case/$name: HTTP %{http_code}\n" "$BASE_URL/case/$name" -o /dev/null
done

echo -e "\n=== done ==="
