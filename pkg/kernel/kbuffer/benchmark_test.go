package kbuffer

import (
	"fmt"
	"sync"
	"testing"

	"github.com/kitsunium/sdk/pkg/kernel/kcache"
)

// Benchmark current sync.Map implementation
func BenchmarkPoolWithSyncMap(b *testing.B) {
	pool := NewBufferPool()
	pool.Prewarm([]int{256, 512, 1024, 4096}, 10)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			size := 1024
			buf := pool.Get(size)
			// Simulate some work
			buf[0] = 1
			buf[len(buf)-1] = 2
			pool.Put(buf)
		}
	})
}

// Benchmark with kcache ShardedLRU
func BenchmarkPoolWithKcacheSharded(b *testing.B) {
	// Mock pool with kcache
	type poolWithKcache struct {
		pools      kcache.Cache[int, *sync.Pool]
		stats      PoolStats
		maxSize    int
		clearOnPut bool
	}

	p := &poolWithKcache{
		pools:   kcache.NewShardedLRU[int, *sync.Pool](100, 16),
		maxSize: 1 << 20,
	}

	// Pre-warm
	sizes := []int{256, 512, 1024, 4096}
	for _, size := range sizes {
		pool := &sync.Pool{New: nil}
		p.pools.Set(size, pool)
		for i := 0; i < 10; i++ {
			buf := make([]byte, size)
			pool.Put(buf)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			size := 1024
			poolSize := nextPowerOf2(size)
			
			var buf []byte
			if pool, ok := p.pools.Get(poolSize); ok {
				if b := pool.Get(); b != nil {
					buf = b.([]byte)[:size]
				} else {
					buf = make([]byte, size, poolSize)
				}
			} else {
				buf = make([]byte, size, poolSize)
			}
			
			// Simulate some work
			buf[0] = 1
			buf[len(buf)-1] = 2
			
			if pool, ok := p.pools.Get(poolSize); ok {
				pool.Put(buf[:poolSize])
			}
		}
	})
}

// Benchmark with kcache AtomicCache
func BenchmarkPoolWithKcacheAtomic(b *testing.B) {
	// Mock pool with kcache
	type poolWithKcache struct {
		pools      kcache.Cache[int, *sync.Pool]
		stats      PoolStats
		maxSize    int
		clearOnPut bool
	}

	p := &poolWithKcache{
		pools:   kcache.NewAtomicCache[int, *sync.Pool](100),
		maxSize: 1 << 20,
	}

	// Pre-warm
	sizes := []int{256, 512, 1024, 4096}
	for _, size := range sizes {
		pool := &sync.Pool{New: nil}
		p.pools.Set(size, pool)
		for i := 0; i < 10; i++ {
			buf := make([]byte, size)
			pool.Put(buf)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			size := 1024
			poolSize := nextPowerOf2(size)
			
			var buf []byte
			if pool, ok := p.pools.Get(poolSize); ok {
				if b := pool.Get(); b != nil {
					buf = b.([]byte)[:size]
				} else {
					buf = make([]byte, size, poolSize)
				}
			} else {
				buf = make([]byte, size, poolSize)
			}
			
			// Simulate some work
			buf[0] = 1
			buf[len(buf)-1] = 2
			
			if pool, ok := p.pools.Get(poolSize); ok {
				pool.Put(buf[:poolSize])
			}
		}
	})
}

// Benchmark Get/Put operations directly
func BenchmarkPoolGetPut(b *testing.B) {
	pool := NewBufferPool()
	pool.Prewarm([]int{1024}, 100)

	b.Run("SyncMap", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := pool.Get(1024)
				pool.Put(buf)
			}
		})
	})
}

// Benchmark concurrent access patterns
func BenchmarkPoolConcurrentAccess(b *testing.B) {
	sizes := []int{256, 512, 1024, 2048, 4096, 8192, 16384, 32768}
	
	b.Run("SyncMap", func(b *testing.B) {
		pool := NewBufferPool()
		pool.Prewarm(sizes, 10)
		
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				size := sizes[i%len(sizes)]
				buf := pool.Get(size)
				// Simulate work
				if len(buf) > 0 {
					buf[0] = byte(i)
				}
				pool.Put(buf)
				i++
			}
		})
	})
}

// Benchmark Buffer operations
func BenchmarkBufferOperations(b *testing.B) {
	b.Run("Write", func(b *testing.B) {
		buf := NewBuffer(4096)
		data := []byte("Hello, World!")
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			buf.Free()
			buf.Write(data)
		}
	})
	
	b.Run("WriteString", func(b *testing.B) {
		buf := NewBuffer(4096)
		data := "Hello, World!"
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			buf.Free()
			buf.WriteString(data)
		}
	})
	
	b.Run("TryWrite", func(b *testing.B) {
		buf := NewBuffer(4096)
		data := []byte("Hello, World!")
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			buf.Free()
			buf.TryWrite(data)
		}
	})
}

// Benchmark memory allocation patterns
func BenchmarkMemoryPatterns(b *testing.B) {
	b.Run("DirectAllocation", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := make([]byte, 1024)
				buf[0] = 1
			}
		})
	})
	
	b.Run("PooledAllocation", func(b *testing.B) {
		pool := NewBufferPool()
		pool.Prewarm([]int{1024}, 100)
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := pool.Get(1024)
				buf[0] = 1
				pool.Put(buf)
			}
		})
	})
}

// Benchmark different pool sizes
func BenchmarkPoolSizes(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096, 16384, 65536}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			pool := NewBufferPool()
			pool.Prewarm([]int{size}, 50)
			
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					buf := pool.Get(size)
					buf[0] = 1
					pool.Put(buf)
				}
			})
		})
	}
}