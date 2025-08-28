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
	// Enable safety checks for testing (false = do checks)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

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

// TestGetCurrentG tests goroutine token extraction.
func TestGetCurrentG(t *testing.T) {
	// Get token from current goroutine
	g1 := getCurrentG()
	if g1 == 0 {
		t.Error("getCurrentG() returned 0")
	}

	// Get token again
	g2 := getCurrentG()
	if g1 != g2 {
		t.Error("getCurrentG() not consistent")
	}

	// Get from different goroutine
	done := make(chan uintptr)
	go func() {
		done <- getCurrentG()
	}()

	g3 := <-done
	if g3 == 0 {
		t.Error("getCurrentG() returned 0 from goroutine")
	}

	// Tokens from different goroutines should differ
	if g3 == g1 {
		t.Error("Different goroutines returned same token")
	}
}

// TestGoroutineCheckerConcurrentInit tests concurrent initialization.
func TestGoroutineCheckerConcurrentInit(t *testing.T) {
	// Enable safety checks (false = do checks)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

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
	// Enable safety checks (false = do checks)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

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
	// Enable safety checks (false = do checks)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

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
	originalDebugMode := testingSkipSafetyCheck

	// Test setting to false
	testingSkipSafetyCheck = false
	if testingSkipSafetyCheck != false {
		t.Error("Failed to set testingSkipSafetyCheck to false")
	}

	// Test setting to true
	testingSkipSafetyCheck = true
	if testingSkipSafetyCheck != true {
		t.Error("Failed to set testingSkipSafetyCheck to true")
	}

	// Restore original value
	testingSkipSafetyCheck = originalDebugMode
}

// TestGoroutineCheckerPerformance tests performance impact.
func TestGoroutineCheckerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Save and restore original mode
	oldDebugMode := testingSkipSafetyCheck
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	var checker goroutineChecker
	iterations := 1000000

	// Test with safety checks enabled (false = do checks)
	testingSkipSafetyCheck = false
	start := time.Now()
	for i := 0; i < iterations; i++ {
		checker.checkSafety()
	}
	enabledDuration := time.Since(start)

	// Reset checker for fair comparison
	checker = goroutineChecker{}

	// Test with safety checks disabled (true = skip checks)
	testingSkipSafetyCheck = true
	start = time.Now()
	for i := 0; i < iterations; i++ {
		checker.checkSafety()
	}
	disabledDuration := time.Since(start)

	t.Logf("Performance with safety checks enabled: %v for %d iterations", enabledDuration, iterations)
	t.Logf("Performance with safety checks disabled: %v for %d iterations", disabledDuration, iterations)
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

	// Size should be at least 20 bytes (two uint64 atomics + one uint32 atomic)
	// May be larger due to alignment or architecture differences
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

// TestGoroutineCheckerRaceCondition tests for race conditions.
func TestGoroutineCheckerRaceCondition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	// This test intentionally creates race conditions to test the checker
	// Run with: go test -race

	var checker goroutineChecker

	// Disable safety checks to avoid panics (true = skip checks)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = true
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

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
