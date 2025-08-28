//go:build !unsafe_no_check
// +build !unsafe_no_check

package kbuffer

import (
	"testing"
	"time"
)

// TestGoroutineCheckerCheckSafety tests conditional checking.
// This test only runs in development mode where safety checks can be toggled.
func TestGoroutineCheckerCheckSafety(t *testing.T) {
	// Save original mode
	oldDebugMode := testingSkipSafetyCheck
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	var checker goroutineChecker

	// Test with safety checks disabled (testingSkipSafetyCheck = true means SKIP checks)
	testingSkipSafetyCheck = true
	checker.checkSafety() // Should skip check
	if checker.writes.Load() != 0 {
		t.Error("checkSafety() should NOT increment writes when testingSkipSafetyCheck=true (skip mode)")
	}

	// Test with safety checks enabled (testingSkipSafetyCheck = false means DO checks)
	testingSkipSafetyCheck = false
	checker2 := goroutineChecker{}
	checker2.checkSafety() // Should perform check
	if checker2.writes.Load() != 1 {
		t.Error("checkSafety() should increment writes when testingSkipSafetyCheck=false (check mode)")
	}
}

// TestGoroutineCheckerPanic tests panic on cross-goroutine access.
// This test only runs in development mode where safety checks are enabled.
func TestGoroutineCheckerPanic(t *testing.T) {
	// Enable safety checks (false = do checks)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	var checker goroutineChecker

	// Set goroutine ID from current goroutine
	checker.checkGoroutineSafety()

	// Try to access from different goroutine
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic
				done <- true
			} else {
				done <- false
			}
		}()
		// This should panic
		checker.checkGoroutineSafety()
	}()

	select {
	case panicked := <-done:
		if !panicked {
			t.Error("Expected panic when accessing from different goroutine")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for panic")
	}
}

// TestUnsafeBufferGoroutineSafety tests goroutine safety checking.
// This test only runs in development mode where safety checks are enabled.
func TestUnsafeBufferGoroutineSafety(t *testing.T) {
	// Enable safety checks for this test (false = do checks)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(100).(*unsafeBuffer)

	// First write should set goroutine ID
	buf.Write([]byte("first"))

	// Writing from same goroutine should work
	buf.Write([]byte(" second"))

	// Test panic when accessed from different goroutine
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic
				done <- true
			} else {
				done <- false
			}
		}()
		// This should panic
		buf.Write([]byte(" from another goroutine"))
	}()

	select {
	case panicked := <-done:
		if !panicked {
			t.Error("Expected panic when accessing unsafe buffer from different goroutine")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for goroutine safety check")
	}
}

// TestUnsafeShardedBufferGoroutineSafety tests goroutine safety checking.
// This test only runs in development mode where safety checks are enabled.
func TestUnsafeShardedBufferGoroutineSafety(t *testing.T) {
	// Enable safety checks for this test (false = do checks)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// First write should set goroutine ID
	buf.Write([]byte("first"))

	// Writing from same goroutine should work
	buf.Write([]byte(" second"))

	// Test panic when accessed from different goroutine
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic
				done <- true
			} else {
				done <- false
			}
		}()
		// This should panic
		buf.Write([]byte(" from another goroutine"))
	}()

	select {
	case panicked := <-done:
		if !panicked {
			t.Error("Expected panic when accessing unsafe buffer from different goroutine")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for goroutine safety check")
	}
}
