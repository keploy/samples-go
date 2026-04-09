#!/bin/bash

# Proxy Stress Test — exercises:
# 1. Concurrent HTTPS through CONNECT tunnel (TLS cert caching)
# 2. Large PostgreSQL DataRow responses (wire protocol reassembly)
# 3. HTTP POST through forward proxy (MatchType validation)
#
# This test verifies that Keploy recording and replay work correctly under
# proxy stress conditions that previously caused 322s p99 latency and hangs.

set -euo pipefail

source ./../../.github/workflows/test_workflow_scripts/test-iid.sh

docker compose build

sudo rm -rf keploy/

$RECORD_BIN config --generate

container_kill() {
    REC_PID="$(pgrep -n -f 'keploy record' || true)"
    echo "Keploy record PID: $REC_PID"
    sudo kill -INT "$REC_PID" 2>/dev/null || true
}

send_request(){
    sleep 15
    app_started=false
    while [ "$app_started" = false ]; do
        if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            app_started=true
        fi
        sleep 3
    done
    echo "App started"

    # Test 1: Concurrent HTTPS through CONNECT tunnel + large PG rows
    echo "=== Recording: /api/transfer (10 concurrent HTTPS + 50 large PG rows) ==="
    curl -sf http://localhost:8080/api/transfer | tee /tmp/transfer.json
    echo ""

    # Test 2: Batch concurrent HTTPS (triggers cert storm without caching)
    echo "=== Recording: /api/batch-transfer (20 concurrent HTTPS + 300 PG rows) ==="
    curl -sf http://localhost:8080/api/batch-transfer | tee /tmp/batch.json
    echo ""

    # Test 3: POST through proxy (MatchType validation)
    echo "=== Recording: /api/post-transfer (POST through CONNECT tunnel) ==="
    curl -sf http://localhost:8080/api/post-transfer | tee /tmp/post.json
    echo ""

    # Health check
    curl -sf http://localhost:8080/health

    sleep 5
    container_kill
    wait
}

container_name="proxyStressApp"

# Record phase
send_request &
$RECORD_BIN record -c "docker compose up" --container-name "$container_name" --generateGithubActions=false |& tee "${container_name}.txt"

if grep "WARNING: DATA RACE" "${container_name}.txt"; then
    echo "Race condition detected in recording"
    cat "${container_name}.txt"
    exit 1
fi

# Check that recording succeeded (no fatal errors)
if grep -q "panic:" "${container_name}.txt"; then
    echo "Panic detected during recording"
    cat "${container_name}.txt"
    exit 1
fi

echo "Recording completed successfully"
sleep 5

# Shutdown services before replay
docker compose down

# Replay phase
$REPLAY_BIN test -c 'docker compose up' --containerName "$container_name" --apiTimeout 60 --delay 20 --generate-github-actions=false &> "${container_name}-replay.txt"

if grep "WARNING: DATA RACE" "${container_name}-replay.txt"; then
    echo "Race condition detected in replay"
    cat "${container_name}-replay.txt"
    exit 1
fi

if grep -q "panic:" "${container_name}-replay.txt"; then
    echo "Panic detected during replay"
    cat "${container_name}-replay.txt"
    exit 1
fi

# Verify test results
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
