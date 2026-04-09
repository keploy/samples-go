#!/bin/bash
set -euo pipefail

# Proxy Stress Test — e2e regression test for:
# 1. TLS cert caching (concurrent HTTPS through CONNECT tunnel)
# 2. Large PostgreSQL DataRow responses (wire protocol reassembly)
# 3. HTTP POST through forward proxy (MatchType validation)

docker compose build
rm -rf keploy/

container_kill() {
    REC_PID="$(pgrep -n -f 'keploy record' || true)"
    echo "Keploy record PID: $REC_PID"
    kill -INT "$REC_PID" 2>/dev/null || true
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
    curl -sf http://localhost:8080/api/transfer
    curl -sf http://localhost:8080/api/batch-transfer
    curl -sf http://localhost:8080/api/post-transfer
    curl -sf http://localhost:8080/health
    sleep 5
    container_kill
    wait
}

container_name="proxyStressApp"

send_request &
keploy record -c "docker compose up" --container-name "$container_name" --generateGithubActions=false |& tee "${container_name}.txt"

if grep -q "panic:" "${container_name}.txt"; then
    echo "Panic detected during recording"
    cat "${container_name}.txt"
    exit 1
fi

echo "Recording completed"
sleep 5
docker compose down

keploy test -c 'docker compose up' --containerName "$container_name" --apiTimeout 60 --delay 20 --generate-github-actions=false &> "${container_name}-replay.txt"

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
