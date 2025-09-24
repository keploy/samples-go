#!/bin/bash
# Script to run randomized endpoints 1000 times with 200 status code validation

echo "--- Running Randomized Endpoints 1000 times ---"
BASE_URL="http://localhost:8080"

# Array of endpoints that return 200 status codes
endpoints=(
    "/"
    "/someone"
    "/noone"
    "/everyone"
    "/status"
    "/everything"
    "/somewhere"
    "/api/v1/users"
    "/api/v2/data"
    "/somebody"
    "/ping"
    "/healthz"
    "/info"
    "/anything"
    "/nothing"
    "/nowhere"
    "/products"
    "/search?q=test&limit=5"
    "/items"
    "/api/v1/data"
    "/api/v2/users"
    "/system/logs"
    "/system/metrics"
    "/proxy"
    "/anybody"
    "/everybody"
    "/user/123/profile"
)

# Counter for tracking progress
count=0
total=1000
success_count=0
error_count=0

# Start time for performance measurement
start_time=$(date +%s)

echo "Testing ${#endpoints[@]} different endpoints randomly..."

for i in $(seq 1 $total); do
    # Randomly select an endpoint
    random_index=$((RANDOM % ${#endpoints[@]}))
    selected_endpoint="${endpoints[$random_index]}"
    
    # Make the request and capture status code
    status_code=$(curl -X GET "${BASE_URL}${selected_endpoint}" -s -o /dev/null -w "%{http_code}")
    
    if [ "$status_code" = "200" ]; then
        success_count=$((success_count + 1))
    else
        error_count=$((error_count + 1))
        echo "ERROR: Request $i failed with status $status_code for endpoint $selected_endpoint"
    fi
    
    count=$((count + 1))
    
    # Show progress every 100 requests
    if [ $((count % 100)) -eq 0 ]; then
        echo "Completed $count requests... (Success: $success_count, Errors: $error_count)"
    fi
done

# Calculate and display performance metrics
end_time=$(date +%s)
duration=$((end_time - start_time))
requests_per_second=$((total / duration))

echo -e "\n--- Performance Summary ---"
echo "Total requests: $total"
echo "Successful requests (200): $success_count"
echo "Failed requests: $error_count"
echo "Success rate: $(( (success_count * 100) / total ))%"
echo "Total time: ${duration}s"
echo "Requests per second: $requests_per_second"
echo "--- Complete ---"
