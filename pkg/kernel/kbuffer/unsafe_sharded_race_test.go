//go:build !race
// +build !race

package kbuffer

import (
	"runtime"
	"sync"
	"testing"
)

// TestUnsafeShardedBufferDataRace intentionally tests for data races.
// This test is excluded when the race detector is enabled.
func TestUnsafeShardedBufferDataRace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	// Disable safety checks to allow concurrent access (for testing only!)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = true // true = skip checks
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(10000, 8).(*unsafeShardedBuffer)

	// WARNING: This deliberately creates a data race
	var wg sync.WaitGroup
	const goroutines = 10
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			data := []byte{byte(id)}
			for j := 0; j < 100; j++ {
				// This will corrupt data or panic
				_, _ = buf.Write(data)
				runtime.Gosched()
			}
		}(i)
	}

	wg.Wait()

	// Buffer is likely corrupted
	t.Logf("Data race test completed (buffer likely corrupted): Len=%d", buf.Len())
}
