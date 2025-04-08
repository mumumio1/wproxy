#!/bin/bash

# Health check script for wproxy

PROXY_URL="${PROXY_URL:-http://localhost:8080}"
TIMEOUT=5

check_health() {
    response=$(curl -s -o /dev/null -w "%{http_code}" --max-time $TIMEOUT "${PROXY_URL}/health")
    if [ "$response" = "200" ]; then
        echo "✓ Health check passed"
        return 0
    else
        echo "✗ Health check failed (HTTP $response)"
        return 1
    fi
}

check_ready() {
    response=$(curl -s -o /dev/null -w "%{http_code}" --max-time $TIMEOUT "${PROXY_URL}/ready")
    if [ "$response" = "200" ]; then
        echo "✓ Readiness check passed"
        return 0
    else
        echo "✗ Readiness check failed (HTTP $response)"
        return 1
    fi
}

check_metrics() {
    response=$(curl -s -o /dev/null -w "%{http_code}" --max-time $TIMEOUT "http://localhost:9090/metrics")
    if [ "$response" = "200" ]; then
        echo "✓ Metrics endpoint accessible"
        return 0
    else
        echo "✗ Metrics endpoint failed (HTTP $response)"
        return 1
    fi
}

echo "Checking wproxy at ${PROXY_URL}..."
echo ""

all_passed=true

if ! check_health; then
    all_passed=false
fi

if ! check_ready; then
    all_passed=false
fi

if ! check_metrics; then
    all_passed=false
fi

echo ""
if [ "$all_passed" = true ]; then
    echo "All checks passed ✓"
    exit 0
else
    echo "Some checks failed ✗"
    exit 1
fi

