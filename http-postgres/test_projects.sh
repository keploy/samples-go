#!/bin/bash
# Test script for the UUID-based projects endpoint.
# During recording, the server generates UUIDs at runtime.
# During replay, the server generates DIFFERENT UUIDs.
# The value substitution feature should handle this transparently.
#
# Test flow:
#   1. Create a project (server generates UUID)
#   2. Get that project by UUID
#   3. Update that project by UUID
#   4. Get it again to verify update
#   5. List all projects
#   6. Create another project

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "=== Test: Create project ==="
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/projects" \
  -H "Content-Type: application/json" \
  -d '{"name": "Alpha Project", "status": "active"}')
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -1)
echo "Status: $CODE"
echo "Body: $BODY"
PROJECT_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Project ID: $PROJECT_ID"

echo ""
echo "=== Test: Get project by UUID ==="
curl -s -X GET "$BASE_URL/projects/$PROJECT_ID" | python3 -m json.tool 2>/dev/null || true

echo ""
echo "=== Test: Update project ==="
curl -s -X PUT "$BASE_URL/projects/$PROJECT_ID" \
  -H "Content-Type: application/json" \
  -d '{"name": "Alpha Project (Updated)", "status": "completed"}' | python3 -m json.tool 2>/dev/null || true

echo ""
echo "=== Test: Get project after update ==="
curl -s -X GET "$BASE_URL/projects/$PROJECT_ID" | python3 -m json.tool 2>/dev/null || true

echo ""
echo "=== Test: List all projects ==="
curl -s -X GET "$BASE_URL/projects" | python3 -m json.tool 2>/dev/null || true

echo ""
echo "=== Test: Create another project ==="
curl -s -X POST "$BASE_URL/projects" \
  -H "Content-Type: application/json" \
  -d '{"name": "Beta Project", "status": "pending"}' | python3 -m json.tool 2>/dev/null || true

echo ""
echo "Done."
