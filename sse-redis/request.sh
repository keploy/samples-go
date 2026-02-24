#!/usr/bin/env bash
#
# request.sh — fire all HTTP requests to generate keploy test cases covering
# SSE, NDJSON, multipart, and plain-text streaming responses.
#
# Usage:
#   ./request.sh                           # default http://localhost:8080
#   BASE_URL=http://host:port ./request.sh
#
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "==> Waiting for server at $BASE_URL ..."
for i in $(seq 1 30); do
  if curl -sf "$BASE_URL/health" >/dev/null 2>&1; then
    echo "    Server is ready."
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "    ERROR: server did not become ready in 30 s" >&2
    exit 1
  fi
  sleep 1
done

echo ""
echo "==> Clearing old messages"
curl -s -X DELETE "$BASE_URL/messages" | python3 -m json.tool 2>/dev/null || true

echo ""
echo "==> Creating messages (3 different categories)"
curl -s -X POST "$BASE_URL/messages" \
  -H "Content-Type: application/json" \
  -d '{"text":"hello world","category":"greeting"}'
echo ""

curl -s -X POST "$BASE_URL/messages" \
  -H "Content-Type: application/json" \
  -d '{"text":"streaming is awesome","category":"demo"}'
echo ""

curl -s -X POST "$BASE_URL/messages" \
  -H "Content-Type: application/json" \
  -d '{"text":"keploy rocks","category":"test"}'
echo ""

echo ""
echo "==> Fetching messages as SSE (text/event-stream)"
curl -s -N -H "Accept: text/event-stream" "$BASE_URL/events/sse"

echo ""
echo "==> Fetching messages as NDJSON (application/x-ndjson)"
curl -s -N "$BASE_URL/events/ndjson"

echo ""
echo "==> Fetching messages as multipart (multipart/x-mixed-replace)"
curl -s -N "$BASE_URL/events/multipart"

echo ""
echo "==> Fetching messages as plain text (text/plain)"
curl -s -N "$BASE_URL/events/plain"

echo ""
echo "==> Health check"
curl -s "$BASE_URL/health"
echo ""

echo ""
echo "==> Done. All requests recorded."
