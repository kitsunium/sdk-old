package kcache

import (
	"fmt"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// ============= LRU Cache Benchmarks =============

func BenchmarkLRU_Set(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewLRU[string, int](size)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
		})
	}
}

func BenchmarkLRU_Get_Hit(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewLRU[string, int](size)
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

func BenchmarkLRU_Get_Miss(b *testing.B) {
	cache := NewLRU[string, int](1000)
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

func BenchmarkLRU_SetWithTTL(b *testing.B) {
	ttls := []time.Duration{0, time.Millisecond, time.Second, time.Hour}
	for _, ttl := range ttls {
		b.Run(fmt.Sprintf("ttl=%v", ttl), func(b *testing.B) {
			cache := NewLRU[string, int](10000)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cache.SetWithTTL(strconv.Itoa(i), i, ttl)
			}
		})
	}
}

func BenchmarkLRU_Get_Expired(b *testing.B) {
	cache := NewLRU[string, int](1000)
	// Pre-populate with expired entries
	for i := 0; i < 1000; i++ {
		cache.SetWithTTL(strconv.Itoa(i), i, time.Nanosecond)
	}
	time.Sleep(time.Microsecond)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i % 1000))
	}
}

func BenchmarkLRU_Delete(b *testing.B) {
	b.Run("existing", func(b *testing.B) {
		cache := NewLRU[string, int](10000)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			cache.Set(strconv.Itoa(i), i)
			b.StartTimer()
			cache.Delete(strconv.Itoa(i))
		}
	})

	b.Run("non-existing", func(b *testing.B) {
		cache := NewLRU[string, int](10000)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			cache.Delete(strconv.Itoa(i))
		}
	})
}

func BenchmarkLRU_Has(b *testing.B) {
	cache := NewLRU[string, int](10000)
	for i := 0; i < 5000; i++ {
		cache.Set(strconv.Itoa(i), i)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Has(strconv.Itoa(i % 10000))
	}
}

func BenchmarkLRU_Clear(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		cache := NewLRU[string, int](1000)
		for j := 0; j < 1000; j++ {
			cache.Set(strconv.Itoa(j), j)
		}
		b.StartTimer()
		cache.Clear()
	}
}

func BenchmarkLRU_Eviction(b *testing.B) {
	sizes := []int{10, 100, 1000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("capacity=%d", size), func(b *testing.B) {
			cache := NewLRU[string, int](size)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cache.Set(strconv.Itoa(i), i) // Forces eviction after capacity
			}
		})
	}
}

func BenchmarkLRU_MixedWorkload(b *testing.B) {
	ratios := []struct {
		name  string
		read  int
		write int
	}{
		{"90r10w", 90, 10},
		{"50r50w", 50, 50},
		{"10r90w", 10, 90},
	}

	for _, ratio := range ratios {
		b.Run(ratio.name, func(b *testing.B) {
			cache := NewLRU[string, int](10000)
			// Pre-populate
			for i := 0; i < 5000; i++ {
				cache.Set(strconv.Itoa(i), i)
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if i%100 < ratio.read {
					cache.Get(strconv.Itoa(i % 5000))
				} else {
					cache.Set(strconv.Itoa(i%10000), i)
				}
			}
		})
	}
}

// ============= ShardedLRU Benchmarks =============

func BenchmarkShardedLRU_Set(b *testing.B) {
	shardCounts := []int{1, 16, 64, 256, 512}
	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("shards=%d", shards), func(b *testing.B) {
			cache := NewShardedLRU[string, int](10000, shards)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
		})
	}
}

func BenchmarkShardedLRU_Get(b *testing.B) {
	shardCounts := []int{1, 16, 64, 256, 512}
	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("shards=%d", shards), func(b *testing.B) {
			cache := NewShardedLRU[string, int](10000, shards)
			for i := 0; i < 10000; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cache.Get(strconv.Itoa(i % 10000))
			}
		})
	}
}

func BenchmarkShardedLRU_Distribution(b *testing.B) {
	// Test hash distribution quality
	cache := NewShardedLRU[string, int](10000, 256)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Sequential keys should distribute evenly
		cache.Set(fmt.Sprintf("key-%d", i), i)
	}
}

// ============= AtomicCache Benchmarks =============

func BenchmarkAtomicCache_Set(b *testing.B) {
	cache := NewAtomicCache[string, int](10000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Set(strconv.Itoa(i), i)
	}
}

func BenchmarkAtomicCache_Get(b *testing.B) {
	cache := NewAtomicCache[string, int](10000)
	for i := 0; i < 10000; i++ {
		cache.Set(strconv.Itoa(i), i)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i % 10000))
	}
}

func BenchmarkAtomicCache_BatchSet(b *testing.B) {
	cache := NewAtomicCache[string, int](10000)
	batch := make(map[string]int, 100)
	for i := 0; i < 100; i++ {
		batch[strconv.Itoa(i)] = i
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.BatchSet(batch)
	}
}

func BenchmarkAtomicCache_BatchGet(b *testing.B) {
	cache := NewAtomicCache[string, int](10000)
	for i := 0; i < 10000; i++ {
		cache.Set(strconv.Itoa(i), i)
	}
	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = strconv.Itoa(i)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.BatchGet(keys)
	}
}

// ============= Concurrent Benchmarks =============

func BenchmarkConcurrent_LRU_ReadHeavy(b *testing.B) {
	cache := NewLRU[string, int](10000)
	for i := 0; i < 10000; i++ {
		cache.Set(strconv.Itoa(i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				cache.Set(strconv.Itoa(i%10000), i)
			} else {
				cache.Get(strconv.Itoa(i % 10000))
			}
			i++
		}
	})
}

func BenchmarkConcurrent_LRU_WriteHeavy(b *testing.B) {
	cache := NewLRU[string, int](10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 < 9 {
				cache.Set(strconv.Itoa(i), i)
			} else {
				cache.Get(strconv.Itoa(i % 10000))
			}
			i++
		}
	})
}

func BenchmarkConcurrent_ShardedLRU_Scaling(b *testing.B) {
	for _, numGoroutines := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("goroutines=%d", numGoroutines), func(b *testing.B) {
			cache := NewShardedLRU[string, int](10000, numGoroutines*16)

			b.ResetTimer()
			b.SetParallelism(numGoroutines)
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := strconv.Itoa(i % 10000)
					if i%2 == 0 {
						cache.Set(key, i)
					} else {
						cache.Get(key)
					}
					i++
				}
			})
		})
	}
}

func BenchmarkConcurrent_AtomicCache(b *testing.B) {
	cache := NewAtomicCache[string, int](10000)
	for i := 0; i < 10000; i++ {
		cache.Set(strconv.Itoa(i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Get(strconv.Itoa(i % 10000))
			i++
		}
	})
}

// ============= Contention Benchmarks =============

func BenchmarkContention_SingleKey(b *testing.B) {
	b.Run("LRU", func(b *testing.B) {
		cache := NewLRU[string, int](1000)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Set("hotkey", 42)
				cache.Get("hotkey")
			}
		})
	})

	b.Run("ShardedLRU", func(b *testing.B) {
		cache := NewShardedLRU[string, int](1000, 256)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Set("hotkey", 42)
				cache.Get("hotkey")
			}
		})
	})

	b.Run("AtomicCache", func(b *testing.B) {
		cache := NewAtomicCache[string, int](1000)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Set("hotkey", 42)
				cache.Get("hotkey")
			}
		})
	})
}

// ============= Memory Benchmarks =============

func BenchmarkMemory_LRU(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache := NewLRU[string, string](1000)
		for j := 0; j < 1000; j++ {
			cache.Set(fmt.Sprintf("key%d", j), fmt.Sprintf("value%d", j))
		}
	}
}

func BenchmarkMemory_ShardedLRU(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache := NewShardedLRU[string, string](1000, 64)
		for j := 0; j < 1000; j++ {
			cache.Set(fmt.Sprintf("key%d", j), fmt.Sprintf("value%d", j))
		}
	}
}

func BenchmarkMemory_AtomicCache(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache := NewAtomicCache[string, string](1000)
		for j := 0; j < 1000; j++ {
			cache.Set(fmt.Sprintf("key%d", j), fmt.Sprintf("value%d", j))
		}
	}
}

// ============= Real-world Scenario Benchmarks =============

func BenchmarkRealWorld_SessionCache(b *testing.B) {
	// Simulate session cache with 1 hour TTL
	cache := NewLRU[string, map[string]interface{}](100000)
	sessionData := map[string]interface{}{
		"user_id": 12345,
		"email":   "user@example.com",
		"roles":   []string{"user", "admin"},
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sessionID := fmt.Sprintf("session_%d", i%50000)
			if i%10 < 8 { // 80% reads
				cache.Get(sessionID)
			} else { // 20% writes
				cache.SetWithTTL(sessionID, sessionData, time.Hour)
			}
			i++
		}
	})
}

func BenchmarkRealWorld_APIRateLimit(b *testing.B) {
	// Simulate API rate limiting cache
	cache := NewShardedLRU[string, int](10000, 256)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			clientIP := fmt.Sprintf("192.168.1.%d", i%256)
			if count, ok := cache.Get(clientIP); ok {
				cache.Set(clientIP, count+1)
			} else {
				cache.SetWithTTL(clientIP, 1, time.Minute)
			}
			i++
		}
	})
}

// ============= CPU Core Scaling =============

func BenchmarkScaling_SingleCore(b *testing.B) {
	old := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(old)

	cache := NewLRU[string, int](10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(i % 10000)
		if i%2 == 0 {
			cache.Set(key, i)
		} else {
			cache.Get(key)
		}
	}
}

func BenchmarkScaling_MultiCore(b *testing.B) {
	for _, cores := range []int{2, 4, 8} {
		if cores > runtime.NumCPU() {
			continue
		}
		b.Run(fmt.Sprintf("cores=%d", cores), func(b *testing.B) {
			old := runtime.GOMAXPROCS(cores)
			defer runtime.GOMAXPROCS(old)

			cache := NewShardedLRU[string, int](10000, cores*32)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := strconv.Itoa(i % 10000)
					if i%2 == 0 {
						cache.Set(key, i)
					} else {
						cache.Get(key)
					}
					i++
				}
			})
		})
	}
}

// ============= Stress Tests =============

func BenchmarkStress_MaxThroughput(b *testing.B) {
	cache := NewShardedLRU[int, int](100000, 512)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Set(i, i)
			i++
		}
	})
}

func BenchmarkStress_LargeValues(b *testing.B) {
	cache := NewLRU[int, []byte](1000)
	largeValue := make([]byte, 4096) // 4KB value

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Set(i%1000, largeValue)
	}
}

// ============= Comparison Suite =============

func BenchmarkComparison_All(b *testing.B) {
	operations := []struct {
		name string
		fn   func(Cache[string, int])
	}{
		{"Set", func(c Cache[string, int]) { c.Set("key", 42) }},
		{"Get", func(c Cache[string, int]) { c.Get("key") }},
		{"Delete", func(c Cache[string, int]) { c.Delete("key") }},
		{"Has", func(c Cache[string, int]) { c.Has("key") }},
	}

	caches := []struct {
		name  string
		cache Cache[string, int]
	}{
		{"LRU", NewLRU[string, int](10000)},
		{"ShardedLRU-256", NewShardedLRU[string, int](10000, 256)},
		{"AtomicCache", NewAtomicCache[string, int](10000)},
	}

	for _, op := range operations {
		b.Run(op.name, func(b *testing.B) {
			for _, c := range caches {
				b.Run(c.name, func(b *testing.B) {
					// Pre-populate for Get/Delete/Has
					c.cache.Set("key", 42)

					b.ResetTimer()
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						op.fn(c.cache)
					}
				})
			}
		})
	}
}

// ============= Latency Percentiles (for analysis) =============

func BenchmarkLatency_P99(b *testing.B) {
	cache := NewLRU[string, int](10000)
	for i := 0; i < 10000; i++ {
		cache.Set(strconv.Itoa(i), i)
	}

	latencies := make([]time.Duration, 0, 1000)
	mu := &sync.Mutex{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			start := time.Now()
			cache.Get(strconv.Itoa(b.N % 10000))
			elapsed := time.Since(start)

			mu.Lock()
			if len(latencies) < 1000 {
				latencies = append(latencies, elapsed)
			}
			mu.Unlock()
		}
	})

	// Analysis would happen here in real scenario
}
