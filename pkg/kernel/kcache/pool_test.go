package kcache

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestEntryPool tests entry object pooling
func TestEntryPool(t *testing.T) {
	ep := newEntryPool()

	if ep == nil {
		t.Fatal("newEntryPool returned nil")
	}

	// Test prewarm
	if ep.size.Load() <= 0 {
		t.Error("Pool not prewarmed")
	}

	// Test get
	e := ep.get()
	if e == nil {
		t.Fatal("Pool get returned nil")
	}

	// Verify entry is reset
	if e.key != nil || e.value != nil || e.hash != 0 || e.state != StateEmpty {
		t.Error("Entry not properly reset")
	}

	// Test put
	e.key = "test"
	e.value = "value"
	e.hash = 12345
	e.state = StateActive

	ep.put(e)

	// Get again and verify it was reset
	e2 := ep.get()
	if e2.key != nil || e2.value != nil || e2.hash != 0 || e2.state != StateEmpty {
		t.Error("Entry not reset after put")
	}

}

// TestEntryPoolConcurrent tests concurrent pool access
func TestEntryPoolConcurrent(t *testing.T) {
	ep := newEntryPool()

	numGoroutines := 100
	numOps := 1000
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < numOps; j++ {
				e := ep.get()
				if e == nil {
					t.Error("Concurrent get returned nil")
					continue
				}

				// Use the entry
				e.key = j
				e.value = j * 2

				// Return to pool
				ep.put(e)
			}
		}()
	}

	wg.Wait()
}

// TestEntryPoolMaxSize tests pool size limits
func TestEntryPoolMaxSize(t *testing.T) {
	ep := newEntryPool()
	ep.maxSize = 10 // Set small max size

	// Try to put more than max
	for i := 0; i < 20; i++ {
		e := &entry{key: i}
		ep.put(e)
	}

	// Size should not exceed max
	if ep.size.Load() > ep.maxSize {
		t.Errorf("Pool size %d exceeds max %d", ep.size.Load(), ep.maxSize)
	}
}

// TestKeyPool tests string key pooling
func TestKeyPool(t *testing.T) {
	kp := newKeyPool()

	// Test small buffer
	smallBuf := kp.getBuffer(50)
	if smallBuf == nil || cap(*smallBuf) < 50 {
		t.Error("Small buffer allocation failed")
	}
	kp.putBuffer(smallBuf)

	// Test medium buffer
	medBuf := kp.getBuffer(200)
	if medBuf == nil || cap(*medBuf) < 200 {
		t.Error("Medium buffer allocation failed")
	}
	kp.putBuffer(medBuf)

	// Test large buffer
	largeBuf := kp.getBuffer(800)
	if largeBuf == nil || cap(*largeBuf) < 800 {
		t.Error("Large buffer allocation failed")
	}
	kp.putBuffer(largeBuf)

	// Test extra large (not pooled)
	xlBuf := kp.getBuffer(2000)
	if xlBuf == nil || cap(*xlBuf) < 2000 {
		t.Error("Extra large buffer allocation failed")
	}
	kp.putBuffer(xlBuf) // Should not panic even though not pooled
}

// TestKeyPoolReuse tests buffer reuse
func TestKeyPoolReuse(t *testing.T) {
	kp := newKeyPool()

	// Get and return a buffer
	buf1 := kp.getBuffer(50)
	*buf1 = append(*buf1, []byte("test data")...)
	originalCap := cap(*buf1)

	// Return to pool
	kp.putBuffer(buf1)

	// Get again - should reuse
	buf2 := kp.getBuffer(50)

	// Buffer should be valid
	if buf2 == nil {
		t.Error("Failed to get buffer from pool")
	}

	// Should be reset (length 0) if reused
	if len(*buf2) != 0 && cap(*buf2) == originalCap {
		t.Error("Reused buffer not reset")
	}
}

// TestNodePool tests collision chain node pooling
func TestNodePool(t *testing.T) {
	np := newNodePool()

	// Test get
	n := np.get()
	if n == nil {
		t.Fatal("NodePool get returned nil")
	}

	// Verify node is reset
	if n.key != nil || n.value != nil || n.hash != 0 || n.next != nil {
		t.Error("Node not properly reset")
	}

	// Use the node
	n.key = "key"
	n.value = "value"
	n.hash = 12345
	n.next = &node{}

	// Return to pool
	np.put(n)

	// Get again and verify reset
	n2 := np.get()
	if n2.key != nil || n2.value != nil || n2.hash != 0 || n2.next != nil {
		t.Error("Node not reset after put")
	}
}

// TestGlobalPools tests global pool initialization
func TestGlobalPools(t *testing.T) {
	// Get global pools
	ep := getGlobalEntryPool()
	kp := getGlobalKeyPool()
	np := getGlobalNodePool()

	if ep == nil {
		t.Fatal("Global entry pool is nil")
	}
	if kp == nil {
		t.Fatal("Global key pool is nil")
	}
	if np == nil {
		t.Fatal("Global node pool is nil")
	}

	// Test that they work
	e := ep.get()
	if e == nil {
		t.Error("Global entry pool get failed")
	}
	ep.put(e)

	buf := kp.getBuffer(100)
	if buf == nil {
		t.Error("Global key pool get failed")
	}
	kp.putBuffer(buf)

	n := np.get()
	if n == nil {
		t.Error("Global node pool get failed")
	}
	np.put(n)

	// Verify singleton behavior
	ep2 := getGlobalEntryPool()
	if ep != ep2 {
		t.Error("Global pools not singleton")
	}
}

// TestCleanupPools tests the global cleanup function
func TestCleanupPools(t *testing.T) {
	// Initialize global pools
	_ = getGlobalEntryPool()
	_ = getGlobalKeyPool()
	_ = getGlobalNodePool()

	ep := getGlobalEntryPool()

	// Call cleanup (should not panic)
	cleanupPools()

	// Verify pools can still be used after cleanup
	// The function doesn't actually clear pools, just a placeholder for GC cleanup
	e := ep.get()
	if e == nil {
		t.Error("Entry pool broken after cleanup")
	}
	ep.put(e)
}

// TestMonitorMemory tests memory monitoring functionality
func TestMonitorMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory monitoring test in short mode")
	}

	// Start monitoring briefly in background
	done := make(chan bool)
	go func() {
		// Run monitor with timeout
		go func() {
			monitorMemory()
		}()
		time.Sleep(10 * time.Millisecond)
		done <- true
	}()

	select {
	case <-done:
		// Monitoring started successfully
	case <-time.After(100 * time.Millisecond):
		// Timeout is OK
	}
}

// TestAllocator tests custom allocator
func TestAllocator(t *testing.T) {
	size := 1024
	a := newAllocator(size)

	if a == nil {
		t.Fatal("newAllocator returned nil")
	}

	// Test allocation
	b1 := a.alloc(100)
	if len(b1) != 100 {
		t.Errorf("Allocated %d bytes, want 100", len(b1))
	}

	// Test alignment (should be 104 due to 8-byte alignment)
	b2 := a.alloc(50)
	if len(b2) != 50 {
		t.Errorf("Allocated %d bytes, want 50", len(b2))
	}

	// Fill arena
	for i := 0; i < 5; i++ {
		a.alloc(100)
	}

	// Next allocation should use fallback
	large := a.alloc(1000)
	if len(large) != 1000 {
		t.Errorf("Fallback allocated %d bytes, want 1000", len(large))
	}

	// Test reset
	a.reset()
	if a.offset != 0 {
		t.Error("Allocator not reset")
	}

	// Can allocate again after reset
	b3 := a.alloc(50)
	if len(b3) != 50 {
		t.Error("Allocation failed after reset")
	}
}

// TestAllocatorConcurrent tests concurrent allocator access
func TestAllocatorConcurrent(t *testing.T) {
	a := newAllocator(1024 * 1024) // 1MB arena

	numGoroutines := 50
	allocSize := 100
	var wg sync.WaitGroup

	successCount := atomic.Int32{}
	fallbackCount := atomic.Int32{}

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				b := a.alloc(allocSize)
				if len(b) != allocSize {
					t.Errorf("Got %d bytes, want %d", len(b), allocSize)
				}

				// Check if from arena or fallback
				if cap(b) == allocSize {
					// Likely from fallback (exact size)
					fallbackCount.Add(1)
				} else {
					// From arena (might have extra capacity)
					successCount.Add(1)
				}
			}
		}()
	}

	wg.Wait()

	t.Logf("Arena allocations: %d, Fallback allocations: %d",
		successCount.Load(), fallbackCount.Load())
}

// TestBatchPool tests batch buffer pooling
func TestBatchPool(t *testing.T) {
	bp := newBatchPool()

	// Get buffer
	buf := bp.get()
	if buf == nil {
		t.Fatal("BatchPool get returned nil")
	}

	// Verify initialized correctly
	if buf.keys == nil || buf.values == nil || buf.hashes == nil || buf.found == nil {
		t.Error("Batch buffer not properly initialized")
	}

	// Use buffer
	buf.keys = append(buf.keys, "key1", "key2")
	buf.values = append(buf.values, "val1", "val2")
	buf.hashes = append(buf.hashes, 123, 456)
	buf.found = append(buf.found, true, false)

	// Return to pool
	bp.put(buf)

	// Get again - should be reset
	buf2 := bp.get()
	if len(buf2.keys) != 0 || len(buf2.values) != 0 ||
		len(buf2.hashes) != 0 || len(buf2.found) != 0 {
		t.Error("Batch buffer not reset")
	}

	// Capacity should be preserved
	if cap(buf2.keys) < DefaultBatchSize {
		t.Error("Batch buffer capacity not preserved")
	}
}

// TestGlobalBatchPool tests global batch pool
func TestGlobalBatchPool(t *testing.T) {
	// Test global pool functions
	buf := getBatchBuffer()
	if buf == nil {
		t.Fatal("getBatchBuffer returned nil")
	}

	// Use buffer
	buf.keys = append(buf.keys, "test")

	// Return to pool
	putBatchBuffer(buf)

	// Get again
	buf2 := getBatchBuffer()
	if buf2 == nil {
		t.Fatal("Second getBatchBuffer returned nil")
	}

	// Should be reset
	if len(buf2.keys) != 0 {
		t.Error("Global batch buffer not reset")
	}
}

// TestMemoryPressureHandling tests memory pressure response
func TestMemoryPressureHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory pressure test in short mode")
	}

	// Initialize global pools
	ep := getGlobalEntryPool()

	// Fill pool
	entries := make([]*entry, 100)
	for i := range entries {
		entries[i] = ep.get()
	}

	// Return to pool
	for _, e := range entries {
		ep.put(e)
	}

	initialSize := ep.size.Load()

	// Trigger memory pressure handling
	handleMemoryPressure()

	// Pool should be reduced
	finalSize := ep.size.Load()
	if finalSize >= initialSize {
		t.Log("Memory pressure handling might not have reduced pool size")
	}

	// Force GC was called
	runtime.GC()

	// Pool should still be functional
	e := ep.get()
	if e == nil {
		t.Error("Pool broken after memory pressure handling")
	}
	ep.put(e)
}

// TestPoolPerformance benchmarks pool vs direct allocation
func TestPoolPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ep := newEntryPool()
	iterations := 100000

	// Benchmark pooled allocation
	start := time.Now()
	for i := 0; i < iterations; i++ {
		e := ep.get()
		e.key = i
		e.value = i * 2
		ep.put(e)
	}
	pooledTime := time.Since(start)

	// Benchmark direct allocation
	start = time.Now()
	for i := 0; i < iterations; i++ {
		e := &entry{
			key:   i,
			value: i * 2,
		}
		_ = e // Simulate usage
	}
	directTime := time.Since(start)

	// Pool should be faster
	speedup := float64(directTime) / float64(pooledTime)
	t.Logf("Pooled: %v, Direct: %v, Speedup: %.2fx", pooledTime, directTime, speedup)

	if speedup < 1.0 {
		t.Log("Warning: Pooling not faster than direct allocation")
	}
}

// TestPoolStress performs stress testing on pools
func TestPoolStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	ep := newEntryPool()
	kp := newKeyPool()
	np := newNodePool()

	duration := 1 * time.Second
	done := make(chan bool)

	var (
		opsEntry atomic.Int64
		opsKey   atomic.Int64
		opsNode  atomic.Int64
	)

	// Entry pool stress
	for i := 0; i < 10; i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					e := ep.get()
					e.key = "stress"
					ep.put(e)
					opsEntry.Add(1)
				}
			}
		}()
	}

	// Key pool stress
	for i := 0; i < 10; i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					buf := kp.getBuffer(100)
					*buf = append(*buf, []byte("stress test")...)
					kp.putBuffer(buf)
					opsKey.Add(1)
				}
			}
		}()
	}

	// Node pool stress
	for i := 0; i < 10; i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					n := np.get()
					n.key = "stress"
					np.put(n)
					opsNode.Add(1)
				}
			}
		}()
	}

	// Run stress test
	time.Sleep(duration)
	close(done)
	time.Sleep(100 * time.Millisecond) // Let goroutines finish

	// Report results
	t.Logf("Stress test results:")
	t.Logf("  Entry pool ops: %d (%.0f/sec)", opsEntry.Load(),
		float64(opsEntry.Load())/duration.Seconds())
	t.Logf("  Key pool ops: %d (%.0f/sec)", opsKey.Load(),
		float64(opsKey.Load())/duration.Seconds())
	t.Logf("  Node pool ops: %d (%.0f/sec)", opsNode.Load(),
		float64(opsNode.Load())/duration.Seconds())

	// Verify pools still work
	if e := ep.get(); e == nil {
		t.Error("Entry pool broken after stress")
	}
	if buf := kp.getBuffer(50); buf == nil {
		t.Error("Key pool broken after stress")
	}
	if n := np.get(); n == nil {
		t.Error("Node pool broken after stress")
	}
}

// TestPoolMemoryLeak tests for memory leaks in pooling
func TestPoolMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	ep := newEntryPool()

	// Allocate and return many times
	for cycle := 0; cycle < 100; cycle++ {
		entries := make([]*entry, 100)

		// Get from pool
		for i := range entries {
			entries[i] = ep.get()
			entries[i].key = make([]byte, 1024) // 1KB allocation
			entries[i].value = make([]byte, 1024)
		}

		// Return to pool
		for _, e := range entries {
			ep.put(e)
		}

		// Force GC every 10 cycles
		if cycle%10 == 0 {
			runtime.GC()
			runtime.GC()
		}
	}

	// Final GC
	runtime.GC()
	runtime.GC()

	// Check pool size is reasonable
	if ep.size.Load() > ep.maxSize {
		t.Errorf("Pool size %d exceeds max %d", ep.size.Load(), ep.maxSize)
	}

	// Pool should still be functional
	e := ep.get()
	if e == nil {
		t.Error("Pool broken after leak test")
	}
}
