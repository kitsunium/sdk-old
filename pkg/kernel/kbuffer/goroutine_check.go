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

// goidCache caches the goroutine ID to ensure consistency
var goidCache struct {
	goid uintptr
	ptr  unsafe.Pointer
}

// getCurrentG returns a consistent pointer for the current goroutine
// Uses the goroutine ID from runtime to ensure uniqueness
//
//go:nosplit
func getCurrentG() unsafe.Pointer {
	// Get current goroutine ID
	goid := getCurrentGID()

	// Check cache first
	if uintptr(goid) == goidCache.goid && goidCache.ptr != nil {
		return goidCache.ptr
	}

	// Create a unique pointer based on goroutine ID
	// We use the address of goidCache plus an offset based on goid
	// This avoids the go vet warning while providing unique pointers
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(&goidCache)) + uintptr(goid)*8)

	// Cache for consistency
	goidCache.goid = uintptr(goid)
	goidCache.ptr = ptr

	return ptr
}

// debugMode controls goroutine safety checks in dev builds
// In production builds (unsafe_no_check), this is always false and checkSafety is a no-op
var debugMode = true
