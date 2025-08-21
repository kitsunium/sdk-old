#!/bin/bash
# Profile analysis script for kbuffer package

echo "=== KBuffer Performance Analysis ==="
echo

# Run comprehensive benchmarks
echo "1. Running comprehensive benchmarks..."
go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof -benchtime=1s -run=^$ > bench_results.txt 2>&1

# Analyze CPU profile
echo "2. Analyzing CPU profile..."
echo "Top CPU consumers:"
go tool pprof -top -cum cpu.prof | head -15

echo
echo "3. Analyzing memory profile..."
echo "Top memory allocators:"
go tool pprof -top -alloc_space mem.prof | head -10

echo
echo "4. Benchmark results summary:"
grep "Benchmark" bench_results.txt | grep -E "ns/op|MB/s" | head -20

echo
echo "5. Allocation analysis:"
grep "allocs/op" bench_results.txt | sort -k5 -n | head -10

echo
echo "6. Performance comparison (kbuffer vs stdlib):"
grep -E "kbuffer|bytes\.Buffer" bench_results.txt

echo
echo "7. Generating flame graph (if go-torch is installed)..."
if command -v go-torch &> /dev/null; then
    go-torch -b cpu.prof -f cpu_flame.svg
    echo "CPU flame graph saved to cpu_flame.svg"
else
    echo "go-torch not found. Install with: go get -u github.com/uber/go-torch"
fi

echo
echo "8. Optimization opportunities:"
echo "- Check for bounds check elimination"
go build -gcflags="-d=ssa/check_bce/debug=1" . 2>&1 | grep -E "Found|Bounds" | head -10

echo
echo "9. Inline decisions:"
go build -gcflags="-m=2" . 2>&1 | grep -E "inline|escape" | grep -E "Buffer|Pool" | head -15

echo
echo "=== Analysis Complete ==="
echo "Files generated:"
echo "  - bench_results.txt: Full benchmark results"
echo "  - cpu.prof: CPU profile"
echo "  - mem.prof: Memory profile"
echo "  - cpu_flame.svg: Flame graph (if go-torch installed)"