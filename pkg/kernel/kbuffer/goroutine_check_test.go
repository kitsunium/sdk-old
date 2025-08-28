package kbuffer

import (
	"runtime"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// TestGoroutineChecker tests the goroutine safety checker.
func TestGoroutineChecker(t *testing.T) {
	// Enable debug mode for testing
	oldDebugMode := debugMode
	debugMode = true
	defer func() { debugMode = oldDebugMode }()

	var checker goroutineChecker

	// First call should set the goroutine ID
	checker.checkGoroutineSafety()

	// Verify writes counter incremented
	if checker.writes.Load() != 1 {
		t.Errorf("writes counter = %d, want 1", checker.writes.Load())
	}

	// Second call from same goroutine should succeed
	checker.checkGoroutineSafety()

	if checker.writes.Load() != 2 {
		t.Errorf("writes counter = %d, want 2", checker.writes.Load())
	}

	// Verify goroutine ID was set
	if checker.gid.Load() == 0 {
		t.Error("goroutine ID not set")
	}
}

// TestGoroutineCheckerPanic tests panic on cross-goroutine access.
func TestGoroutineCheckerPanic(t *testing.T) {
	// Enable debug mode
	oldDebugMode := debugMode
	debugMode = true
	defer func() { debugMode = oldDebugMode }()

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

// TestGoroutineCheckerCheckSafety tests conditional checking.
func TestGoroutineCheckerCheckSafety(t *testing.T) {
	var checker goroutineChecker

	// Test with debug mode enabled
	debugMode = true
	checker.checkSafety() // Should perform check
	if checker.writes.Load() != 1 {
		t.Error("checkSafety() should increment writes when debugMode=true")
	}

	// Test with debug mode disabled
	debugMode = false
	checker2 := goroutineChecker{}
	checker2.checkSafety() // Should skip check
	if checker2.writes.Load() != 0 {
		t.Error("checkSafety() should not increment writes when debugMode=false")
	}

	// Reset debug mode
	debugMode = true
}

// TestGetCurrentGID tests goroutine ID extraction.
func TestGetCurrentGID(t *testing.T) {
	// Get ID from current goroutine
	id1 := getCurrentGID()
	if id1 == 0 {
		t.Error("getCurrentGID() returned 0")
	}

	// Get ID again from same goroutine
	id2 := getCurrentGID()
	if id1 != id2 {
		t.Errorf("getCurrentGID() not consistent: %d != %d", id1, id2)
	}

	// Get ID from different goroutine
	done := make(chan uint32)
	go func() {
		done <- getCurrentGID()
	}()

	id3 := <-done
	if id3 == 0 {
		t.Error("getCurrentGID() returned 0 from goroutine")
	}

	// IDs from different goroutines should differ
	if id3 == id1 {
		t.Errorf("Different goroutines returned same ID: %d", id3)
	}
}

// TestGetCurrentG tests goroutine pointer extraction.
func TestGetCurrentG(t *testing.T) {
	// Get pointer from current goroutine
	g1 := getCurrentG()
	if g1 == nil {
		t.Error("getCurrentG() returned nil")
	}

	// Get pointer again
	g2 := getCurrentG()
	if g1 != g2 {
		t.Error("getCurrentG() not consistent")
	}

	// Get from different goroutine
	done := make(chan unsafe.Pointer)
	go func() {
		done <- getCurrentG()
	}()

	g3 := <-done
	if g3 == nil {
		t.Error("getCurrentG() returned nil from goroutine")
	}

	// Pointers from different goroutines should differ
	if g3 == g1 {
		t.Error("Different goroutines returned same pointer")
	}
}

// TestGoroutineCheckerConcurrentInit tests concurrent initialization.
func TestGoroutineCheckerConcurrentInit(t *testing.T) {
	// Enable debug mode
	oldDebugMode := debugMode
	debugMode = true
	defer func() { debugMode = oldDebugMode }()

	var checker goroutineChecker
	var wg sync.WaitGroup
	const goroutines = 10

	// Track which goroutine wins the race
	winnerChan := make(chan int, goroutines)
	panicChan := make(chan int, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					// This goroutine panicked (lost the race)
					panicChan <- id
				}
			}()

			// Try to initialize
			checker.checkGoroutineSafety()

			// If we get here, we won the race
			winnerChan <- id
		}(i)
	}

	wg.Wait()
	close(winnerChan)
	close(panicChan)

	// Count winners and panics
	winners := 0
	panics := 0

	for range winnerChan {
		winners++
	}
	for range panicChan {
		panics++
	}

	// Exactly one should win, others should panic
	if winners != 1 {
		t.Errorf("Expected 1 winner, got %d", winners)
	}
	if panics != goroutines-1 {
		t.Errorf("Expected %d panics, got %d", goroutines-1, panics)
	}
}

// TestGoroutineCheckerCompareAndSwap tests atomic initialization.
func TestGoroutineCheckerCompareAndSwap(t *testing.T) {
	// Enable debug mode
	oldDebugMode := debugMode
	debugMode = true
	defer func() { debugMode = oldDebugMode }()

	var checker goroutineChecker
	currentGID := getCurrentGID()

	// First CAS should succeed (0 -> currentGID)
	if !checker.gid.CompareAndSwap(0, uint64(currentGID)) {
		t.Error("First CompareAndSwap should succeed")
	}

	// Second CAS with same value should fail
	if checker.gid.CompareAndSwap(0, uint64(currentGID)) {
		t.Error("Second CompareAndSwap with 0 should fail")
	}

	// CAS with current value should succeed
	if !checker.gid.CompareAndSwap(uint64(currentGID), uint64(currentGID)) {
		t.Error("CompareAndSwap with current value should succeed")
	}
}

// TestGoroutineCheckerWriteCounter tests write counting.
func TestGoroutineCheckerWriteCounter(t *testing.T) {
	// Enable debug mode
	oldDebugMode := debugMode
	debugMode = true
	defer func() { debugMode = oldDebugMode }()

	var checker goroutineChecker

	// Multiple writes from same goroutine
	for i := 0; i < 100; i++ {
		checker.checkGoroutineSafety()
	}

	// Verify counter
	if checker.writes.Load() != 100 {
		t.Errorf("writes counter = %d, want 100", checker.writes.Load())
	}
}

// TestDebugModeGlobal tests the global debug mode flag.
func TestDebugModeGlobal(t *testing.T) {
	// Store original value
	originalDebugMode := debugMode

	// Test setting to false
	debugMode = false
	if debugMode != false {
		t.Error("Failed to set debugMode to false")
	}

	// Test setting to true
	debugMode = true
	if debugMode != true {
		t.Error("Failed to set debugMode to true")
	}

	// Restore original value
	debugMode = originalDebugMode
}

// TestGoroutineCheckerPerformance tests performance impact.
func TestGoroutineCheckerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	var checker goroutineChecker
	iterations := 1000000

	// Test with debug mode enabled
	debugMode = true
	start := time.Now()
	for i := 0; i < iterations; i++ {
		checker.checkSafety()
	}
	enabledDuration := time.Since(start)

	// Reset checker for fair comparison
	checker = goroutineChecker{}

	// Test with debug mode disabled
	debugMode = false
	start = time.Now()
	for i := 0; i < iterations; i++ {
		checker.checkSafety()
	}
	disabledDuration := time.Since(start)

	// Reset debug mode
	debugMode = true

	t.Logf("Performance with debug mode enabled: %v for %d iterations", enabledDuration, iterations)
	t.Logf("Performance with debug mode disabled: %v for %d iterations", disabledDuration, iterations)
	t.Logf("Overhead ratio: %.2fx", float64(enabledDuration)/float64(disabledDuration))

	// Disabled should be significantly faster
	if disabledDuration > enabledDuration/2 {
		t.Log("Warning: Debug mode disabled not significantly faster")
	}
}

// TestGoroutineCheckerMemoryUsage tests memory footprint.
func TestGoroutineCheckerMemoryUsage(t *testing.T) {
	// Test size of goroutineChecker struct
	var checker goroutineChecker
	size := unsafe.Sizeof(checker)

	// Should be exactly 16 bytes (two uint64 atomics)
	if size != 16 {
		t.Errorf("goroutineChecker size = %d bytes, want 16", size)
	}

	// Test alignment
	if uintptr(unsafe.Pointer(&checker.gid))%8 != 0 {
		t.Error("gid field not 8-byte aligned")
	}
	if uintptr(unsafe.Pointer(&checker.writes))%8 != 0 {
		t.Error("writes field not 8-byte aligned")
	}
}

// TestGoroutineCheckerRaceCondition tests for race conditions.
func TestGoroutineCheckerRaceCondition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	// This test intentionally creates race conditions to test the checker
	// Run with: go test -race

	var checker goroutineChecker

	// Disable debug mode to avoid panics
	debugMode = false
	defer func() { debugMode = true }()

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
				runtime.Gosched()
			}
		}()
	}

	wg.Wait()

	// If we get here without race detector complaints, atomics are working
	finalWrites := checker.writes.Load()
	expectedWrites := uint64(goroutines * iterations)
	if finalWrites != expectedWrites {
		t.Errorf("Final writes = %d, want %d", finalWrites, expectedWrites)
	}
}
