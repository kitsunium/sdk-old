// Package kcache provides high-performance caching implementations.
// This file contains comprehensive benchmarks for all cache implementations.
package kcache

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

// Production mode flag for Bazel benchmarks
var productionMode = false

func init() {
	// Check if we're running in production mode (for Bazel)
	// This can be set via build tags or environment variables
	if runtime.GOMAXPROCS(0) > 0 {
		productionMode = true
	}
}

// Benchmark test data sizes
const (
	benchSmallSize  = 100
	benchMediumSize = 10000
	benchLargeSize  = 100000
)

// generateBenchData generates test data for benchmarks
func generateBenchData(size int) ([]interface{}, []interface{}) {
	keys := make([]interface{}, size)
	values := make([]interface{}, size)
	for i := 0; i < size; i++ {
		keys[i] = fmt.Sprintf("benchmark-key-%d", i)
		values[i] = fmt.Sprintf("benchmark-value-%d", i)
	}
	return keys, values
}

// =============================================================================
// Cache Creation Benchmarks
// =============================================================================

// BenchmarkCacheCreation benchmarks cache creation for different types
func BenchmarkCacheCreation(b *testing.B) {
	b.Run("SafeCache", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewSafeCache(10000)
		}
	})

	b.Run("UnsafeCache", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewUnsafeCache(10000)
		}
	})

	b.Run("SafeShardedCache", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewSafeShardedCache(10000, 16)
		}
	})

	b.Run("UnsafeShardedCache", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewUnsafeShardedCache(10000, 16)
		}
	})
}

// =============================================================================
// Single-threaded Benchmarks
// =============================================================================

// BenchmarkSingleThreadedSet benchmarks single-threaded set operations
func BenchmarkSingleThreadedSet(b *testing.B) {
	sizes := []int{benchSmallSize, benchMediumSize, benchLargeSize}
	cacheTypes := []struct {
		name   string
		create func() Cache
	}{
		{"SafeCache", func() Cache { return NewSafeCache(benchLargeSize) }},
		{"UnsafeCache", func() Cache { return NewUnsafeCache(benchLargeSize) }},
		{"SafeSharded", func() Cache { return NewSafeShardedCache(benchLargeSize, 16) }},
		{"UnsafeSharded", func() Cache { return NewUnsafeShardedCache(benchLargeSize, 16) }},
	}

	for _, ct := range cacheTypes {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("%s/%d", ct.name, size), func(b *testing.B) {
				cache := ct.create()
				keys, values := generateBenchData(size)

				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for j := 0; j < size; j++ {
						cache.Set(keys[j], values[j])
					}
				}
			})
		}
	}
}

// BenchmarkSingleThreadedGet benchmarks single-threaded get operations
func BenchmarkSingleThreadedGet(b *testing.B) {
	sizes := []int{benchSmallSize, benchMediumSize}
	cacheTypes := []struct {
		name   string
		create func() Cache
	}{
		{"SafeCache", func() Cache { return NewSafeCache(benchLargeSize) }},
		{"UnsafeCache", func() Cache { return NewUnsafeCache(benchLargeSize) }},
		{"SafeSharded", func() Cache { return NewSafeShardedCache(benchLargeSize, 16) }},
		{"UnsafeSharded", func() Cache { return NewUnsafeShardedCache(benchLargeSize, 16) }},
	}

	for _, ct := range cacheTypes {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("%s/%d", ct.name, size), func(b *testing.B) {
				cache := ct.create()
				keys, values := generateBenchData(size)

				// Pre-populate cache
				for i := 0; i < size; i++ {
					cache.Set(keys[i], values[i])
				}

				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for j := 0; j < size; j++ {
						cache.Get(keys[j])
					}
				}
			})
		}
	}
}

// BenchmarkSingleThreadedDelete benchmarks single-threaded delete operations
func BenchmarkSingleThreadedDelete(b *testing.B) {
	sizes := []int{benchSmallSize, benchMediumSize}
	cacheTypes := []struct {
		name   string
		create func() Cache
	}{
		{"SafeCache", func() Cache { return NewSafeCache(benchLargeSize) }},
		{"UnsafeCache", func() Cache { return NewUnsafeCache(benchLargeSize) }},
		{"SafeSharded", func() Cache { return NewSafeShardedCache(benchLargeSize, 16) }},
		{"UnsafeSharded", func() Cache { return NewUnsafeShardedCache(benchLargeSize, 16) }},
	}

	for _, ct := range cacheTypes {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("%s/%d", ct.name, size), func(b *testing.B) {
				keys, values := generateBenchData(size)

				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					cache := ct.create()
					// Pre-populate cache
					for j := 0; j < size; j++ {
						cache.Set(keys[j], values[j])
					}
					b.StartTimer()

					for j := 0; j < size; j++ {
						cache.Delete(keys[j])
					}
				}
			})
		}
	}
}

// =============================================================================
// Multi-threaded Benchmarks
// =============================================================================

// BenchmarkMultiThreadedSet benchmarks concurrent set operations
func BenchmarkMultiThreadedSet(b *testing.B) {
	cacheTypes := []struct {
		name   string
		create func() Cache
	}{
		{"SafeCache", func() Cache { return NewSafeCache(benchLargeSize) }},
		{"SafeSharded", func() Cache { return NewSafeShardedCache(benchLargeSize, 16) }},
	}

	parallelisms := []int{2, 4, 8, 16}

	for _, ct := range cacheTypes {
		for _, p := range parallelisms {
			b.Run(fmt.Sprintf("%s/p%d", ct.name, p), func(b *testing.B) {
				cache := ct.create()

				b.SetParallelism(p)
				b.ResetTimer()
				b.ReportAllocs()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						cache.Set(fmt.Sprintf("key-%d", i), i)
						i++
					}
				})
			})
		}
	}
}

// BenchmarkMultiThreadedGet benchmarks concurrent get operations
func BenchmarkMultiThreadedGet(b *testing.B) {
	cacheTypes := []struct {
		name   string
		create func() Cache
	}{
		{"SafeCache", func() Cache { return NewSafeCache(benchLargeSize) }},
		{"SafeSharded", func() Cache { return NewSafeShardedCache(benchLargeSize, 16) }},
	}

	parallelisms := []int{2, 4, 8, 16}

	for _, ct := range cacheTypes {
		for _, p := range parallelisms {
			b.Run(fmt.Sprintf("%s/p%d", ct.name, p), func(b *testing.B) {
				cache := ct.create()
				keys, values := generateBenchData(benchMediumSize)

				// Pre-populate cache
				for i := 0; i < benchMediumSize; i++ {
					cache.Set(keys[i], values[i])
				}

				b.SetParallelism(p)
				b.ResetTimer()
				b.ReportAllocs()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						cache.Get(keys[i%benchMediumSize])
						i++
					}
				})
			})
		}
	}
}

// BenchmarkMultiThreadedMixed benchmarks mixed concurrent operations
func BenchmarkMultiThreadedMixed(b *testing.B) {
	cacheTypes := []struct {
		name   string
		create func() Cache
	}{
		{"SafeCache", func() Cache { return NewSafeCache(benchLargeSize) }},
		{"SafeSharded", func() Cache { return NewSafeShardedCache(benchLargeSize, 16) }},
	}

	for _, ct := range cacheTypes {
		b.Run(ct.name, func(b *testing.B) {
			cache := ct.create()

			b.ResetTimer()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("key-%d", i)
					switch i % 4 {
					case 0:
						cache.Set(key, i)
					case 1:
						cache.Get(key)
					case 2:
						cache.Has(key)
					case 3:
						cache.Delete(key)
					}
					i++
				}
			})
		})
	}
}

// =============================================================================
// Contention Benchmarks
// =============================================================================

// BenchmarkHighContention benchmarks cache performance under high contention
func BenchmarkHighContention(b *testing.B) {
	cacheTypes := []struct {
		name   string
		create func() Cache
	}{
		{"SafeCache", func() Cache { return NewSafeCache(100) }},
		{"SafeSharded", func() Cache { return NewSafeShardedCache(100, 16) }},
	}

	for _, ct := range cacheTypes {
		b.Run(ct.name, func(b *testing.B) {
			cache := ct.create()
			// Use only 10 keys to create high contention
			keys := []string{"k0", "k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9"}

			b.ResetTimer()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := keys[i%10]
					cache.Set(key, i)
					cache.Get(key)
					i++
				}
			})
		})
	}
}

// =============================================================================
// Batch Operation Benchmarks
// =============================================================================

// BenchmarkBatchOperations benchmarks batch operations
func BenchmarkBatchOperations(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("BatchSet/%d", size), func(b *testing.B) {
			cache := NewSafeShardedCache(benchLargeSize, 16)
			keys, values := generateBenchData(size)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				OptimizedSetBatch(cache, keys, values)
			}
		})

		b.Run(fmt.Sprintf("BatchGet/%d", size), func(b *testing.B) {
			cache := NewSafeShardedCache(benchLargeSize, 16)
			keys, values := generateBenchData(size)
			OptimizedSetBatch(cache, keys, values)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				OptimizedGetBatch(cache, keys)
			}
		})

		b.Run(fmt.Sprintf("BatchDelete/%d", size), func(b *testing.B) {
			keys, values := generateBenchData(size)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				cache := NewSafeShardedCache(benchLargeSize, 16)
				OptimizedSetBatch(cache, keys, values)
				b.StartTimer()
				OptimizedDeleteBatch(cache, keys)
			}
		})
	}
}

// =============================================================================
// Memory and Allocation Benchmarks
// =============================================================================

// BenchmarkMemoryUsage benchmarks memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("SetNoAlloc", func(b *testing.B) {
		cache := NewUnsafeCache(10000)
		key := "static-key"
		value := "static-value"

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			cache.Set(key, value)
		}
	})

	b.Run("GetNoAlloc", func(b *testing.B) {
		cache := NewUnsafeCache(10000)
		key := "static-key"
		cache.Set(key, "static-value")

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			cache.Get(key)
		}
	})

	b.Run("HasNoAlloc", func(b *testing.B) {
		cache := NewUnsafeCache(10000)
		key := "static-key"
		cache.Set(key, "static-value")

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			cache.Has(key)
		}
	})
}

// =============================================================================
// Hasher Benchmarks
// =============================================================================

// BenchmarkHasherImplementations benchmarks different hasher implementations
func BenchmarkHasherImplementations(b *testing.B) {
	// Use the FNV hasher that's available in the package
	hasher := newFNVHasher()

	testKeys := []interface{}{
		"short",
		"medium-length-key",
		"very-long-key-that-simulates-real-world-usage-patterns-with-many-characters",
		int64(12345),
		[]byte("byte-slice-key"),
	}

	for _, key := range testKeys {
		keyStr := fmt.Sprintf("%T", key)
		b.Run(fmt.Sprintf("FNV/%s", keyStr), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				hasher.Hash(key)
			}
		})
	}
}

// =============================================================================
// Production Mode Benchmarks (for Bazel)
// =============================================================================

// BenchmarkProductionMode runs production-optimized benchmarks
func BenchmarkProductionMode(b *testing.B) {
	if !productionMode {
		b.Skip("Skipping production mode benchmarks")
	}

	b.Run("ProdSet", func(b *testing.B) {
		cache := NewUnsafeShardedCache(100000, 32)

		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			var i int64
			for pb.Next() {
				atomic.AddInt64(&i, 1)
				cache.Set(i, i)
			}
		})
	})

	b.Run("ProdGet", func(b *testing.B) {
		cache := NewUnsafeShardedCache(100000, 32)
		// Pre-populate
		for i := 0; i < 10000; i++ {
			cache.Set(i, i)
		}

		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			var i int64
			for pb.Next() {
				cache.Get(atomic.AddInt64(&i, 1) % 10000)
			}
		})
	})

	b.Run("ProdMixed", func(b *testing.B) {
		cache := NewUnsafeShardedCache(100000, 32)

		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			var i int64
			for pb.Next() {
				n := atomic.AddInt64(&i, 1)
				switch n % 10 {
				case 0, 1, 2, 3: // 40% writes
					cache.Set(n, n)
				case 4, 5, 6, 7, 8: // 50% reads
					cache.Get(n % 10000)
				case 9: // 10% deletes
					cache.Delete(n % 10000)
				}
			}
		})
	})
}

// =============================================================================
// Scalability Benchmarks
// =============================================================================

// BenchmarkScalability measures cache scalability with increasing cores
func BenchmarkScalability(b *testing.B) {
	maxProcs := runtime.GOMAXPROCS(0)
	if maxProcs < 2 {
		b.Skip("Scalability benchmark requires at least 2 cores")
	}

	for p := 1; p <= maxProcs; p *= 2 {
		b.Run(fmt.Sprintf("Cores%d", p), func(b *testing.B) {
			oldProcs := runtime.GOMAXPROCS(p)
			defer runtime.GOMAXPROCS(oldProcs)

			cache := NewSafeShardedCache(100000, 32)
			// Pre-populate
			for i := 0; i < 10000; i++ {
				cache.Set(i, i)
			}

			var ops int64
			b.ResetTimer()

			start := time.Now()
			var wg sync.WaitGroup
			for i := 0; i < p; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for time.Since(start) < time.Second {
						cache.Get(int(atomic.AddInt64(&ops, 1)) % 10000)
					}
				}()
			}
			wg.Wait()

			b.ReportMetric(float64(ops)/time.Since(start).Seconds(), "ops/s")
			b.ReportMetric(float64(ops)/time.Since(start).Seconds()/float64(p), "ops/s/core")
		})
	}
}

// =============================================================================
// Cache Miss Benchmarks
// =============================================================================

// BenchmarkCacheMiss benchmarks cache miss scenarios
func BenchmarkCacheMiss(b *testing.B) {
	cacheTypes := []struct {
		name   string
		create func() Cache
	}{
		{"SafeCache", func() Cache { return NewSafeCache(1000) }},
		{"UnsafeCache", func() Cache { return NewUnsafeCache(1000) }},
		{"SafeSharded", func() Cache { return NewSafeShardedCache(1000, 16) }},
		{"UnsafeSharded", func() Cache { return NewUnsafeShardedCache(1000, 16) }},
	}

	for _, ct := range cacheTypes {
		b.Run(ct.name, func(b *testing.B) {
			cache := ct.create()

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				// Always miss - key doesn't exist
				cache.Get(fmt.Sprintf("missing-key-%d", i))
			}
		})
	}
}

// =============================================================================
// Resize Benchmarks
// =============================================================================

// BenchmarkCacheResize benchmarks cache resize operations
func BenchmarkCacheResize(b *testing.B) {
	b.Run("SafeCache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			cache := NewSafeCache(16) // Start small to trigger resize
			b.StartTimer()

			// Add enough items to trigger multiple resizes
			for j := 0; j < 1000; j++ {
				cache.Set(j, j)
			}
		}
	})

	b.Run("UnsafeCache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			cache := NewUnsafeCache(16) // Start small to trigger resize
			b.StartTimer()

			// Add enough items to trigger multiple resizes
			for j := 0; j < 1000; j++ {
				cache.Set(j, j)
			}
		}
	})
}

// =============================================================================
// Unsafe Optimization Benchmarks
// =============================================================================

// BenchmarkUnsafeOptimizations benchmarks unsafe optimizations
func BenchmarkUnsafeOptimizations(b *testing.B) {
	b.Run("StringToBytes", func(b *testing.B) {
		s := "benchmark-string-for-conversion"

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Unsafe string to bytes conversion
			b := *(*[]byte)(unsafe.Pointer(&s))
			_ = b
		}
	})

	b.Run("BytesToString", func(b *testing.B) {
		bytes := []byte("benchmark-bytes-for-conversion")

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Unsafe bytes to string conversion
			s := *(*string)(unsafe.Pointer(&bytes))
			_ = s
		}
	})

	b.Run("DirectMemoryAccess", func(b *testing.B) {
		type testStruct struct {
			a int64
			b int64
			c int64
		}

		s := &testStruct{1, 2, 3}

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Direct memory access via unsafe
			ptr := unsafe.Pointer(s)
			*(*int64)(ptr) = int64(i)
		}
	})
}
