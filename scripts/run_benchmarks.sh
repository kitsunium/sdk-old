#!/bin/bash
# Script to run all configured benchmarks for Kitsunium SDK packages

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Kitsunium SDK Benchmark Runner ===${NC}"
echo

# Configuration options
CONFIG="${1:-prod}"
BENCHTIME="${2:-10000x}"
PATTERN="${3:-.}"

echo -e "${YELLOW}Configuration:${NC}"
echo "  Config: $CONFIG"
echo "  Benchtime: $BENCHTIME"
echo "  Pattern: $PATTERN"
echo

# List of benchmark targets
TARGETS=(
    "//pkg/kernel/kcache:bench"
    "//pkg/kernel/kcache:bench_multi"
    "//pkg/kernel/kbuffer:bench"
    "//pkg/kernel/kbuffer:bench_multi"
    "//pkg/kernel/fs:bench"
    "//pkg/kernel/fs:bench_multi"
    "//pkg/kernel/kerror:bench"
    "//pkg/kernel/kerror:bench_multi"
    "//pkg/core/config/parser:bench"
    "//pkg/core/config/parser:bench_multi"
)

# Function to run a single benchmark
run_benchmark() {
    local target=$1
    local name=$(echo $target | sed 's/.*\/\///g' | sed 's/:/_/g')
    
    echo -e "${GREEN}Running benchmark: $target${NC}"
    
    # Run the benchmark
    bazel run $target --config=$CONFIG -- \
        -test.bench=$PATTERN \
        -test.benchtime=$BENCHTIME \
        -test.run=^$ \
        -test.benchmem \
        2>/dev/null | grep -E "^(Benchmark|goos|goarch|cpu|PASS|FAIL)" || true
    
    echo
}

# Run all benchmarks
for target in "${TARGETS[@]}"; do
    run_benchmark "$target"
done

echo -e "${BLUE}=== Benchmark run complete ===${NC}"

# Usage examples
echo
echo "Usage examples:"
echo "  ./scripts/run_benchmarks.sh                    # Run all benchmarks in prod mode"
echo "  ./scripts/run_benchmarks.sh dev                # Run all benchmarks in dev mode"
echo "  ./scripts/run_benchmarks.sh prod 1000x         # Run with custom iteration count"
echo "  ./scripts/run_benchmarks.sh prod 10s           # Run for 10 seconds each"
echo "  ./scripts/run_benchmarks.sh prod 10000x Cache  # Run only Cache benchmarks"
echo
echo "Individual benchmark commands:"
echo "  bazel run //pkg/kernel/kcache:bench --config=prod"
echo "  bazel run //pkg/kernel/kcache:bench_multi --config=prod"
echo "  bazel run //pkg/kernel/kbuffer:bench --config=prod"
echo "  bazel run //pkg/kernel/kbuffer:bench_multi --config=prod"
echo "  bazel run //pkg/kernel/fs:bench --config=prod"
echo "  bazel run //pkg/kernel/fs:bench_multi --config=prod"
echo "  bazel run //pkg/kernel/kerror:bench --config=prod"
echo "  bazel run //pkg/kernel/kerror:bench_multi --config=prod"
echo "  bazel run //pkg/core/config/parser:bench --config=prod"
echo "  bazel run //pkg/core/config/parser:bench_multi --config=prod"