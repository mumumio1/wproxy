#!/bin/bash

# Benchmark script for wproxy

set -e

PROXY_URL="${PROXY_URL:-http://localhost:8080}"
OUTPUT_DIR="./benchmark-results"

mkdir -p "$OUTPUT_DIR"

echo "=== wproxy Benchmark Suite ==="
echo "Target: $PROXY_URL"
echo "Results will be saved to: $OUTPUT_DIR"
echo ""

# Benchmark 1: Baseline (no cache, no rate limit)
echo "1. Baseline Test (1k concurrent, 10k requests)"
ab -n 10000 -c 1000 -g "${OUTPUT_DIR}/baseline.tsv" "${PROXY_URL}/test" > "${OUTPUT_DIR}/baseline.txt"

# Benchmark 2: Cache Hit Performance
echo "2. Cache Hit Test (priming cache first)"
# Prime cache
for i in {1..10}; do
    curl -s "${PROXY_URL}/test" > /dev/null
done
sleep 1
ab -n 10000 -c 1000 -g "${OUTPUT_DIR}/cache-hit.tsv" "${PROXY_URL}/test" > "${OUTPUT_DIR}/cache-hit.txt"

# Benchmark 3: Rate Limit Threshold
echo "3. Rate Limit Test (high load)"
ab -n 50000 -c 2000 -g "${OUTPUT_DIR}/rate-limit.tsv" "${PROXY_URL}/test" > "${OUTPUT_DIR}/rate-limit.txt"

# Benchmark 4: Different Payload Sizes
echo "4. Various Payload Sizes"
for size in 1k 10k 100k 1m; do
    echo "  Testing ${size} payload"
    ab -n 1000 -c 100 "${PROXY_URL}/payload/${size}" > "${OUTPUT_DIR}/payload-${size}.txt"
done

# Benchmark 5: Latency Percentiles
echo "5. Latency Distribution (detailed)"
ab -n 100000 -c 100 -g "${OUTPUT_DIR}/latency-dist.tsv" "${PROXY_URL}/test" > "${OUTPUT_DIR}/latency-dist.txt"

# Generate summary
echo ""
echo "=== Benchmark Summary ==="
echo ""
echo "Baseline:"
grep "Requests per second" "${OUTPUT_DIR}/baseline.txt"
grep "Time per request" "${OUTPUT_DIR}/baseline.txt" | head -1
grep "Transfer rate" "${OUTPUT_DIR}/baseline.txt"
echo ""
echo "Cache Hit:"
grep "Requests per second" "${OUTPUT_DIR}/cache-hit.txt"
grep "Time per request" "${OUTPUT_DIR}/cache-hit.txt" | head -1
echo ""
echo "Results saved to ${OUTPUT_DIR}/"

