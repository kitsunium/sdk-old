package kbuffer

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// =============================================================================
// Multi-Core Benchmarks - Only safe concurrent operations
// =============================================================================

// BenchmarkParallel_ShardedBuffer_Write benchmarks concurrent writes to sharded buffer
func BenchmarkParallel_ShardedBuffer_Write(b *testing.B) {
	buf := NewSafeShardedBuffer(1000000, 16)
	data := make([]byte, 100)

	b.SetBytes(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf.Write(data)
			if buf.Available() < 1000 {
				buf.Reset()
			}
		}
	})
}

// BenchmarkParallel_ShardedBuffer_Mixed benchmarks mixed operations
func BenchmarkParallel_ShardedBuffer_Mixed(b *testing.B) {
	buf := NewSafeShardedBuffer(1000000, 16)
	data := []byte("test data")

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 3 {
			case 0:
				buf.Write(data)
			case 1:
				buf.WriteString("string data")
			case 2:
				_ = buf.Len()
			}
			if buf.Available() < 1000 {
				buf.Reset()
			}
			i++
		}
	})
}

// BenchmarkParallel_Pool_GetPut benchmarks concurrent pool operations
func BenchmarkParallel_Pool_GetPut(b *testing.B) {
	pool := NewPool()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(1024)
			// Simulate some work
			for i := 0; i < 10 && i < len(buf); i++ {
				buf[i] = byte(i)
			}
			pool.Put(buf)
		}
	})
}

// BenchmarkParallel_Pool_BufferGetPut benchmarks concurrent buffer pool operations
func BenchmarkParallel_Pool_BufferGetPut(b *testing.B) {
	pool := NewPool()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.GetBuffer(1024)
			buf.Write([]byte("test"))
			pool.PutBuffer(buf)
		}
	})
}

// BenchmarkParallel_Pool_Contention benchmarks pool under contention
func BenchmarkParallel_Pool_Contention(b *testing.B) {
	pool := NewPool()
	sizes := []int{64, 256, 1024, 4096}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			size := sizes[i%len(sizes)]
			buf := pool.Get(size)
			// Minimal work
			if len(buf) > 0 {
				buf[0] = byte(i)
			}
			pool.Put(buf)
			i++
		}
	})
}

// BenchmarkParallel_ShardedBuffer_Balance benchmarks concurrent balance operations
func BenchmarkParallel_ShardedBuffer_Balance(b *testing.B) {
	buf := NewSafeShardedBuffer(100000, 16)
	data := make([]byte, 100)

	// Create some imbalance
	for i := 0; i < 100; i++ {
		buf.WriteToShard(0, data)
	}

	var wg sync.WaitGroup
	var writeOps atomic.Int64

	// Start writers
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				buf.Write(data)
				writeOps.Add(1)
			}
		}()
	}

	// Benchmark balance operations
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf.Balance()
		}
	})

	wg.Wait()
	b.ReportMetric(float64(writeOps.Load()), "writes_during_balance")
}

// BenchmarkParallel_UnsafeBuffer_SingleWriter benchmarks unsafe buffer with single writer
func BenchmarkParallel_UnsafeBuffer_SingleWriter(b *testing.B) {
	if testing.Short() || runtime.GOOS == "js" {
		b.Skip("Skipping unsafe parallel benchmark in short/js mode")
	}

	buf := NewUnsafeBuffer(100000)
	var mu sync.Mutex
	data := make([]byte, 100)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%100 == 0 {
				// Only 1% writes, with mutex protection
				mu.Lock()
				buf.Write(data)
				if buf.Available() < 100 {
					buf.Reset()
				}
				mu.Unlock()
			} else {
				// 99% reads (safe without mutex)
				_ = buf.Len()
			}
			i++
		}
	})
}

// NOTE: We explicitly DO NOT include benchmarks for:
// - Unsafe buffers with concurrent writes (data races)
// - Complex resize operations under concurrency
// These are unsafe and will cause data races
