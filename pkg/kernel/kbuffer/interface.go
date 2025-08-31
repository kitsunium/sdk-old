// Package kbuffer provides ultra-optimized, lock-free byte buffers for kernel operations.
// This implementation provides maximum performance through extensive unsafe operations,
// atomic primitives, and CPU cache optimization.
//
// # Architecture Overview
//
// The package offers four distinct buffer implementations, each optimized for specific use cases:
//
//   - UnsafeBuffer: Single-threaded, zero-overhead buffer with maximum performance (~2-3 ns/op writes)
//   - SafeBuffer: Thread-safe buffer using spinlocks for concurrent access (~15-25 ns/op writes)
//   - UnsafeShardedBuffer: Single-threaded sharded buffer for data distribution algorithms
//   - SafeShardedBuffer: Thread-safe sharded buffer for high-contention scenarios (~70-85 ns/op with 100 goroutines)
//
// # Buffer Selection Guide
//
// Choose your buffer based on concurrency requirements:
//
//   - Single-threaded contexts: Use UnsafeBuffer for maximum performance
//   - Low-contention concurrent: Use SafeBuffer with spinlock optimization
//   - High-contention concurrent: Use SafeShardedBuffer to distribute load
//   - Single-threaded with sharding needs: Use UnsafeShardedBuffer
//
// # Safety Model
//
// The package enforces explicit safety choices:
//   - All constructors require explicit "Unsafe" or "Safe" prefix
//   - Unsafe buffers include goroutine safety checks in development builds
//   - Production builds can disable checks with 'unsafe_no_check' build tag
//
// # Performance Characteristics
//
// All implementations are optimized for:
//   - Zero allocations in hot paths
//   - CPU cache-line alignment (64 bytes)
//   - False sharing prevention through padding
//   - Lock-free reads where possible
//   - Direct memory operations using unsafe package
//
// # Example Usage
//
//	// Single-threaded high-performance buffer
//	buf := kbuffer.NewUnsafeBuffer(4096)
//	buf.WriteString("Hello, World!")
//	data := buf.Bytes()
//
//	// Thread-safe buffer for concurrent access
//	safeBuf := kbuffer.NewSafeBuffer(4096)
//	go safeBuf.Write([]byte("concurrent"))
//	go safeBuf.Write([]byte("writes"))
//
//	// High-contention scenario with sharding
//	shardedBuf := kbuffer.NewSafeShardedBuffer(65536, 16) // 64KB, 16 shards
//	for i := 0; i < 100; i++ {
//		go shardedBuf.Write([]byte(fmt.Sprintf("goroutine %d", i)))
//	}
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
//
// All Buffer implementations provide:
//   - Zero-copy operations where possible
//   - Efficient memory management
//   - Consistent API across safe and unsafe variants
//
// Thread-safety depends on the implementation:
//   - UnsafeBuffer: NOT thread-safe, will panic on concurrent access
//   - SafeBuffer: Fully thread-safe with spinlock protection
//
// The interface is designed for maximum performance with methods optimized
// for common use cases in kernel-level operations.
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
//
// Pooling reduces allocation overhead by reusing buffers. The pool uses
// size classes (powers of 2) for efficient memory management:
//   - Reduces GC pressure
//   - Improves allocation performance
//   - Prevents memory fragmentation
//
// Pools are thread-safe and can be shared across goroutines.
// Each application should create its own pool instances rather than
// relying on global state.
type Pool interface {
	// Get retrieves a byte slice of at least the requested size.
	// The returned slice may be larger than requested for better reuse.
	// The slice length is set to the requested size, but capacity may be larger.
	Get(size int) []byte

	// Put returns a byte slice to the pool for reuse.
	// The slice capacity is used to determine the appropriate size class.
	// Slices larger than MaxSize are not pooled.
	Put(buf []byte)

	// GetBuffer retrieves a Buffer instance with at least the requested capacity.
	// The buffer is reset before being returned.
	GetBuffer(size int) Buffer

	// PutBuffer returns a Buffer instance to the pool.
	// The buffer is reset and optionally cleared based on pool configuration.
	PutBuffer(b Buffer)

	// SetClearOnPut configures whether buffers are zeroed when returned to the pool.
	// Enable for security-sensitive data, disable for maximum performance.
	SetClearOnPut(clear bool)

	// SetMaxSize sets the maximum buffer size that will be pooled.
	// Larger buffers are allocated directly and not reused.
	SetMaxSize(size int64)
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
//
// Sharding distributes data across multiple internal buffers to:
//   - Reduce contention in concurrent scenarios (SafeShardedBuffer)
//   - Improve cache locality for large data sets
//   - Enable parallel processing algorithms
//
// Sharded buffers automatically distribute writes across shards and can
// consolidate data when reading. The sharding strategy uses round-robin
// or goroutine-affinity based selection depending on the implementation.
type Sharded interface {
	Buffer // Extends Buffer interface

	// WriteToShard writes directly to a specific shard.
	// Returns the number of bytes written and any error.
	// Useful for manual load distribution in advanced use cases.
	WriteToShard(shard int, p []byte) (int, error)

	// ShardCount returns the number of shards in this buffer.
	// The count is fixed at creation time and typically a power of 2.
	ShardCount() int

	// Balance redistributes existing data evenly across all shards.
	// Useful after skewed write patterns to optimize future access.
	// This operation may be expensive for large amounts of data.
	Balance()
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
//
// ⚠️ WARNING: NOT thread-safe! Use ONLY in single-threaded contexts.
//
// This buffer provides maximum performance with zero synchronization overhead:
//   - Write performance: ~2-3 ns/op (10x faster than safe version)
//   - Zero allocations after initial creation
//   - Direct memory operations without locks
//   - Goroutine safety checks in development builds
//
// Use cases:
//   - Request-scoped buffers in single-threaded handlers
//   - Protocol parsing and serialization
//   - Temporary buffers in non-concurrent algorithms
//
// The buffer will panic if accessed from multiple goroutines in development
// builds. Production builds with 'unsafe_no_check' tag disable these checks.
//
// Example:
//
//	buf := kbuffer.NewUnsafeBuffer(4096)
//	buf.WriteString("fast writes")
//	data := buf.Bytes() // Zero-copy access
func NewUnsafeBuffer(capacity int, opts ...Option) Buffer {
	return newUnsafeBuffer(capacity, opts...)
}

// NewSafeBuffer creates a THREAD-SAFE buffer with spinlock optimization.
//
// ✅ SAFE: Can be used concurrently from multiple goroutines.
//
// This buffer provides thread-safety with optimized performance:
//   - Write performance: ~15-25 ns/op (faster than mutex-based solutions)
//   - Spinlock for short critical sections
//   - Lock-free reads for some operations
//   - Suitable for low to moderate contention
//
// Use cases:
//   - Shared logging buffers
//   - Concurrent data collection
//   - Multi-producer scenarios with moderate contention
//
// For high-contention scenarios (>10 concurrent writers), consider using
// NewSafeShardedBuffer instead for better scalability.
//
// Example:
//
//	buf := kbuffer.NewSafeBuffer(4096)
//	var wg sync.WaitGroup
//	for i := 0; i < 10; i++ {
//		wg.Add(1)
//		go func(id int) {
//			defer wg.Done()
//			buf.WriteString(fmt.Sprintf("Writer %d\n", id))
//		}(i)
//	}
//	wg.Wait()
func NewSafeBuffer(capacity int, opts ...Option) Buffer {
	return newSafeBuffer(capacity, opts...)
}

// NewUnsafeShardedBuffer creates a NON-THREAD-SAFE sharded buffer.
//
// ⚠️ WARNING: NOT thread-safe! Use ONLY in single-threaded contexts.
//
// This buffer provides sharding benefits without synchronization overhead:
//   - Improved cache locality for large data sets
//   - Data distribution for algorithmic purposes
//   - Foundation for custom parallel processing
//   - Zero synchronization overhead
//
// Use cases:
//   - Single-threaded algorithms requiring data partitioning
//   - Cache-optimized sequential processing
//   - Building blocks for custom concurrent structures
//
// The sharding is useful even in single-threaded contexts for:
//   - Reducing cache misses on large buffers
//   - Preparing data for parallel processing
//   - Implementing scatter-gather patterns
//
// Example:
//
//	buf := kbuffer.NewUnsafeShardedBuffer(65536, 8) // 64KB across 8 shards
//	for i := 0; i < 8; i++ {
//		buf.WriteToShard(i, []byte(fmt.Sprintf("Shard %d data", i)))
//	}
//	buf.Balance() // Redistribute data evenly
func NewUnsafeShardedBuffer(capacity, shards int, opts ...Option) Sharded {
	return newUnsafeShardedBuffer(capacity, shards, opts...)
}

// NewSafeShardedBuffer creates a THREAD-SAFE sharded buffer.
//
// ✅ SAFE: Thread-safe through sharding (each shard uses safe buffers).
//
// This buffer excels in high-contention scenarios:
//   - Write performance: ~70-85 ns/op even with 100 goroutines
//   - 7x faster than SafeBuffer under high contention
//   - Reduces lock contention through data distribution
//   - Scales linearly with shard count up to CPU cores
//
// Use cases:
//   - High-throughput logging systems
//   - Multi-producer queues
//   - Concurrent metrics collection
//   - Any scenario with >10 concurrent writers
//
// Sharding strategy:
//   - Automatic round-robin distribution
//   - Work-stealing when primary shard is full
//   - Optional rebalancing with Balance() method
//
// Recommended shard counts:
//   - Light contention (2-10 goroutines): 4-8 shards
//   - Moderate contention (10-50 goroutines): 16 shards
//   - Heavy contention (50+ goroutines): 32-64 shards
//
// Example:
//
//	buf := kbuffer.NewSafeShardedBuffer(1048576, 16) // 1MB across 16 shards
//	var wg sync.WaitGroup
//	for i := 0; i < 100; i++ {
//		wg.Add(1)
//		go func(id int) {
//			defer wg.Done()
//			for j := 0; j < 1000; j++ {
//				buf.WriteString(fmt.Sprintf("G%d-M%d\n", id, j))
//			}
//		}(i)
//	}
//	wg.Wait()
//	data := buf.Bytes() // Consolidated view of all shards
func NewSafeShardedBuffer(capacity, shards int, opts ...Option) Sharded {
	return newSafeShardedBuffer(capacity, shards, opts...)
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
