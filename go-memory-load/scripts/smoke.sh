#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
TMP_DIR="${TMP_DIR:-/tmp/load-test-smoke}"
mkdir -p "$TMP_DIR"

suffix="$(date +%s)"

customer_payload="$(jq -n \
  --arg email "smoke-${suffix}@example.com" \
  --arg name "Smoke Test ${suffix}" \
  '{email:$email, full_name:$name, segment:"startup"}')"

customer_status="$(curl -sS -o "$TMP_DIR/customer.json" -w '%{http_code}' \
  -X POST "${BASE_URL}/customers" \
  -H 'Content-Type: application/json' \
  -d "$customer_payload")"
[[ "$customer_status" == "201" ]]
customer_id="$(jq -r '.id' "$TMP_DIR/customer.json")"

product_a_payload="$(jq -n \
  --arg sku "SMOKE-${suffix}-A" \
  '{sku:$sku, name:"Smoke Product A", category:"compute", price_cents:2500, inventory_count:200}')"

product_a_status="$(curl -sS -o "$TMP_DIR/product-a.json" -w '%{http_code}' \
  -X POST "${BASE_URL}/products" \
  -H 'Content-Type: application/json' \
  -d "$product_a_payload")"
[[ "$product_a_status" == "201" ]]
product_a_id="$(jq -r '.id' "$TMP_DIR/product-a.json")"

product_b_payload="$(jq -n \
  --arg sku "SMOKE-${suffix}-B" \
  '{sku:$sku, name:"Smoke Product B", category:"analytics", price_cents:3200, inventory_count:200}')"

product_b_status="$(curl -sS -o "$TMP_DIR/product-b.json" -w '%{http_code}' \
  -X POST "${BASE_URL}/products" \
  -H 'Content-Type: application/json' \
  -d "$product_b_payload")"
[[ "$product_b_status" == "201" ]]
product_b_id="$(jq -r '.id' "$TMP_DIR/product-b.json")"

order_payload="$(jq -n \
  --argjson customer_id "$customer_id" \
  --argjson product_a_id "$product_a_id" \
  --argjson product_b_id "$product_b_id" \
  '{customer_id:$customer_id, status:"paid", items:[{product_id:$product_a_id, quantity:2}, {product_id:$product_b_id, quantity:1}]}')"

order_status="$(curl -sS -o "$TMP_DIR/order.json" -w '%{http_code}' \
  -X POST "${BASE_URL}/orders" \
  -H 'Content-Type: application/json' \
  -d "$order_payload")"
[[ "$order_status" == "201" ]]
order_id="$(jq -r '.id' "$TMP_DIR/order.json")"

detail_status="$(curl -sS -o "$TMP_DIR/detail.json" -w '%{http_code}' "${BASE_URL}/orders/${order_id}")"
[[ "$detail_status" == "200" ]]

summary_status="$(curl -sS -o "$TMP_DIR/summary.json" -w '%{http_code}' "${BASE_URL}/customers/${customer_id}/summary")"
[[ "$summary_status" == "200" ]]

search_status="$(curl -sS -o "$TMP_DIR/search.json" -w '%{http_code}' \
  "${BASE_URL}/orders?status=paid&customer_id=${customer_id}&min_total_cents=1000&limit=5")"
[[ "$search_status" == "200" ]]

analytics_status="$(curl -sS -o "$TMP_DIR/analytics.json" -w '%{http_code}' \
  "${BASE_URL}/analytics/top-products?days=30&limit=5")"
[[ "$analytics_status" == "200" ]]

large_payload_blob="$(head -c 262144 /dev/zero | tr '\0' 'L')"
large_payload_request="$(jq -n \
  --arg name "Large Smoke Payload ${suffix}" \
  --arg content_type "text/plain" \
  --arg payload "$large_payload_blob" \
  '{name:$name, content_type:$content_type, payload:$payload}')"

large_create_status="$(curl -sS -o "$TMP_DIR/large-create.json" -w '%{http_code}' \
  -X POST "${BASE_URL}/large-payloads" \
  -H 'Content-Type: application/json' \
  -d "$large_payload_request")"
[[ "$large_create_status" == "201" ]]
large_payload_id="$(jq -r '.id' "$TMP_DIR/large-create.json")"

large_get_status="$(curl -sS -o "$TMP_DIR/large-get.json" -w '%{http_code}' \
  "${BASE_URL}/large-payloads/${large_payload_id}")"
[[ "$large_get_status" == "200" ]]
[[ "$(jq -r '.payload_size_bytes' "$TMP_DIR/large-get.json")" == "262144" ]]

large_delete_status="$(curl -sS -o "$TMP_DIR/large-delete.json" -w '%{http_code}' \
  -X DELETE "${BASE_URL}/large-payloads/${large_payload_id}")"
[[ "$large_delete_status" == "200" ]]

jq -n \
  --arg customer_id "$customer_id" \
  --arg order_id "$order_id" \
  --arg order_total "$(jq -r '.total_cents' "$TMP_DIR/detail.json")" \
  --arg favorite "$(jq -r '.favorite_category' "$TMP_DIR/summary.json")" \
  --arg search_count "$(jq 'length' "$TMP_DIR/search.json")" \
  --arg top_sku "$(jq -r '.[0].sku' "$TMP_DIR/analytics.json")" \
  --arg large_payload_bytes "$(jq -r '.payload_size_bytes' "$TMP_DIR/large-create.json")" \
  '{
    customer_id: ($customer_id | tonumber),
    order_id: $order_id,
    order_total_cents: ($order_total | tonumber),
    favorite_category: $favorite,
    search_count: ($search_count | tonumber),
    top_sku: $top_sku,
    large_payload_bytes: ($large_payload_bytes | tonumber)
  }'
