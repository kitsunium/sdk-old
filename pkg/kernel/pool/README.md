# pool - High-Performance Kernel Buffer Package

## ⚠️ CRITICAL: Thread-Safety Choice

**YOU MUST CHOOSE THE RIGHT BUFFER TYPE FOR YOUR USE CASE:**

| Buffer Type                    | Thread-Safe | Performance          | Use Case                         |
| ------------------------------ | ----------- | -------------------- | -------------------------------- |
| **`NewUnsafeBuffer()`**        | ❌ **NO**   | ~2-3 ns/op (FASTEST) | Single-threaded ONLY             |
| **`NewSafeBuffer()`**          | ✅ **YES**  | ~15-25 ns/op (FAST)  | Multi-threaded (2-10 goroutines) |
| **`NewUnsafeShardedBuffer()`** | ❌ **NO**   | ~5-10 ns/op (FAST)   | Single-threaded with sharding    |
| **`NewSafeShardedBuffer()`**   | ✅ **YES**  | ~70-85 ns/op         | High contention (10+ goroutines) |

**⚠️ WARNING: Using `NewUnsafeBuffer()` in concurrent contexts WILL cause data
corruption and crashes!**

## Overview

`pool` is an ultra-optimized kernel buffer package providing maximum
performance through:

- Zero-allocation operations
- Unsafe memory operations for speed
- CPU cache-line alignment
- Lock-free algorithms where possible
- Choice between unsafe (fast) and safe (thread-safe) implementations

## Quick Start

### Single-Threaded Usage (FASTEST)

```go
import "github.com/kitsunium/sdk/pkg/kernel/pool"

// ⚠️ UNSAFE: Use ONLY in single-threaded context!
buf := pool.NewUnsafeBuffer(1024)

// Write operations - ~2-3 ns/op
buf.Write([]byte("hello"))
buf.WriteString(" world")
buf.WriteByte('!')

// Read operations - zero-copy
data := buf.Bytes()  // []byte("hello world!")
str := buf.String()  // "hello world!"
```

### Multi-Threaded Usage (SAFE)

```go
import "github.com/kitsunium/sdk/pkg/kernel/pool"

// ✅ SAFE: Thread-safe with spinlock optimization
buf := pool.NewSafeBuffer(1024)

// Safe for concurrent access from multiple goroutines
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        buf.Write([]byte(fmt.Sprintf("goroutine %d\n", id)))
    }(i)
}
wg.Wait()
```

## Performance Benchmarks

### Write Performance Comparison

| Operation  | Unsafe Buffer (Dev) | Unsafe Buffer (Prod)\* | std bytes.Buffer | Speed vs std |
| ---------- | ------------------- | ---------------------- | ---------------- | ------------ |
| Write 64B  | 3548 ns/op          | **4.13 ns/op**         | ~16.5 ns/op      | 4.0x faster  |
| Write 256B | 3497 ns/op          | **7.10 ns/op**         | ~16.5 ns/op      | 2.3x faster  |
| Write 1KB  | 3503 ns/op          | **17.80 ns/op**        | ~16.5 ns/op      | Similar      |
| Write 4KB  | 3588 ns/op          | **58.70 ns/op**        | ~60 ns/op        | Similar      |
| Write 16KB | 3817 ns/op          | **225.0 ns/op**        | ~230 ns/op       | Similar      |
| Write 64KB | 4628 ns/op          | **846.8 ns/op**        | ~850 ns/op       | Similar      |
| WriteByte  | N/A                 | **2.24 ns/op**         | ~3.5 ns/op       | 1.6x faster  |

\*Production mode: compiled with `-tags=unsafe_no_check` to disable safety
checks

### Memory & Allocation Performance

| Operation          | Time/op (Dev) | Time/op (Prod) | Allocs/op | Bytes/op | Description             |
| ------------------ | ------------- | -------------- | --------- | -------- | ----------------------- |
| Write operations   | 3500+ ns      | 4-850 ns       | 0         | 0 B      | Zero allocations always |
| Pool Get/Put (1KB) | 42.97 ns      | 49.86 ns       | 1         | 24 B     | Buffer pooling overhead |
| Buffer Bytes()     | N/A           | 2.35 ns        | 0         | 0 B      | Zero-copy read          |
| Buffer String()    | N/A           | 2.76 ns        | 0         | 0 B      | Zero-alloc conversion   |
| Reset()            | N/A           | 13.20 ns       | 0         | 0 B      | Fast reset              |

### Throughput Performance (Production Mode)

| Buffer Size | pool Throughput | Speed     | Notes                       |
| ----------- | ------------------ | --------- | --------------------------- |
| 64 bytes    | 15,498 MB/s        | Very Fast | Optimal for small writes    |
| 256 bytes   | 36,052 MB/s        | Very Fast | Excellent cache utilization |
| 1 KB        | 57,518 MB/s        | Very Fast | Peak throughput             |
| 4 KB        | 69,783 MB/s        | Very Fast | Large buffer optimization   |
| 16 KB       | 72,810 MB/s        | Very Fast | Near memory bandwidth limit |
| 64 KB       | 77,393 MB/s        | Very Fast | Maximum throughput achieved |

**Important Notes**:

- Development mode includes goroutine safety checks that add ~3500ns overhead
- Production mode (`-tags=unsafe_no_check`) provides maximum performance
- Zero allocations for all operations
- Performance scales linearly with buffer size

### Reproducibility

To reproduce these benchmark results exactly:

- **Go Version**: 1.22.x or later
- **OS**: macOS 14.5 / Linux kernel 5.15+
- **CPU**: Apple M1 Pro / Intel Core i7-9750H @ 2.60GHz
- **CPU Settings**: Performance governor, Turbo/SMT enabled
- **Benchmark Commands**:
  - Development: `go test -bench=. -benchmem -count=5`
  - Production: `go test -tags=unsafe_no_check -bench=. -benchmem -count=5`
- **Date**: August 2025
- **Commit**: ad9ce1e (feat/kbuffer branch)
- **Number of Runs**: 5 iterations per benchmark

## Buffer Types

### 1. Unsafe Buffer

- **Function**: `NewUnsafeBuffer()`
- **Thread-Safe**: ❌ NO
- **Performance**: ~2-3 ns/op
- **Use When**:
  - Single-threaded execution guaranteed
  - Maximum performance required
  - You manage synchronization externally

### 2. Safe Buffer

- **Function**: `NewSafeBuffer()`
- **Thread-Safe**: ✅ YES (spinlock)
- **Performance**: ~15-25 ns/op
- **Use When**:
  - 2-10 goroutines access buffer
  - Thread-safety required
  - Good performance with safety

### 3. Unsafe Sharded Buffer

- **Function**: `NewUnsafeShardedBuffer()`
- **Thread-Safe**: ❌ NO
- **Performance**: ~5-10 ns/op
- **Use When**:
  - Single-threaded but need sharding for data organization
  - Maximum performance with data distribution
  - External synchronization if needed

### 4. Safe Sharded Buffer

- **Function**: `NewSafeShardedBuffer()`
- **Thread-Safe**: ✅ YES
- **Performance**: ~70-85 ns/op even with 100 goroutines
- **Use When**:
  - High contention (10+ goroutines)
  - Write-heavy workloads
  - Need horizontal scaling
  - **7x faster than SafeBuffer with 100 goroutines!**

## Advanced Features

### Buffer Pooling

```go
// Global pool for buffer reuse
pool := pool.GetGlobalPool()

// Get buffer from pool
buf := pool.GetBuffer(1024)
defer pool.PutBuffer(buf)  // Return to pool when done

// Use buffer...
buf.Write([]byte("pooled buffer"))
```

### Zero-Copy Operations

```go
buf := pool.NewSafeBuffer(1024)
buf.WriteString("hello")

// Zero-copy read - shares memory with buffer
data := buf.Bytes()

// Direct memory access (unsafe)
// WARNING: ptr becomes invalid after any Write/Reset/Clear operation!
// The returned memory is NOT copied - use with extreme caution.
ptr, len := buf.BytesUnsafe()
```

### Sharded Buffer for High Contention

```go
// Safe sharded buffer for concurrent writes
buf := pool.NewSafeShardedBuffer(10000, 16)

// Concurrent writes distributed across shards using sync.WaitGroup
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        buf.Write(item.Bytes())  // Automatically sharded
    }(item)
}
wg.Wait()

// Rebalance if needed
buf.Balance()
```

### Unsafe Sharded Buffer for Single-Threaded Use

```go
// Unsafe sharded buffer for organized data distribution
buf := pool.NewUnsafeShardedBuffer(10000, 16)

// Write to specific shards for data organization
buf.WriteToShard(0, headerData)
buf.WriteToShard(1, bodyData)
buf.WriteToShard(2, footerData)
```

## API Reference

### Buffer Interface

```go
type Buffer interface {
    // Write operations
    Write(p []byte) (n int, err error)
    WriteString(s string) (n int, err error)
    WriteByte(c byte) error
    WriteAt(p []byte, off int64) (n int, err error)
    TryWrite(p []byte) bool  // Non-blocking

    // Read operations (lock-free)
    Bytes() []byte
    String() string
    BytesUnsafe() (ptr uintptr, len int)

    // Management
    Len() int
    Cap() int
    Available() int
    Reset()
    Clear()
    Truncate(n int)
    Grow(n int) error
    Extend(n int) error

    // Advanced
    Clone() Buffer
    RemainingSlice() []byte
    AppendBytes(data ...byte) error
}
```

## 🛡️ Automatic Safety Protection

**NEW: Goroutine safety checks prevent silent corruption!**

When using `UnsafeBuffer` or `UnsafeShardedBuffer`, the package automatically
detects concurrent access and **panics with a clear error message** to prevent
data corruption:

```go
buf := pool.NewUnsafeBuffer(1024)

// First goroutine - OK
go func() {
    buf.Write([]byte("data"))  // ✅ Works
}()

// Second goroutine - PANIC!
go func() {
    buf.Write([]byte("more"))  // 💥 PANIC: "UNSAFE buffer accessed from multiple goroutines!"
}()
```

This protection:

- ✅ Prevents silent data corruption
- ✅ Forces explicit safety choices
- ✅ Can be disabled in production builds with `-tags=unsafe_no_check`

## Safety Guidelines

### ❌ DON'T Do This (Will Panic)

```go
// WRONG: Unsafe buffer in concurrent context
buf := pool.NewUnsafeBuffer(1024)  // ❌ UNSAFE

go func() {
    buf.Write([]byte("goroutine 1"))  // ❌ DATA RACE
}()

go func() {
    buf.Write([]byte("goroutine 2"))  // ❌ DATA RACE
}()
```

### ✅ DO This Instead

```go
// CORRECT: Safe buffer for concurrent access
buf := pool.NewSafeBuffer(1024)  // ✅ SAFE

go func() {
    buf.Write([]byte("goroutine 1"))  // ✅ Thread-safe
}()

go func() {
    buf.Write([]byte("goroutine 2"))  // ✅ Thread-safe
}()
```

### Or This (External Synchronization)

```go
// CORRECT: Unsafe buffer with external synchronization
buf := pool.NewUnsafeBuffer(1024)  // Fast but unsafe
var mu sync.Mutex

go func() {
    mu.Lock()
    buf.Write([]byte("goroutine 1"))  // ✅ Protected by mutex
    mu.Unlock()
}()
```

## Implementation Details

### Memory Layout

```text
Cache Line 1 (64 bytes) - Hot Path:
[data ptr][length][capacity][flags][spinlock][padding...]

Cache Line 2 (64 bytes) - Cold Path:
[origin ptr][pooled flag][padding...]
```

### Unsafe Operations

The package uses unsafe operations for maximum performance:

- Direct memory pointers
- Zero-copy string conversions
- Pointer arithmetic for writes
- Memory-mapped operations

### Spinlock vs Mutex

Safe buffers use custom spinlock instead of `sync.Mutex`:

- Lower overhead for short critical sections
- Better cache locality
- Exponential backoff for contention
- ~2-3x faster than mutex

## Compiler Optimizations

The package uses several compiler directives:

- `//go:nosplit` - Prevent stack splits for low-level functions
- `//go:noinline` - Prevent inlining when explicit control is needed
- Cache-line alignment for structs to avoid false sharing

## Testing

```bash
# Run all tests
bazel test //pkg/kernel/pool:test

# Run benchmarks
bazel run //pkg/kernel/pool:bench

# Run benchmarks in production mode (no safety checks)
go test -bench=. -benchmem -tags=unsafe_no_check

# Run with race detector (safe buffers only)
bazel test //pkg/kernel/pool:test --features=race
```

## Production Deployment

For maximum performance in production:

```bash
# Build with production mode (disables safety checks)
go build -tags=unsafe_no_check

# Or with Bazel
bazel build --define=production=true //your/target
```

**⚠️ WARNING**: Production mode disables goroutine safety checks. Ensure your
code is properly tested before deploying with `unsafe_no_check` tag.

## License

Part of Kitsunium SDK - Kernel packages for maximum performance.

## Contributing

When contributing, ensure:

1. Zero allocations for all operations
2. Maintain thread-safety guarantees
3. Document unsafe operations clearly
4. Add benchmarks for new features
5. Test with race detector enabled
