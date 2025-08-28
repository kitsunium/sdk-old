package kbuffer

import (
	"bytes"
	"fmt"
	"testing"
)

// Benchmark standard buffer operations

func BenchmarkStandardBuffer_Write(b *testing.B) {
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

func BenchmarkStandardBuffer_WriteString(b *testing.B) {
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

func BenchmarkStandardBuffer_WriteByte(b *testing.B) {
	buf := NewUnsafeBuffer(b.N)
	b.ResetTimer()
	b.SetBytes(1)

	for i := 0; i < b.N; i++ {
		buf.WriteByte(byte(i))
	}
}

func BenchmarkStandardBuffer_TryWrite(b *testing.B) {
	data := make([]byte, 256)
	buf := NewUnsafeBuffer(1024)

	b.ResetTimer()
	b.SetBytes(256)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.TryWrite(data)
	}
}

func BenchmarkStandardBuffer_Bytes(b *testing.B) {
	buf := NewUnsafeBuffer(1024)
	buf.Write(bytes.Repeat([]byte("x"), 1024))

	b.ResetTimer()
	b.SetBytes(1024)

	for i := 0; i < b.N; i++ {
		_ = buf.Bytes()
	}
}

func BenchmarkStandardBuffer_String(b *testing.B) {
	buf := NewUnsafeBuffer(1024)
	buf.Write(bytes.Repeat([]byte("x"), 1024))

	b.ResetTimer()
	b.SetBytes(1024)

	for i := 0; i < b.N; i++ {
		_ = buf.String()
	}
}

func BenchmarkStandardBuffer_Reset(b *testing.B) {
	buf := NewUnsafeBuffer(1024)
	data := make([]byte, 512)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Write(data)
		buf.Reset()
	}
}

func BenchmarkStandardBuffer_Clear(b *testing.B) {
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
	pool := GetGlobalPool()

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
	pool := GetGlobalPool()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := pool.GetBuffer(1024)
		buf.Write([]byte("test"))
		pool.PutBuffer(buf)
	}
}

// Concurrent benchmarks

func BenchmarkConcurrent_StandardBuffer(b *testing.B) {
	data := make([]byte, 100)

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

func BenchmarkConcurrent_ShardedBuffer(b *testing.B) {
	buf := NewSafeShardedBuffer(100000, 16)
	data := make([]byte, 100)

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

func BenchmarkPool_ParallelGetPut(b *testing.B) {
	sizes := []int{256, 1024, 4096}
	p := newPool()

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					buf := p.Get(size)
					copy(buf[:min(10, size)], []byte("test data"))
					p.Put(buf)
				}
			})
		})
	}
}

func BenchmarkBuffer_Concurrent(b *testing.B) {
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

func BenchmarkBuffer_ParallelWrite(b *testing.B) {
	sizes := []int{64, 256, 1024}

	for _, size := range sizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.RunParallel(func(pb *testing.PB) {
				buf := NewBuffer(size * 2)
				for pb.Next() {
					buf.Reset()
					buf.Write(data)
				}
			})
		})
	}
}

func BenchmarkBuffer_ParallelMixed(b *testing.B) {
	data := []byte("test data")
	str := "string data"

	b.RunParallel(func(pb *testing.PB) {
		buf := NewBuffer(4096)
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
	pool := GetGlobalPool()
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

// Scalability benchmarks

func BenchmarkScalability_ShardedBuffer(b *testing.B) {
	data := make([]byte, 100)

	for _, numCPU := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("cpu_%d", numCPU), func(b *testing.B) {
			runtime.GOMAXPROCS(numCPU)
			defer runtime.GOMAXPROCS(runtime.NumCPU())

			buf := NewSafeShardedBuffer(1000000, numCPU*2)
			var wg sync.WaitGroup
			var ops atomic.Int64

			b.ResetTimer()

			for i := 0; i < numCPU; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < b.N/numCPU; j++ {
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
		})
	}
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

func BenchmarkContention_Sharded(b *testing.B) {
	data := []byte("test")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := p.Get(1024)
			copy(buf[:4], []byte("test"))
			p.Put(buf)
		}
	})
}

func BenchmarkPool_ParallelContention(b *testing.B) {
	p := newPool()
	sizes := []int{64, 256, 1024, 4096}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			size := sizes[i%len(sizes)]
			buf := p.Get(size)
			for j := 0; j < min(10, size); j++ {
				buf[j] = byte(j)
			}
			p.Put(buf)
			i++
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
