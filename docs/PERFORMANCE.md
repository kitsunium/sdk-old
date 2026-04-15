# Performance Optimization Guide

## Build Configurations

The SDK provides several build configurations optimized for different use cases:

### For SDK Users

When importing the SDK as a library, the optimizations are automatically
applied. No special configuration needed.

```go
import "github.com/kitsunium/sdk/pkg/kernel/pool"
```

### For Building Applications

#### Development Build

```bash
bazel build //cmd/myapp
```

#### Production Build (Maximum Performance)

```bash
bazel build --config=perf //cmd/myapp
```

This enables (from `.bazelrc`):

- Compiler optimizations (`--copt=-O2`)
- Pure Go builds (`--@rules_go//go/config:pure`)
- Static linking (`--@rules_go//go/config:static`)
- Stripped debug symbols (`--@rules_go//go/config:gc_linkopts=-s,-w`)
- Optimized compilation mode (`--compilation_mode=opt`)

#### SDK Library Build (Balanced)

```bash
bazel build --config=sdk //pkg/...
```

This provides:

- Standard optimizations (`--copt=-O2`)
- Debug symbols retained (`--strip=never`)
- All runtime safety checks preserved
- Pure Go builds for compatibility

## Performance Features

### pool Package

The `pool` package is heavily optimized with:

1. **Zero-allocation operations**
   - String conversions use unsafe.String
   - WriteString uses unsafe.StringData

2. **Bounds check elimination**
   - Strategic hints to compiler
   - Local variable caching

3. **Compiler directives**
   - `//go:inline` for hot paths
   - Maximum inlining level in Bazel

4. **Memory pool**
   - Size-classed pools
   - Power-of-2 allocations
   - Pre-warmed buffers

## Benchmark Results

Latest optimizations show:

- **2x faster** than standard `bytes.Buffer`
- **35% faster** WriteByte operations
- **30-50% faster** Write operations
- **Zero allocations** for most operations

## Usage Examples

### High-Performance Buffer Usage

```go
// Use pooled buffers for maximum performance
buf := pool.GetBuffer(1024)
defer pool.PutBuffer(buf)

// Zero-allocation string operations
buf.WriteString("Hello ")
buf.WriteString("World")
result := buf.String() // No allocation
```

### Building with Maximum Optimizations

```bash
# For production binaries
bazel build --config=perf //cmd/yourapp

# For benchmarking
bazel run --config=perf //pkg/kernel/pool:bench

# For development with race detection
bazel build --config=debug //cmd/yourapp
```

## Compiler Flags Reference

### Go Compiler Optimizations

**Note about inlining**:

- No `-l` flag: Default inlining behavior (recommended)
- `-gcflags="-l"`: Disable inlining completely (useful for debugging)
- `-gcflags="-l=N"`: Control inlining level where N is 0-4 (higher = more
  aggressive)
  - `-l=0`: Disable inlining
  - `-l=1`: Default level
  - `-l=2-4`: Increasingly aggressive inlining

Other optimization flags:

- `-gcflags="-m"`: Print optimization decisions
- `-ldflags="-s -w"`: Strip debug symbols and DWARF info (reduces binary size)

**Important**: There is no supported public build flag to disable bounds checks
globally. The Go compiler automatically applies bounds check elimination (BCE)
where it can prove safety.

### C Compiler Optimizations

- `-O2`: Standard optimization level (balanced performance/size)
- `-O3`: Maximum optimization (may increase binary size)

## Safety Considerations

The `perf` configuration achieves optimizations through:

- **Bounds Check Elimination (BCE)**: The Go compiler automatically eliminates
  bounds checks where it can prove safety
- **Carefully scoped `unsafe` operations**: Used only with explicit bounds
  validation
- **Object pooling**: Reduces allocations and GC pressure

The `perf` configuration is recommended for:

- Production binaries after thorough testing
- Performance-critical paths
- When you're confident about code correctness

For library distribution, use `--config=sdk` which retains all runtime safety
checks while still providing good optimizations.

## Monitoring Performance

Run benchmarks to verify optimizations:

```bash
# Run benchmarks
make bench

# Compare with previous commits
make bench/compare <commit>

# Profile CPU usage
go test -cpuprofile=cpu.prof -bench=. ./pkg/kernel/pool
go tool pprof cpu.prof
```
