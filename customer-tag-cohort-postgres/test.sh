#!/usr/bin/env bash
# test.sh — drives the homogeneous-bucket cohort pattern that reproduces
# issue #4 (intra-bucket FIFO desync).
#
# The five GET /customers/11/tags?offset=N calls all share:
#   * identical SQL hash (same prepared text)
#   * identical primary bind ($1 = 11)
# But each returns a DIFFERENT rowset (because OFFSET shifts the window).
# That's the canonical reproducer shape — five mocks in one bucket,
# heterogeneous payloads. If the v3 cohort consumer holds FIFO order on
# replay, all five match. If it desyncs, body diff fails.
#
# Two more customers (202, 203) are hit once each so the test set has at
# least three distinct buckets. /tag/:id and POST /tags exercise different
# SQL hashes for parity with the upstream sap_demo_java reproducer.
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "=== customer-tag-cohort-postgres exerciser ==="

# Wait for app readiness via /health. The app brings up its HTTP listener
# before initDB completes, so /health returns "starting" until the seed is
# done — poll until "ok".
echo "--- waiting for /health to flip to ok ---"
for i in $(seq 1 60); do
  body=$(curl -fsS "$BASE_URL/health" 2>/dev/null || echo '')
  if echo "$body" | grep -q '"status":"ok"'; then
    echo "  app ready after ${i}s"
    break
  fi
  sleep 1
done

# ------ Cohort window: same customer, same SQL, varying OFFSET ------
# Five calls that all bucket on (sql_hash, customer_id=11). Each call's
# response is rows[offset..offset+50) — all distinct payloads. This is
# the homogeneous-bucket pattern that triggers #4 if cohort consumption
# desyncs.
echo ""
echo "--- Cohort window: 5x GET /customers/11/tags with shifting offset ---"
for offset in 0 1 2 3 4; do
  echo "  GET /customers/11/tags?limit=50&offset=${offset}:"
  curl -fSs "$BASE_URL/customers/11/tags?limit=50&offset=${offset}" | head -c 300
  echo ""
  sleep 0.2
done

# Two distinct buckets, one call each — confirms the per-bucket consumer
# stays scoped to its (sql_hash, primary_bind) pair.
echo ""
echo "--- Other customers (distinct buckets, single call each) ---"
for cid in 202 203; do
  echo "  GET /customers/${cid}/tags?limit=50&offset=0:"
  curl -fSs "$BASE_URL/customers/${cid}/tags?limit=50&offset=0" | head -c 200
  echo ""
  sleep 0.2
done

# Different SQL hash — single-row SELECT. Uses a primary bind (id=1) that
# does not collide with the cohort bucket's bind (customer_id=11).
echo ""
echo "--- Single-row SELECT (different SQL hash) ---"
echo "  GET /tag/1:"
curl -fSs "$BASE_URL/tag/1"
echo ""

echo "  GET /tag/52:"  # second customer's first tag
curl -fSs "$BASE_URL/tag/52"
echo ""
sleep 0.2

# INSERT path — yet another SQL hash, returns the new id.
echo ""
echo "--- INSERT (different SQL hash) ---"
echo "  POST /tags:"
curl -fSs -X POST "$BASE_URL/tags" \
  -H 'Content-Type: application/json' \
  -d '{"customer_id": 11, "tag": "exerciser-injected", "priority": 2}'
echo ""

# Final cohort sweep with the SAME bucket again — five MORE calls into the
# bucket already loaded above. This stretches the cohort to 10 entries on
# the same (sql_hash, customer_id=11) key, and any FIFO desync becomes
# even more likely to surface across the 10-deep sequence.
echo ""
echo "--- Cohort window 2: 5 more GET /customers/11/tags (different offsets) ---"
for offset in 5 6 7 8 9; do
  echo "  GET /customers/11/tags?limit=50&offset=${offset}:"
  curl -fSs "$BASE_URL/customers/11/tags?limit=50&offset=${offset}" | head -c 300
  echo ""
  sleep 0.2
done

echo ""
echo "=== exerciser done ==="
