package pool

import (
	"bytes"
	"testing"
	"time"
)

// TestNewUnsafeBuffer tests unsafe buffer creation with various capacities.
func TestNewUnsafeBuffer(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		opts     []Option
		wantCap  int
	}{
		{"zero capacity", 0, nil, defaultBufferSize},
		{"negative capacity", -100, nil, defaultBufferSize},
		{"below minimum", minBufferSize - 1, nil, minBufferSize},
		{"normal capacity", 2048, nil, 2048},
		{"above maximum", maxBufferSize + 5000, nil, maxBufferSize},
		{"with options", 1024, []Option{func(b Buffer) error { return nil }}, 1024},
		{"with failing option", 512, []Option{func(b Buffer) error { return errBufferFull }}, 512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := newUnsafeBuffer(tt.capacity, tt.opts...)
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

// TestUnsafeBufferWrite tests write operations on unsafe buffer.
func TestUnsafeBufferWrite(t *testing.T) {
	// Temporarily disable debug mode to avoid goroutine check
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(100).(*unsafeBuffer)

	// Test empty write
	n, err := buf.Write([]byte{})
	if err != nil || n != 0 {
		t.Errorf("Write(empty) = %d, %v; want 0, nil", n, err)
	}

	// Test normal write
	data := []byte("hello unsafe world")
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

	// Verify full flag is set
	if buf.flag&stateFlagFull == 0 {
		t.Error("Buffer should have full flag set")
	}

	// Test buffer full error
	_, err = buf.Write([]byte{1})
	if err != errBufferFull {
		t.Errorf("Write(overflow) error = %v, want errBufferFull", err)
	}
}

// TestUnsafeBufferWriteString tests string write operations.
func TestUnsafeBufferWriteString(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(60).(*unsafeBuffer)

	// Empty string
	n, err := buf.WriteString("")
	if err != nil || n != 0 {
		t.Errorf("WriteString(empty) = %d, %v; want 0, nil", n, err)
	}

	// Normal string
	str := "unsafe string write"
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

	// Verify full flag
	if buf.flag&stateFlagFull == 0 {
		t.Error("Buffer should have full flag set")
	}

	// Overflow
	_, err = buf.WriteString("x")
	if err != errBufferFull {
		t.Errorf("WriteString(overflow) error = %v, want errBufferFull", err)
	}
}

// TestUnsafeBufferWriteByte tests single byte write operations.
func TestUnsafeBufferWriteByte(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(4).(*unsafeBuffer) // Will be rounded to minBufferSize
	actualCap := buf.Cap()

	// Write bytes up to capacity
	for i := byte(0); i < byte(actualCap); i++ {
		if err := buf.WriteByte(i); err != nil {
			t.Fatalf("WriteByte(%d) error = %v", i, err)
		}
	}

	// Check that we filled the buffer
	if buf.Len() != actualCap {
		t.Errorf("Len() = %d, want %d", buf.Len(), actualCap)
	}

	// Check full flag
	if buf.flag&stateFlagFull == 0 {
		t.Error("Buffer should have full flag set")
	}

	// Buffer full
	if err := buf.WriteByte(100); err != errBufferFull {
		t.Errorf("WriteByte(overflow) error = %v, want errBufferFull", err)
	}
}

// TestUnsafeBufferWriteAt tests positional write operations.
func TestUnsafeBufferWriteAt(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(100).(*unsafeBuffer)

	// Write at beginning
	data1 := []byte("start")
	n, err := buf.WriteAt(data1, 0)
	if err != nil || n != len(data1) {
		t.Errorf("WriteAt(0) = %d, %v; want %d, nil", n, err, len(data1))
	}

	// Write at middle
	data2 := []byte("middle")
	n, err = buf.WriteAt(data2, 40)
	if err != nil || n != len(data2) {
		t.Errorf("WriteAt(40) = %d, %v; want %d, nil", n, err, len(data2))
	}

	// Write at end
	data3 := []byte("end")
	n, err = buf.WriteAt(data3, 97)
	if err != nil || n != len(data3) {
		t.Errorf("WriteAt(97) = %d, %v; want %d, nil", n, err, len(data3))
	}

	// Invalid offset (negative)
	_, err = buf.WriteAt([]byte("test"), -5)
	if err != errInvalidOffset {
		t.Errorf("WriteAt(-5) error = %v, want errInvalidOffset", err)
	}

	// Invalid offset (beyond capacity)
	_, err = buf.WriteAt([]byte("test"), int64(buf.Cap()))
	if err != errInvalidOffset {
		t.Errorf("WriteAt(cap) error = %v, want errInvalidOffset", err)
	}

	// Partial write (truncated to available space)
	longData := make([]byte, 60)
	n, err = buf.WriteAt(longData, 60)
	if err != nil {
		t.Errorf("WriteAt(partial) error = %v", err)
	}
	if n != 40 { // Should write only 40 bytes (100 - 60)
		t.Errorf("WriteAt(partial) = %d, want 40", n)
	}
}

// TestUnsafeBufferTryWrite tests non-blocking write attempts.
func TestUnsafeBufferTryWrite(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(10).(*unsafeBuffer)

	// Normal write should succeed
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
}

// TestUnsafeBufferReadOperations tests all read methods.
func TestUnsafeBufferReadOperations(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	// Test with empty buffer
	buf := newUnsafeBuffer(100).(*unsafeBuffer)
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
	data := []byte("unsafe read test")
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

// TestUnsafeBufferStateOperations tests Len, Cap, Available.
func TestUnsafeBufferStateOperations(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(100).(*unsafeBuffer)

	// Initial state
	if buf.Len() != 0 {
		t.Errorf("Initial Len() = %d, want 0", buf.Len())
	}
	if buf.Cap() != 100 {
		t.Errorf("Cap() = %d, want 100", buf.Cap())
	}
	if buf.Available() != 100 {
		t.Errorf("Initial Available() = %d, want 100", buf.Available())
	}

	// After writing
	buf.Write([]byte("12345"))
	if buf.Len() != 5 {
		t.Errorf("Len() after write = %d, want 5", buf.Len())
	}
	if buf.Cap() != 100 {
		t.Errorf("Cap() after write = %d, want 100", buf.Cap())
	}
	if buf.Available() != 95 {
		t.Errorf("Available() after write = %d, want 95", buf.Available())
	}
}

// TestUnsafeBufferReset tests reset operation.
func TestUnsafeBufferReset(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(100).(*unsafeBuffer)
	buf.Write([]byte("test data"))
	buf.flag = stateFlagFull | stateFlagPooled

	// Reset should clear position but keep capacity
	oldCap := buf.Cap()
	buf.Reset()

	if buf.Len() != 0 {
		t.Errorf("Len() after Reset = %d, want 0", buf.Len())
	}
	if buf.Cap() != oldCap {
		t.Errorf("Cap() after Reset = %d, want %d", buf.Cap(), oldCap)
	}
	if buf.flag != stateFlagNormal {
		t.Errorf("flag after Reset = %d, want stateFlagNormal", buf.flag)
	}
}

// TestUnsafeBufferClear tests clear operation.
func TestUnsafeBufferClear(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(100).(*unsafeBuffer)
	data := []byte("sensitive data to clear")
	buf.Write(data)

	// Clear should zero memory
	buf.Clear()

	if buf.Len() != 0 {
		t.Errorf("Len() after Clear = %d, want 0", buf.Len())
	}
	if buf.flag != stateFlagCleared {
		t.Errorf("flag after Clear = %d, want stateFlagCleared", buf.flag)
	}

	// Verify buffer is cleared (no data visible)
	if buf.Bytes() != nil {
		t.Error("Bytes() after Clear should return nil")
	}
}

// TestUnsafeBufferTruncate tests truncate operation.
func TestUnsafeBufferTruncate(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(100).(*unsafeBuffer)
	buf.Write([]byte("hello unsafe world"))

	// Truncate to smaller size
	buf.Truncate(5)
	if buf.Len() != 5 {
		t.Errorf("Len() after Truncate(5) = %d, want 5", buf.Len())
	}
	if string(buf.Bytes()) != "hello" {
		t.Errorf("Bytes() after Truncate = %q, want %q", buf.Bytes(), "hello")
	}

	// Truncate to negative (should become 0)
	buf.Truncate(-20)
	if buf.Len() != 0 {
		t.Errorf("Len() after Truncate(-20) = %d, want 0", buf.Len())
	}

	// Truncate to larger than current (no effect)
	buf.Write([]byte("test"))
	buf.Truncate(200)
	if buf.Len() != 4 {
		t.Errorf("Len() after Truncate(200) = %d, want 4", buf.Len())
	}

	// Test flag update when truncating from full buffer
	buf2 := newUnsafeBuffer(5).(*unsafeBuffer) // Will be rounded to minBufferSize
	// Fill the actual capacity
	fillData := make([]byte, buf2.Cap())
	buf2.Write(fillData)
	if buf2.flag&stateFlagFull == 0 {
		t.Error("Buffer should be marked as full")
	}
	buf2.Truncate(3)
	if buf2.flag&stateFlagFull != 0 {
		t.Error("Buffer should not be marked as full after truncate")
	}
}

// TestUnsafeBufferGrow tests grow operation.
func TestUnsafeBufferGrow(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(10).(*unsafeBuffer) // Will be rounded to minBufferSize
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

// TestUnsafeBufferExtend tests extend operation.
func TestUnsafeBufferExtend(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(10).(*unsafeBuffer) // Will be rounded to minBufferSize
	actualCap := buf.Cap()

	// Extend by positive amount
	if err := buf.Extend(3); err != nil {
		t.Errorf("Extend(3) error = %v", err)
	}
	if buf.Len() != 3 {
		t.Errorf("Len() after Extend(3) = %d, want 3", buf.Len())
	}

	// Extend by negative (error)
	if err := buf.Extend(-5); err != errInvalidSize {
		t.Errorf("Extend(-5) error = %v, want errInvalidSize", err)
	}

	// Extend by zero (no effect)
	if err := buf.Extend(0); err != nil {
		t.Errorf("Extend(0) error = %v", err)
	}
	if buf.Len() != 3 {
		t.Errorf("Len() after Extend(0) = %d, want 3", buf.Len())
	}

	// Extend beyond capacity
	if err := buf.Extend(actualCap); err != errBufferFull {
		t.Errorf("Extend(%d) error = %v, want errBufferFull", actualCap, err)
	}
}

// TestUnsafeBufferClone tests clone operation.
func TestUnsafeBufferClone(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(100).(*unsafeBuffer)
	data := []byte("data to clone")
	buf.Write(data)
	buf.flag = stateFlagFull | stateFlagPooled
	buf.pooled = true

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
	if cloneUnsafe, ok := clone.(*unsafeBuffer); ok {
		if cloneUnsafe.flag&stateFlagPooled != 0 {
			t.Error("Clone should not have pooled flag")
		}
		if cloneUnsafe.pooled {
			t.Error("Clone pooled field should be false")
		}
	}

	// Modify original and verify clone is unaffected
	buf.Write([]byte(" modified"))
	if bytes.Equal(clone.Bytes(), buf.Bytes()) {
		t.Error("Clone should be independent of original")
	}
}

// TestUnsafeBufferRemainingSlice tests remaining slice operation.
func TestUnsafeBufferRemainingSlice(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(10).(*unsafeBuffer) // Will be rounded to minBufferSize
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

// TestUnsafeBufferAppendBytes tests variadic append operation.
func TestUnsafeBufferAppendBytes(t *testing.T) {
	// Temporarily disable debug mode
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(10).(*unsafeBuffer)

	// Empty append
	if err := buf.AppendBytes(); err != nil {
		t.Errorf("AppendBytes(empty) error = %v", err)
	}

	// Single byte
	if err := buf.AppendBytes('x'); err != nil {
		t.Errorf("AppendBytes('x') error = %v", err)
	}

	// Multiple bytes
	if err := buf.AppendBytes('y', 'z'); err != nil {
		t.Errorf("AppendBytes('y','z') error = %v", err)
	}

	expected := []byte("xyz")
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("Bytes() = %v, want %v", buf.Bytes(), expected)
	}

	// Fill to capacity
	actualCap := buf.Cap()
	currentLen := buf.Len() // Currently have "xyz"
	padding := make([]byte, actualCap-currentLen)
	for i := range padding {
		padding[i] = byte('a' + i)
	}
	if err := buf.AppendBytes(padding...); err != nil {
		t.Errorf("AppendBytes(padding) error = %v", err)
	}

	// Should be full now
	if err := buf.AppendBytes('!'); err != errBufferFull {
		t.Errorf("AppendBytes(overflow) error = %v, want errBufferFull", err)
	}
}

// TestUnsafeBufferGoroutineSafetyDisabled tests with safety check disabled.
func TestUnsafeBufferGoroutineSafetyDisabled(t *testing.T) {
	// Disable safety checks (true = skip checks, false = do checks)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = true
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(100).(*unsafeBuffer)

	// Write from main goroutine
	buf.Write([]byte("main"))

	// Write from different goroutine should not panic when disabled
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- false // Unexpected panic
			} else {
				done <- true // No panic as expected
			}
		}()
		// This should NOT panic when debug mode is disabled
		buf.Write([]byte(" other"))
	}()

	select {
	case noPanic := <-done:
		if !noPanic {
			t.Error("Unexpected panic when debug mode is disabled")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for goroutine")
	}
}

// TestUnsafeBufferPerformance tests performance characteristics.
func TestUnsafeBufferPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Enable safety checks (testingSkipSafetyCheck=false enables checks)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(1 << 20).(*unsafeBuffer) // 1MB buffer
	data := make([]byte, 1024)                      // 1KB chunks
	for i := range data {
		data[i] = byte(i % 256)
	}

	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		buf.Reset()
		for j := 0; j < 100; j++ {
			buf.Write(data[:100]) // Write 100 bytes at a time
		}
	}

	elapsed := time.Since(start)
	opsPerSec := float64(iterations*100) / elapsed.Seconds()

	t.Logf("Unsafe buffer performance: %d iterations in %v", iterations, elapsed)
	t.Logf("Operations per second: %.0f", opsPerSec)

	// Verify it's significantly faster than safe version
	// This is just a sanity check, actual numbers depend on hardware
	if opsPerSec < 100000 {
		t.Logf("Warning: Performance seems low (%.0f ops/sec)", opsPerSec)
	}
}

// TestUnsafeBufferDataRace is moved to a separate file with !race build tag
// to exclude it when the race detector is enabled

// TestUnsafeBufferEdgeCases tests various edge cases.
func TestUnsafeBufferEdgeCases(t *testing.T) {
	// Enable safety checks (false = do checks)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	// Test with minimum size buffer
	minBuf := newUnsafeBuffer(1).(*unsafeBuffer)
	if minBuf.Cap() < minBufferSize {
		t.Errorf("Minimum buffer capacity = %d, expected at least %d", minBuf.Cap(), minBufferSize)
	}

	// Test writing exactly to capacity
	exactBuf := newUnsafeBuffer(10).(*unsafeBuffer) // Will be rounded to minBufferSize
	exactCap := exactBuf.Cap()
	exactData := make([]byte, exactCap)
	n, err := exactBuf.Write(exactData)
	if err != nil || n != exactCap {
		t.Errorf("Write(exact) = %d, %v; want %d, nil", n, err, exactCap)
	}
	if exactBuf.flag&stateFlagFull == 0 {
		t.Error("Buffer should be marked as full")
	}

	// Test multiple resets
	resetBuf := newUnsafeBuffer(10).(*unsafeBuffer)
	for i := 0; i < 10; i++ {
		resetBuf.Write([]byte("test"))
		resetBuf.Reset()
	}
	if resetBuf.Len() != 0 {
		t.Errorf("Len after multiple resets = %d, want 0", resetBuf.Len())
	}
}
