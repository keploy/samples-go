#!/bin/bash
set -euo pipefail

# Proxy Stress Test — e2e regression test for Keploy proxy fixes

docker compose build
rm -rf keploy/

cleanup() {
    docker compose down 2>/dev/null || true
}
trap cleanup EXIT

container_kill() {
    REC_PID="$(pgrep -n -f 'keploy record' || true)"
    if [ -n "$REC_PID" ]; then
        kill -INT "$REC_PID" 2>/dev/null || true
    fi
}

send_request(){
    sleep 15
    max_attempts=30
    attempt=0
    while ! curl -sf --max-time 5 http://localhost:8080/health > /dev/null 2>&1; do
        attempt=$((attempt + 1))
        if [ "$attempt" -ge "$max_attempts" ]; then
            echo "App failed to start after $max_attempts attempts"
            exit 1
        fi
        sleep 3
    done
    echo "App started"
    curl -sf --max-time 30 http://localhost:8080/api/transfer
    curl -sf --max-time 30 http://localhost:8080/api/batch-transfer
    curl -sf --max-time 30 http://localhost:8080/api/post-transfer
    curl -sf --max-time 5 http://localhost:8080/health
    sleep 5
    container_kill
    wait
}

container_name="proxyStressApp"

send_request &
keploy record -c "docker compose up" --container-name "$container_name" --generate-github-actions=false |& tee "${container_name}.txt"

if grep -q "panic:" "${container_name}.txt"; then
    echo "Panic detected during recording"
    cat "${container_name}.txt"
    exit 1
fi

echo "Recording completed"
sleep 5
docker compose down

keploy test -c 'docker compose up' --container-name "$container_name" --apiTimeout 60 --delay 20 --generate-github-actions=false &> "${container_name}-replay.txt"

docker compose down

if grep -q "panic:" "${container_name}-replay.txt"; then
    echo "Panic detected during replay"
    cat "${container_name}-replay.txt"
    exit 1
fi

all_passed=true
for report_file in ./keploy/reports/test-run-0/test-set-*-report.yaml; do
    if [ ! -f "$report_file" ]; then
        echo "No test reports found"
        exit 1
    fi
    test_status=$(grep 'status:' "$report_file" | head -n 1 | awk '{print $2}')
    echo "$(basename "$report_file"): $test_status"
    if [ "$test_status" != "PASSED" ]; then
        all_passed=false
    fi
done

if [ "$all_passed" = false ]; then
    echo "Some test sets FAILED"
    cat "${container_name}-replay.txt"
    exit 1
fi

echo "All proxy stress tests PASSED"
