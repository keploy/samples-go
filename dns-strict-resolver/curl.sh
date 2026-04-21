#!/bin/bash
# Traffic generation for the dns-strict-resolver E2E test.
# Exercises the unconnected-UDP + RFC 5452 strict-source-validation path
# that surfaces keploy/keploy#4092.
#
# NAMESERVER (optional): ip:port of the DNS server to query explicitly.
# When set, it is passed through to /resolve so the sample does not
# have to rely on /etc/resolv.conf (which, under keploy's docker-mode
# --network=container:<keploy-v3> rewrite, may not be a useful
# address). The CI harness sets it to the fixture CoreDNS container.

set -euo pipefail

BASE="http://localhost:8086"
NS_QUERY=""
if [[ -n "${NAMESERVER:-}" ]]; then
  NS_QUERY="&nameserver=${NAMESERVER}"
fi

echo "=== strict resolve: google.com ==="
curl -sS --max-time 10 "$BASE/resolve?domain=google.com${NS_QUERY}"
echo

echo "=== strict resolve: cloudflare.com ==="
curl -sS --max-time 10 "$BASE/resolve?domain=cloudflare.com${NS_QUERY}"
echo

echo "=== strict resolve: example.com ==="
curl -sS --max-time 10 "$BASE/resolve?domain=example.com${NS_QUERY}"
echo

echo "=== Done ==="
