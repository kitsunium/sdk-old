// Package cache provides thread-safe caching implementations.
package cache

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"sync"
	"testing"
	"time"
)

func BenchmarkLRU_Set(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			c := NewLRU[string, int](size)
			b.ResetTimer()
			for i := range b.N {
				c.Set(strconv.Itoa(i), i)
			}
		})
	}
}

func BenchmarkLRU_Get(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			c := NewLRU[string, int](size)
			for i := range size {
				c.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			for i := range b.N {
				c.Get(strconv.Itoa(i % size))
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
				c := NewLRU[string, int](size)
				b.ResetTimer()
				for i := range b.N {
					c.SetWithTTL(strconv.Itoa(i), i, ttl)
				}
			})
		}
	}
}

func BenchmarkLRU_Delete(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			c := NewLRU[string, int](size)
			for i := range size {
				c.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			for i := range b.N {
				key := strconv.Itoa(i % size)
				c.Delete(key)
				c.Set(key, i)
			}
		})
	}
}

func BenchmarkLRU_Has(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			c := NewLRU[string, int](size)
			for i := range size {
				c.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			for i := range b.N {
				c.Has(strconv.Itoa(i % size))
			}
		})
	}
}

func BenchmarkLRU_Clear(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			c := NewLRU[string, int](size)
			b.ResetTimer()
			for range b.N {
				b.StopTimer()
				for j := range size {
					c.Set(strconv.Itoa(j), j)
				}
				b.StartTimer()
				c.Clear()
			}
		})
	}
}

func BenchmarkLRU_ConcurrentSet(b *testing.B) {
	concurrencies := []int{1, 10, 100}
	cacheSize := 1000

	for _, concurrency := range concurrencies {
		b.Run(fmt.Sprintf("concurrency=%d", concurrency), func(b *testing.B) {
			c := NewLRU[string, int](cacheSize)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					c.Set(strconv.Itoa(i), i)
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
			c := NewLRU[string, int](cacheSize)
			for i := range cacheSize {
				c.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					c.Get(strconv.Itoa(i % cacheSize))
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
			c := NewLRU[string, int](size)
			for i := range size / 2 {
				c.Set(strconv.Itoa(i), i)
			}

			b.ResetTimer()
			for i := range b.N {
				switch i % 4 {
				case 0:
					c.Set(strconv.Itoa(i), i)
				case 1:
					c.Get(strconv.Itoa(i % size))
				case 2:
					c.Has(strconv.Itoa(i % size))
				case 3:
					c.Delete(strconv.Itoa(i % size))
				}
			}
		})
	}
}

func BenchmarkLRU_Eviction(b *testing.B) {
	sizes := []int{10, 100, 1000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			c := NewLRU[string, int](size)
			b.ResetTimer()
			for i := range b.N {
				c.Set(strconv.Itoa(i), i)
			}
			stats := c.Stats()
			b.Logf("Evictions: %d", stats.Evictions.Load())
		})
	}
}

func BenchmarkShardedCache_Set(b *testing.B) {
	shardCounts := []int{4, 16, 64}
	cacheSize := 10000

	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("shards=%d", shards), func(b *testing.B) {
			c := NewShardedLRU[string, int](cacheSize, shards)
			b.ResetTimer()
			for i := range b.N {
				c.Set(strconv.Itoa(i), i)
			}
		})
	}
}

func BenchmarkShardedCache_Get(b *testing.B) {
	shardCounts := []int{4, 16, 64}
	cacheSize := 10000

	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("shards=%d", shards), func(b *testing.B) {
			c := NewShardedLRU[string, int](cacheSize, shards)
			for i := range cacheSize {
				c.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			for i := range b.N {
				c.Get(strconv.Itoa(i % cacheSize))
			}
		})
	}
}

func BenchmarkShardedCache_ConcurrentSet(b *testing.B) {
	shardCounts := []int{4, 16, 64}
	cacheSize := 10000

	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("shards=%d", shards), func(b *testing.B) {
			c := NewShardedLRU[string, int](cacheSize, shards)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					c.Set(strconv.Itoa(i), i)
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
			c := NewShardedLRU[string, int](cacheSize, shards)
			for i := range cacheSize {
				c.Set(strconv.Itoa(i), i)
			}
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					c.Get(strconv.Itoa(i % cacheSize))
					i++
				}
			})
		})
	}
}

func BenchmarkComparisonConcurrentReadHeavy(b *testing.B) {
	const (
		cacheSize int = 1000
		numKeys   int = 100
		workers   int = 10
	)

	keys := make([]string, numKeys)
	for i := range numKeys {
		keys[i] = strconv.Itoa(i)
	}

	b.Run("LRU", func(b *testing.B) {
		c := NewLRU[string, int](cacheSize)
		for i := range numKeys {
			c.Set(keys[i], i)
		}

		var wg sync.WaitGroup
		b.ResetTimer()

		for i := range workers {
			workerID := i
			wg.Go(func() {
				for range b.N / workers {
					idx := rand.IntN(numKeys)
					c.Get(keys[idx])
				}
				_ = workerID
			})
		}
		wg.Wait()
	})

	b.Run("ShardedCache", func(b *testing.B) {
		c := NewShardedLRU[string, int](cacheSize, 16)
		for i := range numKeys {
			c.Set(keys[i], i)
		}

		var wg sync.WaitGroup
		b.ResetTimer()

		for i := range workers {
			workerID := i
			wg.Go(func() {
				for range b.N / workers {
					idx := rand.IntN(numKeys)
					c.Get(keys[idx])
				}
				_ = workerID
			})
		}
		wg.Wait()
	})
}

func BenchmarkComparisonConcurrentWriteHeavy(b *testing.B) {
	const (
		cacheSize int = 1000
		workers   int = 10
	)

	b.Run("LRU", func(b *testing.B) {
		c := NewLRU[string, int](cacheSize)

		var wg sync.WaitGroup
		b.ResetTimer()

		for i := range workers {
			workerID := i
			wg.Go(func() {
				base := workerID * (b.N / workers)
				for j := range b.N / workers {
					key := strconv.Itoa(base + j)
					c.Set(key, base+j)
				}
			})
		}
		wg.Wait()
	})

	b.Run("ShardedCache", func(b *testing.B) {
		c := NewShardedLRU[string, int](cacheSize, 16)

		var wg sync.WaitGroup
		b.ResetTimer()

		for i := range workers {
			workerID := i
			wg.Go(func() {
				base := workerID * (b.N / workers)
				for j := range b.N / workers {
					key := strconv.Itoa(base + j)
					c.Set(key, base+j)
				}
			})
		}
		wg.Wait()
	})
}
