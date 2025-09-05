# Safe/Unsafe Implementation Pattern

## Purpose

Provide both thread-safe and high-performance implementations with clear performance tradeoffs.

## When to Use

- Implementing kernel packages where performance is critical
- When benchmarks show >30% improvement with unsafe version
- For hot-path code in performance-critical systems

## The 30% Rule

**Only create unsafe versions when performance gain ≥30%**

```bash
# Measure performance difference
go test -bench="(Safe|Unsafe)" -benchmem | benchstat -
```

## Implementation Pattern

### 1. Always Start with Safe Version

```go
// widget.go

type Widget struct {
    mu   sync.Mutex  // Thread safety
    data []byte
    size int
}

// NewWidget creates a thread-safe widget.
// Safe for concurrent use.
func NewWidget(size int) *Widget {
    return &Widget{
        data: make([]byte, size),
        size: size,
    }
}

func (b *Widget) Read(p []byte) (int, error) {
    b.mu.Lock()
    defer b.mu.Unlock()
    return b.readImpl(p)
}

// Shared implementation logic
func (b *Widget) readImpl(p []byte) (int, error) {
    n := copy(p, b.data[:b.size])
    return n, nil
}
```

### 2. Add Unsafe Version Only if Justified

```go
// unsafe_widget.go

type UnsafeWidget struct {
    data []byte
    size int
    // Concurrency checker for development
    checker *concurrencyChecker
}

// NewUnsafeWidget creates a high-performance non-thread-safe widget.
// WARNING: Not safe for concurrent use.
// Panics on concurrent access in development builds.
// Performance: ~40% faster than safe version.
func NewUnsafeWidget(size int) *UnsafeWidget {
    ub := &UnsafeWidget{
        data: make([]byte, size),
        size: size,
    }

    // Enable checker in development
    if !isProduction() {
        ub.checker = newConcurrencyChecker()
    }

    return ub
}

func (b *UnsafeWidget) Read(p []byte) (int, error) {
    if b.checker != nil {
        b.checker.check() // Panic on concurrent access
    }
    n := copy(p, b.data[:b.size])
    return n, nil
}
```

### 3. Concurrency Detection

```go
// concurrency_check.go
// +build !production

type concurrencyChecker struct {
    goroutineID uint64
}

func newConcurrencyChecker() *concurrencyChecker {
    return &concurrencyChecker{
        goroutineID: getCurrentGoroutineID(),
    }
}

func (c *concurrencyChecker) check() {
    current := getCurrentGoroutineID()
    if !atomic.CompareAndSwapUint64(&c.goroutineID, c.goroutineID, current) {
        panic("concurrent access detected on unsafe object")
    }
}
```

## Benchmark Requirements

### Mandatory Comparison Benchmarks

```go
func BenchmarkWidget_Read_Safe(b *testing.B) {
    buf := NewWidget(1024)
    p := make([]byte, 256)

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        buf.Read(p)
    }
}

func BenchmarkWidget_Read_Unsafe(b *testing.B) {
    buf := NewUnsafeWidget(1024)
    p := make([]byte, 256)

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        buf.Read(p)
    }
}

// Parallel benchmark for safe version
func BenchmarkWidget_Read_Safe_Parallel(b *testing.B) {
    buf := NewWidget(1024)

    b.RunParallel(func(pb *testing.PB) {
        p := make([]byte, 256)
        for pb.Next() {
            buf.Read(p)
        }
    })
}
```

## Decision Matrix

| Criteria         | Keep Safe Only | Add Unsafe Version |
| ---------------- | -------------- | ------------------ |
| Performance Gain | <30%           | ≥30%               |
| Allocations      | Acceptable     | Must be zero       |
| Complexity       | N/A            | Low to medium      |
| Usage Pattern    | Concurrent     | Single-threaded    |
| Maintenance Cost | Low            | Acceptable         |

## Do's and Don'ts

### Do's

- ✅ Always implement safe version first
- ✅ Benchmark before creating unsafe version
- ✅ Document performance gains clearly
- ✅ Add concurrency detection in development
- ✅ Share implementation logic between versions
- ✅ Test both versions thoroughly

### Don'ts

- ❌ Create unsafe version without benchmarking
- ❌ Skip concurrency detection
- ❌ Duplicate all logic (use shared impl methods)
- ❌ Forget to document thread-safety guarantees
- ❌ Use unsafe version in concurrent contexts

## Build Tags

```bash
# Development build (with checks)
go build

# Production build (without checks)
go build -tags production

# Benchmark mode
go test -bench=. -tags benchmark
```

## Related Documents

- [02-concurrency-detection.md](02-concurrency-detection.md) - Concurrency detection details
- [../03-testing/02-benchmarks.md](../03-testing/02-benchmarks.md) - Benchmark patterns
- [../01-architecture/01-interfaces.md](../01-architecture/01-interfaces.md) - Interface design
