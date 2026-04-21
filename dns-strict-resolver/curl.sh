#!/bin/bash
# Traffic generation for the dns-strict-resolver E2E test.
# Exercises the unconnected-UDP + RFC 5452 strict-source-validation path
# that surfaces keploy/keploy#4092.

set -euo pipefail

BASE="http://localhost:8086"

echo "=== strict resolve: google.com ==="
curl -sS --max-time 10 "$BASE/resolve?domain=google.com"
echo

echo "=== strict resolve: cloudflare.com ==="
curl -sS --max-time 10 "$BASE/resolve?domain=cloudflare.com"
echo

echo "=== strict resolve: example.com ==="
curl -sS --max-time 10 "$BASE/resolve?domain=example.com"
echo

echo "=== Done ==="
