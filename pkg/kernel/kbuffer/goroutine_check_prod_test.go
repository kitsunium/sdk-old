//go:build unsafe_no_check
// +build unsafe_no_check

package kbuffer

import (
	"testing"
)

// TestProductionBuildTag tests that safety checks are disabled in production builds.
func TestProductionBuildTag(t *testing.T) {
	// Create an unsafe buffer which uses goroutine checking internally
	buf := NewUnsafeBuffer(1024)

	// Perform multiple writes that would trigger safety checks in dev mode
	data := []byte("test")
	for i := 0; i < 10; i++ {
		n, err := buf.Write(data)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		if n != len(data) {
			t.Fatalf("Write returned %d, want %d", n, len(data))
		}
	}

	// In production mode, safety checks should be disabled
	// The buffer should function normally without any panics
	// and without incrementing internal counters
	if buf.Len() != len(data)*10 {
		t.Errorf("Buffer length = %d, want %d", buf.Len(), len(data)*10)
	}
}

// TestProductionPerformance verifies no overhead in production.
func TestProductionPerformance(t *testing.T) {
	var checker goroutineChecker

	// In production, checkSafety should be a no-op
	for i := 0; i < 1000000; i++ {
		checker.checkSafety()
	}

	// Writes counter should remain at 0 (no checks performed)
	if checker.writes.Load() != 0 {
		t.Errorf("writes counter = %d in production, want 0", checker.writes.Load())
	}
}

// TestProductionInit verifies behavior in production mode.
func TestProductionInit(t *testing.T) {
	// Create multiple unsafe buffers and verify they work without safety checks
	buf1 := NewUnsafeBuffer(256)
	buf2 := NewUnsafeBuffer(256)

	// Write to both buffers
	buf1.Write([]byte("buffer1"))
	buf2.Write([]byte("buffer2"))

	// In production, concurrent access from different goroutines should not panic
	// (though it's still unsafe and may corrupt data)
	done := make(chan bool)
	go func() {
		defer func() {
			// Should not panic in production mode
			if r := recover(); r != nil {
				t.Errorf("Unexpected panic in production mode: %v", r)
			}
			done <- true
		}()
		buf1.Write([]byte("goroutine"))
	}()

	<-done

	// Buffers should still be functional
	if buf1.Len() == 0 || buf2.Len() == 0 {
		t.Error("Buffers should contain data")
	}
}
