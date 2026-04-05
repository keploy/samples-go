#!/bin/bash
# Generate traffic against the connect-tunnel sample app.
# Called by the CI test script during the record phase.

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "=== GET /health ==="
curl -fsS "$BASE_URL/health"
echo ""

echo "=== GET /via-proxy ==="
curl -fsS --max-time 15 "$BASE_URL/via-proxy"
echo ""
