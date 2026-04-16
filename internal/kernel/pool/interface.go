// Package pool provides high-performance byte buffers with a global sync.Pool.
//
// Two flavors are exposed:
//
//   - safeBuffer / safeShardedBuffer  — thread-safe via spinlock or sharding.
//   - unsafeBuffer                    — single-goroutine, amd64/arm64 only.
//
// The package also exposes a package-level sync.Pool via Get/Put/GetBuffer/PutBuffer.
package pool

// Sizing and shape constants. All values are referenced by the live code.
const (
	cacheLineSize     = 64
	minBufferSize     = 64
	defaultBufferSize = 4096
	maxBufferSize     = 16 << 20 // 16MB

	poolMinSize      = 1 << 6  // 64
	poolMaxSize      = 1 << 22 // 4MB
	poolClassCount   = 17      // log2(poolMaxSize/poolMinSize) + 1
	poolPrewarmCount = 4

	defaultShardCount = 16
	maxShardCount     = 256
	shardCachePadding = cacheLineSize
)

// Sentinel errors. Using a named string type keeps these allocation-free.
const (
	errBufferFull       = bufferError("buffer full")
	errInvalidSize      = bufferError("invalid size")
	errInvalidOffset    = bufferError("invalid offset")
	errShardOutOfBounds = bufferError("shard index out of bounds")
)

// State flags set on buffers. `stateFlagNormal` is the zero value.
const (
	stateFlagNormal  uint32 = 0
	stateFlagFull    uint32 = 1 << 0
	stateFlagPooled  uint32 = 1 << 1
	stateFlagCleared uint32 = 1 << 2
)

// bufferError is a compile-time constant error type.
type bufferError string

// Error returns the error message.
// Implements the error interface without allocations.
func (e bufferError) Error() string { return string(e) }

// ============================================================================
// INTERFACES
// ============================================================================

// Buffer defines the interface for a high-performance byte buffer.
// All implementations MUST guarantee thread-safety for concurrent reads
// and provide lock-free operations where possible.
type Buffer interface {
	// Write operations - optimized for zero-allocation
	Write(p []byte) (n int, err error)              // Write bytes to buffer
	WriteString(s string) (n int, err error)        // Write string without allocation
	WriteByte(c byte) error                         // Write single byte
	WriteAt(p []byte, off int64) (n int, err error) // Write at specific offset
	TryWrite(p []byte) bool                         // Non-blocking write attempt

	// Read operations - lock-free for concurrent access
	Bytes() []byte                       // Get written bytes (zero-copy)
	String() string                      // Get as string (zero-allocation)
	BytesUnsafe() (ptr uintptr, len int) // Direct memory access

	// Buffer management
	Len() int           // Current data length
	Cap() int           // Buffer capacity
	Available() int     // Available write space
	Reset()             // Reset position (keep capacity)
	Clear()             // Zero memory and reset
	Truncate(n int)     // Set total length to n bytes (absolute, not relative)
	Grow(n int) error   // Ensure n bytes available
	Extend(n int) error // Advance position by n

	// Advanced operations
	Clone() Buffer                  // Deep copy with new memory
	RemainingSlice() []byte         // Get unused portion
	AppendBytes(data ...byte) error // Variadic byte append
}

// Pool defines the interface for buffer pooling with size classes.
// Implementations MUST be thread-safe and lock-free where possible.
type Pool interface {
	// Buffer acquisition and release
	Get(size int) []byte       // Get raw byte slice
	Put(buf []byte)            // Return byte slice
	GetBuffer(size int) Buffer // Get Buffer instance
	PutBuffer(b Buffer)        // Return Buffer instance

	// Configuration
	SetClearOnPut(clear bool) // Security clearing option
	SetMaxSize(size int64)    // Maximum pooled size
}

// Sharded extends Buffer with shard-local operations.
type Sharded interface {
	Buffer
	WriteToShard(shard int, p []byte) (int, error)
	ShardCount() int
	Balance()
}

// Option defines functional options for buffer configuration.
// Allows extensible configuration without breaking API compatibility.
type Option func(Buffer) error

// ============================================================================
// BUFFER CREATION - EXPLICIT SAFETY CHOICE REQUIRED
// ============================================================================
//
// ⚠️ CRITICAL: You MUST explicitly choose between:
//
// BASIC BUFFERS:
// 1. NewUnsafeBuffer() - NOT thread-safe
//    - Use ONLY in single-threaded contexts
//    - ~2-3 ns/op writes (FASTEST)
//    - Will cause data corruption if used concurrently!
//
// 2. NewSafeBuffer() - Thread-safe with spinlock
//    - Use for concurrent access
//    - ~15-25 ns/op writes (FAST)
//    - Safe for multiple goroutines
//
// SHARDED BUFFERS (for high contention):
// 3. NewUnsafeShardedBuffer() - NOT thread-safe
//    - Sharded but NOT safe for concurrent use
//    - Use when you need sharding but manage sync externally
//
// 4. NewSafeShardedBuffer() - Thread-safe via sharding
//    - Best for high-contention scenarios (10+ goroutines)
//    - ~70-85 ns/op even with 100 goroutines
//
// ============================================================================

// NewUnsafeBuffer explicitly creates a non-thread-safe buffer.
// ⚠️ WARNING: NOT thread-safe! Use ONLY in single-threaded contexts.
// Performance: ~2-3 ns/op for writes (10x faster than safe version).
func NewUnsafeBuffer(capacity int, opts ...Option) Buffer {
	return newUnsafeBuffer(capacity, opts...)
}

// NewSafeBuffer creates a THREAD-SAFE buffer with spinlock optimization.
// ✅ SAFE: Can be used concurrently from multiple goroutines.
// Performance: ~15-25 ns/op
// for writes (faster than mutex).
func NewSafeBuffer(capacity int, opts ...Option) Buffer {
	return newSafeBuffer(capacity, opts...)
}

// NewSafeShardedBuffer creates a THREAD-SAFE sharded buffer.
// ✅ SAFE: Thread-safe through sharding (each shard uses safe buffers).
// Best for high-contention scenarios with many goroutines.
// Performance: ~70-85 ns/op even with 100 goroutines (7x faster than SafeBuffer).
func NewSafeShardedBuffer(capacity, shards int, opts ...Option) Sharded {
	return newSafeShardedBuffer(capacity, shards, opts...)
}

// GetGlobalPool returns the global buffer pool instance.
// The global pool is shared across the entire application and provides
// optimized buffer pooling with size classes from 64 bytes to 4MB.
// It is thread-safe and can be used concurrently from multiple goroutines.
func GetGlobalPool() Pool {
	// Implementation will be in global.go
	return globalPool
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// nextPowerOf2 rounds up to next power of 2.
// Uses efficient bit manipulation for O(1) calculation.
// Returns the smallest power of 2 that is >= n.
//
//go:inline
//go:nosplit
func nextPowerOf2(n uint32) uint32 {
	if n == 0 { // Handle zero case
		return 1 // Minimum power of 2
	}
	n--          // Decrement to handle exact powers
	n |= n >> 1  // Fill bits to the right
	n |= n >> 2  // Continue filling
	n |= n >> 4  // Continue filling
	n |= n >> 8  // Continue filling
	n |= n >> 16 // Final fill for 32-bit
	n++          // Increment to next power
	return n     // Return result
}

// min returns the smaller of two integers.
// Simple utility function for bounds checking.
//
//go:inline
//go:nosplit
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two integers.
// Simple utility function for bounds checking.
//
//go:inline
//go:nosplit
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
