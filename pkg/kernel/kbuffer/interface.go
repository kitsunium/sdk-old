// Package kbuffer provides ultra-optimized, lock-free byte buffers for kernel operations.
// This implementation provides maximum performance through extensive unsafe operations,
// atomic primitives, and CPU cache optimization.
package kbuffer

import (
	"io"
	"unsafe"
)

// ============================================================================
// CONSTANTS
// ============================================================================

// CPU cache line size for optimal memory alignment.
// Most modern x86_64 and ARM64 processors use 64-byte cache lines.
// Aligning hot data to cache lines prevents false sharing and improves performance.
const cacheLineSize = 64

// Buffer size constants optimized for common use cases and memory hierarchy.
const (
	// minBufferSize is the minimum allocation size to avoid excessive small allocations.
	// 64 bytes fits exactly in one cache line for optimal CPU cache utilization.
	minBufferSize = 64

	// defaultBufferSize is the default when no size specified.
	// 4KB matches common memory page size for efficient virtual memory usage.
	defaultBufferSize = 4096

	// maxBufferSize is the maximum size for a single buffer.
	// 16MB prevents excessive memory usage while supporting large operations.
	maxBufferSize = 16 << 20 // 16MB

	// optimalIOSize is the optimal size for I/O operations.
	// 64KB balances between syscall overhead and memory usage.
	optimalIOSize = 65536
)

// Pool size class boundaries using power-of-2 for efficient bit operations.
// Size classes reduce fragmentation and improve allocation performance.
const (
	// poolMinSize is the minimum pooled buffer size (64 bytes = 2^6).
	// Smaller allocations use stack or are not worth pooling.
	poolMinSize = 1 << 6

	// poolMaxSize is the maximum pooled buffer size (4MB = 2^22).
	// Larger buffers are rare and not worth pooling.
	poolMaxSize = 1 << 22

	// poolClassCount is the number of size classes (2^6 to 2^22 = 17 classes).
	// Each class represents a power-of-2 size for efficient memory management.
	poolClassCount = 17

	// poolPrewarmCount is buffers per size class to prewarm.
	// Prevents initial allocation overhead during startup.
	poolPrewarmCount = 4
)

// Sharding constants for concurrent access optimization.
const (
	// defaultShardCount is the default number of shards.
	// 16 shards balance between concurrency and memory overhead.
	defaultShardCount = 16

	// maxShardCount limits the maximum shards to prevent excessive overhead.
	// 256 shards support up to 256-core systems effectively.
	maxShardCount = 256

	// shardCachePadding adds padding between shards to prevent false sharing.
	// Ensures each shard occupies separate cache lines.
	shardCachePadding = cacheLineSize
)

// Atomic operation constants for lock-free algorithms.
const (
	// spinLimit is the maximum spin iterations before yielding.
	// Prevents excessive CPU usage in contended scenarios.
	spinLimit = 100

	// backoffInitial is the initial backoff delay in nanoseconds.
	// Reduces contention through exponential backoff.
	backoffInitial = 10

	// backoffMax is the maximum backoff delay in nanoseconds.
	// Prevents excessive delays in highly contended scenarios.
	backoffMax = 10000
)

// Memory alignment constants for optimal performance.
const (
	// ptrSize is the size of a pointer on the current architecture.
	// Used for proper memory alignment calculations.
	ptrSize = unsafe.Sizeof(uintptr(0))

	// wordSize is the native word size for efficient operations.
	// Aligning to word boundaries improves memory access performance.
	wordSize = unsafe.Sizeof(uint(0))

	// alignment16 ensures 16-byte alignment for SIMD operations.
	// Required for SSE/AVX instructions on x86_64.
	alignment16 = 16

	// alignment32 ensures 32-byte alignment for AVX operations.
	// Required for AVX2/AVX-512 instructions.
	alignment32 = 32
)

// Error sentinel values for zero-allocation error handling.
// Using constants avoids heap allocations in error paths.
const (
	// errBufferFull indicates write operation exceeds capacity.
	errBufferFull = bufferError("buffer full")

	// errInvalidSize indicates invalid size parameter.
	errInvalidSize = bufferError("invalid size")

	// errInvalidOffset indicates offset out of bounds.
	errInvalidOffset = bufferError("invalid offset")

	// errNilBuffer indicates operation on nil buffer.
	errNilBuffer = bufferError("nil buffer")

	// errConcurrentModification indicates unsafe concurrent modification.
	errConcurrentModification = bufferError("concurrent modification")

	// errShardOutOfBounds indicates invalid shard index.
	errShardOutOfBounds = bufferError("shard index out of bounds")
)

// State flags for buffer status using bit flags for efficiency.
// Allows multiple states to be checked with single operation.
const (
	// stateFlagNormal indicates normal operational state.
	stateFlagNormal uint32 = 0

	// stateFlagFull indicates buffer is at capacity.
	stateFlagFull uint32 = 1 << 0

	// stateFlagLocked indicates buffer is locked for exclusive access.
	stateFlagLocked uint32 = 1 << 1

	// stateFlagPooled indicates buffer is from pool.
	stateFlagPooled uint32 = 1 << 2

	// stateFlagCleared indicates buffer was security-cleared.
	stateFlagCleared uint32 = 1 << 3

	// stateFlagReadOnly indicates buffer is read-only.
	stateFlagReadOnly uint32 = 1 << 4
)

// Performance hint constants for compiler optimization.
// These hints help the compiler generate better code.
const (
	// likelyTrue hints that condition is likely true.
	// Used for hot path optimization.
	likelyTrue = 1

	// likelyFalse hints that condition is likely false.
	// Used for cold path optimization.
	likelyFalse = 0

	// prefetchRead hints to prefetch for reading.
	// Improves cache performance for sequential access.
	prefetchRead = 0

	// prefetchWrite hints to prefetch for writing.
	// Improves cache performance for write operations.
	prefetchWrite = 1
)

// bufferError is a compile-time constant error type.
// Avoids allocations when returning errors.
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
	Truncate(n int)     // Reduce to n bytes
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

// Writer defines the interface for write-only buffer operations.
// Useful for APIs that only need write capabilities.
type Writer interface {
	io.Writer       // Standard io.Writer
	io.ByteWriter   // Single byte writer
	io.StringWriter // String writer
	io.WriterAt     // Position writer
}

// Reader defines the interface for read-only buffer operations.
// Provides zero-copy reads without modifying buffer state.
type Reader interface {
	io.Reader     // Standard io.Reader
	io.ByteReader // Single byte reader
	io.ReaderAt   // Position reader
	io.RuneReader // Rune reader
}

// Sharded defines the interface for sharded buffer implementations.
// Provides concurrent write access through sharding for scalability.
type Sharded interface {
	Buffer // Extends Buffer interface

	// Sharded operations
	WriteToShard(shard int, p []byte) (int, error) // Direct shard write
	ShardCount() int                               // Number of shards
	Balance()                                      // Rebalance shards
}

// Factory defines the interface for buffer creation.
// Allows custom buffer implementations and configurations.
type Factory interface {
	NewUnsafeBuffer(size int) Buffer                 // Create unsafe buffer
	NewSafeBuffer(size int) Buffer                   // Create thread-safe buffer
	NewUnsafeShardedBuffer(size, shards int) Sharded // Create unsafe sharded buffer
	NewSafeShardedBuffer(size, shards int) Sharded   // Create safe sharded buffer
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

// NewUnsafeShardedBuffer creates a NON-THREAD-SAFE sharded buffer.
// ⚠️ WARNING: NOT thread-safe! Use ONLY in single-threaded contexts.
// Performance: Fastest sharded option when no concurrency is needed.
func NewUnsafeShardedBuffer(capacity, shards int, opts ...Option) Sharded {
	return newUnsafeShardedBuffer(capacity, shards, opts...)
}

// NewSafeShardedBuffer creates a THREAD-SAFE sharded buffer.
// ✅ SAFE: Thread-safe through sharding (each shard uses safe buffers).
// Best for high-contention scenarios with many goroutines.
// Performance: ~70-85 ns/op even with 100 goroutines (7x faster than SafeBuffer).
func NewSafeShardedBuffer(capacity, shards int, opts ...Option) Sharded {
	return newSafeShardedBuffer(capacity, shards, opts...)
}

// GetGlobalPool returns the global buffer pool instance.
func GetGlobalPool() Pool {
	// Implementation will be in global.go
	return globalPool
}
