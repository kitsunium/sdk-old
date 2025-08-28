//go:build !unsafe_no_check
// +build !unsafe_no_check

package kbuffer

import (
	"runtime"
	"sync/atomic"
)

const (
	// sampleMask determines sampling frequency (1 in 512 checks)
	sampleMask = uint32(511)
)

// testingSkipSafetyCheck is used only in tests to temporarily disable safety checks
var testingSkipSafetyCheck bool

// goroutineChecker tracks goroutine ID to detect concurrent access
type goroutineChecker struct {
	gid     atomic.Uint64 // Current goroutine ID (0 = unset)
	writes  atomic.Uint64 // Write counter for detection
	counter atomic.Uint32 // Sampling counter for amortized checks
}

// checkSafety performs goroutine safety checks in development builds
func (g *goroutineChecker) checkSafety() {
	// Allow tests to skip safety checks
	if !testingSkipSafetyCheck {
		g.checkGoroutineSafety()
	}
}

// checkGoroutineSafety panics if called from different goroutine
// Uses amortized sampling to reduce overhead of getCurrentGID calls
func (g *goroutineChecker) checkGoroutineSafety() {
	// Load current owner
	currentOwner := g.gid.Load()

	// First access - always check and set ownership
	if currentOwner == 0 {
		currentGID := getCurrentGID()
		// Safeguard against unlikely case where getCurrentGID returns 0
		if currentGID == 0 {
			currentGID = 1 // Use non-zero fallback
		}
		if g.gid.CompareAndSwap(0, uint64(currentGID)) {
			g.writes.Add(1)
			return
		}
		// Lost race, re-check ownership
		currentOwner = g.gid.Load()
		if currentOwner != uint64(currentGID) {
			panic("kbuffer: UNSAFE buffer accessed from multiple goroutines! " +
				"Use NewSafeBuffer() or NewSafeShardedBuffer() for concurrent access. " +
				"This panic prevents data corruption and undefined behavior.")
		}
		g.writes.Add(1)
		return
	}

	// Increment counter for sampling
	count := g.counter.Add(1)

	// Always check on first few accesses to catch early violations
	// Then sample periodically to reduce overhead
	if count <= 10 || (count-1)&sampleMask == 0 {
		currentGID := getCurrentGID()
		if currentOwner != uint64(currentGID) {
			// DIFFERENT GOROUTINE DETECTED - PANIC!
			panic("kbuffer: UNSAFE buffer accessed from multiple goroutines! " +
				"Use NewSafeBuffer() or NewSafeShardedBuffer() for concurrent access. " +
				"This panic prevents data corruption and undefined behavior.")
		}
	}

	// Increment write counter
	g.writes.Add(1)
}

// getCurrentGID returns current goroutine ID for safety checking
func getCurrentGID() uint32 {
	// Use runtime.Stack to get goroutine info
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	if n > 0 {
		// Hash only first 16 bytes which contain goroutine ID
		// This avoids expensive hashing of entire stack trace
		limit := n
		if limit > 16 {
			limit = 16
		}
		hash := uint32(0)
		for i := 0; i < limit; i++ {
			hash = hash*31 + uint32(buf[i])
		}
		return hash
	}
	return 0
}
