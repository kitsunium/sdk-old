package kbuffer

import (
	"runtime"
	"sync/atomic"
)

const (
	// sampleMask determines sampling frequency (1 in 512 checks)
	sampleMask = uint32(511)
)

// goroutineChecker tracks goroutine ID to detect concurrent access
type goroutineChecker struct {
	gid     atomic.Uint64 // Current goroutine ID (0 = unset)
	writes  atomic.Uint64 // Write counter for detection
	counter atomic.Uint32 // Sampling counter for amortized checks
}

// checkGoroutineSafety panics if called from different goroutine
// Uses amortized sampling to reduce overhead of getCurrentGID calls
func (g *goroutineChecker) checkGoroutineSafety() {
	// Increment counter atomically
	count := g.counter.Add(1)

	// Load current owner
	currentOwner := g.gid.Load()

	// First access - always check and set ownership
	if currentOwner == 0 {
		currentGID := getCurrentGID()
		if g.gid.CompareAndSwap(0, uint64(currentGID)) {
			g.writes.Add(1)
			return
		}
		// Lost race, re-load owner
		currentOwner = g.gid.Load()
	}

	// Sample check: only call expensive getCurrentGID periodically
	if (count-1)&sampleMask == 0 {
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
// Note: This function calls runtime.Stack and must not be marked nosplit
func getCurrentGID() uint32 {
	// Use runtime.Stack to get goroutine info
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	if n > 0 {
		// Hash stack trace for pseudo-ID
		hash := uint32(0)
		for i := 0; i < n; i++ {
			hash = hash*31 + uint32(buf[i])
		}
		return hash
	}
	return 0
}

// getCurrentG returns a deterministic token for the current goroutine
// Returns a uintptr-based token instead of an unsafe.Pointer to avoid GC issues
func getCurrentG() uintptr {
	// Get current goroutine ID and return as token
	// This avoids unsafe pointer forging and global state mutation
	goid := getCurrentGID()
	// Create deterministic token from goroutine ID
	// Use a simple transformation that provides uniqueness
	return uintptr(goid) * 0x9E3779B9 // Golden ratio prime for better distribution
}
