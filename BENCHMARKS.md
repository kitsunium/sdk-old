# Benchmark System

This project includes a comprehensive benchmarking system with SQLite storage
and commit comparison capabilities.

## Features

- **Dual-mode benchmarks**: Separate single-core and multi-core benchmarks for
  accurate performance comparison
- **SQLite storage**: All benchmark results are stored in a local SQLite
  database
- **Git isolation**: Option to clone the repository for benchmarking specific
  commits
- **Performance comparison**: Compare benchmark results between different
  commits
- **Parallel scaling analysis**: Analyze how well your code scales across
  multiple cores

## Quick Start

### Run benchmarks for current commit

```bash
make bench
```

### Run benchmarks for a specific commit (with isolation)

```bash
make bench COMMIT=abc123
```

### Compare benchmarks between commits

```bash
make bench/compare COMMIT1=abc123 COMMIT2=def456
```

### Analyze parallel scaling

```bash
make bench/scaling
```

### List all saved benchmarks

```bash
make bench/list
```

## Benchmark Structure

The benchmarks are organized to clearly distinguish between single-core and
multi-core performance:

- **Single-core benchmarks** (`Benchmark*_SingleCore`): Run with GOMAXPROCS=1
  for consistent baseline measurements
- **Multi-core benchmarks** (`Benchmark*_MultiCore`): Run with GOMAXPROCS set to
  system CPU count for parallel performance testing

Example benchmark naming:

- `BenchmarkSet_SingleCore` - Single-threaded set operations
- `BenchmarkSet_MultiCore` - Multi-threaded set operations with parallel
  execution

## Database Schema

The SQLite database (`benchmarks.sqlite`) stores:

- Commit information (hash, date, author, message)
- Package and benchmark names
- Performance metrics (ns/op, MB/s, allocations)
- Execution mode (single/multi) and core count
- System information (Go version, OS, architecture)

## Advanced Usage

### Direct script usage

```bash
# Run benchmarks with isolation
python3 scripts/bench_runner.py run HEAD --clone

# Compare specific commits
python3 scripts/bench_runner.py compare commit1 commit2

# Analyze scaling
python3 scripts/bench_runner.py scaling HEAD

# List saved benchmarks
python3 scripts/bench_runner.py list
```

### Interpreting Results

When comparing benchmarks, the system will show:

- **Performance changes**: Percentage improvement/degradation between commits
- **Parallel efficiency**: How well the code scales (speedup vs core count)
- **Memory metrics**: Allocation changes between versions

Example output:

```
Benchmark Comparison
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Base:    abc123 - Fix memory leak
Compare: def456 - Optimize cache

kcache
─────────────────────────────
Benchmark            Mode     Time/op        Change      Allocs
Set_SingleCore       single   125.3 ns       ↓ 15.2%     0 allocs
Set_MultiCore        multi    45.6 ns        ↓ 22.1%     0 allocs
```

## Best Practices

1. **Consistent environment**: Run benchmarks on the same machine for accurate
   comparisons
2. **Multiple runs**: Consider running benchmarks multiple times for statistical
   significance
3. **Isolate commits**: Use the `--clone` option when benchmarking specific
   commits to avoid workspace interference
4. **Monitor scaling**: Regularly check parallel scaling to identify concurrency
   bottlenecks

## Troubleshooting

If benchmarks aren't found:

1. Ensure your benchmark functions follow the naming pattern
2. Check that BUILD.bazel files have the correct benchmark targets
3. Verify bazel configuration includes benchmark configs

For database issues:

- Delete `benchmarks.sqlite` to start fresh
- The database schema is automatically created on first use
