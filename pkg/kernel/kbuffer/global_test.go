package kbuffer

import (
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

	// Set small max size
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

	// Standard buffer
	stdBuf := NewUnsafeBuffer(256)
	pool.PutBuffer(stdBuf)

	// Sharded buffer
	shardedBuf := NewSafeShardedBuffer(1024, 4)
	pool.PutBuffer(shardedBuf) // Should pool individual shards
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

	// Try to get common sizes that should be prewarmed
	buf := pool.Get(256)
	if buf == nil {
		t.Log("Warning: Pool prewarming may not be working")
	} else {
		pool.Put(buf)
	}
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
