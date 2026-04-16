#!/usr/bin/env bash
set -euo pipefail

BASE_URL="http://localhost:8080"

echo "=== PS-Cache Mock Mismatch Test ==="
echo ""
echo "Pattern: 4 /account requests with /evict in the middle."
echo "Requests 1-2 go through PgBouncer connection A."
echo "/evict forces a new PgBouncer connection B."
echo "Requests 3-4 go through connection B."
echo ""
echo "All 4 queries use the same SQL with different member_ids."
echo "PS cache warms on connection A for request 1, reuses for request 2."
echo "Connection B starts cold, PS cache warms for request 3, reuses for request 4."
echo ""

echo "--- Window 1: Connection A ---"
echo "  /account?member=19:"
curl -s "$BASE_URL/account?member=19"
echo ""
sleep 0.3

echo "  /account?member=23:"
curl -s "$BASE_URL/account?member=23"
echo ""
sleep 0.3

echo ""
echo "--- Evict (force new PgBouncer connection) ---"
echo "  /evict:"
curl -s "$BASE_URL/evict"
echo ""
sleep 1

echo ""
echo "--- Window 2: Connection B ---"
echo "  /account?member=31:"
curl -s "$BASE_URL/account?member=31"
echo ""
sleep 0.3

echo "  /account?member=42:"
curl -s "$BASE_URL/account?member=42"
echo ""

echo ""
echo "=== Done ==="
