# Benchmarking for Kernel Packages

## Purpose

Create comprehensive performance benchmarks that validate optimization claims, compare safe vs unsafe implementations, and ensure performance regressions are detected.

## When to Use

- Measuring performance characteristics
- Comparing implementation strategies
- Validating optimization effectiveness
- Detecting performance regressions
- Profiling memory allocations
- Measuring concurrency scalability

## Benchmark Structure

### Basic Benchmark Pattern

```go
// foo_bench_test.go
package foo

import (
    "testing"
)

func BenchmarkWidget_Write(b *testing.B) {
    // Setup phase - not timed
    buf := NewWidget(1024 * 1024)
    data := make([]byte, 1024)

    // Reset timer after setup
    b.ResetTimer()

    // Benchmark loop
    for i := 0; i < b.N; i++ {
        buf.Write(data)

        // Reset state if needed
        if buf.Len() > 1024*1024-1024 {
            b.StopTimer()
            buf.Reset()
            b.StartTimer()
        }
    }

    // Report metrics
    b.ReportAllocs()
    b.SetBytes(int64(len(data)))
}
```

### Safe vs Unsafe Comparison

```go
func BenchmarkSafeVsUnsafe(b *testing.B) {
    sizes := []int{64, 256, 1024, 4096, 65536}

    for _, size := range sizes {
        data := make([]byte, size)

        b.Run(fmt.Sprintf("Safe_%d", size), func(b *testing.B) {
            buf := NewSafeWidget(size * 2)
            b.ResetTimer()
            b.ReportAllocs()
            b.SetBytes(int64(size))

            for i := 0; i < b.N; i++ {
                buf.Write(data)
                buf.Reset()
            }
        })

        b.Run(fmt.Sprintf("Unsafe_%d", size), func(b *testing.B) {
            buf := NewUnsafeWidget(size * 2)
            b.ResetTimer()
            b.ReportAllocs()
            b.SetBytes(int64(size))

            for i := 0; i < b.N; i++ {
                buf.Write(data)
                buf.Reset()
            }
        })
    }
}

// Helper to calculate improvement
func calculateImprovement(safeBench, unsafeBench testing.BenchmarkResult) float64 {
    safeNs := float64(safeBench.NsPerOp())
    unsafeNs := float64(unsafeBench.NsPerOp())
    return (safeNs - unsafeNs) / safeNs * 100
}
```

### Allocation Benchmarks

```go
func BenchmarkAllocations(b *testing.B) {
    b.Run("WithManager", func(b *testing.B) {
        manager := NewWidgetManager()
        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
            buf := manager.Get(1024)
            buf.Write([]byte("test"))
            manager.Put(buf)
        }
    })

    b.Run("WithoutManager", func(b *testing.B) {
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
            buf := NewWidget(1024)
            buf.Write([]byte("test"))
        }
    })

    b.Run("ZeroAlloc", func(b *testing.B) {
        buf := NewWidget(1024)
        data := []byte("test")
        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
            buf.Write(data)
            buf.Reset()
        }

        // Verify zero allocations
        if b.AllocsPerOp() > 0 {
            b.Errorf("expected zero allocations, got %d", b.AllocsPerOp())
        }
    })
}
```

## Concurrency Benchmarks

### Parallel Scaling

```go
func BenchmarkConcurrentScaling(b *testing.B) {
    for _, goroutines := range []int{1, 2, 4, 8, 16, 32} {
        b.Run(fmt.Sprintf("Goroutines_%d", goroutines), func(b *testing.B) {
            buf := NewShardedWidget(1024 * 1024)
            data := make([]byte, 1024)

            b.SetParallelism(goroutines)
            b.ResetTimer()

            b.RunParallel(func(pb *testing.PB) {
                localData := make([]byte, 1024)
                for pb.Next() {
                    buf.Write(localData)
                }
            })

            // Report ops/sec per goroutine
            opsPerSec := float64(b.N) / b.Elapsed().Seconds()
            b.ReportMetric(opsPerSec/float64(goroutines), "ops/sec/goroutine")
        })
    }
}

func BenchmarkContention(b *testing.B) {
    b.Run("HighContention", func(b *testing.B) {
        // Single shared resource
        buf := NewSafeWidget(1024)

        b.RunParallel(func(pb *testing.PB) {
            data := []byte("x")
            for pb.Next() {
                buf.Write(data)
            }
        })
    })

    b.Run("LowContention", func(b *testing.B) {
        // Sharded resources
        bufs := make([]*SafeWidget, runtime.NumCPU())
        for i := range bufs {
            bufs[i] = NewSafeWidget(1024)
        }

        var counter uint32
        b.RunParallel(func(pb *testing.PB) {
            // Round-robin selection
            idx := atomic.AddUint32(&counter, 1) % uint32(len(bufs))
            buf := bufs[idx]
            data := []byte("x")

            for pb.Next() {
                buf.Write(data)
            }
        })
    })
}
```

## Memory and Cache Benchmarks

### Cache Efficiency

```go
func BenchmarkCacheEfficiency(b *testing.B) {
    b.Run("Sequential", func(b *testing.B) {
        data := make([]byte, 1<<20) // 1MB
        b.ResetTimer()

        for i := 0; i < b.N; i++ {
            sum := byte(0)
            // Sequential access - cache friendly
            for j := 0; j < len(data); j++ {
                sum += data[j]
            }
            _ = sum
        }
    })

    b.Run("Random", func(b *testing.B) {
        data := make([]byte, 1<<20) // 1MB
        indices := make([]int, 1024)
        for i := range indices {
            indices[i] = rand.Intn(len(data))
        }
        b.ResetTimer()

        for i := 0; i < b.N; i++ {
            sum := byte(0)
            // Random access - cache unfriendly
            for _, idx := range indices {
                sum += data[idx]
            }
            _ = sum
        }
    })

    b.Run("Strided", func(b *testing.B) {
        data := make([]byte, 1<<20) // 1MB
        stride := 64 // Cache line size
        b.ResetTimer()

        for i := 0; i < b.N; i++ {
            sum := byte(0)
            // Strided access
            for j := 0; j < len(data); j += stride {
                sum += data[j]
            }
            _ = sum
        }
    })
}
```

### Memory Layout Impact

```go
func BenchmarkStructLayout(b *testing.B) {
    // Poorly aligned struct
    type BadLayout struct {
        a bool    // 1 byte + 7 padding
        b int64   // 8 bytes
        c bool    // 1 byte + 7 padding
        d int64   // 8 bytes
    } // Total: 32 bytes

    // Well aligned struct
    type GoodLayout struct {
        b int64   // 8 bytes
        d int64   // 8 bytes
        a bool    // 1 byte
        c bool    // 1 byte + 6 padding
    } // Total: 24 bytes

    b.Run("BadLayout", func(b *testing.B) {
        s := &BadLayout{}
        b.ResetTimer()

        for i := 0; i < b.N; i++ {
            s.b = int64(i)
            s.d = s.b * 2
            _ = s.d
        }
    })

    b.Run("GoodLayout", func(b *testing.B) {
        s := &GoodLayout{}
        b.ResetTimer()

        for i := 0; i < b.N; i++ {
            s.b = int64(i)
            s.d = s.b * 2
            _ = s.d
        }
    })
}
```

## Comparative Benchmarks

### Against Standard Library

```go
func BenchmarkVsStdlib(b *testing.B) {
    data := make([]byte, 1024)

    b.Run("KWidget", func(b *testing.B) {
        buf := NewUnsafeWidget(4096)
        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
            buf.Write(data)
            buf.Reset()
        }
    })

    b.Run("bytes.Widget", func(b *testing.B) {
        buf := bytes.NewWidget(make([]byte, 0, 4096))
        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
            buf.Write(data)
            buf.Reset()
        }
    })

    b.Run("strings.Builder", func(b *testing.B) {
        var sb strings.Builder
        sb.Grow(4096)
        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
            sb.Write(data)
            sb.Reset()
        }
    })
}
```

## Benchmark Analysis

### Performance Validation

```go
func BenchmarkPerformanceRequirements(b *testing.B) {
    b.Run("MeetsLatencyTarget", func(b *testing.B) {
        targetNs := int64(100) // 100ns target
        buf := NewUnsafeWidget(1024)
        data := []byte("test")

        b.ResetTimer()

        for i := 0; i < b.N; i++ {
            start := time.Now()
            buf.Write(data)
            elapsed := time.Since(start).Nanoseconds()

            if elapsed > targetNs {
                b.Errorf("operation took %dns, exceeds %dns target",
                    elapsed, targetNs)
            }
            buf.Reset()
        }
    })

    b.Run("MeetsThroughputTarget", func(b *testing.B) {
        targetMBps := 1000.0 // 1GB/s target
        buf := NewUnsafeWidget(1<<20)
        data := make([]byte, 1024)

        b.ResetTimer()
        b.SetBytes(int64(len(data)))

        for i := 0; i < b.N; i++ {
            buf.Write(data)
            if buf.Len() > 1<<20-1024 {
                buf.Reset()
            }
        }

        mbps := float64(b.Bytes()) / 1e6 / b.Elapsed().Seconds()
        if mbps < targetMBps {
            b.Errorf("throughput %.2f MB/s below %.2f MB/s target",
                mbps, targetMBps)
        }
    })
}
```

## Profiling Integration

### CPU Profiling

```go
func BenchmarkWithProfile(b *testing.B) {
    // Run with: go test -bench=BenchmarkWithProfile -cpuprofile=cpu.prof
    // Analyze: go tool pprof cpu.prof

    buf := NewWidget(1<<20)
    data := make([]byte, 4096)

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        processData(buf, data)
    }
}

// Memory profiling
func BenchmarkMemoryProfile(b *testing.B) {
    // Run with: go test -bench=BenchmarkMemoryProfile -memprofile=mem.prof
    // Analyze: go tool pprof -alloc_space mem.prof

    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        buf := NewWidget(1024)
        buf.Write(make([]byte, 512))
        _ = buf
    }
}
```

## Benchmark Utilities

### Custom Metrics

```go
func BenchmarkCustomMetrics(b *testing.B) {
    buf := NewShardedWidget(1<<20)
    data := make([]byte, 1024)

    var totalBytes int64
    var maxLatency time.Duration

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        start := time.Now()
        n, _ := buf.Write(data)
        latency := time.Since(start)

        totalBytes += int64(n)
        if latency > maxLatency {
            maxLatency = latency
        }
    }

    // Report custom metrics
    b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
    b.ReportMetric(float64(maxLatency.Nanoseconds()), "ns/max-latency")
    b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
}
```

### Regression Detection

```go
// baseline_test.go
var baselineResults = map[string]float64{
    "Widget_Write": 50.0,  // ns/op
    "Widget_Read":  45.0,  // ns/op
    "Manager_Get":     25.0,  // ns/op
}

func TestPerformanceRegression(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping regression test in short mode")
    }

    results := testing.Benchmark(BenchmarkWidget_Write)
    nsPerOp := float64(results.NsPerOp())

    baseline := baselineResults["Widget_Write"]
    threshold := baseline * 1.1 // Allow 10% regression

    if nsPerOp > threshold {
        t.Errorf("performance regression: %.2fns/op exceeds %.2fns threshold",
            nsPerOp, threshold)
    }
}
```

## Do's and Don'ts

### Do's

- ✅ Reset timer after setup
- ✅ Report allocations with ReportAllocs()
- ✅ Use SetBytes() for throughput metrics
- ✅ Run with multiple input sizes
- ✅ Compare safe vs unsafe implementations
- ✅ Use sub-benchmarks for organization
- ✅ Validate performance requirements
- ✅ Profile CPU and memory usage

### Don'ts

- ❌ Don't include setup in timing
- ❌ Don't modify b.N
- ❌ Don't use fmt.Print in benchmarks
- ❌ Don't ignore benchmark results
- ❌ Don't benchmark with tiny datasets only
- ❌ Don't forget to reset state between iterations

## Benchmark Commands

```bash
# Run all benchmarks
go test -bench=.

# Run specific benchmark
go test -bench=BenchmarkWidget_Write

# Run with memory stats
go test -bench=. -benchmem

# Run for specific duration
go test -bench=. -benchtime=10s

# Run with specific count
go test -bench=. -count=10

# Compare benchmarks
go test -bench=. > new.txt
benchcmp old.txt new.txt

# With CPU profiling
go test -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof

# With memory profiling
go test -bench=. -memprofile=mem.prof
go tool pprof -alloc_space mem.prof

# With trace
go test -bench=. -trace=trace.out
go tool trace trace.out
```

## Related Documents

- [01-unit-tests.md](01-unit-tests.md) - Testing patterns
- [03-integration-tests.md](03-integration-tests.md) - Integration benchmarks
- [../02-implementation/01-safe-unsafe-pattern.md](../02-implementation/01-safe-unsafe-pattern.md) - Performance comparison
- [../05-commands/02-validation.md](../05-commands/02-validation.md) - Performance validation
