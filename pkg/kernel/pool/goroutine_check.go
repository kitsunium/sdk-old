//go:build !unsafe_no_check
// +build !unsafe_no_check

// Package pool provides ultra-optimized, lock-free byte buffers for kernel operations.
// This file contains goroutine safety checking for development builds.
// Safety checks are enabled by default and can detect concurrent access to unsafe buffers.
package pool

import (
	"runtime"
	"sync/atomic"
)

const (
	// sampleMask determines sampling frequency (1 in 512 checks).
	// This reduces the overhead of safety checking by sampling operations.
	// Set to 511 (binary: 111111111) for efficient bitwise AND operation.
	sampleMask = uint32(511)
)

// testingSkipSafetyCheck is used only in tests to temporarily disable safety checks.
// This allows tests to bypass goroutine checking when testing specific scenarios.
var testingSkipSafetyCheck bool

// goroutineChecker tracks goroutine ID to detect concurrent access.
// Uses atomic operations to detect when an unsafe buffer is accessed
// from multiple goroutines, which would cause data corruption.
type goroutineChecker struct {
	gid     atomic.Uint64 // Current goroutine ID (0 = unset)
	writes  atomic.Uint64 // Write counter for detection
	counter atomic.Uint32 // Sampling counter for amortized checks
}

// checkSafety performs goroutine safety checks in development builds.
// This is the main entry point for safety checking. In development builds,
// it will panic if concurrent access is detected. In production builds,
// this becomes a no-op for maximum performance.
func (g *goroutineChecker) checkSafety() {
	// Allow tests to skip safety checks
	if !testingSkipSafetyCheck {
		g.checkGoroutineSafety()
	}
}

// checkGoroutineSafety panics if called from different goroutine.
// Uses amortized sampling to reduce overhead of getCurrentGID calls.
// This is the core safety checking logic that detects concurrent access
// and panics with a helpful error message to prevent data corruption.
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
			panic("pool: UNSAFE buffer accessed from multiple goroutines! " +
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
			panic("pool: UNSAFE buffer accessed from multiple goroutines! " +
				"Use NewSafeBuffer() or NewSafeShardedBuffer() for concurrent access. " +
				"This panic prevents data corruption and undefined behavior.")
		}
	}

	// Increment write counter
	g.writes.Add(1)
}

// getCurrentGID returns current goroutine ID for safety checking.
// Uses runtime.Stack to get goroutine information and hashes it to create
// a unique identifier. This is more expensive than atomic operations but
// necessary for detecting goroutine switches.
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
