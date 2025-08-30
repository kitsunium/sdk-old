// Package kcache provides high-performance caching implementations.
// This file contains comprehensive benchmarks for all cache implementations.
package kcache

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
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

// BenchmarkSet_SingleCore benchmarks single-threaded set operations
func BenchmarkSet_SingleCore(b *testing.B) {
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

// BenchmarkGet_SingleCore benchmarks single-threaded get operations
func BenchmarkGet_SingleCore(b *testing.B) {
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

// BenchmarkDelete_SingleCore benchmarks single-threaded delete operations
func BenchmarkDelete_SingleCore(b *testing.B) {
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

// Multi-threaded benchmarks moved to kcache_bench_multi_test.go

// Contention benchmarks moved to kcache_bench_multi_test.go

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
		cache := NewSafeShardedCache(100000, 32) // Use safe version for parallel

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
		cache := NewSafeShardedCache(100000, 32) // Use safe version for parallel
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
		cache := NewSafeShardedCache(100000, 32) // Use safe version for parallel

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

// BenchmarkScalability removed - takes too long for regular runs
// Use specialized performance testing tools for scalability analysis

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
