package kcache

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

// ============= Cache Benchmarks =============

func BenchmarkCache_Set(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewCache(WithCapacity(size))
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
		})
	}
}

func BenchmarkCache_Get_Hit(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewCache(WithCapacity(size))
			// Pre-populate
			for i := 0; i < size; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cache.Get(strconv.Itoa(i % size))
			}
		})
	}
}

func BenchmarkCache_Get_Miss(b *testing.B) {
	cache := NewCache(WithCapacity(1000))
	// Pre-populate with different keys
	for i := 0; i < 1000; i++ {
		cache.Set(fmt.Sprintf("key%d", i), i)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Get(fmt.Sprintf("miss%d", i))
	}
}

func BenchmarkCache_TTL_Set(b *testing.B) {
	capacities := []int{100, 1000, 10000}
	for _, capacity := range capacities {
		b.Run(fmt.Sprintf("capacity=%d", capacity), func(b *testing.B) {
			cache := NewCache(WithCapacity(capacity))
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
		})
	}
}

func BenchmarkCache_Delete(b *testing.B) {
	cache := NewCache(WithCapacity(10000))
	// Pre-populate
	for i := 0; i < 10000; i++ {
		cache.Set(strconv.Itoa(i), i)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Delete(strconv.Itoa(i % 10000))
		if i%10000 == 9999 {
			// Repopulate when all deleted
			for j := 0; j < 10000; j++ {
				cache.Set(strconv.Itoa(j), j)
			}
		}
	}
}

// ============= Sharded Cache Benchmarks =============

func BenchmarkSharded_Set(b *testing.B) {
	shardCounts := []int{4, 8, 16, 32}
	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("shards=%d", shards), func(b *testing.B) {
			cache := NewUnsafeShardedCache(10000, shards)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
		})
	}
}

func BenchmarkSharded_Get_Hit(b *testing.B) {
	cache := NewUnsafeShardedCache(10000, 16)
	// Pre-populate
	for i := 0; i < 10000; i++ {
		cache.Set(strconv.Itoa(i), i)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i % 10000))
	}
}

func BenchmarkSharded_Parallel_Mixed(b *testing.B) {
	cache := NewUnsafeShardedCache(10000, 16)
	// Pre-populate
	for i := 0; i < 5000; i++ {
		cache.Set(strconv.Itoa(i), i)
	}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%3 == 0 {
				cache.Set(strconv.Itoa(i), i)
			} else {
				cache.Get(strconv.Itoa(i % 5000))
			}
			i++
		}
	})
}

// ============= Standard Map Benchmarks (for comparison) =============

func BenchmarkMap_Set(b *testing.B) {
	m := make(map[string]interface{})
	var mu sync.RWMutex
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m[strconv.Itoa(i)] = i
		mu.Unlock()
	}
}

func BenchmarkMap_Get_Hit(b *testing.B) {
	m := make(map[string]interface{})
	var mu sync.RWMutex
	// Pre-populate
	for i := 0; i < 1000; i++ {
		m[strconv.Itoa(i)] = i
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		_ = m[strconv.Itoa(i%1000)]
		mu.RUnlock()
	}
}

// ============= Complex Operations Benchmarks =============

func BenchmarkEviction_Cache(b *testing.B) {
	cache := NewCache(WithCapacity(100))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// This will cause evictions after 100 entries
		cache.Set(strconv.Itoa(i), i)
	}
}

func BenchmarkEviction_Sharded(b *testing.B) {
	cache := NewUnsafeShardedCache(100, 16)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// This will cause evictions after 100 entries
		cache.Set(strconv.Itoa(i), i)
	}
}

// ============= Parallel Benchmarks =============

func BenchmarkCache_Parallel_Set(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewCache(WithCapacity(size), WithShards(16))
			b.ResetTimer()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					cache.Set(strconv.Itoa(i), i)
					i++
				}
			})
		})
	}
}

func BenchmarkCache_Parallel_Get(b *testing.B) {
	cache := NewCache(WithCapacity(10000), WithShards(16))
	// Pre-populate
	for i := 0; i < 10000; i++ {
		cache.Set(strconv.Itoa(i), i)
	}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Get(strconv.Itoa(i % 10000))
			i++
		}
	})
}

// ============= Memory Benchmarks =============

func BenchmarkCacheMemoryUsage(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			var m1, m2 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)

			cache := NewCache(WithCapacity(size))
			for i := 0; i < size; i++ {
				cache.Set(strconv.Itoa(i), i)
			}

			runtime.GC()
			runtime.ReadMemStats(&m2)

			bytesPerEntry := float64(m2.Alloc-m1.Alloc) / float64(size)
			b.ReportMetric(bytesPerEntry, "bytes/entry")
		})
	}
}

// ============= Contention Benchmarks =============

func BenchmarkCacheHighContention(b *testing.B) {
	cache := NewUnsafeShardedCache(1000, 32)
	// Pre-populate with hot keys
	for i := 0; i < 100; i++ {
		cache.Set(strconv.Itoa(i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 90% reads on hot keys, 10% writes
			for i := 0; i < 9; i++ {
				cache.Get(strconv.Itoa(i % 100))
			}
			cache.Set(strconv.Itoa(100), 100)
		}
	})
}

func BenchmarkLowContention(b *testing.B) {
	cache := NewUnsafeShardedCache(100000, 32)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Each goroutine works on different keys
			key := strconv.Itoa(i*runtime.NumCPU() + runtime.NumCPU())
			if i%2 == 0 {
				cache.Set(key, i)
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}
