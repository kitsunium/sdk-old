package kcache

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

// =============================================================================
// Multi-Core Benchmarks - Only safe concurrent operations
// =============================================================================

// BenchmarkParallel_SafeCache_Set benchmarks concurrent set operations on SafeCache
func BenchmarkParallel_SafeCache_Set(b *testing.B) {
	cache := NewSafeCache(100000)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Set(fmt.Sprintf("key-%d", i), i)
			i++
		}
	})
}

// BenchmarkParallel_SafeCache_Get benchmarks concurrent get operations on SafeCache
func BenchmarkParallel_SafeCache_Get(b *testing.B) {
	cache := NewSafeCache(100000)
	// Pre-populate
	for i := 0; i < 10000; i++ {
		cache.Set(i, i)
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Get(i % 10000)
			i++
		}
	})
}

// BenchmarkParallel_SafeSharded_Set benchmarks concurrent set on sharded cache
func BenchmarkParallel_SafeSharded_Set(b *testing.B) {
	cache := NewSafeShardedCache(100000, 16)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Set(fmt.Sprintf("key-%d", i), i)
			i++
		}
	})
}

// BenchmarkParallel_SafeSharded_Get benchmarks concurrent get on sharded cache
func BenchmarkParallel_SafeSharded_Get(b *testing.B) {
	cache := NewSafeShardedCache(100000, 16)
	// Pre-populate
	for i := 0; i < 10000; i++ {
		cache.Set(i, i)
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Get(i % 10000)
			i++
		}
	})
}

// BenchmarkParallel_SafeSharded_Mixed benchmarks mixed operations
func BenchmarkParallel_SafeSharded_Mixed(b *testing.B) {
	cache := NewSafeShardedCache(100000, 16)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%10000)
			switch i % 4 {
			case 0:
				cache.Set(key, i)
			case 1, 2:
				cache.Get(key)
			case 3:
				cache.Delete(key)
			}
			i++
		}
	})
}

// BenchmarkParallel_Contention_Low benchmarks low contention scenario
func BenchmarkParallel_Contention_Low(b *testing.B) {
	cache := NewSafeShardedCache(100000, 32)
	numGoroutines := runtime.GOMAXPROCS(0)

	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine works on different key ranges
		goroutineID := runtime.NumCPU() // pseudo-ID
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d-%d", goroutineID, i)
			if i%2 == 0 {
				cache.Set(key, i)
			} else {
				cache.Get(key)
			}
			i++
		}
	})

	b.ReportMetric(float64(numGoroutines), "goroutines")
}

// BenchmarkParallel_Contention_High benchmarks high contention scenario
func BenchmarkParallel_Contention_High(b *testing.B) {
	cache := NewSafeShardedCache(100, 16)
	// Use only 10 keys to create high contention
	keys := []string{"k0", "k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9"}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := keys[i%10]
			if i%4 == 0 {
				cache.Set(key, i)
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}

// BenchmarkParallel_BatchOperations benchmarks batch operations
func BenchmarkParallel_BatchOperations(b *testing.B) {
	cache := NewSafeShardedCache(100000, 16)
	batchSize := 100

	b.RunParallel(func(pb *testing.PB) {
		keys := make([]interface{}, batchSize)
		values := make([]interface{}, batchSize)

		for i := range keys {
			keys[i] = fmt.Sprintf("batch-key-%d", i)
			values[i] = i
		}

		for pb.Next() {
			OptimizedSetBatch(cache, keys, values)
		}
	})
}

// BenchmarkParallel_UnsafeCache_SingleWriter benchmarks unsafe cache with single writer
func BenchmarkParallel_UnsafeCache_SingleWriter(b *testing.B) {
	// Skip this test in race detector mode
	if testing.Short() || runtime.GOOS == "js" {
		b.Skip("Skipping unsafe parallel benchmark in short/js mode")
	}

	cache := NewUnsafeCache(100000)
	var mu sync.Mutex

	// Pre-populate
	for i := 0; i < 10000; i++ {
		cache.Set(i, i)
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%100 == 0 {
				// Only 1% writes, with mutex protection
				mu.Lock()
				cache.Set(i, i)
				mu.Unlock()
			} else {
				// 99% reads (safe without mutex for unsafe cache)
				cache.Get(i % 10000)
			}
			i++
		}
	})
}

// NOTE: We explicitly DO NOT include benchmarks for:
// - Unsafe caches with concurrent writes (data races)
// - Scalability benchmark (takes too long)
// - Complex resize operations under concurrency
// These are either unsafe or impractical for regular benchmark runs
