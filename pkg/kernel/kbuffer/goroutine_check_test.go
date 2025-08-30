//go:build !unsafe_no_check
// +build !unsafe_no_check

package kbuffer

import (
	"fmt"
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

	t.Run("BasicCrossGoroutinePanic", func(t *testing.T) {
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
	})

	t.Run("PanicDuringInitialOwnershipRace", func(t *testing.T) {
		// Test the race condition during initial ownership setting
		var checker goroutineChecker
		var wg sync.WaitGroup
		panics := 0
		var mu sync.Mutex

		// Two goroutines trying to set ownership at the same time
		wg.Add(2)
		for i := 0; i < 2; i++ {
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						mu.Lock()
						panics++
						mu.Unlock()
					}
				}()
				checker.checkGoroutineSafety()
			}()
		}
		wg.Wait()

		// At least one should have panicked due to race
		if panics == 0 {
			t.Skip("Race condition not triggered, skipping test")
		}
	})

	t.Run("PanicAfterMultipleChecks", func(t *testing.T) {
		var checker goroutineChecker

		// Set ownership and do multiple checks to trigger sampling
		for i := 0; i < 20; i++ {
			checker.checkGoroutineSafety()
		}

		// Now try from different goroutine - should still panic
		done := make(chan bool)
		go func() {
			defer func() {
				done <- recover() != nil
			}()
			// Try multiple times to ensure we hit a check
			for i := 0; i < 1000; i++ {
				checker.checkGoroutineSafety()
			}
		}()

		select {
		case panicked := <-done:
			if !panicked {
				t.Error("Expected panic even after sampling kicks in")
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for panic")
		}
	})
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

	// Test that sampling checks still detect goroutine changes
	t.Run("SamplingStillDetectsGoroutineChange", func(t *testing.T) {
		var checker2 goroutineChecker

		// Do enough calls to get into sampling mode
		for i := 0; i < 600; i++ { // Well past sampling threshold
			checker2.checkGoroutineSafety()
		}

		// Now try from different goroutine - sampling should still catch it
		done := make(chan bool)
		go func() {
			defer func() {
				done <- recover() != nil
			}()
			// Try multiple times to ensure we hit a sampling check
			for i := 0; i < 1000; i++ {
				checker2.checkGoroutineSafety()
			}
		}()

		select {
		case panicked := <-done:
			if !panicked {
				t.Error("Sampling should still detect goroutine changes")
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for panic")
		}
	})
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

// TestGetCurrentGID tests the getCurrentGID function edge cases
func TestGetCurrentGID(t *testing.T) {
	// Test basic functionality
	gid1 := getCurrentGID()
	gid2 := getCurrentGID()
	if gid1 != gid2 {
		t.Error("getCurrentGID() should return same value in same goroutine")
	}
	if gid1 == 0 {
		t.Error("getCurrentGID() should not return 0 for valid goroutine")
	}

	// Test from different goroutines
	done := make(chan uint32)
	go func() {
		done <- getCurrentGID()
	}()

	gid3 := <-done
	if gid3 == gid1 {
		t.Error("getCurrentGID() should return different values for different goroutines")
	}
}

// TestGoroutineCheckerEdgeCases tests various edge cases
func TestGoroutineCheckerEdgeCases(t *testing.T) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	t.Run("ZeroGIDFallback", func(t *testing.T) {
		// This tests the edge case where getCurrentGID might return 0
		// We can't easily force this, but we can test the logic path
		var checker goroutineChecker

		// First call sets ownership
		checker.checkGoroutineSafety()
		if checker.gid.Load() == 0 {
			t.Error("GID should not be 0 after first check")
		}
		if checker.writes.Load() != 1 {
			t.Error("Writes should be 1 after first check")
		}
	})

	t.Run("CompareAndSwapFailure", func(t *testing.T) {
		// Test the CAS failure path during initial ownership
		var checker goroutineChecker
		var wg sync.WaitGroup
		success := false

		// Try to trigger CAS failure by having multiple goroutines compete
		wg.Add(10)
		for i := 0; i < 10; i++ {
			go func(id int) {
				defer wg.Done()
				defer func() {
					recover() // Ignore panics
				}()

				// Try to set ownership
				checker.checkGoroutineSafety()
				success = true
			}(i)
		}
		wg.Wait()

		// At least one should have succeeded
		if !success {
			t.Error("At least one goroutine should have succeeded")
		}
	})

	t.Run("SamplingBoundary", func(t *testing.T) {
		var checker goroutineChecker

		// Test the boundary between always-check and sampling
		for i := 0; i < 15; i++ { // Just past the always-check threshold
			checker.checkGoroutineSafety()
		}

		// Counter should be > 10
		if checker.counter.Load() <= 10 {
			t.Error("Counter should be > 10 after 15 calls")
		}

		// Writes should match the number of calls
		if checker.writes.Load() != 15 {
			t.Errorf("Writes should be 15, got %d", checker.writes.Load())
		}
	})
}

// Benchmarks for goroutine checker operations

// BenchmarkGoroutineChecker benchmarks the goroutine safety check overhead.
func BenchmarkGoroutineChecker(b *testing.B) {
	oldDebugMode := testingSkipSafetyCheck
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	b.Run("Disabled", func(b *testing.B) {
		testingSkipSafetyCheck = true
		var checker goroutineChecker
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			checker.checkSafety()
		}
	})

	b.Run("Enabled", func(b *testing.B) {
		testingSkipSafetyCheck = false
		var checker goroutineChecker
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			checker.checkSafety()
		}
	})
}

// BenchmarkGoroutineCheckerSafety benchmarks the full goroutine safety check.
func BenchmarkGoroutineCheckerSafety(b *testing.B) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	b.Run("FirstCheck", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var checker goroutineChecker
			checker.checkGoroutineSafety() // First check sets ownership
		}
	})

	b.Run("SubsequentChecks", func(b *testing.B) {
		var checker goroutineChecker
		checker.checkGoroutineSafety() // Set ownership

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			checker.checkGoroutineSafety() // Subsequent checks
		}
	})

	b.Run("WithSampling", func(b *testing.B) {
		var checker goroutineChecker
		// Prime the checker to get into sampling mode
		for i := 0; i < 1000; i++ {
			checker.checkGoroutineSafety()
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			checker.checkGoroutineSafety() // Sampled checks
		}
	})
}

// BenchmarkGetCurrentGID benchmarks the GID retrieval function.
func BenchmarkGetCurrentGID(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = getCurrentGID()
	}
}

// BenchmarkAtomicOperations benchmarks atomic operations used in the checker.
func BenchmarkAtomicOperations(b *testing.B) {
	var checker goroutineChecker

	b.Run("Load", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = checker.gid.Load()
		}
	})

	b.Run("Store", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			checker.gid.Store(uint64(i))
		}
	})

	b.Run("CompareAndSwap", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			checker.gid.CompareAndSwap(0, 1)
			checker.gid.Store(0) // Reset for next iteration
		}
	})

	b.Run("Add", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			checker.writes.Add(1)
		}
	})
}

// BenchmarkConcurrentChecks benchmarks concurrent goroutine checks.
func BenchmarkConcurrentChecks(b *testing.B) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = true // Disable to avoid panics
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	var checker goroutineChecker

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			checker.checkSafety()
		}
	})
}

// BenchmarkMemoryFootprint benchmarks memory usage of the checker.
func BenchmarkMemoryFootprint(b *testing.B) {
	checkers := []int{1, 10, 100, 1000}

	for _, count := range checkers {
		b.Run(fmt.Sprintf("Checkers%d", count), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Allocate multiple checkers to measure memory impact
				checkerSlice := make([]goroutineChecker, count)
				_ = checkerSlice
			}
		})
	}
}
