#!/usr/bin/env bash
set -euo pipefail

BASE_URL="http://localhost:8080"

# Deterministic per-iteration company names. Previously this used a random
# generator (head -c 8 /dev/urandom | base64 | tr -dc 'a-zA-Z' | head -c 10)
# producing names like "Company_<8-10 random alphas>_<i>", which differ
# between record and replay runs of the Keploy http-postgres CI pipeline.
# The postgres-v3 replayer matches mocks by bind value, so divergent random
# names triggered intermittent "no invocation matched" errors on test-4 /
# test-21 (the POST /companies inserts). A deterministic counter keeps
# inserts unique within a single run while guaranteeing record/replay agree
# byte-for-byte on the bind payload.

echo "=== Rapid-fire API test (25+ calls) ==="

# 1-10: Create 10 unique companies in parallel
echo -e "\n--- Creating 10 unique companies (parallel) ---"
pids=()
for i in $(seq 1 10); do
  name="Company_${i}"
  curl -s -w "  HTTP %{http_code}\n" -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -d "{\"name\": \"$name\"}" &
  pids+=($!)
done
wait "${pids[@]}"

# 11-15: Create 5 companies we'll reuse for duplicate tests
echo -e "\n--- Creating 5 known companies ---"
KNOWN=("DupeAlpha" "DupeBeta" "DupeGamma" "DupeDelta" "DupeEpsilon")
for name in "${KNOWN[@]}"; do
  curl -s -w "  HTTP %{http_code}\n" -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -d "{\"name\": \"$name\"}" &
done
wait

# 16-20: Try to create the same 5 again (should all 409)
echo -e "\n--- Duplicate creates (expect 409) ---"
for name in "${KNOWN[@]}"; do
  curl -s -w "  HTTP %{http_code}\n" -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -d "{\"name\": \"$name\"}" &
done
wait

# 21: List all companies
echo -e "\n--- List companies ---"
curl -s -w "\n  HTTP %{http_code}\n" "$BASE_URL/companies"

# 22-26: Get each known company by name
echo -e "\n--- Get companies by name ---"
for name in "${KNOWN[@]}"; do
  curl -s -w "  HTTP %{http_code}\n" "$BASE_URL/companies/$name" &
done
wait

# 27: Get a company that doesn't exist (expect 404)
echo -e "\n--- Get non-existent company (expect 404) ---"
curl -s -w "  HTTP %{http_code}\n" "$BASE_URL/companies/NoSuchCorp"

# 28: Empty name (expect 400)
echo -e "\n--- Empty name (expect 400) ---"
curl -s -w "  HTTP %{http_code}\n" -X POST "$BASE_URL/companies" \
  -H 'Content-Type: application/json' \
  -d '{"name": ""}'

# 29: Invalid JSON (expect 400)
echo -e "\n--- Invalid JSON (expect 400) ---"
curl -s -w "  HTTP %{http_code}\n" -X POST "$BASE_URL/companies" \
  -H 'Content-Type: application/json' \
  -d 'not-json'

echo -e "\n=== Done ==="
