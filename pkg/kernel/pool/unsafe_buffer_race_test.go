//go:build !race
// +build !race

package pool

import (
	"runtime"
	"sync"
	"testing"
)

// TestUnsafeBufferDataRace intentionally tests for data races when misused.
// This test is excluded when the race detector is enabled.
func TestUnsafeBufferDataRace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	// Disable safety checks to allow concurrent access (for testing only!)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = true // true = skip checks
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeBuffer(10000).(*unsafeBuffer)

	// WARNING: This deliberately creates a data race for testing
	// DO NOT do this in production code!

	var wg sync.WaitGroup
	const goroutines = 10
	wg.Add(goroutines)

	// Multiple goroutines writing concurrently (UNSAFE!)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			data := []byte{byte(id)}
			for j := 0; j < 100; j++ {
				// This may corrupt data or panic
				_, _ = buf.Write(data)
				runtime.Gosched()
			}
		}(i)
	}

	wg.Wait()

	// The buffer is likely corrupted at this point
	// This test demonstrates why unsafe buffers should never be used concurrently
	t.Logf("Data race test completed (buffer likely corrupted): Len=%d", buf.Len())
}
