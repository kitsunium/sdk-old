package kbuffer

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// Benchmark standard buffer operations

func BenchmarkWrite_SingleCore(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			buf := NewUnsafeBuffer(size * 2)
			b.ResetTimer()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				buf.Reset()
				buf.Write(data)
			}
		})
	}
}

func BenchmarkWriteString_SingleCore(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096}

	for _, size := range sizes {
		str := string(make([]byte, size))

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			buf := NewUnsafeBuffer(size * 2)
			b.ResetTimer()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				buf.Reset()
				buf.WriteString(str)
			}
		})
	}
}

func BenchmarkWriteByte_SingleCore(b *testing.B) {
	buf := NewUnsafeBuffer(b.N)
	b.ResetTimer()
	b.SetBytes(1)

	for i := 0; i < b.N; i++ {
		buf.WriteByte(byte(i))
	}
}

func BenchmarkTryWrite_SingleCore(b *testing.B) {
	data := make([]byte, 256)
	buf := NewUnsafeBuffer(1024)

	b.ResetTimer()
	b.SetBytes(256)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.TryWrite(data)
	}
}

func BenchmarkBytes_SingleCore(b *testing.B) {
	buf := NewUnsafeBuffer(1024)
	buf.Write(bytes.Repeat([]byte("x"), 1024))

	b.ResetTimer()
	b.SetBytes(1024)

	for i := 0; i < b.N; i++ {
		_ = buf.Bytes()
	}
}

func BenchmarkString_SingleCore(b *testing.B) {
	buf := NewUnsafeBuffer(1024)
	buf.Write(bytes.Repeat([]byte("x"), 1024))

	b.ResetTimer()
	b.SetBytes(1024)

	for i := 0; i < b.N; i++ {
		_ = buf.String()
	}
}

func BenchmarkReset_SingleCore(b *testing.B) {
	buf := NewUnsafeBuffer(1024)
	data := make([]byte, 512)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Write(data)
		buf.Reset()
	}
}

func BenchmarkClear_SingleCore(b *testing.B) {
	buf := NewUnsafeBuffer(1024)
	data := make([]byte, 512)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Write(data)
		buf.Clear()
	}
}

// Benchmark sharded buffer operations

func BenchmarkShardedBuffer_Write(b *testing.B) {
	shardCounts := []int{4, 8, 16, 32}

	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("shards_%d", shards), func(b *testing.B) {
			buf := NewSafeShardedBuffer(65536, shards)
			data := make([]byte, 256)

			b.ResetTimer()
			b.SetBytes(256)

			for i := 0; i < b.N; i++ {
				buf.Write(data)
				if i%(1000) == 0 {
					buf.Reset()
				}
			}
		})
	}
}

func BenchmarkShardedBuffer_Balance(b *testing.B) {
	buf := NewSafeShardedBuffer(4096, 16)

	// Create imbalanced distribution
	for i := 0; i < 10; i++ {
		buf.WriteToShard(0, make([]byte, 100))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Balance()
	}
}

// Benchmark pool operations

func BenchmarkPool_GetPut(b *testing.B) {
	pool := NewPool()

	sizes := []int{256, 1024, 4096, 16384}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf := pool.Get(size)
				pool.Put(buf)
			}
		})
	}
}

func BenchmarkPool_GetBufferPutBuffer(b *testing.B) {
	pool := NewPool()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := pool.GetBuffer(1024)
		buf.Write([]byte("test"))
		pool.PutBuffer(buf)
	}
}

// Multi-core benchmarks

func BenchmarkWrite_MultiCore(b *testing.B) {
	data := make([]byte, 100)
	cores := runtime.GOMAXPROCS(0)
	if cores < 2 {
		cores = 2
	}

	b.SetParallelism(cores)
	b.RunParallel(func(pb *testing.PB) {
		buf := NewUnsafeBuffer(10000)
		for pb.Next() {
			buf.Write(data)
			if buf.Available() < 100 {
				buf.Reset()
			}
		}
	})
}

func BenchmarkShardedWrite_MultiCore(b *testing.B) {
	buf := NewSafeShardedBuffer(100000, 16)
	data := make([]byte, 100)
	cores := runtime.GOMAXPROCS(0)
	if cores < 2 {
		cores = 2
	}

	b.SetParallelism(cores)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf.Write(data)
			if buf.Available() < 1000 {
				buf.Reset()
			}
		}
	})
}

func BenchmarkPool_MultiCore(b *testing.B) {
	sizes := []int{256, 1024, 4096}
	pool := NewPool()

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					buf := pool.Get(size)
					limit := 10
					if size < limit {
						limit = size
					}
					copy(buf[:limit], []byte("test data"))
					pool.Put(buf)
				}
			})
		})
	}
}

func BenchmarkPoolUsage_MultiCore(b *testing.B) {
	pool := NewPool()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(1024)
			// Simulate some work
			for i := 0; i < len(buf) && i < 100; i++ {
				buf[i] = byte(i)
			}
			pool.Put(buf)
		}
	})
}

func BenchmarkParallelWrite_MultiCore(b *testing.B) {
	sizes := []int{64, 256, 1024}

	for _, size := range sizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.RunParallel(func(pb *testing.PB) {
				buf := NewUnsafeBuffer(size * 2)
				for pb.Next() {
					buf.Reset()
					buf.Write(data)
				}
			})
		})
	}
}

func BenchmarkMixed_MultiCore(b *testing.B) {
	data := []byte("test data")
	str := "string data"

	b.RunParallel(func(pb *testing.PB) {
		buf := NewUnsafeBuffer(4096)
		i := 0
		for pb.Next() {
			buf.Reset()
			switch i % 3 {
			case 0:
				buf.Write(data)
			case 1:
				buf.WriteString(str)
			case 2:
				buf.WriteByte('x')
			}
			_ = buf.String()
			i++
		}
	})
}

// Benchmark memory allocations

func BenchmarkAllocation_StandardBuffer(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := NewUnsafeBuffer(1024)
		buf.Write([]byte("test"))
		_ = buf.String()
	}
}

func BenchmarkAllocation_WithPool(b *testing.B) {
	pool := NewPool()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.GetBuffer(1024)
		buf.Write([]byte("test"))
		_ = buf.String()
		pool.PutBuffer(buf)
	}
}

func BenchmarkAllocation_ZeroCopy(b *testing.B) {
	buf := NewUnsafeBuffer(1024)
	data := []byte("test data")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.Write(data)
		_ = buf.Bytes()  // Zero-copy
		_ = buf.String() // Zero-allocation
	}
}

// Comparison benchmarks with standard library

func BenchmarkComparison_StandardBuffer_vs_BytesBuffer(b *testing.B) {
	data := make([]byte, 1024)

	b.Run("kbuffer", func(b *testing.B) {
		buf := NewUnsafeBuffer(2048)
		b.ResetTimer()
		b.SetBytes(1024)

		for i := 0; i < b.N; i++ {
			buf.Reset()
			buf.Write(data)
			_ = buf.Bytes()
		}
	})

	b.Run("bytes.Buffer", func(b *testing.B) {
		buf := bytes.NewBuffer(make([]byte, 0, 2048))
		b.ResetTimer()
		b.SetBytes(1024)

		for i := 0; i < b.N; i++ {
			buf.Reset()
			buf.Write(data)
			_ = buf.Bytes()
		}
	})
}

// Scalability benchmarks for multi-core

func BenchmarkScalability_MultiCore(b *testing.B) {
	data := make([]byte, 100)
	cores := runtime.GOMAXPROCS(0)
	if cores < 2 {
		cores = 2
	}

	buf := NewSafeShardedBuffer(1000000, cores*2)
	var wg sync.WaitGroup
	var ops atomic.Int64

	b.SetParallelism(cores)
	b.ResetTimer()

	for i := 0; i < cores; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/cores; j++ {
				buf.Write(data)
				ops.Add(1)
				if j%1000 == 0 && buf.Available() < 10000 {
					buf.Reset()
				}
			}
		}()
	}

	wg.Wait()
	b.SetBytes(100 * ops.Load() / int64(b.N))
}

// Cache efficiency benchmarks

func BenchmarkCacheEfficiency_Sequential(b *testing.B) {
	buf := NewUnsafeBuffer(1 << 20) // 1MB
	data := make([]byte, cacheLineSize)

	b.ResetTimer()
	b.SetBytes(int64(cacheLineSize))

	for i := 0; i < b.N; i++ {
		buf.Write(data)
		if buf.Available() < cacheLineSize {
			buf.Reset()
		}
	}
}

func BenchmarkCacheEfficiency_Random(b *testing.B) {
	const bufSize = 1 << 20
	buf := NewUnsafeBuffer(bufSize)
	data := make([]byte, cacheLineSize)

	// Fill buffer first
	for buf.Available() >= cacheLineSize {
		buf.Write(data)
	}

	b.ResetTimer()
	b.SetBytes(int64(cacheLineSize))

	for i := 0; i < b.N; i++ {
		offset := int64((i * 7919) % (bufSize - cacheLineSize))
		buf.WriteAt(data, offset)
	}
}

// Contention benchmarks

func BenchmarkContention_MultiCore(b *testing.B) {
	pool := NewPool()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(1024)
			copy(buf[:4], []byte("test"))
			pool.Put(buf)
		}
	})
}

func BenchmarkPoolContention_MultiCore(b *testing.B) {
	pool := NewPool()
	sizes := []int{64, 256, 1024, 4096}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			size := sizes[i%len(sizes)]
			buf := pool.Get(size)
			limit := 10
			if size < limit {
				limit = size
			}
			for j := 0; j < limit; j++ {
				buf[j] = byte(j)
			}
			pool.Put(buf)
			i++
		}
	})
}

func BenchmarkUnsafe_BytesAccess(b *testing.B) {
	buf := NewUnsafeBuffer(1024)
	buf.Write(make([]byte, 1024))

	b.Run("unsafe_pointer", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ptr, length := buf.BytesUnsafe()
			// Simply verify we got the pointer and length
			// Don't convert to avoid go vet warnings
			if ptr == 0 || length == 0 {
				b.Fatal("BytesUnsafe returned invalid values")
			}
		}
	})

	b.Run("standard", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = buf.Bytes()
		}
	})
}
