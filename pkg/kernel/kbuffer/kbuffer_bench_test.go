package kbuffer

import (
	"bytes"
	"fmt"
	"testing"
)

// Benchmark buffer write operations

func BenchmarkBuffer_Write(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			buf := NewBuffer(size * 2)
			b.ResetTimer()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				buf.Reset()
				buf.Write(data)
			}
		})
	}
}

func BenchmarkBuffer_WriteString(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096}

	for _, size := range sizes {
		str := string(make([]byte, size))

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			buf := NewBuffer(size * 2)
			b.ResetTimer()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				buf.Reset()
				buf.WriteString(str)
			}
		})
	}
}

func BenchmarkBuffer_WriteByte(b *testing.B) {
	buf := NewBuffer(b.N)
	b.ResetTimer()
	b.SetBytes(1)

	for i := 0; i < b.N; i++ {
		buf.WriteByte(byte(i))
	}
}

func BenchmarkBuffer_TryWrite(b *testing.B) {
	data := make([]byte, 256)
	buf := NewBuffer(1024)

	b.ResetTimer()
	b.SetBytes(256)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.TryWrite(data)
	}
}

func BenchmarkBuffer_String(b *testing.B) {
	buf := NewBuffer(1024)
	buf.Write([]byte("hello world"))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = buf.String()
	}
}

func BenchmarkBuffer_Bytes(b *testing.B) {
	buf := NewBuffer(1024)
	buf.Write([]byte("hello world"))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = buf.Bytes()
	}
}

// Benchmark pool operations

func BenchmarkPool_Get(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096, 16384, 65536}
	p := newPool()

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf := p.Get(size)
				_ = buf[0] // Prevent optimization
			}
		})
	}
}

func BenchmarkPool_GetPut(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096, 16384, 65536}
	p := newPool()

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf := p.Get(size)
				p.Put(buf)
			}
		})
	}
}

func BenchmarkPool_GetBuffer(b *testing.B) {
	p := newPool()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := p.GetBuffer(1024)
		p.PutBuffer(buf)
	}
}

// Benchmark vs standard library

func BenchmarkComparison_Buffer_vs_BytesBuffer(b *testing.B) {
	data := make([]byte, 1024)

	b.Run("kbuffer.Buffer", func(b *testing.B) {
		b.SetBytes(1024)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := NewBuffer(1024)
			buf.Write(data)
			_ = buf.String()
		}
	})

	b.Run("bytes.Buffer", func(b *testing.B) {
		b.SetBytes(1024)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := &bytes.Buffer{}
			buf.Write(data)
			_ = buf.String()
		}
	})
}

func BenchmarkComparison_Pool_vs_Make(b *testing.B) {
	p := newPool()

	b.Run("pool.Get", func(b *testing.B) {
		b.SetBytes(4096)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := p.Get(4096)
			buf[0] = 1 // Use buffer
			p.Put(buf)
		}
	})

	b.Run("make", func(b *testing.B) {
		b.SetBytes(4096)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := make([]byte, 4096)
			buf[0] = 1 // Use buffer
		}
	})
}

// Benchmark concurrent access

func BenchmarkPool_Concurrent(b *testing.B) {
	p := newPool()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := p.Get(1024)
			copy(buf, []byte("test data"))
			p.Put(buf)
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
		buf := NewBuffer(1024)
		data := []byte("test data")

		for pb.Next() {
			buf.Reset()
			buf.Write(data)
			_ = buf.String()
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

func BenchmarkAllocations_Buffer(b *testing.B) {
	data := []byte("hello world")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := NewBuffer(100)
		buf.Write(data)
		_ = buf.String()
	}
}

func BenchmarkAllocations_Pool(b *testing.B) {
	p := newPool()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := p.Get(100)
		copy(buf, []byte("hello"))
		p.Put(buf)
	}
}

func BenchmarkAllocations_PoolReuse(b *testing.B) {
	p := newPool()

	// Pre-warm pool
	for i := 0; i < 100; i++ {
		buf := make([]byte, 1024)
		p.Put(buf)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := p.Get(1024)
		copy(buf, []byte("test"))
		p.Put(buf)
	}
}

// Benchmark with various workloads

func BenchmarkWorkload_SmallWrites(b *testing.B) {
	buf := NewBuffer(1024)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		for j := 0; j < 100; j++ {
			buf.WriteByte(byte(j))
		}
	}
}

func BenchmarkWorkload_MixedOperations(b *testing.B) {
	buf := NewBuffer(4096)
	data := []byte("test data")
	str := "string data"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.Write(data)
		buf.WriteString(str)
		buf.WriteByte('x')
		_ = buf.Len()
		_ = buf.Available()
		_ = buf.String()
	}
}

func BenchmarkWorkload_PoolChurn(b *testing.B) {
	p := newPool()
	sizes := []int{64, 128, 256, 512, 1024, 2048, 4096}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		size := sizes[i%len(sizes)]
		buf := p.Get(size)
		for j := 0; j < size; j++ {
			buf[j] = byte(j)
		}
		p.Put(buf)
	}
}

// Benchmark pool efficiency

func BenchmarkPoolEfficiency(b *testing.B) {
	p := newPool()

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
