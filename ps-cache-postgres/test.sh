#!/usr/bin/env bash
set -euo pipefail

BASE_URL="http://localhost:8080"

echo "=== PS-Cache Mock Mismatch Test ==="

echo "--- Window 1: Connection A ---"
echo "  /account?member=19:"
curl -fSs "$BASE_URL/account?member=19"
echo ""
sleep 0.3

echo "  /account?member=23:"
curl -fSs "$BASE_URL/account?member=23"
echo ""
sleep 0.3

echo ""
echo "--- Evict (force new connection) ---"
echo "  /evict:"
curl -fSs "$BASE_URL/evict"
echo ""
sleep 1

echo ""
echo "--- Window 2: Connection B ---"
echo "  /account?member=31:"
curl -fSs "$BASE_URL/account?member=31"
echo ""
sleep 0.3

echo "  /account?member=42:"
curl -fSs "$BASE_URL/account?member=42"
echo ""

echo ""
echo "=== Done ==="
