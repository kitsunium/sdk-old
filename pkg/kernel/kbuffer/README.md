# kbuffer - High-Performance Buffer Management

Zero-allocation, CPU-optimized buffer management with advanced pooling for Go
applications.

## Features

- **Zero-allocation operations** in hot paths
- **Lock-free buffer pools** with power-of-2 size classes
- **CPU cache-aligned** data structures
- **Unsafe optimizations** for maximum performance
- **Security-hardened** bounds checking
- **Atomic statistics** tracking

## Installation

```bash
go get github.com/kitsunium/sdk/pkg/kernel/kbuffer
```

## Usage

### Basic Buffer Operations

```go
import "github.com/kitsunium/sdk/pkg/kernel/kbuffer"

// Create a new buffer
buf := kbuffer.NewBuffer(1024)

// Write data
buf.Write([]byte("hello"))
buf.WriteString(" world")
buf.WriteByte('!')

// Read data
data := buf.Bytes()  // []byte("hello world!")
str := buf.String()  // "hello world!"

// Reset for reuse
buf.Reset()
```

### Buffer Pool

```go
// Get buffer from global pool
buf := kbuffer.Get(4096)
defer kbuffer.Put(buf)

// Use the buffer
copy(buf, []byte("data"))

// Get Buffer object from pool
b := kbuffer.GetBuffer(1024)
defer kbuffer.PutBuffer(b)

b.WriteString("pooled buffer")
```

### Pool Statistics

```go
stats := kbuffer.Stats()
fmt.Printf("Hit rate: %.2f%%\n", stats.HitRate())
fmt.Printf("Alloc rate: %.2f%%\n", stats.AllocRate())
```

## Performance

Benchmark results on Apple M1 Pro:

| Operation          | Time    | Throughput | Allocations |
| ------------------ | ------- | ---------- | ----------- |
| Buffer.Write(64B)  | 3.22 ns | 19.9 GB/s  | 0 allocs    |
| Buffer.Write(1KB)  | 18.8 ns | 54.6 GB/s  | 0 allocs    |
| Buffer.WriteString | 3.31 ns | 19.3 GB/s  | 0 allocs    |
| Buffer.String      | 1.10 ns | -          | 0 allocs    |
| Pool.GetPut        | 35.4 ns | 116 GB/s   | 1 alloc     |

### Comparison vs stdlib

- **1.75x faster** than bytes.Buffer
- **Zero allocations** in Write operations (vs 2 allocs in bytes.Buffer)
- **100% pool hit rate** after warmup

## Build

### Using Go

```bash
go test ./...
go test -bench=. -benchmem
```

### Using Bazel

```bash
bazel test //pkg/kernel/kbuffer:all
bazel test //pkg/kernel/kbuffer:kbuffer_bench_test --test_arg=-test.bench=.
```

### Using Make

```bash
make test      # Run tests
make bench     # Run benchmarks
make coverage  # Generate coverage report
```

## License

See LICENSE file in the repository root.
