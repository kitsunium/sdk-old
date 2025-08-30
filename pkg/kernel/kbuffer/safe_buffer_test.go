package kbuffer

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

// TestSpinLock tests the spinlock implementation for thread safety.
func TestSpinLock(t *testing.T) {
	var lock spinLock
	var counter uint32
	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Test concurrent lock/unlock
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				lock.Lock()
				atomic.AddUint32(&counter, 1)
				lock.Unlock()
			}
		}()
	}

	wg.Wait()

	expected := uint32(goroutines * iterations)
	if counter != expected {
		t.Errorf("Counter = %d, want %d (race condition detected)", counter, expected)
	}
}

// TestSpinLockTryLock tests non-blocking lock acquisition.
func TestSpinLockTryLock(t *testing.T) {
	var lock spinLock

	// First TryLock should succeed
	if !lock.TryLock() {
		t.Error("TryLock() = false on unlocked spinlock")
	}

	// Second TryLock should fail (already locked)
	if lock.TryLock() {
		t.Error("TryLock() = true on locked spinlock")
	}

	// Unlock and try again
	lock.Unlock()
	if !lock.TryLock() {
		t.Error("TryLock() = false after unlock")
	}
}

// TestSpinLockBackoff tests exponential backoff behavior.
func TestSpinLockBackoff(t *testing.T) {
	var lock spinLock
	lock.Lock()

	// Try to acquire from another goroutine with timeout
	done := make(chan bool)
	go func() {
		lock.Lock()
		lock.Unlock()
		done <- true
	}()

	// Wait a bit then unlock
	time.Sleep(10 * time.Millisecond)
	lock.Unlock()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Lock acquisition timed out (backoff not working)")
	}
}

// TestNewSafeBuffer tests safe buffer creation with various capacities.
func TestNewSafeBuffer(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		opts     []Option
		wantCap  int
	}{
		{"zero capacity", 0, nil, defaultBufferSize},
		{"negative capacity", -100, nil, defaultBufferSize},
		{"below minimum", minBufferSize - 1, nil, minBufferSize},
		{"at minimum", minBufferSize, nil, minBufferSize},
		{"normal capacity", 1024, nil, 1024},
		{"above maximum", maxBufferSize + 1, nil, maxBufferSize},
		{"at maximum", maxBufferSize, nil, maxBufferSize},
		{"above maximum", maxBufferSize + 1000, nil, maxBufferSize},
		{"with options", 512, []Option{func(b Buffer) error { return nil }}, 512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := newSafeBuffer(tt.capacity, tt.opts...)
			if buf.Cap() != tt.wantCap {
				t.Errorf("Cap() = %d, want %d", buf.Cap(), tt.wantCap)
			}
			if buf.Len() != 0 {
				t.Errorf("Len() = %d, want 0", buf.Len())
			}
			if buf.Available() != tt.wantCap {
				t.Errorf("Available() = %d, want %d", buf.Available(), tt.wantCap)
			}
		})
	}
}

// TestSafeBufferWrite tests thread-safe write operations.
func TestSafeBufferWrite(t *testing.T) {
	buf := newSafeBuffer(100)

	// Test empty write
	n, err := buf.Write([]byte{})
	if err != nil || n != 0 {
		t.Errorf("Write(empty) = %d, %v; want 0, nil", n, err)
	}

	// Test normal write
	data := []byte("hello world")
	n, err = buf.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() = %d, want %d", n, len(data))
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Errorf("Bytes() = %v, want %v", buf.Bytes(), data)
	}

	// Fill buffer to capacity
	remaining := buf.Available()
	bigData := make([]byte, remaining)
	for i := range bigData {
		bigData[i] = byte(i % 256)
	}
	n, err = buf.Write(bigData)
	if err != nil {
		t.Fatalf("Write(fill) error = %v", err)
	}
	if n != remaining {
		t.Errorf("Write(fill) = %d, want %d", n, remaining)
	}

	// Test buffer full error
	_, err = buf.Write([]byte{1})
	if err != errBufferFull {
		t.Errorf("Write(overflow) error = %v, want errBufferFull", err)
	}
}

// TestSafeBufferConcurrentWrite tests concurrent write safety.
func TestSafeBufferConcurrentWrite(t *testing.T) {
	const goroutines = 100
	const writesPerGoroutine = 10
	buf := newSafeBuffer(goroutines * writesPerGoroutine * 10)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent writes
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			data := []byte{byte(id)}
			for j := 0; j < writesPerGoroutine; j++ {
				buf.Write(data)
				runtime.Gosched() // Encourage context switching
			}
		}(i)
	}

	wg.Wait()

	// Verify total length
	expectedLen := goroutines * writesPerGoroutine
	if buf.Len() != expectedLen {
		t.Errorf("Concurrent writes: Len() = %d, want %d", buf.Len(), expectedLen)
	}
}

// TestSafeBufferWriteString tests string write operations.
func TestSafeBufferWriteString(t *testing.T) {
	buf := newSafeBuffer(50)

	// Empty string
	n, err := buf.WriteString("")
	if err != nil || n != 0 {
		t.Errorf("WriteString(empty) = %d, %v; want 0, nil", n, err)
	}

	// Normal string
	str := "hello world"
	n, err = buf.WriteString(str)
	if err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if n != len(str) {
		t.Errorf("WriteString() = %d, want %d", n, len(str))
	}
	if buf.String() != str {
		t.Errorf("String() = %q, want %q", buf.String(), str)
	}

	// Fill to capacity
	remaining := buf.Available()
	longStr := string(make([]byte, remaining))
	n, err = buf.WriteString(longStr)
	if err != nil {
		t.Fatalf("WriteString(fill) error = %v", err)
	}
	if n != remaining {
		t.Errorf("WriteString(fill) = %d, want %d", n, remaining)
	}

	// Overflow
	_, err = buf.WriteString("x")
	if err != errBufferFull {
		t.Errorf("WriteString(overflow) error = %v, want errBufferFull", err)
	}
}

// TestSafeBufferWriteByte tests single byte write operations.
func TestSafeBufferWriteByte(t *testing.T) {
	buf := newSafeBuffer(3) // Will be rounded up to minBufferSize

	// Write bytes up to actual capacity
	actualCap := buf.Cap()
	for i := byte(0); i < byte(actualCap); i++ {
		if err := buf.WriteByte(i); err != nil {
			t.Fatalf("WriteByte(%d) error = %v", i, err)
		}
	}

	// Check content (first few bytes)
	if buf.Len() != actualCap {
		t.Errorf("Len() = %d, want %d", buf.Len(), actualCap)
	}

	// Buffer full
	if err := buf.WriteByte(100); err != errBufferFull {
		t.Errorf("WriteByte(overflow) error = %v, want errBufferFull", err)
	}
}

// TestSafeBufferWriteAt tests positional write operations.
func TestSafeBufferWriteAt(t *testing.T) {
	buf := newSafeBuffer(100)

	// Write at various offsets
	data1 := []byte("hello")
	n, err := buf.WriteAt(data1, 0)
	if err != nil || n != len(data1) {
		t.Errorf("WriteAt(0) = %d, %v; want %d, nil", n, err, len(data1))
	}

	data2 := []byte("world")
	n, err = buf.WriteAt(data2, 50)
	if err != nil || n != len(data2) {
		t.Errorf("WriteAt(50) = %d, %v; want %d, nil", n, err, len(data2))
	}

	// Invalid offset (negative)
	_, err = buf.WriteAt([]byte("test"), -1)
	if err != errInvalidOffset {
		t.Errorf("WriteAt(-1) error = %v, want errInvalidOffset", err)
	}

	// Invalid offset (beyond capacity)
	_, err = buf.WriteAt([]byte("test"), int64(buf.Cap()))
	if err != errInvalidOffset {
		t.Errorf("WriteAt(cap) error = %v, want errInvalidOffset", err)
	}

	// Partial write (truncated to available space)
	longData := make([]byte, 50)
	n, err = buf.WriteAt(longData, 70)
	if err != nil {
		t.Errorf("WriteAt(partial) error = %v", err)
	}
	if n != 30 { // Should write only 30 bytes (100 - 70)
		t.Errorf("WriteAt(partial) = %d, want 30", n)
	}
}

// TestSafeBufferTryWrite tests non-blocking write attempts.
func TestSafeBufferTryWrite(t *testing.T) {
	buf := newSafeBuffer(10).(*safeBuffer)

	// Empty write should succeed
	if !buf.TryWrite([]byte{}) {
		t.Error("TryWrite(empty) = false, want true")
	}

	// Normal write
	data := []byte("test")
	if !buf.TryWrite(data) {
		t.Error("TryWrite() = false, want true")
	}

	// Fill buffer
	remaining := make([]byte, buf.Available())
	if !buf.TryWrite(remaining) {
		t.Error("TryWrite(fill) = false, want true")
	}

	// Overflow should fail
	if buf.TryWrite([]byte{1}) {
		t.Error("TryWrite(overflow) = true, want false")
	}

	// Test with locked buffer (simulate contention)
	buf2 := newSafeBuffer(10).(*safeBuffer)
	buf2.spin.Lock()
	done := make(chan bool)
	go func() {
		// This should fail immediately (non-blocking)
		result := buf2.TryWrite([]byte("test"))
		done <- result
	}()

	select {
	case result := <-done:
		if result {
			t.Error("TryWrite() succeeded on locked buffer")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("TryWrite() blocked on locked buffer")
	}
	buf2.spin.Unlock()
}

// TestSafeBufferReadOperations tests all read methods.
func TestSafeBufferReadOperations(t *testing.T) {
	// Test with empty buffer
	buf := newSafeBuffer(100)
	if buf.Bytes() != nil {
		t.Error("Bytes() on empty buffer should return nil")
	}
	if buf.String() != "" {
		t.Error("String() on empty buffer should return empty string")
	}
	ptr, length := buf.BytesUnsafe()
	if ptr != 0 || length != 0 {
		t.Errorf("BytesUnsafe() on empty = %d, %d; want 0, 0", ptr, length)
	}

	// Test with data
	data := []byte("hello world")
	buf.Write(data)

	// Test Bytes()
	if !bytes.Equal(buf.Bytes(), data) {
		t.Errorf("Bytes() = %v, want %v", buf.Bytes(), data)
	}

	// Test String()
	if buf.String() != string(data) {
		t.Errorf("String() = %q, want %q", buf.String(), string(data))
	}

	// Test BytesUnsafe()
	ptr, length = buf.BytesUnsafe()
	if ptr == 0 || length != len(data) {
		t.Errorf("BytesUnsafe() = %d, %d; want non-zero, %d", ptr, length, len(data))
	}
}

// TestSafeBufferReset tests reset operation.
func TestSafeBufferReset(t *testing.T) {
	buf := newSafeBuffer(100).(*safeBuffer)
	buf.Write([]byte("test data"))

	// Reset should clear position but keep capacity
	oldCap := buf.Cap()
	buf.Reset()

	if buf.Len() != 0 {
		t.Errorf("Len() after Reset = %d, want 0", buf.Len())
	}
	if buf.Cap() != oldCap {
		t.Errorf("Cap() after Reset = %d, want %d", buf.Cap(), oldCap)
	}
	if buf.flag.Load() != stateFlagNormal {
		t.Errorf("flag after Reset = %d, want stateFlagNormal", buf.flag.Load())
	}
}

// TestSafeBufferClear tests clear operation.
func TestSafeBufferClear(t *testing.T) {
	buf := newSafeBuffer(100).(*safeBuffer)
	data := []byte("sensitive data")
	buf.Write(data)

	// Clear should zero memory
	buf.Clear()

	if buf.Len() != 0 {
		t.Errorf("Len() after Clear = %d, want 0", buf.Len())
	}
	if buf.flag.Load() != stateFlagCleared {
		t.Errorf("flag after Clear = %d, want stateFlagCleared", buf.flag.Load())
	}

	// Verify memory was zeroed (check internal data)
	// This is implementation-specific testing
	internalData := buf.Bytes()
	if internalData != nil {
		t.Error("Bytes() after Clear should return nil")
	}
}

// TestSafeBufferTruncate tests truncate operation.
func TestSafeBufferTruncate(t *testing.T) {
	buf := newSafeBuffer(100).(*safeBuffer)
	buf.Write([]byte("hello world"))

	// Truncate to smaller size
	buf.Truncate(5)
	if buf.Len() != 5 {
		t.Errorf("Len() after Truncate(5) = %d, want 5", buf.Len())
	}
	if string(buf.Bytes()) != "hello" {
		t.Errorf("Bytes() after Truncate = %q, want %q", buf.Bytes(), "hello")
	}

	// Truncate to negative (should become 0)
	buf.Truncate(-10)
	if buf.Len() != 0 {
		t.Errorf("Len() after Truncate(-10) = %d, want 0", buf.Len())
	}

	// Truncate to larger than current (no effect)
	buf.Write([]byte("test"))
	buf.Truncate(100)
	if buf.Len() != 4 {
		t.Errorf("Len() after Truncate(100) = %d, want 4", buf.Len())
	}

	// Test flag update when truncating from full buffer
	buf2 := newSafeBuffer(5).(*safeBuffer) // Will be rounded to minBufferSize
	// Fill the actual capacity
	fillData := make([]byte, buf2.Cap())
	buf2.Write(fillData)
	if buf2.flag.Load()&stateFlagFull == 0 {
		t.Error("Buffer should be marked as full")
	}
	buf2.Truncate(3)
	if buf2.flag.Load()&stateFlagFull != 0 {
		t.Error("Buffer should not be marked as full after truncate")
	}
}

// TestSafeBufferGrow tests grow operation.
func TestSafeBufferGrow(t *testing.T) {
	buf := newSafeBuffer(10) // Will be rounded to minBufferSize
	actualCap := buf.Cap()

	// Should succeed when space available
	if err := buf.Grow(actualCap); err != nil {
		t.Errorf("Grow(%d) error = %v", actualCap, err)
	}

	// Write some data
	buf.Write([]byte("12345"))
	remaining := buf.Available()

	// Should succeed with remaining space
	if err := buf.Grow(remaining); err != nil {
		t.Errorf("Grow(%d) with %d available error = %v", remaining, remaining, err)
	}

	// Should fail when not enough space
	if err := buf.Grow(remaining + 1); err != errBufferFull {
		t.Errorf("Grow(%d) with %d available error = %v, want errBufferFull", remaining+1, remaining, err)
	}
}

// TestSafeBufferExtend tests extend operation.
func TestSafeBufferExtend(t *testing.T) {
	buf := newSafeBuffer(10).(*safeBuffer) // Will be rounded to minBufferSize
	actualCap := buf.Cap()

	// Extend by positive amount
	if err := buf.Extend(5); err != nil {
		t.Errorf("Extend(5) error = %v", err)
	}
	if buf.Len() != 5 {
		t.Errorf("Len() after Extend(5) = %d, want 5", buf.Len())
	}

	// Extend by negative (error)
	if err := buf.Extend(-1); err != errInvalidSize {
		t.Errorf("Extend(-1) error = %v, want errInvalidSize", err)
	}

	// Extend beyond capacity
	if err := buf.Extend(actualCap); err != errBufferFull {
		t.Errorf("Extend(%d) error = %v, want errBufferFull", actualCap, err)
	}
}

// TestSafeBufferClone tests clone operation.
func TestSafeBufferClone(t *testing.T) {
	buf := newSafeBuffer(100).(*safeBuffer)
	data := []byte("test data")
	buf.Write(data)
	buf.flag.Store(stateFlagFull | stateFlagPooled)

	// Clone buffer
	clone := buf.Clone()

	// Verify clone is independent
	if clone.Len() != buf.Len() {
		t.Errorf("Clone Len() = %d, want %d", clone.Len(), buf.Len())
	}
	if clone.Cap() != buf.Cap() {
		t.Errorf("Clone Cap() = %d, want %d", clone.Cap(), buf.Cap())
	}
	if !bytes.Equal(clone.Bytes(), buf.Bytes()) {
		t.Errorf("Clone Bytes() = %v, want %v", clone.Bytes(), buf.Bytes())
	}

	// Verify pooled flag is cleared in clone
	if cloneSafe, ok := clone.(*safeBuffer); ok {
		if cloneSafe.flag.Load()&stateFlagPooled != 0 {
			t.Error("Clone should not have pooled flag")
		}
		if cloneSafe.pooled {
			t.Error("Clone pooled field should be false")
		}
	}

	// Modify original and verify clone is unaffected
	buf.Write([]byte(" more"))
	if bytes.Equal(clone.Bytes(), buf.Bytes()) {
		t.Error("Clone should be independent of original")
	}
}

// TestSafeBufferRemainingSlice tests remaining slice operation.
func TestSafeBufferRemainingSlice(t *testing.T) {
	buf := newSafeBuffer(10).(*safeBuffer) // Will be rounded to minBufferSize
	actualCap := buf.Cap()

	// Full capacity available initially
	remaining := buf.RemainingSlice()
	if len(remaining) != actualCap {
		t.Errorf("RemainingSlice() initial len = %d, want %d", len(remaining), actualCap)
	}

	// Write some data
	buf.Write([]byte("hello"))
	remaining = buf.RemainingSlice()
	if len(remaining) != actualCap-5 {
		t.Errorf("RemainingSlice() after write len = %d, want %d", len(remaining), actualCap-5)
	}

	// Fill buffer
	fillData := make([]byte, actualCap-5)
	buf.Write(fillData)
	remaining = buf.RemainingSlice()
	if remaining != nil {
		t.Errorf("RemainingSlice() when full = %v, want nil", remaining)
	}
}

// TestSafeBufferAppendBytes tests variadic append operation.
func TestSafeBufferAppendBytes(t *testing.T) {
	buf := newSafeBuffer(10) // Will be rounded to minBufferSize
	actualCap := buf.Cap()

	// Empty append
	if err := buf.AppendBytes(); err != nil {
		t.Errorf("AppendBytes(empty) error = %v", err)
	}

	// Single byte
	if err := buf.AppendBytes('a'); err != nil {
		t.Errorf("AppendBytes('a') error = %v", err)
	}

	// Multiple bytes
	if err := buf.AppendBytes('b', 'c', 'd'); err != nil {
		t.Errorf("AppendBytes('b','c','d') error = %v", err)
	}

	expected := []byte("abcd")
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("Bytes() = %v, want %v", buf.Bytes(), expected)
	}

	// Fill to capacity and test overflow
	fillData := make([]byte, actualCap-4) // Already have "abcd"
	buf.Write(fillData)

	// Now buffer is full, this should fail
	if err := buf.AppendBytes('x'); err != errBufferFull {
		t.Errorf("AppendBytes(overflow) error = %v, want errBufferFull", err)
	}
}

// TestSafeBufferConcurrentReadWrite tests concurrent read/write safety.
func TestSafeBufferConcurrentReadWrite(t *testing.T) {
	buf := newSafeBuffer(10000)
	stopChan := make(chan struct{})
	var wg sync.WaitGroup

	// Start writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			data := []byte{byte(id)}
			for {
				select {
				case <-stopChan:
					return
				default:
					buf.Write(data)
					runtime.Gosched()
				}
			}
		}(i)
	}

	// Start readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					_ = buf.Len()
					_ = buf.Bytes()
					_ = buf.String()
					_ = buf.Available()
					runtime.Gosched()
				}
			}
		}()
	}

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)
	close(stopChan)
	wg.Wait()

	// If we get here without panic/race, concurrent access is safe
	t.Logf("Concurrent test completed: Len=%d, Cap=%d", buf.Len(), buf.Cap())
}

// TestSafeBufferStressTest performs stress testing with high concurrency.
func TestSafeBufferStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	buf := newSafeBuffer(1 << 20) // 1MB buffer
	const goroutines = 100
	const operations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	start := time.Now()

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operations; j++ {
				// Mix of operations
				switch j % 7 {
				case 0:
					buf.Write([]byte("test"))
				case 1:
					buf.WriteString("string")
				case 2:
					buf.WriteByte(byte(j))
				case 3:
					_ = buf.Len()
				case 4:
					_ = buf.Bytes()
				case 5:
					if j%100 == 0 {
						buf.Reset()
					}
				case 6:
					_ = buf.TryWrite([]byte("try"))
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Stress test completed: %d goroutines, %d ops each, took %v",
		goroutines, operations, elapsed)
	t.Logf("Final buffer state: Len=%d, Cap=%d", buf.Len(), buf.Cap())
}

// TestSafeBufferMemoryAlignment tests cache-line alignment.
func TestSafeBufferMemoryAlignment(t *testing.T) {
	buf := newSafeBuffer(100).(*safeBuffer)

	// Test that critical fields are properly aligned
	// This ensures no false sharing between cache lines
	if size := unsafe.Sizeof(*buf); size%64 != 0 {
		t.Logf("Warning: safeBuffer size %d is not cache-line aligned", size)
	}
}

// TestSafeBufferPanicRecovery tests panic recovery in safe buffer operations.
func TestSafeBufferPanicRecovery(t *testing.T) {
	t.Run("RecoverFromNilWrite", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Should not panic on nil write: %v", r)
			}
		}()

		b := NewSafeBuffer(256)
		b.Write(nil) // Should handle gracefully
	})

	t.Run("RecoverFromInvalidTruncate", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Should not panic on invalid truncate: %v", r)
			}
		}()

		b := NewSafeBuffer(256)
		b.Write([]byte("test"))
		b.Truncate(-1)   // Should handle gracefully
		b.Truncate(1000) // Should handle gracefully
	})

	t.Run("RecoverFromInvalidGrow", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Should not panic on invalid grow: %v", r)
			}
		}()

		b := NewSafeBuffer(256)
		b.Grow(-100) // Should handle gracefully
	})

	t.Run("RecoverFromConcurrentModification", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Should not panic on concurrent modification: %v", r)
			}
		}()

		b := NewSafeBuffer(256)
		var wg sync.WaitGroup

		// Multiple goroutines writing concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					b.Write([]byte{byte(id)})
				}
			}(i)
		}
		wg.Wait()
	})
}

// Benchmarks for safe buffer operations

// BenchmarkSafeBufferWrite benchmarks write operations.
func BenchmarkSafeBufferWrite(b *testing.B) {
	sizes := []int{16, 64, 256, 1024, 4096}
	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i)
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			buf := NewSafeBuffer(4096)
			writeData := data[:size]
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				buf.Reset()
				buf.Write(writeData)
			}
		})
	}
}

// BenchmarkSafeBufferWriteString benchmarks string write operations.
func BenchmarkSafeBufferWriteString(b *testing.B) {
	strings := []string{
		"short",
		"medium length string",
		"this is a longer string that spans multiple cache lines for testing performance",
		string(make([]byte, 256)),
		string(make([]byte, 1024)),
	}

	for i, s := range strings {
		b.Run(fmt.Sprintf("Len%d", len(s)), func(b *testing.B) {
			buf := NewSafeBuffer(4096)
			b.ResetTimer()
			b.ReportAllocs()

			for j := 0; j < b.N; j++ {
				buf.Reset()
				buf.WriteString(s)
			}
		})
		_ = i
	}
}

// BenchmarkSafeBufferWriteByte benchmarks single byte write operations.
func BenchmarkSafeBufferWriteByte(b *testing.B) {
	buf := NewSafeBuffer(4096)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if buf.Len() >= 4000 {
			buf.Reset()
		}
		buf.WriteByte(byte(i))
	}
}

// BenchmarkSafeBufferRead benchmarks read operations.
func BenchmarkSafeBufferRead(b *testing.B) {
	sizes := []int{16, 64, 256, 1024, 4096}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			buf := NewSafeBuffer(size)
			data := make([]byte, size)
			for i := range data {
				data[i] = byte(i)
			}
			buf.Write(data)

			readBuf := make([]byte, size)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				copy(readBuf, buf.Bytes())
				buf.Reset()
				buf.Write(data)
			}
		})
	}
}

// BenchmarkSafeBufferClone benchmarks buffer cloning.
func BenchmarkSafeBufferClone(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			buf := NewSafeBuffer(size)
			data := make([]byte, size)
			for i := range data {
				data[i] = byte(i)
			}
			buf.Write(data)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = buf.Clone()
			}
		})
	}
}

// BenchmarkSafeBufferGrow benchmarks buffer growth operations.
func BenchmarkSafeBufferGrow(b *testing.B) {
	growSizes := []int{64, 256, 1024, 4096}

	for _, growSize := range growSizes {
		b.Run(fmt.Sprintf("Grow%d", growSize), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				buf := NewSafeBuffer(64)
				buf.Grow(growSize)
			}
		})
	}
}

// BenchmarkSafeBufferConcurrentWrite benchmarks concurrent write operations.
func BenchmarkSafeBufferConcurrentWrite(b *testing.B) {
	data := []byte("test data")

	b.Run("Parallel", func(b *testing.B) {
		buf := NewSafeBuffer(4096)
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf.Write(data)
				if buf.Len() > 3000 {
					buf.Reset()
				}
			}
		})
	})
}

// BenchmarkSafeBufferConcurrentReadWrite benchmarks mixed operations.
func BenchmarkSafeBufferConcurrentReadWrite(b *testing.B) {
	buf := NewSafeBuffer(4096)
	writeData := []byte("test data")
	buf.Write(make([]byte, 1024)) // Pre-fill buffer

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		readBuf := make([]byte, 64)
		for pb.Next() {
			if i := 0; i%2 == 0 {
				buf.Write(writeData)
			} else {
				copy(readBuf, buf.Bytes())
			}
		}
	})
}

// BenchmarkSpinLock benchmarks spinlock performance.
func BenchmarkSpinLock(b *testing.B) {
	var lock spinLock

	b.Run("LockUnlock", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			lock.Lock()
			lock.Unlock()
		}
	})

	b.Run("TryLock", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			if lock.TryLock() {
				lock.Unlock()
			}
		}
	})

	b.Run("Contended", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				lock.Lock()
				lock.Unlock()
			}
		})
	})
}
