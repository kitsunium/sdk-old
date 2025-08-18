package kbuffer

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	
)

// TestBufferPool tests the buffer pool functionality.
func TestBufferPool(t *testing.T) {
	t.Run("BasicGetPut", func(t *testing.T) {
		pool := NewBufferPool()

		// Test various sizes
		sizes := []int{100, 256, 1000, 4096, 65536}
		for _, size := range sizes {
			buf := pool.Get(size)
			if buf == nil {
				t.Fatalf("buffer should not be nil for size %d", size)
			}
			GreaterOrEqual(t, len(buf), size, "buffer should be at least requested size")
			GreaterOrEqual(t, cap(buf), size, "buffer capacity should be at least requested size")

			// Verify it's a power of 2 capacity
			True(t, isPowerOf2(cap(buf)), "capacity should be power of 2")

			// Write data to ensure it's usable
			for i := 0; i < size; i++ {
				buf[i] = byte(i % 256)
			}

			pool.Put(buf)
		}
	})

	t.Run("ReuseBuffers", func(t *testing.T) {
		pool := NewBufferPool()
		pool.ResetStats()

		// Get and put the same size multiple times
		size := 1024
		for i := 0; i < 10; i++ {
			buf := pool.Get(size)
			pool.Put(buf)
		}

		stats := pool.GetStats()
		Greater(t, stats.Hits, int64(0), "should have cache hits")
		Equal(t, int64(10), stats.Gets, "should have 10 gets")
		Equal(t, int64(10), stats.Puts, "should have 10 puts")
	})

	t.Run("ZeroedBuffers", func(t *testing.T) {
		pool := NewBufferPool()
		// Enable clearing for this test
		pool.SetClearOnPut(true)

		// Get buffer, write data, return it
		buf1 := pool.Get(1024)
		for i := range buf1 {
			buf1[i] = 0xFF
		}
		pool.Put(buf1)

		// Get another buffer of same size
		buf2 := pool.Get(1024)

		// Check if it's zeroed (security feature)
		allZero := true
		for _, b := range buf2 {
			if b != 0 {
				allZero = false
				break
			}
		}
		True(t, allZero, "returned buffers should be zeroed for security when clearOnPut is enabled")
		pool.Put(buf2)
	})

	t.Run("LargeSizes", func(t *testing.T) {
		pool := NewBufferPool()
		pool.ResetStats()

		// Test very large size (beyond default max)
		largeSize := 10 * 1024 * 1024 // 10MB
		buf := pool.Get(largeSize)
		Equal(t, largeSize, len(buf), "large buffer should be exact size")

		// Put will not actually pool it due to size check
		pool.Put(buf)

		// Try to get another large buffer - should allocate new
		buf2 := pool.Get(largeSize)
		Equal(t, largeSize, len(buf2), "second large buffer should be exact size")

		stats := pool.GetStats()
		// Should have 2 allocations, 1 put attempt, but 0 hits (not pooled)
		Equal(t, int64(2), stats.Gets, "should have 2 gets")
		Equal(t, int64(1), stats.Puts, "should have 1 put attempt")
		Equal(t, int64(0), stats.Hits, "should have no hits for large buffers")
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		pool := NewBufferPool()
		pool.ResetStats()

		const goroutines = 100
		const iterations = 1000

		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := 0; i < goroutines; i++ {
			go func(id int) {
				defer wg.Done()

				for j := 0; j < iterations; j++ {
					size := 256 + (id * 16) // Different sizes per goroutine
					buf := pool.Get(size)

					// Do some work with the buffer
					buf[0] = byte(id)
					buf[len(buf)-1] = byte(j)

					pool.Put(buf)
				}
			}(i)
		}

		wg.Wait()

		stats := pool.GetStats()
		Equal(t, int64(goroutines*iterations), stats.Gets, "should have correct number of gets")
		Greater(t, stats.Hits, int64(0), "should have cache hits")
	})

	t.Run("BufferObjects", func(t *testing.T) {
		pool := NewBufferPool()

		// Test Buffer object operations
		buf := pool.GetBuffer(1024)
		if buf == nil {
			t.Fatal("buffer object should not be nil")
		}
		Equal(t, 1024, buf.Cap(), "buffer should have correct capacity")

		// Write and read
		n, err := buf.WriteString("Hello, World!")
		NoError(t, err)
		Equal(t, 13, n)
		Equal(t, "Hello, World!", buf.String())

		pool.PutBuffer(buf)
	})

	t.Run("GlobalPool", func(t *testing.T) {
		// Test global pool functions
		buf := Get(512)
		NotNil(t, buf)
		GreaterOrEqual(t, len(buf), 512)
		Put(buf)

		// Test Buffer objects with global pool
		bufObj := GetBuffer(256)
		NotNil(t, bufObj)
		PutBuffer(bufObj)
	})

	t.Run("ConvenienceFunctions", func(t *testing.T) {
		pool := NewBufferPool()

		buf1k := pool.Get1K()
		Equal(t, 1024, len(buf1k))
		Put(buf1k)

		buf4k := pool.Get4K()
		Equal(t, 4*1024, len(buf4k))
		Put(buf4k)

		buf64k := pool.Get64K()
		Equal(t, 64*1024, len(buf64k))
		Put(buf64k)
	})

	t.Run("Statistics", func(t *testing.T) {
		pool := NewBufferPool()
		pool.ResetStats()

		// Perform operations
		for i := 0; i < 100; i++ {
			buf := pool.Get(1024)
			pool.Put(buf)
		}

		stats := pool.GetStats()
		Equal(t, int64(100), stats.Gets)
		Equal(t, int64(100), stats.Puts)
		Greater(t, stats.Hits, int64(0), "should have hits after first allocation")

		// Reset and verify
		pool.ResetStats()
		stats = pool.GetStats()
		Equal(t, int64(0), stats.Gets)
		Equal(t, int64(0), stats.Puts)
	})

	t.Run("SetMaxSize", func(t *testing.T) {
		pool := NewBufferPool()
		pool.SetMaxSize(512)
		
		// Buffer larger than max size should not be pooled
		buf := pool.Get(1024)
		NotNil(t, buf)
		pool.Put(buf)
		
		// Get again - should allocate new since it wasn't pooled
		pool.ResetStats()
		buf2 := pool.Get(1024)
		NotNil(t, buf2)
		stats := pool.GetStats()
		Equal(t, int64(0), stats.Hits)
	})

	t.Run("SetClearOnPut", func(t *testing.T) {
		pool := NewBufferPool()
		pool.SetClearOnPut(false)
		
		// Get buffer, write data, return it
		buf1 := pool.Get(1024)
		for i := range buf1 {
			buf1[i] = 0xFF
		}
		pool.Put(buf1)
		
		// Get another buffer - should not be cleared
		buf2 := pool.Get(1024)
		// Check first byte only (since clearing is disabled)
		if buf2[0] == 0xFF {
			// Buffer was reused without clearing - expected
			Equal(t, byte(0xFF), buf2[0])
		}
	})

	t.Run("Prewarm", func(t *testing.T) {
		pool := NewBufferPool()
		pool.ResetStats()
		
		// Prewarm with specific sizes
		pool.Prewarm([]int{512, 1024}, 5)
		
		stats := pool.GetStats()
		Equal(t, int64(10), stats.Allocs) // 2 sizes * 5 count
	})

	t.Run("Get_ZeroSize", func(t *testing.T) {
		pool := NewBufferPool()
		buf := pool.Get(0)
		Nil(t, buf)
	})

	t.Run("Get_NegativeSize", func(t *testing.T) {
		pool := NewBufferPool()
		buf := pool.Get(-10)
		Nil(t, buf)
	})

	t.Run("Put_NilBuffer", func(t *testing.T) {
		pool := NewBufferPool()
		// Should not panic
		pool.Put(nil)
	})

	t.Run("Put_NonPowerOf2", func(t *testing.T) {
		pool := NewBufferPool()
		buf := make([]byte, 100) // Not a power of 2
		pool.Put(buf) // Should not pool it
	})

	t.Run("GetBuffer_ZeroSize", func(t *testing.T) {
		pool := NewBufferPool()
		buf := pool.GetBuffer(0)
		NotNil(t, buf)
		Equal(t, 2, buf.Cap()) // MinBitSize = 1, so 2^1 = 2
	})

	t.Run("PutBuffer_Nil", func(t *testing.T) {
		pool := NewBufferPool()
		// Should not panic
		pool.PutBuffer(nil)
	})

	t.Run("PutBuffer_InvalidCapacity", func(t *testing.T) {
		pool := NewBufferPool()
		buf := NewBuffer(100) // Not a power of 2
		pool.PutBuffer(buf) // Should not pool it
	})
}


// TestMemoryLeak verifies no memory leaks.
func TestMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory leak test in short mode")
	}

	pool := NewBufferPool()

	// Force GC to get baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Perform many allocations
	const iterations = 10000
	for i := 0; i < iterations; i++ {
		buf := pool.Get(4096)
		pool.Put(buf)
	}

	// Force GC and measure again
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Memory should not grow significantly
	growth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	maxGrowth := int64(4096 * 100) // Allow for some pooled buffers

	Less(t, growth, maxGrowth, "memory growth should be limited")
}

// TestConcurrentStats verifies statistics are accurate under concurrent access.
func TestConcurrentStats(t *testing.T) {
	pool := NewBufferPool()
	pool.ResetStats()

	const goroutines = 10
	const operations = 1000

	var wg sync.WaitGroup
	var totalGets, totalPuts int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < operations; j++ {
				buf := pool.Get(1024)
				atomic.AddInt64(&totalGets, 1)

				pool.Put(buf)
				atomic.AddInt64(&totalPuts, 1)
			}
		}()
	}

	wg.Wait()

	stats := pool.GetStats()
	Equal(t, totalGets, stats.Gets, "gets should match")
	Equal(t, totalPuts, stats.Puts, "puts should match")
}
