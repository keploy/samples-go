#!/bin/bash

# Static DNS API Server Test Commands
# Pure curl commands with no dynamic fields

echo "===== Testing DNS API Server ====="
echo ""

# Health Check
echo "1. Health Check:"
curl -s "http://localhost:8086/health" | jq
echo ""

# A Record (IPv4)
echo "2. A Record Query (google.com):"
curl -s "http://localhost:8086/dns/a?domain=google.com" | jq
echo ""

# AAAA Record (IPv6)
echo "3. AAAA Record Query (google.com):"
curl -s "http://localhost:8086/dns/aaaa?domain=google.com" | jq
echo ""

# CNAME Record
echo "4. CNAME Record Query (www.github.com):"
curl -s "http://localhost:8086/dns/cname?domain=www.github.com" | jq
echo ""

# TXT Record
echo "5. TXT Record Query (google.com):"
curl -s "http://localhost:8086/dns/txt?domain=google.com" | jq
echo ""

# MX Record
echo "6. MX Record Query (gmail.com):"
curl -s "http://localhost:8086/dns/mx?domain=gmail.com" | jq
echo ""

# SRV Record
echo "7. SRV Record Query (_mongodb._tcp.cluster0.sjlpojg.mongodb.net):"
curl -s "http://localhost:8086/dns/srv?service=mongodb&proto=tcp&name=cluster0.sjlpojg.mongodb.net" | jq
echo ""

echo "===== All tests completed ====="
