// Package pool provides ultra-optimized, lock-free byte buffers for kernel operations.
// This file contains the global buffer pool implementation with size-class based pooling.
package pool

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Ensure bufferPool implements Pool interface at compile time.
var _ Pool = (*bufferPool)(nil)

// globalPool is the singleton pool instance for package-level operations.
// Initialized once at package load time with optimal configuration.
var globalPool = newGlobalPool()

// bufferPool implements high-performance buffer pooling with size classes.
// Uses sync.Pool with power-of-2 size classes for efficient memory management.
// The pool is optimized for concurrent access and memory efficiency.
type bufferPool struct {
	// Cache line 1 (64 bytes) - Pool arrays
	pools [poolClassCount]*sync.Pool // Power-of-2 pools from 2^6 to 2^22 (8 bytes * 17 = 136 bytes split across lines)

	// Cache line 2+ - Configuration
	clearOnPut atomic.Bool  // Security clearing flag (1 byte)
	maxSize    atomic.Int64 // Maximum pooled size (8 bytes)
}

// newGlobalPool creates and initializes the global buffer pool.
// Pre-warms pools with buffers for common sizes to avoid startup latency.
//
//go:nosplit
func newGlobalPool() *bufferPool {
	p := &bufferPool{}

	// Set default configuration
	p.maxSize.Store(poolMaxSize)
	p.clearOnPut.Store(false) // Performance over security by default

	// Initialize pools for each size class
	for i := 0; i < poolClassCount; i++ {
		size := 1 << (i + 6) // Size classes from 2^6 to 2^22
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

	// Pre-warm pools for common sizes
	p.prewarm()

	return p
}

// Get retrieves a buffer of at least the requested size from pool.
// Returns larger buffer if exact size not available for better reuse.
// The buffer is automatically resized to the requested length.
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
	poolIdx := class - 6 // Adjust for starting at 2^6

	// Validate pool index
	if poolIdx < 0 || poolIdx >= poolClassCount {
		return make([]byte, size)
	}

	// Get from pool
	bufPtr := p.pools[poolIdx].Get().(*[]byte)

	// Track statistics
	if bufPtr != nil {
		// Resize to requested length
		return (*bufPtr)[:size]
	}

	// Shouldn't happen due to New function, but handle gracefully
	return make([]byte, size)
}

// Put returns a buffer to pool for reuse.
// Clears buffer if security mode enabled to prevent information leakage.
// Only pools buffers with power-of-2 capacities that fit within size limits.
func (p *bufferPool) Put(buf []byte) {
	// Validate buffer
	if buf == nil || cap(buf) == 0 {
		return
	}

	capacity := cap(buf)

	// Don't pool oversized buffers
	if capacity > int(p.maxSize.Load()) {
		return
	}

	// Don't pool non-power-of-2 sizes (indicates non-pooled allocation)
	if !isPowerOfTwo(capacity) {
		return
	}

	// Clear if security mode enabled
	if p.clearOnPut.Load() {
		clear(buf[:capacity])
	}

	// Calculate pool index
	class := sizeToClass(capacity)
	poolIdx := class - 6

	// Validate pool index
	if poolIdx < 0 || poolIdx >= poolClassCount {
		return
	}

	// Reset to full capacity and return to pool
	buf = buf[:capacity]
	p.pools[poolIdx].Put(&buf)
}

// GetBuffer retrieves a Buffer instance from pool.
// Creates buffer wrapper around pooled byte slice for convenient use.
// The returned buffer is marked as pooled for proper lifecycle management.
func (p *bufferPool) GetBuffer(size int) Buffer {
	// Get byte slice from pool
	data := p.Get(size)
	if data == nil {
		// Fallback to new allocation
		return newSafeBuffer(size)
	}

	// Create safe buffer wrapper with pooled backing
	b := &safeBuffer{
		data:   unsafe.Pointer(&data[0]),
		cap:    uint32(cap(data)),
		origin: unsafe.Pointer(&data[0]),
		pooled: true, // Mark as pooled for lifecycle management
	}

	// Initialize atomic fields
	b.len.Store(0)
	b.flag.Store(stateFlagPooled)

	return b
}

// PutBuffer returns a Buffer instance to pool.
// Extracts underlying byte slice and pools it for reuse.
// Handles all buffer types including sharded buffers by pooling each shard.
func (p *bufferPool) PutBuffer(b Buffer) {
	if b == nil {
		return
	}

	// Type assertion to access internals
	switch buf := b.(type) {
	case *safeBuffer:
		// Reset buffer state
		buf.Reset()

		// Extract and pool underlying data
		if buf.pooled && buf.data != nil {
			// Reconstruct byte slice from pointer
			data := unsafe.Slice((*byte)(buf.data), buf.cap)
			p.Put(data)
		}

	case *unsafeBuffer:
		// Reset buffer state
		buf.Reset()

		// Extract and pool underlying data
		if buf.pooled && buf.data != nil {
			// Reconstruct byte slice from pointer
			data := unsafe.Slice((*byte)(buf.data), buf.cap)
			p.Put(data)
		}

	case *safeShardedBuffer:
		// Pool each shard's buffer
		for i := uint32(0); i < buf.shardCount; i++ {
			if buf.shards[i] != nil && buf.shards[i].buffer != nil {
				p.PutBuffer(buf.shards[i].buffer)
			}
		}
	}
}

// SetClearOnPut configures security clearing on buffer return.
// When enabled, zeros buffer content when returned to pool to prevent information leakage.
// This adds overhead but is necessary for sensitive data handling.
//
//go:inline
func (p *bufferPool) SetClearOnPut(clear bool) {
	p.clearOnPut.Store(clear)
}

// SetMaxSize sets maximum buffer size that will be pooled.
// Larger buffers will be allocated directly without pooling to avoid memory waste.
// The size is automatically clamped to valid pooling bounds.
//
//go:inline
func (p *bufferPool) SetMaxSize(size int64) {
	if size < poolMinSize {
		size = poolMinSize
	}
	if size > poolMaxSize {
		size = poolMaxSize
	}
	p.maxSize.Store(size)
}

// prewarm pre-allocates buffers for common sizes.
// Reduces allocation overhead during initial usage by warming up the pools.
// The number of buffers preallocated scales with CPU count for concurrency.
func (p *bufferPool) prewarm() {
	// Common sizes to prewarm (in bytes)
	sizes := []int{
		256,    // Small messages
		1024,   // 1KB - common buffer size
		4096,   // 4KB - page size
		16384,  // 16KB - medium buffers
		65536,  // 64KB - large buffers
		262144, // 256KB - very large buffers
	}

	// Prewarm based on CPU count for concurrency
	numCPU := runtime.NumCPU()
	prewarmCount := numCPU * poolPrewarmCount

	for _, size := range sizes {
		class := sizeToClass(size)
		poolIdx := class - 6

		if poolIdx < 0 || poolIdx >= poolClassCount {
			continue
		}

		// Pre-allocate buffers
		buffers := make([]*[]byte, prewarmCount)
		for i := 0; i < prewarmCount; i++ {
			buf := make([]byte, 1<<class)
			buffers[i] = &buf
		}

		// Return to pool
		for _, buf := range buffers {
			p.pools[poolIdx].Put(buf)
		}
	}
}

// sizeToClass calculates the size class for a given size.
// Returns the power-of-2 exponent that can hold the size.
// Uses efficient bit manipulation for O(1) calculation.
//
//go:inline
//go:nosplit
func sizeToClass(size int) int {
	if size <= poolMinSize {
		return 6 // 2^6 = 64
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

// isPowerOfTwo checks if n is a power of 2.
// Uses efficient bit manipulation trick: n > 0 && (n & (n-1)) == 0.
// Returns true if n is exactly a power of 2, false otherwise.
//
//go:inline
//go:nosplit
func isPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

// Package-level convenience functions using global pool

// Get retrieves a buffer from the global pool.
// Convenience function that uses the shared global pool instance.
// Returns a byte slice of at least the requested size.
//
//go:inline
func Get(size int) []byte {
	return globalPool.Get(size)
}

// Put returns a buffer to the global pool.
// Convenience function that uses the shared global pool instance.
// The buffer will be reused for future allocations if appropriate.
//
//go:inline
func Put(buf []byte) {
	globalPool.Put(buf)
}

// GetBuffer retrieves a Buffer from the global pool.
// Convenience function that returns a full Buffer interface implementation.
// The buffer is ready for immediate use with all operations available.
//
//go:inline
func GetBuffer(size int) Buffer {
	return globalPool.GetBuffer(size)
}

// PutBuffer returns a Buffer to the global pool.
// Convenience function that recycles a Buffer instance for reuse.
// Works with all buffer types including sharded buffers.
//
//go:inline
func PutBuffer(b Buffer) {
	globalPool.PutBuffer(b)
}

// SetGlobalClearOnPut sets security clearing for global pool.
// When enabled, all buffers are zeroed when returned to prevent data leakage.
// This affects all users of the global pool.
//
//go:inline
func SetGlobalClearOnPut(clear bool) {
	globalPool.SetClearOnPut(clear)
}

// SetGlobalMaxSize sets max size for global pool.
// Buffers larger than this size will not be pooled.
// This affects memory usage patterns for all global pool users.
//
//go:inline
func SetGlobalMaxSize(size int64) {
	globalPool.SetMaxSize(size)
}
