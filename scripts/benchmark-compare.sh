#!/bin/bash
# GoShield Benchmark Comparison Script
# Usage: ./scripts/benchmark-compare.sh [baseline] [current]

set -e

BASELINE=${1:-benchmark-results/baseline.txt}
CURRENT=${2:-benchmark-results/latest.txt}
OUTPUT=${3:-benchmark-results/comparison.md}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== GoShield Benchmark Comparison ==="
echo ""

# Check if files exist
if [ ! -f "$BASELINE" ]; then
    echo -e "${RED}Error: Baseline file not found: $BASELINE${NC}"
    echo "Run 'make bench-save' first to create baseline."
    exit 1
fi

if [ ! -f "$CURRENT" ]; then
    echo -e "${RED}Error: Current file not found: $CURRENT${NC}"
    echo "Run 'make bench-save' first to create current results."
    exit 1
fi

# Extract timestamps
BASELINE_TIME=$(head -1 "$BASELINE" | cut -d' ' -f2-)
CURRENT_TIME=$(head -1 "$CURRENT" | cut -d' ' -f2-)

echo "Baseline: $BASELINE_TIME"
echo "Current:  $CURRENT_TIME"
echo ""

# Function to extract benchmark results
extract_results() {
    local file=$1
    local pattern=$2
    grep -E "^Benchmark${pattern}" "$file" | awk '{
        name=$1
        ns=$3
        # Convert to ns/op
        if ($4 == "ns/op") {
            ns=$3
        } else if ($4 == "µs/op") {
            ns=$3 * 1000
        } else if ($4 == "ms/op") {
            ns=$3 * 1000000
        }
        printf "%s\t%s\n", name, ns
    }'
}

# Function to calculate percentage change
calc_change() {
    local baseline=$1
    local current=$2
    if [ -z "$baseline" ] || [ -z "$current" ]; then
        echo "N/A"
        return
    fi
    local change=$(echo "scale=2; (($current - $baseline) / $baseline) * 100" | bc)
    if (( $(echo "$change > 0" | bc -l) )); then
        echo -e "${RED}+${change}%${NC}"
    elif (( $(echo "$change < 0" | bc -l) )); then
        echo -e "${GREEN}${change}%${NC}"
    else
        echo -e "${YELLOW}${change}%${NC}"
    fi
}

# Generate comparison report
echo "Generating comparison report..."
echo ""

# Create markdown report
cat > "$OUTPUT" << EOF
# GoShield Benchmark Comparison

Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)

## Summary

| Metric | Baseline | Current | Change |
|--------|----------|---------|--------|
EOF

# Compare key metrics
for pattern in "CircuitBreaker_Execute_Closed" "RateLimiter_Allow" "Bulkhead_Execute" "Timeout_Execute" "Retry_Execute_Success" "AdaptiveBreaker_Execute_Closed" "SimpleCounter_Inc"; do
    baseline_ns=$(grep -E "^Benchmark${pattern}-" "$BASELINE" | head -1 | awk '{print $3}')
    current_ns=$(grep -E "^Benchmark${pattern}-" "$CURRENT" | head -1 | awk '{print $3}')
    
    if [ -n "$baseline_ns" ] && [ -n "$current_ns" ]; then
        change=$(calc_change "$baseline_ns" "$current_ns")
        echo "| ${pattern} | ${baseline_ns} ns/op | ${current_ns} ns/op | ${change} |" >> "$OUTPUT"
    fi
done

cat >> "$OUTPUT" << EOF

## Detailed Results

### Circuit Breaker
\`\`\`
$(grep -E "^BenchmarkCircuitBreaker" "$CURRENT" | head -20)
\`\`\`

### Rate Limiter
\`\`\`
$(grep -E "^BenchmarkRateLimiter" "$CURRENT" | head -20)
\`\`\`

### Bulkhead
\`\`\`
$(grep -E "^BenchmarkBulkhead" "$CURRENT" | head -20)
\`\`\`

### Backoff
\`\`\`
$(grep -E "^BenchmarkBackoff" "$CURRENT" | head -20)
\`\`\`

### Timeout
\`\`\`
$(grep -E "^BenchmarkTimeout" "$CURRENT" | head -20)
\`\`\`

### Adaptive Breaker
\`\`\`
$(grep -E "^BenchmarkAdaptiveBreaker" "$CURRENT" | head -20)
\`\`\`

### Metrics
\`\`\`
$(grep -E "^BenchmarkSimple" "$CURRENT" | head -20)
\`\`\`

### Parallel Benchmarks
\`\`\`
$(grep -E "Parallel" "$CURRENT" | head -20)
\`\`\`

### Contention Benchmarks
\`\`\`
$(grep -E "Contention" "$CURRENT" | head -20)
\`\`\`
EOF

echo -e "${GREEN}Comparison report generated: $OUTPUT${NC}"
echo ""

# Print summary to console
echo "=== Quick Summary ==="
echo ""
for pattern in "CircuitBreaker_Execute_Closed" "RateLimiter_Allow" "Bulkhead_Execute" "Timeout_Execute" "Retry_Execute_Success" "AdaptiveBreaker_Execute_Closed"; do
    baseline_ns=$(grep -E "^Benchmark${pattern}-" "$BASELINE" | head -1 | awk '{print $3}')
    current_ns=$(grep -E "^Benchmark${pattern}-" "$CURRENT" | head -1 | awk '{print $3}')
    
    if [ -n "$baseline_ns" ] && [ -n "$current_ns" ]; then
        change=$(calc_change "$baseline_ns" "$current_ns")
        printf "%-40s %10s ns/op -> %10s ns/op %s\n" "$pattern" "$baseline_ns" "$current_ns" "$change"
    fi
done

echo ""
echo "=== Comparison Complete ==="
