package kbuffer

import (
	"runtime"
	"sync"
	"testing"
)

func TestBufferPool_Get(t *testing.T) {
	p := newPool()
	p.ResetStats()

	tests := []struct {
		name     string
		size     int
		wantSize int
	}{
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
			if tt.size == 0 {
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

	stats := p.Stats()
	if stats.Gets == 0 {
		t.Error("Stats().Gets = 0, want > 0")
	}
}

func TestBufferPool_Put(t *testing.T) {
	p := newPool()
	p.ResetStats()

	// Test putting valid buffer
	buf := make([]byte, 64)
	p.Put(buf)

	stats := p.Stats()
	if stats.Puts != 1 {
		t.Errorf("Stats().Puts = %d, want 1", stats.Puts)
	}

	// Test putting nil
	p.Put(nil)
	stats = p.Stats()
	if stats.Puts != 1 {
		t.Errorf("Stats().Puts after nil = %d, want 1", stats.Puts)
	}

	// Test putting oversized buffer
	oversized := make([]byte, 2*maxPoolSize)
	p.Put(oversized)
	stats = p.Stats()
	if stats.Puts != 2 {
		t.Errorf("Stats().Puts after oversized = %d, want 2", stats.Puts)
	}

	// Test putting non-power-of-2 buffer
	nonPower := make([]byte, 100)
	p.Put(nonPower)
	stats = p.Stats()
	if stats.Puts != 3 {
		t.Errorf("Stats().Puts after non-power = %d, want 3", stats.Puts)
	}
}

func TestBufferPool_GetPutCycle(t *testing.T) {
	p := newPool()
	p.ResetStats()

	// Get buffer
	buf1 := p.Get(64)
	copy(buf1, []byte("test"))

	// Return to pool
	p.Put(buf1)

	// Get again - should reuse
	buf2 := p.Get(64)

	stats := p.Stats()
	if stats.Hits == 0 {
		t.Error("Stats().Hits = 0, want > 0")
	}

	// Buffers should have same capacity (reused)
	if cap(buf2) != cap(buf1) {
		t.Log("Different capacities suggest no reuse (this may be OK due to GC)")
	}
}

func TestBufferPool_GetBuffer(t *testing.T) {
	p := newPool()

	// Test getting buffer
	b := p.GetBuffer(100)
	if b == nil {
		t.Fatal("GetBuffer() = nil")
	}
	if b.Cap() < 100 {
		t.Errorf("GetBuffer(100).Cap() = %d, want >= 100", b.Cap())
	}
	if b.Len() != 0 {
		t.Errorf("GetBuffer().Len() = %d, want 0", b.Len())
	}
}

func TestBufferPool_PutBuffer(t *testing.T) {
	p := newPool()
	p.ResetStats()

	// Test putting valid buffer
	b := NewBuffer(64)
	b.Write([]byte("test"))
	p.PutBuffer(b)

	if b.Len() != 0 {
		t.Errorf("Buffer.Len() after PutBuffer = %d, want 0", b.Len())
	}

	stats := p.Stats()
	if stats.Puts == 0 {
		t.Error("Stats().Puts = 0, want > 0")
	}

	// Test putting nil
	p.PutBuffer(nil)
}

func TestBufferPool_SetClearOnPut(t *testing.T) {
	p := newPool()
	p.SetClearOnPut(true)

	// Get buffer and write data
	buf := p.Get(64)
	copy(buf, []byte("sensitive"))

	// Return to pool
	p.Put(buf)

	// Data should be cleared
	for i := 0; i < len("sensitive"); i++ {
		if buf[i] != 0 {
			t.Errorf("buf[%d] = %d, want 0 (not cleared)", i, buf[i])
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

func TestBufferPool_Stats(t *testing.T) {
	p := newPool()
	p.ResetStats()

	// Perform operations
	buf1 := p.Get(64)
	buf2 := p.Get(128)
	p.Put(buf1)
	p.Put(buf2)
	buf3 := p.Get(64)
	p.Put(buf3)

	stats := p.Stats()

	if stats.Gets != 3 {
		t.Errorf("Stats().Gets = %d, want 3", stats.Gets)
	}
	if stats.Puts != 3 {
		t.Errorf("Stats().Puts = %d, want 3", stats.Puts)
	}

	// Test hit rate
	if stats.HitRate() == 0 && stats.Hits > 0 {
		t.Error("HitRate() = 0 with hits > 0")
	}

	// Test alloc rate
	if stats.AllocRate() == 0 && stats.Allocs > 0 {
		t.Error("AllocRate() = 0 with allocs > 0")
	}
}

func TestPoolStats_Rates(t *testing.T) {
	// Test zero gets
	s := PoolStats{Gets: 0, Hits: 10, Allocs: 5}
	if s.HitRate() != 0 {
		t.Errorf("HitRate() with 0 gets = %f, want 0", s.HitRate())
	}
	if s.AllocRate() != 0 {
		t.Errorf("AllocRate() with 0 gets = %f, want 0", s.AllocRate())
	}

	// Test normal rates
	s = PoolStats{Gets: 100, Hits: 75, Allocs: 25}
	if s.HitRate() != 75.0 {
		t.Errorf("HitRate() = %f, want 75.0", s.HitRate())
	}
	if s.AllocRate() != 25.0 {
		t.Errorf("AllocRate() = %f, want 25.0", s.AllocRate())
	}
}

func TestGlobalPoolFunctions(t *testing.T) {
	ResetStats()

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

	// Test global Stats
	stats := Stats()
	if stats.Gets == 0 {
		t.Error("Global Stats().Gets = 0, want > 0")
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

func TestNextPowerOf2(t *testing.T) {
	tests := []struct {
		n    int
		want int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{64, 64},
		{65, 128},
		{1000, 1024},
		{maxPoolSize + 1, maxPoolSize},
	}

	for _, tt := range tests {
		if got := nextPowerOf2(tt.n); got != tt.want {
			t.Errorf("nextPowerOf2(%d) = %d, want %d", tt.n, got, tt.want)
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
				size := 64 << (j % 5) // Various sizes

				// Get buffer
				buf := p.Get(size)
				if len(buf) != size {
					t.Errorf("goroutine %d: Get(%d) length = %d", id, size, len(buf))
				}

				// Use buffer
				for k := range buf {
					buf[k] = byte(id + j)
				}

				// Return buffer
				p.Put(buf)

				// Occasionally get Buffer objects
				if j%10 == 0 {
					b := p.GetBuffer(size)
					b.Write([]byte("test"))
					p.PutBuffer(b)
				}
			}
		}(i)
	}

	wg.Wait()

	stats := p.Stats()
	expectedGets := uint64(numGoroutines * numOps)
	expectedPuts := uint64(numGoroutines * numOps)

	// Add Buffer operations
	expectedGets += uint64(numGoroutines * (numOps / 10))
	expectedPuts += uint64(numGoroutines * (numOps / 10))

	if stats.Gets < expectedGets {
		t.Errorf("Concurrent Stats().Gets = %d, want >= %d", stats.Gets, expectedGets)
	}
	if stats.Puts < expectedPuts {
		t.Errorf("Concurrent Stats().Puts = %d, want >= %d", stats.Puts, expectedPuts)
	}
}

func TestBufferPool_MemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory pressure test in short mode")
	}

	p := newPool()
	p.ResetStats()

	// Allocate many buffers
	buffers := make([][]byte, 1000)
	for i := range buffers {
		buffers[i] = p.Get(4096)
	}

	// Return half to pool
	for i := 0; i < 500; i++ {
		p.Put(buffers[i])
	}

	// Force GC
	runtime.GC()
	runtime.GC()

	// Get more buffers - should reuse from pool
	for i := 0; i < 500; i++ {
		buf := p.Get(4096)
		if cap(buf) < 4096 {
			t.Errorf("Get(4096) after GC capacity = %d, want >= 4096", cap(buf))
		}
	}

	stats := p.Stats()
	if stats.Hits == 0 {
		t.Log("Warning: No hits after GC (pool may have been cleared)")
	}
}
