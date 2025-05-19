# 1. List all products
curl -X GET http://localhost:8080/products

# 2. Create a new product
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name":       "Widget",
    "price":      19.99,
    "quantity":   100,
    "metadata":   {"color":"red","size":"M"},
    "related_ids":["abc123","def456"],
    "categories": ["gadgets","sale"]
}'

# 3. Get a single product by ID
curl -X GET http://localhost:8080/products/<ID>

# 4. Update a product
curl -X PUT http://localhost:8080/products/<ID> \
  -H "Content-Type: application/json" \
  -d '{
    "price":    17.49,
    "quantity": 80
}'

# 5. Delete a product
curl -X DELETE http://localhost:8080/products/<ID>

# 6. Add a rating (score 4.2) to a product
curl -X POST http://localhost:8080/products/<ID>/rate \
  -H "Content-Type: application/json" \
  -d '{"score":4.2}'

# 7. Get aggregated ratings for a product
curl -X GET http://localhost:8080/products/<ID>/ratings

# 8. Add tags to a product
curl -X POST http://localhost:8080/products/<ID>/tags \
  -H "Content-Type: application/json" \
  -d '{"tags":["electronics","new"]}'

# 9. List tags for a product
curl -X GET http://localhost:8080/products/<ID>/tags

# 10. List all products under a tag
curl -X GET http://localhost:8080/tags/<TAG>/products

# 11. Bulk‐create multiple products
curl -X POST http://localhost:8080/products/bulk \
  -H "Content-Type: application/json" \
  -d '[
    {"name":"Gizmo","price":9.99,"quantity":50},
    {"name":"Doohickey","price":5.99,"quantity":200}
]'

# 12. Get the recent activity log
curl -X GET http://localhost:8080/activity

# 13. Record/query unique visitors for a product
#    (this GET just returns the count)
curl -X GET http://localhost:8080/products/<ID>/visitors

# 14. Get top‐selling products leaderboard
curl -X GET http://localhost:8080/leaderboard

# 15. Update a user’s cart
curl -X POST http://localhost:8080/carts/<USER_ID> \
  -H "Content-Type: application/json" \
  -d '{"<PRODUCT_ID_1>":2,"<PRODUCT_ID_2>":1}'

# 16. Retrieve a user’s cart
curl -X GET http://localhost:8080/carts/<USER_ID>

# 17. Ping the health endpoint
curl -X GET http://localhost:8080/health