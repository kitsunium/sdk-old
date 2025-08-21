package kbuffer

import (
	"fmt"
	"math/rand"
	"testing"
)

// BenchmarkOptimizedPool tests the optimized fast pool
func BenchmarkOptimizedPool(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("FastPool_size_%d", size), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)

			b.ReportAllocs()
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf := OptimizedGet(size)
				copy(buf, data)
				OptimizedPut(buf)
			}
		})
	}
}

// BenchmarkOptimizedWrite tests the SIMD-optimized write operations
func BenchmarkOptimizedWrite(b *testing.B) {
	sizes := []int{32, 64, 128, 256, 512, 1024, 2048, 4096}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			buf := NewBuffer(size * 10)
			data := make([]byte, size)
			rand.Read(data)

			b.ReportAllocs()
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf.pos = 0
				buf.writeOptimized(data)
			}
		})
	}
}

// BenchmarkComparisonOptimized compares optimized vs standard operations
func BenchmarkComparisonOptimized(b *testing.B) {
	data := make([]byte, 1024)
	rand.Read(data)

	b.Run("standard_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := Get(1024)
			copy(buf, data)
			Put(buf)
		}
	})

	b.Run("optimized_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := OptimizedGet(1024)
			copy(buf, data)
			OptimizedPut(buf)
		}
	})

	b.Run("standard_write", func(b *testing.B) {
		buf := NewBuffer(10240)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.pos = 0
			buf.Write(data)
		}
	})

	b.Run("optimized_write", func(b *testing.B) {
		buf := NewBuffer(10240)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.pos = 0
			buf.writeOptimized(data)
		}
	})
}

// BenchmarkZeroAllocation verifies zero allocations in hot paths
func BenchmarkZeroAllocation(b *testing.B) {
	// Pre-warm the pools
	for i := 0; i < 100; i++ {
		buf := OptimizedGet(1024)
		OptimizedPut(buf)
	}

	b.Run("optimized_pool_reuse", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := OptimizedGet(256)
			buf[0] = byte(i)
			OptimizedPut(buf)
		}
	})

	b.Run("buffer_write_no_alloc", func(b *testing.B) {
		buf := NewBuffer(4096)
		data := []byte("test data")

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.pos = 0
			buf.writeOptimized(data)
		}
	})
}

// BenchmarkMemoryBandwidth tests memory bandwidth utilization
func BenchmarkMemoryBandwidth(b *testing.B) {
	sizes := []int{4096, 16384, 65536, 262144, 1048576}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			src := make([]byte, size)
			dst := make([]byte, size)
			rand.Read(src)

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Use standard copy for benchmark
				copy(dst, src)
			}
		})
	}
}

// BenchmarkConcurrentOptimized tests concurrent performance
func BenchmarkConcurrentOptimized(b *testing.B) {
	b.Run("FastPool_Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := OptimizedGet(1024)
				buf[0] = 1
				OptimizedPut(buf)
			}
		})
	})

	b.Run("OptimizedWrite_Parallel", func(b *testing.B) {
		data := []byte("concurrent test data")
		b.RunParallel(func(pb *testing.PB) {
			buf := NewBuffer(4096)
			for pb.Next() {
				buf.pos = 0
				buf.writeOptimized(data)
			}
		})
	})
}

// BenchmarkRealWorld simulates real-world usage patterns
func BenchmarkRealWorld(b *testing.B) {
	// Simulate JSON encoding scenario
	b.Run("JSON_encoding", func(b *testing.B) {
		parts := []string{
			`{"id":`, `12345`, `,"name":"`, `test user`,
			`","email":"`, `test@example.com`, `","data":[`,
			`1,2,3,4,5`, `]}`,
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := OptimizedGet(256)
			pos := 0
			for _, part := range parts {
				copy(buf[pos:], part)
				pos += len(part)
			}
			result := string(buf[:pos])
			_ = result
			OptimizedPut(buf)
		}
	})

	// Simulate HTTP response building
	b.Run("HTTP_response", func(b *testing.B) {
		header := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n")
		body := []byte(`{"status":"success","data":{"id":1,"value":"test"}}`)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := NewBuffer(512)
			buf.writeOptimized(header)
			buf.writeOptimized(body)
			_ = buf.Bytes()
			buf.Reset()
		}
	})
}
