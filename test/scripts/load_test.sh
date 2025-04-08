#!/bin/bash

# Load test script for wproxy

set -e

PROXY_URL="${PROXY_URL:-http://localhost:8080}"
DURATION="${DURATION:-30s}"
CONNECTIONS="${CONNECTIONS:-100}"
REQUESTS="${REQUESTS:-1000}"

echo "=== Load Testing wproxy ==="
echo "Proxy URL: $PROXY_URL"
echo "Duration: $DURATION"
echo "Connections: $CONNECTIONS"
echo "Total Requests: $REQUESTS"
echo ""

# Check if wrk is installed
if ! command -v wrk &> /dev/null; then
    echo "wrk not found. Attempting to install..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install wrk
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "Please install wrk manually: https://github.com/wg/wrk"
        exit 1
    fi
fi

# Run load test
echo "Starting load test..."
wrk -t4 -c${CONNECTIONS} -d${DURATION} --latency ${PROXY_URL}/test

echo ""
echo "=== Cache Performance Test ==="
# First, prime the cache
echo "Priming cache..."
for i in {1..10}; do
    curl -s ${PROXY_URL}/test > /dev/null
done

# Test cached performance
echo "Testing cached responses..."
wrk -t4 -c${CONNECTIONS} -d10s --latency ${PROXY_URL}/test

echo ""
echo "Load test completed!"

