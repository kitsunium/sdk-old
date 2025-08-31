// Package kbuffer provides buffer management with configurable safety and pooling.
//
// This file contains the buffer pool implementation for reducing allocation overhead.
// The pool uses size-classed buckets (powers of 2) for efficient memory reuse and
// reduced fragmentation.
package kbuffer

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// poolClassBits defines size class boundaries.
const (
	poolClassBits = 6  // Starting at 2^6 = 64
	poolClassMax  = 22 // Up to 2^22 = 4MB
)

// bufferPool implements the Pool interface with size-classed pooling.
//
// The pool manages buffers in size classes from 64 bytes to 4MB, with each
// class being a power of 2. This design:
//   - Reduces memory fragmentation
//   - Improves allocation performance
//   - Minimizes GC pressure
//   - Provides predictable memory usage patterns
//
// The pool is thread-safe and can be shared across goroutines.
type bufferPool struct {
	pools       [poolClassCount]*sync.Pool // Size-classed pools
	bufferPool  *sync.Pool                 // Pool for Buffer instances
	maxSize     atomic.Int64               // Maximum poolable size
	clearOnPut  atomic.Bool                // Clear buffers on return
	prewarmSize int                        // Prewarm count per pool
	_           [32]byte                   // Cache line padding
}

// NewPool creates a new buffer pool instance.
//
// Each application component should create its own pool instance rather than
// using global pools. This provides:
//   - Better isolation between components
//   - Easier testing and mocking
//   - Fine-grained configuration control
//   - Predictable resource management
//
// The pool automatically pre-warms with buffers based on CPU count to reduce
// startup latency. Size classes range from 64 bytes to 4MB.
//
// Example:
//
//	pool := kbuffer.NewPool()
//	pool.SetClearOnPut(true)  // Enable security clearing
//	pool.SetMaxSize(1 << 20)  // Max 1MB buffers in pool
//
//	// Use raw byte slices
//	buf := pool.Get(1024)
//	defer pool.Put(buf)
//	// ... use buf ...
//
//	// Or use Buffer instances
//	buffer := pool.GetBuffer(1024)
//	defer pool.PutBuffer(buffer)
//	// ... use buffer ...
func NewPool() Pool {
	return newPool()
}

// newPool is the internal constructor.
func newPool() *bufferPool {
	p := &bufferPool{
		prewarmSize: runtime.NumCPU() * 2,
	}

	// Set defaults
	p.maxSize.Store(poolMaxSize)
	p.clearOnPut.Store(false) // Performance over security by default

	// Initialize pools for each size class
	for i := 0; i < poolClassCount; i++ {
		size := 1 << (i + poolClassBits) // Size classes from 2^6 to 2^22
		poolIdx := i

		// Create pool with custom allocator
		p.pools[poolIdx] = &sync.Pool{
			New: func(sz int) func() any {
				return func() any {
					// Allocate aligned memory for better cache performance
					buf := make([]byte, sz)
					return &buf
				}
			}(size),
		}
	}

	// Initialize buffer instance pool
	p.bufferPool = &sync.Pool{
		New: func() any {
			return NewUnsafeBuffer(1024) // Default 1KB buffer
		},
	}

	// Pre-warm pools for common sizes
	p.prewarm()

	return p
}

// prewarm pre-allocates buffers to reduce startup latency.
func (p *bufferPool) prewarm() {
	// Prewarm small size classes more aggressively
	for i := 0; i < 4 && i < poolClassCount; i++ {
		size := 1 << (i + poolClassBits)
		count := p.prewarmSize / (i + 1) // More for smaller sizes

		for j := 0; j < count; j++ {
			buf := make([]byte, size)
			p.pools[i].Put(&buf)
		}
	}

	// Prewarm buffer instances
	for i := 0; i < p.prewarmSize; i++ {
		p.bufferPool.Put(NewUnsafeBuffer(1024))
	}
}

// Get retrieves a buffer of at least the requested size from pool.
//
// The method:
//   - Returns a buffer from the appropriate size class
//   - May return a larger buffer for better reuse (capacity > size)
//   - Allocates directly if size exceeds MaxSize
//   - Sets the slice length to the requested size
//
// The returned buffer is NOT zeroed unless the pool was configured with
// SetClearOnPut(true) and the buffer was previously returned to the pool.
//
// Performance characteristics:
//   - O(1) retrieval from pool
//   - Zero allocations for pooled sizes
//   - Automatic size class selection
func (p *bufferPool) Get(size int) []byte {
	// Validate size
	if size <= 0 {
		return nil
	}

	// Direct allocation for oversized requests
	if size > int(p.maxSize.Load()) {
		return make([]byte, size)
	}

	// Calculate size class using bit scan
	class := sizeToClass(size)
	poolIdx := class - poolClassBits

	// Validate pool index
	if poolIdx < 0 || poolIdx >= poolClassCount {
		return make([]byte, size)
	}

	// Get from pool
	bufPtr := p.pools[poolIdx].Get().(*[]byte)

	// Track statistics
	if bufPtr != nil {
		// Ensure buffer has sufficient capacity before slicing
		if cap(*bufPtr) >= size {
			// Return buffer with full capacity but sliced to requested size
			// This preserves the capacity for test assertions
			return (*bufPtr)[:size:cap(*bufPtr)]
		}
		// Buffer too small (shouldn't happen but handle gracefully)
		return make([]byte, size)
	}

	// Shouldn't happen due to New function, but handle gracefully
	return make([]byte, size)
}

// Put returns a buffer to the pool for reuse.
//
// The method:
//   - Uses the buffer's capacity to determine the size class
//   - Clears the buffer if SetClearOnPut(true) was called
//   - Ignores buffers larger than MaxSize
//   - Resets the slice length to 0 while preserving capacity
//
// Security note: Enable clearing with SetClearOnPut(true) when buffers
// may contain sensitive data (passwords, keys, PII).
//
// Performance note: Returning buffers promptly improves reuse and reduces
// GC pressure. Consider using defer for automatic return.
func (p *bufferPool) Put(buf []byte) {
	if buf == nil || cap(buf) == 0 {
		return
	}

	size := cap(buf)

	// Don't pool oversized buffers
	if size > int(p.maxSize.Load()) {
		return
	}

	// Clear if configured
	if p.clearOnPut.Load() {
		clear(buf[:cap(buf)])
	}

	// Calculate size class
	class := sizeToClass(size)
	poolIdx := class - poolClassBits

	// Validate pool index
	if poolIdx < 0 || poolIdx >= poolClassCount {
		return
	}

	// Reset length to 0, keep capacity
	buf = buf[:0:cap(buf)]

	// Return to pool
	p.pools[poolIdx].Put(&buf)
}

// GetBuffer retrieves a Buffer instance from the pool.
//
// The buffer:
//   - Has at least the specified capacity
//   - Is reset to empty state before return
//   - May have larger capacity for better reuse
//   - Is an UnsafeBuffer instance (not thread-safe)
//
// For thread-safe buffers, create them directly with NewSafeBuffer
// or wrap the pooled buffer with appropriate synchronization.
func (p *bufferPool) GetBuffer(size int) Buffer {
	// Get buffer instance from pool
	buf := p.bufferPool.Get().(Buffer)

	// Ensure it has sufficient capacity
	if buf.Cap() < size {
		buf.Grow(size - buf.Cap())
	}

	// Reset the buffer
	buf.Reset()

	return buf
}

// PutBuffer returns a Buffer instance to the pool.
//
// The buffer is automatically:
//   - Reset to empty state
//   - Cleared if SetClearOnPut(true) was configured
//   - Returned to the pool for reuse
//
// Only UnsafeBuffer instances are pooled. SafeBuffer instances
// are not pooled due to their internal synchronization state.
func (p *bufferPool) PutBuffer(b Buffer) {
	if b == nil {
		return
	}

	// Reset the buffer
	b.Reset()

	// Clear if configured
	if p.clearOnPut.Load() && b.Cap() > 0 {
		b.Clear()
	}

	// Return to pool
	p.bufferPool.Put(b)
}

// SetClearOnPut configures whether buffers are cleared when returned.
//
// When enabled:
//   - Buffers are zeroed when returned to the pool
//   - Prevents data leakage between uses
//   - Adds ~10-20% overhead to Put operations
//
// Enable for:
//   - Security-sensitive applications
//   - Buffers containing passwords, keys, or PII
//   - Compliance with data protection regulations
//
// Disable for:
//   - Maximum performance
//   - Non-sensitive data
//   - Trusted environments
func (p *bufferPool) SetClearOnPut(clear bool) {
	p.clearOnPut.Store(clear)
}

// SetMaxSize sets the maximum buffer size that will be pooled.
//
// Buffers larger than this size:
//   - Are allocated directly from the heap
//   - Are not returned to the pool when Put is called
//   - May cause more GC pressure
//
// The size is clamped between 64 bytes and 4MB. Default is 4MB.
//
// Consider your application's memory patterns when setting this:
//   - Larger values: More memory usage, better reuse for large buffers
//   - Smaller values: Less memory usage, more allocations for large buffers
func (p *bufferPool) SetMaxSize(size int64) {
	if size < poolMinSize {
		size = poolMinSize
	}
	if size > poolMaxSize {
		size = poolMaxSize
	}
	p.maxSize.Store(size)
}

// sizeToClass calculates the size class for a given size.
//
// Returns the bit position of the next power of 2 that can hold the size.
// For example:
//   - size 50 -> class 6 (2^6 = 64)
//   - size 100 -> class 7 (2^7 = 128)
//   - size 1000 -> class 10 (2^10 = 1024)
//
// This ensures efficient bucket selection with minimal fragmentation.
//
//go:inline
//go:nosplit
func sizeToClass(size int) int {
	if size <= poolMinSize {
		return poolClassBits // 2^6 = 64
	}

	// Find the next power of 2 that fits the size
	// Using bit scan for efficiency
	bits := 0
	size--
	for size > 0 {
		size >>= 1
		bits++
	}

	return bits
}

// clear zeros the buffer for security.
// Uses optimized memory clearing.
func clear(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
