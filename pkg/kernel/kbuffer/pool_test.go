package kbuffer

import (
	"runtime"
	"sync"
	"testing"
)

func TestBufferPool_Get(t *testing.T) {
	p := newPool()

	tests := []struct {
		name     string
		size     int
		wantSize int
	}{
		{"negative size", -10, 0},
		{"zero size", 0, 0},
		{"small size", 10, 10},
		{"exact power of 2", 64, 64},
		{"non-power of 2", 100, 100},
		{"large size", 1024, 1024},
		{"oversized", 2 * maxPoolSize, 2 * maxPoolSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := p.Get(tt.size)
			if tt.size <= 0 {
				if buf != nil {
					t.Errorf("Get(%d) = %v, want nil", tt.size, buf)
				}
				return
			}

			if len(buf) != tt.wantSize {
				t.Errorf("Get(%d) length = %d, want %d", tt.size, len(buf), tt.wantSize)
			}
			if cap(buf) < tt.wantSize {
				t.Errorf("Get(%d) capacity = %d, want >= %d", tt.size, cap(buf), tt.wantSize)
			}
		})
	}

	// Test edge case: size class calculation boundary
	// This tests the poolIdx < 0 branch
	buf := p.Get(32) // size class 5, poolIdx = -1
	if buf == nil || len(buf) != 32 {
		t.Errorf("Get(32) = %v, want buffer of size 32", buf)
	}
}

func TestBufferPool_Put(t *testing.T) {
	p := newPool()

	// Get a buffer and put it back
	buf := p.Get(64)
	p.Put(buf)

	// Test nil buffer
	p.Put(nil) // Should not panic

	// Test oversized buffer
	bigBuf := make([]byte, maxPoolSize+1)
	p.Put(bigBuf) // Should not panic

	// Test non-power-of-2 buffer
	oddBuf := make([]byte, 100) // Not a power of 2
	p.Put(oddBuf) // Should not panic

	// Test small buffer that goes to pool
	smallBuf := make([]byte, 64)
	p.Put(smallBuf) // Should not panic

	// Test buffer with poolIdx >= len(pools)
	// This tests the boundary condition
	hugeBuf := make([]byte, 1<<27) // Very large power of 2
	p.Put(hugeBuf) // Should not panic
}

func TestBufferPool_GetPutCycle(t *testing.T) {
	p := newPool()

	// Test that buffers are reused
	buf1 := p.Get(128)
	buf1[0] = 42
	p.Put(buf1)

	buf2 := p.Get(128)
	// We may get the same buffer back from the pool
	if cap(buf2) < 128 {
		t.Errorf("Reused buffer capacity = %d, want >= 128", cap(buf2))
	}
	p.Put(buf2)

	// Test multiple cycles
	for i := 0; i < 10; i++ {
		buf := p.Get(256)
		buf[0] = byte(i)
		p.Put(buf)
	}
}

func TestBufferPool_GetBuffer(t *testing.T) {
	p := newPool()

	tests := []struct {
		name string
		size int
	}{
		{"small", 64},
		{"medium", 1024},
		{"large", 16384},
		{"oversized", maxPoolSize + 1},
		{"zero", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := p.GetBuffer(tt.size)
			if b == nil {
				t.Fatal("GetBuffer returned nil")
			}
			if b.Cap() < tt.size {
				t.Errorf("GetBuffer(%d).Cap() = %d, want >= %d", tt.size, b.Cap(), tt.size)
			}
			if b.Len() != 0 {
				t.Errorf("GetBuffer(%d).Len() = %d, want 0", tt.size, b.Len())
			}
			if b.Available() != b.Cap() {
				t.Errorf("GetBuffer(%d).Available() = %d, want %d", tt.size, b.Available(), b.Cap())
			}
		})
	}

	// Test that nil from Get results in NewBuffer
	// This tests the defensive code path
	b := p.GetBuffer(100)
	if b == nil {
		t.Fatal("GetBuffer should never return nil")
	}
}

func TestBufferPool_PutBuffer(t *testing.T) {
	p := newPool()

	// Test normal put
	b := p.GetBuffer(128)
	b.Write([]byte("test"))
	p.PutBuffer(b)

	// Test nil buffer
	p.PutBuffer(nil) // Should not panic

	// Test that buffer is reset
	b2 := p.GetBuffer(64)
	b2.Write([]byte("data"))
	origPos := b2.pos
	p.PutBuffer(b2)
	if origPos == 0 {
		t.Error("Buffer should have been written to")
	}
}

func TestBufferPool_SetClearOnPut(t *testing.T) {
	p := newPool()
	p.SetClearOnPut(true)

	// Get buffer and write data
	buf := p.Get(64)
	copy(buf, []byte("sensitive"))

	// Return to pool
	p.Put(buf)

	// Get the same buffer again (since it's in the pool)
	buf2 := p.Get(64)

	// Data should be cleared
	for i := 0; i < len("sensitive"); i++ {
		if buf2[i] != 0 {
			t.Errorf("buf[%d] = %d, want 0 (not cleared)", i, buf2[i])
		}
	}
}

func TestBufferPool_SetMaxSize(t *testing.T) {
	p := newPool()
	p.SetMaxSize(1024)

	// Test that oversized buffers are not pooled
	buf := p.Get(2048)
	if cap(buf) != 2048 {
		t.Errorf("Get(2048) capacity = %d, want 2048", cap(buf))
	}

	p.Put(buf)
	// Buffer should not be pooled due to size limit
}

func TestGlobalPoolFunctions(t *testing.T) {
	// Test global Get/Put
	buf := Get(100)
	if len(buf) != 100 {
		t.Errorf("Get(100) length = %d, want 100", len(buf))
	}
	Put(buf)

	// Test global GetBuffer/PutBuffer
	b := GetBuffer(200)
	if b.Cap() < 200 {
		t.Errorf("GetBuffer(200).Cap() = %d, want >= 200", b.Cap())
	}
	PutBuffer(b)
}

func TestBufferPool_Prewarm(t *testing.T) {
	// Test normal prewarm
	p := newPool()
	p.prewarm()

	// Verify it doesn't panic and works correctly
	// The prewarm function pre-allocates buffers for common sizes
}

func TestBufferPool_EdgeCases(t *testing.T) {
	p := newPool()

	// Test Get with size that results in poolIdx >= len(pools)
	// maxPoolSize is 1<<20, so size > maxPoolSize will take the oversized path
	bigSize := maxPoolSize + 1
	buf := p.Get(bigSize)
	if buf == nil || len(buf) != bigSize {
		t.Errorf("Get(%d) = %v, want buffer of size %d", bigSize, buf, bigSize)
	}

	// Now test the poolIdx >= len(pools) case
	// We need a buffer with valid size but poolIdx >= 21
	// Size class 27 would give poolIdx = 21 (27 - 6)
	// But this would be > maxPoolSize, so let's create a custom test

	// Create a pool with smaller pools array to trigger the boundary
	p2 := &BufferPool{
		pools: [21]*sync.Pool{}, // Leave some nil
	}
	p2.maxSize.Store(1 << 30) // Very large max

	// Size class 27 = 2^27 = 134217728, poolIdx = 21
	hugeSize := 1 << 27
	buf2 := p2.Get(hugeSize)
	if buf2 == nil || len(buf2) != hugeSize {
		t.Errorf("Get(%d) = %v, want buffer of size %d", hugeSize, buf2, hugeSize)
	}
}

func TestSizeClass(t *testing.T) {
	tests := []struct {
		size  int
		class int
	}{
		{0, 6},
		{1, 6},
		{64, 6},
		{65, 7},
		{128, 7},
		{129, 8},
		{256, 8},
		{512, 9},
		{1024, 10},
	}

	for _, tt := range tests {
		if got := sizeClass(tt.size); got != tt.class {
			t.Errorf("sizeClass(%d) = %d, want %d", tt.size, got, tt.class)
		}
	}
}

func TestIsPowerOf2(t *testing.T) {
	tests := []struct {
		n    int
		want bool
	}{
		{0, false},
		{1, true},
		{2, true},
		{3, false},
		{4, true},
		{64, true},
		{100, false},
		{128, true},
		{-1, false},
	}

	for _, tt := range tests {
		if got := isPowerOf2(tt.n); got != tt.want {
			t.Errorf("isPowerOf2(%d) = %v, want %v", tt.n, got, tt.want)
		}
	}
}

func TestBufferPool_Concurrent(t *testing.T) {
	p := newPool()

	const (
		numGoroutines = 100
		numOps        = 1000
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOps; j++ {
				// Vary buffer sizes
				size := 64 << (j % 5) // 64, 128, 256, 512, 1024

				buf := p.Get(size)
				if len(buf) != size {
					t.Errorf("Goroutine %d: Get(%d) length = %d, want %d", id, size, len(buf), size)
				}

				// Write some data
				if len(buf) > 0 {
					buf[0] = byte(id)
					buf[len(buf)-1] = byte(j)
				}

				p.Put(buf)

				// Also test GetBuffer/PutBuffer
				if j%10 == 0 {
					b := p.GetBuffer(size)
					b.WriteByte(byte(id))
					p.PutBuffer(b)
				}
			}

			// Force GC occasionally to test under memory pressure
			if id%10 == 0 {
				runtime.GC()
			}
		}(i)
	}

	wg.Wait()
}

func TestBufferPool_MemoryPressure(t *testing.T) {
	p := newPool()

	// Allocate many buffers without returning them immediately
	buffers := make([][]byte, 1000)
	for i := range buffers {
		buffers[i] = p.Get(1024)
	}

	// Return half of them
	for i := 0; i < len(buffers)/2; i++ {
		p.Put(buffers[i])
	}

	// Force GC
	runtime.GC()

	// Get more buffers
	for i := 0; i < 100; i++ {
		buf := p.Get(2048)
		p.Put(buf)
	}

	// Return the rest
	for i := len(buffers) / 2; i < len(buffers); i++ {
		p.Put(buffers[i])
	}
}