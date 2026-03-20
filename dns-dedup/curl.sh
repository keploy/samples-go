#!/bin/bash
# Traffic generation for dns-dedup E2E test.
# Exercises single and bulk DNS resolution endpoints.

set -euo pipefail

BASE="http://localhost:8086"

echo "=== Single resolve (default domain) ==="
curl -sS "$BASE/resolve"
echo

echo "=== Single resolve (google.com) ==="
curl -sS "$BASE/resolve?domain=google.com"
echo

echo "=== Bulk resolve: 30 lookups for default domain ==="
curl -sS "$BASE/resolve-many?n=30"
echo

echo "=== Bulk resolve: 10 lookups for google.com ==="
curl -sS "$BASE/resolve-many?n=10&domain=google.com"
echo

echo "=== Done ==="
