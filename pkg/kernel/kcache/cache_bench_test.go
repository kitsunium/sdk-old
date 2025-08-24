package kcache

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

func BenchmarkLRU_Set(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewLRU[string, int](size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
		})
	}
}

func BenchmarkLRU_Get(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewLRU[string, int](size)
			for i := 0; i < size; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cache.Get(strconv.Itoa(i % size))
			}
		})
	}
}

func BenchmarkLRU_SetWithTTL(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	ttls := []time.Duration{time.Second, time.Minute, time.Hour}
	
	for _, size := range sizes {
		for _, ttl := range ttls {
			b.Run(fmt.Sprintf("size=%d/ttl=%s", size, ttl), func(b *testing.B) {
				cache := NewLRU[string, int](size)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					cache.SetWithTTL(strconv.Itoa(i), i, ttl)
				}
			})
		}
	}
}

func BenchmarkLRU_Delete(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewLRU[string, int](size)
			for i := 0; i < size; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := strconv.Itoa(i % size)
				cache.Delete(key)
				cache.Set(key, i)
			}
		})
	}
}

func BenchmarkLRU_Has(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewLRU[string, int](size)
			for i := 0; i < size; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cache.Has(strconv.Itoa(i % size))
			}
		})
	}
}

func BenchmarkLRU_Clear(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewLRU[string, int](size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				for j := 0; j < size; j++ {
					cache.Set(strconv.Itoa(j), j)
				}
				b.StartTimer()
				cache.Clear()
			}
		})
	}
}

func BenchmarkLRU_ConcurrentSet(b *testing.B) {
	concurrencies := []int{1, 10, 100}
	cacheSize := 1000
	
	for _, concurrency := range concurrencies {
		b.Run(fmt.Sprintf("concurrency=%d", concurrency), func(b *testing.B) {
			cache := NewLRU[string, int](cacheSize)
			b.ResetTimer()
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

func BenchmarkLRU_ConcurrentGet(b *testing.B) {
	concurrencies := []int{1, 10, 100}
	cacheSize := 1000
	
	for _, concurrency := range concurrencies {
		b.Run(fmt.Sprintf("concurrency=%d", concurrency), func(b *testing.B) {
			cache := NewLRU[string, int](cacheSize)
			for i := 0; i < cacheSize; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					cache.Get(strconv.Itoa(i % cacheSize))
					i++
				}
			})
		})
	}
}

func BenchmarkLRU_MixedOperations(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewLRU[string, int](size)
			for i := 0; i < size/2; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				switch i % 4 {
				case 0:
					cache.Set(strconv.Itoa(i), i)
				case 1:
					cache.Get(strconv.Itoa(i % size))
				case 2:
					cache.Has(strconv.Itoa(i % size))
				case 3:
					cache.Delete(strconv.Itoa(i % size))
				}
			}
		})
	}
}

func BenchmarkLRU_Eviction(b *testing.B) {
	sizes := []int{10, 100, 1000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewLRU[string, int](size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
			stats := cache.Stats()
			b.Logf("Evictions: %d", stats.Evictions.Load())
		})
	}
}

func BenchmarkAtomicCache_Set(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewAtomicCache[string, int](size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
		})
	}
}

func BenchmarkAtomicCache_Get(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			cache := NewAtomicCache[string, int](size)
			for i := 0; i < size; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cache.Get(strconv.Itoa(i % size))
			}
		})
	}
}

func BenchmarkShardedCache_Set(b *testing.B) {
	shardCounts := []int{4, 16, 64}
	cacheSize := 10000
	
	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("shards=%d", shards), func(b *testing.B) {
			cache := NewShardedLRU[string, int](cacheSize, shards)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
		})
	}
}

func BenchmarkShardedCache_Get(b *testing.B) {
	shardCounts := []int{4, 16, 64}
	cacheSize := 10000
	
	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("shards=%d", shards), func(b *testing.B) {
			cache := NewShardedLRU[string, int](cacheSize, shards)
			for i := 0; i < cacheSize; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cache.Get(strconv.Itoa(i % cacheSize))
			}
		})
	}
}

func BenchmarkShardedCache_ConcurrentSet(b *testing.B) {
	shardCounts := []int{4, 16, 64}
	cacheSize := 10000
	
	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("shards=%d", shards), func(b *testing.B) {
			cache := NewShardedLRU[string, int](cacheSize, shards)
			b.ResetTimer()
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

func BenchmarkShardedCache_ConcurrentGet(b *testing.B) {
	shardCounts := []int{4, 16, 64}
	cacheSize := 10000
	
	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("shards=%d", shards), func(b *testing.B) {
			cache := NewShardedLRU[string, int](cacheSize, shards)
			for i := 0; i < cacheSize; i++ {
				cache.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					cache.Get(strconv.Itoa(i % cacheSize))
					i++
				}
			})
		})
	}
}

func BenchmarkComparisonConcurrentReadHeavy(b *testing.B) {
	cacheSize := 1000
	numKeys := 100
	
	keys := make([]string, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = strconv.Itoa(i)
	}
	
	b.Run("LRU", func(b *testing.B) {
		cache := NewLRU[string, int](cacheSize)
		for i := 0; i < numKeys; i++ {
			cache.Set(keys[i], i)
		}
		
		var wg sync.WaitGroup
		b.ResetTimer()
		
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < b.N/10; j++ {
					idx := rand.Intn(numKeys)
					cache.Get(keys[idx])
				}
			}(i)
		}
		wg.Wait()
	})
	
	b.Run("AtomicCache", func(b *testing.B) {
		cache := NewAtomicCache[string, int](cacheSize)
		for i := 0; i < numKeys; i++ {
			cache.Set(keys[i], i)
		}
		
		var wg sync.WaitGroup
		b.ResetTimer()
		
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < b.N/10; j++ {
					idx := rand.Intn(numKeys)
					cache.Get(keys[idx])
				}
			}(i)
		}
		wg.Wait()
	})
	
	b.Run("ShardedCache", func(b *testing.B) {
		cache := NewShardedLRU[string, int](cacheSize, 16)
		for i := 0; i < numKeys; i++ {
			cache.Set(keys[i], i)
		}
		
		var wg sync.WaitGroup
		b.ResetTimer()
		
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < b.N/10; j++ {
					idx := rand.Intn(numKeys)
					cache.Get(keys[idx])
				}
			}(i)
		}
		wg.Wait()
	})
}

func BenchmarkComparisonConcurrentWriteHeavy(b *testing.B) {
	cacheSize := 1000
	
	b.Run("LRU", func(b *testing.B) {
		cache := NewLRU[string, int](cacheSize)
		
		var wg sync.WaitGroup
		b.ResetTimer()
		
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				base := workerID * (b.N / 10)
				for j := 0; j < b.N/10; j++ {
					key := strconv.Itoa(base + j)
					cache.Set(key, base+j)
				}
			}(i)
		}
		wg.Wait()
	})
	
	b.Run("AtomicCache", func(b *testing.B) {
		cache := NewAtomicCache[string, int](cacheSize)
		
		var wg sync.WaitGroup
		b.ResetTimer()
		
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				base := workerID * (b.N / 10)
				for j := 0; j < b.N/10; j++ {
					key := strconv.Itoa(base + j)
					cache.Set(key, base+j)
				}
			}(i)
		}
		wg.Wait()
	})
	
	b.Run("ShardedCache", func(b *testing.B) {
		cache := NewShardedLRU[string, int](cacheSize, 16)
		
		var wg sync.WaitGroup
		b.ResetTimer()
		
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				base := workerID * (b.N / 10)
				for j := 0; j < b.N/10; j++ {
					key := strconv.Itoa(base + j)
					cache.Set(key, base+j)
				}
			}(i)
		}
		wg.Wait()
	})
}