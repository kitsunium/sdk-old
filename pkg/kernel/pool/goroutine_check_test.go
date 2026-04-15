//go:build !unsafe_no_check
// +build !unsafe_no_check

package pool

import (
	"sync"
	"testing"
	"time"
	"unsafe"
)

// TestGoroutineCheckerBasic tests basic goroutine safety functionality
func TestGoroutineCheckerBasic(t *testing.T) {
	oldDebugMode := testingSkipSafetyCheck
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	// Test with checks disabled
	testingSkipSafetyCheck = true
	var checker1 goroutineChecker
	checker1.checkSafety()
	if checker1.writes.Load() != 0 {
		t.Error("checkSafety() should not increment when testingSkipSafetyCheck=true")
	}

	// Test with checks enabled
	testingSkipSafetyCheck = false
	var checker2 goroutineChecker
	checker2.checkSafety()
	if checker2.writes.Load() != 1 {
		t.Error("checkSafety() should increment when testingSkipSafetyCheck=false")
	}
}

// TestGoroutineCheckerPanic tests panic on cross-goroutine access
func TestGoroutineCheckerPanic(t *testing.T) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	var checker goroutineChecker
	checker.checkGoroutineSafety() // Set ownership

	// Try access from different goroutine
	done := make(chan bool)
	go func() {
		defer func() {
			done <- recover() != nil
		}()
		checker.checkGoroutineSafety() // Should panic
	}()

	select {
	case panicked := <-done:
		if !panicked {
			t.Error("Expected panic when accessing from different goroutine")
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for panic")
	}
}

// TestGoroutineCheckerSampling tests amortized sampling
func TestGoroutineCheckerSampling(t *testing.T) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	var checker goroutineChecker

	// First 10 calls should always check
	for i := 0; i < 10; i++ {
		checker.checkGoroutineSafety()
		if checker.writes.Load() != uint64(i+1) {
			t.Errorf("Call %d: writes=%d, want %d", i, checker.writes.Load(), i+1)
		}
	}

	// After 10, only sampled calls increment
	startWrites := checker.writes.Load()
	for i := 0; i < 1000; i++ {
		checker.checkGoroutineSafety()
	}

	endWrites := checker.writes.Load()
	totalCalls := endWrites - startWrites

	// Should have done approximately 1000 writes (all calls increment writes)
	// But only ~2 actual GID checks (1/512 sampling)
	if totalCalls != 1000 {
		t.Errorf("Expected 1000 write increments, got %d", totalCalls)
	}
}

// TestGoroutineCheckerMemoryUsage tests memory footprint
func TestGoroutineCheckerMemoryUsage(t *testing.T) {
	var checker goroutineChecker
	size := unsafe.Sizeof(checker)

	// Size should be at least 20 bytes (two uint64 + one uint32)
	if size < 20 {
		t.Errorf("goroutineChecker size = %d bytes, expected at least 20", size)
	} else {
		t.Logf("goroutineChecker size: %d bytes", size)
	}

	// Check alignment - log warnings instead of failing
	if uintptr(unsafe.Pointer(&checker.gid))%8 != 0 {
		t.Logf("Warning: gid field not 8-byte aligned (may impact performance)")
	}
	if uintptr(unsafe.Pointer(&checker.writes))%8 != 0 {
		t.Logf("Warning: writes field not 8-byte aligned (may impact performance)")
	}
	if uintptr(unsafe.Pointer(&checker.counter))%4 != 0 {
		t.Logf("Warning: counter field not 4-byte aligned (may impact performance)")
	}
}

// TestGoroutineCheckerRaceCondition tests for race conditions in the checker itself
func TestGoroutineCheckerRaceCondition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	// Disable safety checks to test atomic operations
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = true
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	var checker goroutineChecker
	var wg sync.WaitGroup
	const goroutines = 100
	const iterations = 1000

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// These operations should be race-free due to atomics
				_ = checker.gid.Load()
				_ = checker.writes.Load()
				checker.writes.Add(1)
			}
		}()
	}

	wg.Wait()

	// Verify atomic operations worked correctly
	finalWrites := checker.writes.Load()
	expectedWrites := uint64(goroutines * iterations)
	if finalWrites != expectedWrites {
		t.Errorf("Final writes = %d, want %d", finalWrites, expectedWrites)
	}
}

// TestUnsafeBufferGoroutineSafety tests unsafe buffer goroutine protection
func TestUnsafeBufferGoroutineSafety(t *testing.T) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(100).(*unsafeBuffer)
	buf.Write([]byte("first"))
	buf.Write([]byte(" second")) // Same goroutine OK

	// Test panic from different goroutine
	done := make(chan bool)
	go func() {
		defer func() {
			done <- recover() != nil
		}()
		buf.Write([]byte(" from another goroutine"))
	}()

	select {
	case panicked := <-done:
		if !panicked {
			t.Error("Expected panic when accessing unsafe buffer from different goroutine")
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for goroutine safety check")
	}
}
