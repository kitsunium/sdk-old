// Package pool provides high-performance byte buffers with a global sync.Pool.
// This file contains the spinLock implementation used by safeBuffer.
package pool

import (
	"runtime"     // For Gosched() in spinlock backoff
	"sync/atomic" // For atomic operations
)

// spinLock is a lightweight spinlock for short critical sections.
// More efficient than mutex for our use case (short writes).
// Uses atomic CAS operations for lock acquisition.
type spinLock struct {
	lock atomic.Uint32 // Lock state: 0=unlocked, 1=locked
}

// Lock acquires the spinlock.
// Uses exponential backoff to reduce contention.
// Blocks until lock is acquired.
//
//go:nosplit
func (s *spinLock) Lock() {
	backoff := 1                       // Initial backoff count
	for !s.lock.CompareAndSwap(0, 1) { // Try to acquire lock
		// Exponential backoff to reduce contention
		for i := 0; i < backoff; i++ { // Backoff loop
			runtime.Gosched() // Yield to other goroutines
		}
		if backoff < 32 { // Cap maximum backoff
			backoff <<= 1 // Double backoff time
		}
	}
}

// Unlock releases the spinlock.
// Must be called after Lock().
//
//go:nosplit
func (s *spinLock) Unlock() {
	s.lock.Store(0) // Release lock atomically
}

// TryLock attempts to acquire without blocking.
// Returns true if lock was acquired, false otherwise.
//
//go:nosplit
func (s *spinLock) TryLock() bool {
	return s.lock.CompareAndSwap(0, 1) // Try atomic CAS
}
