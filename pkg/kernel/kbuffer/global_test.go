package kbuffer

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

// TestGlobalPool tests the global pool singleton.
func TestGlobalPool(t *testing.T) {
	// Get global pool
	pool := GetGlobalPool()
	if pool == nil {
		t.Fatal("GetGlobalPool() returned nil")
	}

	// Verify it's a singleton
	pool2 := GetGlobalPool()
	if pool != pool2 {
		t.Error("GetGlobalPool() not returning singleton")
	}
}

// TestPoolGetPut tests basic pool get/put operations.
func TestPoolGetPut(t *testing.T) {
	pool := GetGlobalPool()

	t.Run("BasicGetPut", func(t *testing.T) {
		// Get buffer
		buf := pool.Get(256)
		if buf == nil {
			t.Fatal("Get(256) returned nil")
		}
		if cap(buf) < 256 {
			t.Errorf("Get(256) capacity = %d, want >= 256", cap(buf))
		}

		// Put buffer back
		pool.Put(buf)

		// Get again - might get same buffer back
		buf2 := pool.Get(256)
		if buf2 == nil {
			t.Fatal("Get(256) after Put returned nil")
		}
	})

	t.Run("GetZeroSize", func(t *testing.T) {
		// Get with zero size should return nil
		buf := pool.Get(0)
		if buf != nil {
			t.Errorf("Get(0) = %v, want nil", buf)
		}
	})

	t.Run("GetNegativeSize", func(t *testing.T) {
		// Get with negative size should return nil
		buf := pool.Get(-10)
		if buf != nil {
			t.Errorf("Get(-10) = %v, want nil", buf)
		}
	})

	t.Run("GetOversizedBuffer", func(t *testing.T) {
		// Get buffer larger than max pool size
		buf := pool.Get(poolMaxSize + 1)
		if buf == nil {
			t.Fatal("Get(poolMaxSize+1) returned nil")
		}
		if cap(buf) < poolMaxSize+1 {
			t.Errorf("Get(poolMaxSize+1) capacity = %d, want >= %d", cap(buf), poolMaxSize+1)
		}
		// Put back - should not be pooled
		pool.Put(buf)
	})

	t.Run("PutNilBuffer", func(t *testing.T) {
		// Put nil should not panic
		pool.Put(nil)
	})

	t.Run("PutZeroCapBuffer", func(t *testing.T) {
		// Put buffer with zero capacity should not panic
		buf := make([]byte, 0, 0)
		pool.Put(buf)
	})

	t.Run("PutOversizedBuffer", func(t *testing.T) {
		// Put buffer larger than max size - should not be pooled
		largeBuf := make([]byte, poolMaxSize+1)
		pool.Put(largeBuf)
	})

	t.Run("PutNonPowerOfTwoBuffer", func(t *testing.T) {
		// Put buffer with non-power-of-2 capacity
		buf := make([]byte, 0, 100) // 100 is not power of 2
		pool.Put(buf)
	})

	t.Run("GetInvalidPoolIndex", func(t *testing.T) {
		// Test edge case where calculated pool index might be invalid
		// This tests the poolIdx < 0 branch
		buf := pool.Get(32) // Below poolMinSize, will have negative poolIdx
		if buf == nil {
			t.Fatal("Get(32) returned nil")
		}
		if cap(buf) < 32 {
			t.Errorf("Get(32) capacity = %d, want >= 32", cap(buf))
		}
	})

	t.Run("PutInvalidPoolIndex", func(t *testing.T) {
		// Test Put with buffer that would have invalid pool index
		buf := make([]byte, 0, 32) // Below poolMinSize
		pool.Put(buf)              // Should handle gracefully

		// Also test with a very large power-of-2 that exceeds pool count
		veryLargeBuf := make([]byte, 0, 1<<30) // Way beyond poolClassCount
		pool.Put(veryLargeBuf)                 // Should handle gracefully
	})
}

// TestPoolSizeClasses tests power-of-2 size class allocation.
func TestPoolSizeClasses(t *testing.T) {
	pool := GetGlobalPool()

	tests := []struct {
		request int
		minCap  int
	}{
		{50, 64},     // Round up to 64
		{64, 64},     // Exact
		{65, 128},    // Round up to 128
		{200, 256},   // Round up to 256
		{1000, 1024}, // Round up to 1024
		{5000, 8192}, // Round up to 8192
	}

	for _, tt := range tests {
		buf := pool.Get(tt.request)
		if cap(buf) < tt.minCap {
			t.Errorf("Get(%d) capacity = %d, want >= %d", tt.request, cap(buf), tt.minCap)
		}
		pool.Put(buf)
	}
}

// TestPoolGetBuffer tests Buffer instance pooling.
func TestPoolGetBuffer(t *testing.T) {
	pool := GetGlobalPool()

	t.Run("BasicGetBuffer", func(t *testing.T) {
		// Get buffer instance
		buf := pool.GetBuffer(512)
		if buf == nil {
			t.Fatal("GetBuffer(512) returned nil")
		}
		if buf.Cap() < 512 {
			t.Errorf("GetBuffer(512) capacity = %d, want >= 512", buf.Cap())
		}

		// Write some data
		buf.Write([]byte("test"))

		// Put back to pool
		pool.PutBuffer(buf)

		// Get another - should be reset
		buf2 := pool.GetBuffer(512)
		if buf2.Len() != 0 {
			t.Errorf("GetBuffer after Put has Len() = %d, want 0", buf2.Len())
		}
	})

	t.Run("GetBufferNilData", func(t *testing.T) {
		// Test GetBuffer when Get returns nil (edge case)
		// This happens when size <= 0
		buf := pool.GetBuffer(0)
		if buf == nil {
			t.Fatal("GetBuffer(0) returned nil")
		}
		// Should fallback to newSafeBuffer
	})

	t.Run("GetBufferPooledFlag", func(t *testing.T) {
		// Verify pooled flag is set correctly
		buf := pool.GetBuffer(256)
		// We can't directly check the pooled flag, but we can verify
		// the buffer works correctly when returned to pool
		buf.Write([]byte("test"))
		pool.PutBuffer(buf)
	})
}

// TestPoolClearOnPut tests security clearing configuration.
func TestPoolClearOnPut(t *testing.T) {
	pool := GetGlobalPool()

	// Enable clearing
	pool.SetClearOnPut(true)
	defer pool.SetClearOnPut(false) // Reset after test

	// Get buffer and write sensitive data
	buf := pool.Get(256)
	sensitive := []byte("password123")
	copy(buf, sensitive)

	// Put back
	pool.Put(buf)

	// The buffer should be cleared (but we can't directly verify this
	// as it might be reallocated to someone else)
	// Just verify the setting works without error
}

// TestPoolMaxSize tests maximum pooled size limit.
func TestPoolMaxSize(t *testing.T) {
	pool := GetGlobalPool()

	// Test SetMaxSize with various edge cases
	t.Run("BelowMinimum", func(t *testing.T) {
		// Set size below minimum - should clamp to poolMinSize
		pool.SetMaxSize(32) // Below poolMinSize (64)
		// The actual test is that it doesn't panic and works correctly
		buf := pool.Get(64)
		if buf == nil {
			t.Fatal("Get(64) after SetMaxSize(32) returned nil")
		}
		pool.Put(buf)
	})

	t.Run("AboveMaximum", func(t *testing.T) {
		// Set size above maximum - should clamp to poolMaxSize
		pool.SetMaxSize(1 << 30) // Way above poolMaxSize
		// The actual test is that it doesn't panic and works correctly
		buf := pool.Get(1024)
		if buf == nil {
			t.Fatal("Get(1024) after SetMaxSize(1<<30) returned nil")
		}
		pool.Put(buf)
	})

	t.Run("NormalRange", func(t *testing.T) {
		// Set small max size within valid range
		oldMax := poolMaxSize
		pool.SetMaxSize(1024)
		defer pool.SetMaxSize(int64(oldMax)) // Reset after test

		// Large buffer should not be pooled
		largeBuf := pool.Get(2048)
		if largeBuf == nil {
			t.Fatal("Get(2048) returned nil")
		}

		// Put back - should not actually be pooled due to size
		pool.Put(largeBuf)
	})
}

// TestPoolStatistics tests pool basic functionality.
func TestPoolStatistics(t *testing.T) {
	pool := GetGlobalPool()

	// Perform operations
	buf1 := pool.Get(256)
	buf2 := pool.Get(512)

	// Verify buffers are not nil
	if buf1 == nil {
		t.Error("Get(256) returned nil")
	}
	if buf2 == nil {
		t.Error("Get(512) returned nil")
	}

	// Return buffers to pool
	pool.Put(buf1)
	pool.Put(buf2)
}

// TestPoolConcurrent tests concurrent pool access.
func TestPoolConcurrent(t *testing.T) {
	pool := GetGlobalPool()
	var wg sync.WaitGroup

	// Many goroutines getting/putting buffers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				// Vary sizes
				size := 64 << (j % 5)

				// Get buffer
				buf := pool.Get(size)
				if buf == nil {
					t.Errorf("Goroutine %d: Get(%d) returned nil", id, size)
					continue
				}

				// Use buffer
				for k := 0; k < len(buf) && k < 10; k++ {
					buf[k] = byte(id + j)
				}

				// Put back
				pool.Put(buf)

				// Yield occasionally
				if j%10 == 0 {
					runtime.Gosched()
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestPoolBufferTypes tests pooling different buffer types.
func TestPoolBufferTypes(t *testing.T) {
	pool := GetGlobalPool()

	t.Run("NilBuffer", func(t *testing.T) {
		// PutBuffer with nil should not panic
		pool.PutBuffer(nil)
	})

	t.Run("UnsafeBuffer", func(t *testing.T) {
		// Standard unsafe buffer
		stdBuf := NewUnsafeBuffer(256)
		stdBuf.Write([]byte("test data"))
		pool.PutBuffer(stdBuf)
	})

	t.Run("SafeBuffer", func(t *testing.T) {
		// Safe buffer
		safeBuf := NewSafeBuffer(256)
		safeBuf.Write([]byte("test data"))
		pool.PutBuffer(safeBuf)
	})

	t.Run("SafeShardedBuffer", func(t *testing.T) {
		// Sharded buffer - should pool individual shards
		shardedBuf := NewSafeShardedBuffer(1024, 4)
		shardedBuf.Write([]byte("test data"))
		pool.PutBuffer(shardedBuf)
	})

	t.Run("UnsafeShardedBuffer", func(t *testing.T) {
		// Unsafe sharded buffer
		unsafeShardedBuf := NewUnsafeShardedBuffer(1024, 4)
		unsafeShardedBuf.Write([]byte("test data"))
		pool.PutBuffer(unsafeShardedBuf)
	})

	t.Run("PooledSafeBuffer", func(t *testing.T) {
		// Test PutBuffer with a pooled safe buffer
		buf := pool.GetBuffer(256)
		buf.Write([]byte("test"))
		pool.PutBuffer(buf)
	})

	t.Run("NonPooledSafeBuffer", func(t *testing.T) {
		// Test PutBuffer with non-pooled safe buffer (no data pointer)
		safeBuf := &safeBuffer{
			cap:    256,
			pooled: false,
			data:   nil, // No data pointer
		}
		pool.PutBuffer(safeBuf)
	})

	t.Run("NonPooledUnsafeBuffer", func(t *testing.T) {
		// Test PutBuffer with non-pooled unsafe buffer (no data pointer)
		unsafeBuf := &unsafeBuffer{
			cap:    256,
			pooled: false,
			data:   nil, // No data pointer
		}
		pool.PutBuffer(unsafeBuf)
	})
}

// TestSizeToClass tests size class calculation.
func TestSizeToClass(t *testing.T) {
	tests := []struct {
		size  int
		class int
	}{
		{0, 6},     // Minimum
		{1, 6},     // Minimum
		{64, 6},    // 2^6
		{65, 7},    // 2^7
		{128, 7},   // 2^7
		{129, 8},   // 2^8
		{256, 8},   // 2^8
		{512, 9},   // 2^9
		{1024, 10}, // 2^10
		{2048, 11}, // 2^11
		{4096, 12}, // 2^12
	}

	for _, tt := range tests {
		got := sizeToClass(tt.size)
		if got != tt.class {
			t.Errorf("sizeToClass(%d) = %d, want %d", tt.size, got, tt.class)
		}
	}
}

// TestIsPowerOfTwo tests power of 2 checking.
func TestIsPowerOfTwo(t *testing.T) {
	tests := []struct {
		n    int
		want bool
	}{
		{0, false},
		{1, true},
		{2, true},
		{3, false},
		{4, true},
		{5, false},
		{7, false},
		{8, true},
		{15, false},
		{16, true},
		{64, true},
		{100, false},
		{128, true},
		{256, true},
		{1024, true},
	}

	for _, tt := range tests {
		got := isPowerOfTwo(tt.n)
		if got != tt.want {
			t.Errorf("isPowerOfTwo(%d) = %v, want %v", tt.n, got, tt.want)
		}
	}
}

// TestPackageLevelFunctions tests global convenience functions.
func TestPackageLevelFunctions(t *testing.T) {
	// Test Get/Put
	buf := Get(256)
	if buf == nil {
		t.Fatal("Get(256) returned nil")
	}
	Put(buf)

	// Test GetBuffer/PutBuffer
	buffer := GetBuffer(512)
	if buffer == nil {
		t.Fatal("GetBuffer(512) returned nil")
	}
	PutBuffer(buffer)

	// Test configuration functions
	SetGlobalClearOnPut(true)
	SetGlobalClearOnPut(false)

	SetGlobalMaxSize(1024)
	SetGlobalMaxSize(poolMaxSize)

	// Stats functionality has been removed for performance
}

// TestPoolPrewarm tests pool prewarming.
func TestPoolPrewarm(t *testing.T) {
	// The pool is prewarmed during initialization
	// We can verify by getting and putting buffers
	pool := GetGlobalPool()

	// Test all prewarmed sizes
	sizes := []int{256, 1024, 4096, 16384, 65536, 262144}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size%d", size), func(t *testing.T) {
			buf := pool.Get(size)
			if buf == nil {
				t.Errorf("Get(%d) returned nil after prewarming", size)
			} else {
				pool.Put(buf)
			}
		})
	}

	// Test prewarm edge cases by calling it directly
	t.Run("PrewarmEdgeCases", func(t *testing.T) {
		// Create a new pool to test prewarm with edge cases
		p := &bufferPool{}
		p.maxSize.Store(poolMaxSize)
		p.clearOnPut.Store(false)

		// Initialize pools
		for i := 0; i < poolClassCount; i++ {
			size := 1 << (i + 6)
			poolIdx := i
			p.pools[poolIdx] = &sync.Pool{
				New: func(sz int) func() any {
					return func() any {
						buf := make([]byte, sz)
						return &buf
					}
				}(size),
			}
		}

		// Call prewarm - should handle edge cases gracefully
		p.prewarm()

		// Verify pool is functional
		buf := p.Get(256)
		if buf == nil {
			t.Error("Pool not functional after prewarm")
		} else {
			p.Put(buf)
		}
	})
}

// TestPoolNonPowerOfTwo tests handling of non-power-of-2 buffers.
func TestPoolNonPowerOfTwo(t *testing.T) {
	pool := GetGlobalPool()

	// Create non-power-of-2 buffer (shouldn't be pooled)
	buf := make([]byte, 100) // 100 is not power of 2

	// This should just be ignored without errors
	pool.Put(buf)

	// Verify pool still works after
	testBuf := pool.Get(128)
	if testBuf == nil {
		t.Error("Pool not working after non-power-of-2 put")
	} else {
		pool.Put(testBuf)
	}
}

// TestPoolStress performs stress testing on the pool.
func TestPoolStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	pool := GetGlobalPool()
	var wg sync.WaitGroup

	// Many goroutines, many operations
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Keep some buffers alive across iterations
			held := make([][]byte, 0, 10)

			for j := 0; j < 1000; j++ {
				switch j % 5 {
				case 0, 1:
					// Get and immediately put
					size := 64 << (j % 8)
					buf := pool.Get(size)
					pool.Put(buf)

				case 2:
					// Get and hold
					size := 256 << (j % 4)
					buf := pool.Get(size)
					held = append(held, buf)

				case 3:
					// Release held buffers
					for _, buf := range held {
						pool.Put(buf)
					}
					held = held[:0]

				case 4:
					// Buffer operations
					buf := pool.GetBuffer(1024)
					buf.Write([]byte("test"))
					pool.PutBuffer(buf)
				}

				// Yield occasionally
				if j%50 == 0 {
					runtime.Gosched()
				}
			}

			// Clean up remaining held buffers
			for _, buf := range held {
				pool.Put(buf)
			}
		}(i)
	}

	wg.Wait()

	// Pool should still be functional
	finalBuf := pool.Get(256)
	if finalBuf == nil {
		t.Error("Pool not functional after stress test")
	}
	pool.Put(finalBuf)
}

// TestPoolPanicRecovery tests that the pool handles panics gracefully.
func TestPoolPanicRecovery(t *testing.T) {
	pool := GetGlobalPool()

	t.Run("RecoverFromInvalidTypeAssertion", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				// Pool should handle invalid types gracefully
			}
		}()

		// Try to put a buffer with invalid type - should be handled gracefully
		var invalidBuf Buffer
		pool.PutBuffer(invalidBuf)
	})

	t.Run("RecoverFromNilDereference", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Pool operations should not panic on nil: %v", r)
			}
		}()

		// These should not panic
		pool.Put(nil)
		pool.PutBuffer(nil)
	})

	t.Run("RecoverFromInvalidPoolIndex", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Pool should handle invalid indices gracefully: %v", r)
			}
		}()

		// Test with size that would produce invalid pool index
		buf := pool.Get(1 << 30) // Very large size
		if buf != nil {
			pool.Put(buf)
		}
	})
}

// Benchmarks for global pool operations

// BenchmarkPoolGet benchmarks buffer retrieval from pool.
func BenchmarkPoolGet(b *testing.B) {
	pool := GetGlobalPool()
	sizes := []int{64, 256, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				buf := pool.Get(size)
				_ = buf
			}
		})
	}
}

// BenchmarkPoolPut benchmarks buffer return to pool.
func BenchmarkPoolPut(b *testing.B) {
	pool := GetGlobalPool()
	sizes := []int{64, 256, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			// Pre-allocate buffers
			bufs := make([][]byte, b.N)
			for i := 0; i < b.N; i++ {
				bufs[i] = make([]byte, size)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				pool.Put(bufs[i])
			}
		})
	}
}

// BenchmarkPoolGetPut benchmarks complete get/put cycle.
func BenchmarkPoolGetPut(b *testing.B) {
	pool := GetGlobalPool()
	sizes := []int{64, 256, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				buf := pool.Get(size)
				pool.Put(buf)
			}
		})
	}
}

// BenchmarkPoolGetBuffer benchmarks Buffer instance retrieval.
func BenchmarkPoolGetBuffer(b *testing.B) {
	pool := GetGlobalPool()
	sizes := []int{256, 1024, 4096, 16384}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				buf := pool.GetBuffer(size)
				_ = buf
			}
		})
	}
}

// BenchmarkPoolPutBuffer benchmarks Buffer instance return.
func BenchmarkPoolPutBuffer(b *testing.B) {
	pool := GetGlobalPool()
	sizes := []int{256, 1024, 4096, 16384}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			// Pre-allocate buffers
			bufs := make([]Buffer, b.N)
			for i := 0; i < b.N; i++ {
				bufs[i] = NewSafeBuffer(size)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				pool.PutBuffer(bufs[i])
			}
		})
	}
}

// BenchmarkPoolGetPutBuffer benchmarks complete Buffer get/put cycle.
func BenchmarkPoolGetPutBuffer(b *testing.B) {
	pool := GetGlobalPool()
	sizes := []int{256, 1024, 4096, 16384}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				buf := pool.GetBuffer(size)
				pool.PutBuffer(buf)
			}
		})
	}
}

// BenchmarkPoolConcurrent benchmarks concurrent pool operations.
func BenchmarkPoolConcurrent(b *testing.B) {
	pool := GetGlobalPool()
	sizes := []int{256, 1024, 4096}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					buf := pool.Get(size)
					pool.Put(buf)
				}
			})
		})
	}
}

// BenchmarkPoolConcurrentBuffer benchmarks concurrent Buffer operations.
func BenchmarkPoolConcurrentBuffer(b *testing.B) {
	pool := GetGlobalPool()
	sizes := []int{256, 1024, 4096}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					buf := pool.GetBuffer(size)
					buf.Write([]byte("test"))
					pool.PutBuffer(buf)
				}
			})
		})
	}
}

// BenchmarkSizeToClass benchmarks size class calculation.
func BenchmarkSizeToClass(b *testing.B) {
	sizes := []int{32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = sizeToClass(size)
			}
		})
	}
}

// BenchmarkIsPowerOfTwo benchmarks power-of-2 checking.
func BenchmarkIsPowerOfTwo(b *testing.B) {
	values := []int{0, 1, 2, 3, 4, 7, 8, 16, 31, 32, 64, 100, 128, 256, 1024}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, v := range values {
			_ = isPowerOfTwo(v)
		}
	}
}

// BenchmarkPackageLevelFunctions benchmarks package-level convenience functions.
func BenchmarkPackageLevelFunctions(b *testing.B) {
	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = Get(256)
		}
	})

	b.Run("GetPut", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := Get(256)
			Put(buf)
		}
	})

	b.Run("GetBuffer", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = GetBuffer(256)
		}
	})

	b.Run("GetPutBuffer", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := GetBuffer(256)
			PutBuffer(buf)
		}
	})
}
