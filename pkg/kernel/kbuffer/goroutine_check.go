package kbuffer

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

// goroutineChecker tracks goroutine ID to detect concurrent access
type goroutineChecker struct {
	gid    atomic.Uint64 // Current goroutine ID (0 = unset)
	writes atomic.Uint64 // Write counter for detection
}

// checkGoroutineSafety panics if called from different goroutine
//
//go:inline
func (g *goroutineChecker) checkGoroutineSafety() {
	currentGID := getCurrentGID()

	// First access - set the goroutine ID
	if g.gid.CompareAndSwap(0, uint64(currentGID)) {
		g.writes.Add(1)
		return
	}

	// Check if same goroutine
	if g.gid.Load() == uint64(currentGID) {
		g.writes.Add(1)
		return
	}

	// DIFFERENT GOROUTINE DETECTED - PANIC!
	panic("kbuffer: UNSAFE buffer accessed from multiple goroutines! " +
		"Use NewSafeBuffer() or NewSafeShardedBuffer() for concurrent access. " +
		"This panic prevents data corruption and undefined behavior.")
}

// getCurrentGID returns current goroutine ID for safety checking
//
//go:nosplit
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

// getCurrentG returns current goroutine pointer from runtime internals
// This function exists for compatibility but is not used in the safe path
//
//go:nosplit
func getCurrentG() unsafe.Pointer {
	// Return a dummy non-nil pointer to avoid nil checks
	// The actual value doesn't matter as we use getCurrentGID() for safety checks
	return unsafe.Pointer(&debugMode)
}

// debugMode controls goroutine safety checks
// Set via build tag: -tags=unsafe_no_check for production builds
var debugMode = true // Default: enabled for safety

// checkSafety is a conditional check based on debug mode
//
//go:inline
func (g *goroutineChecker) checkSafety() {
	if debugMode {
		g.checkGoroutineSafety()
	}
}
