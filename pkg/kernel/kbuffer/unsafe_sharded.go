// Package kbuffer provides ultra-optimized, lock-free byte buffers for kernel operations.
//
// This file contains the unsafe sharded buffer implementation for maximum performance
// in single-threaded contexts where sharding is needed for algorithmic reasons,
// such as cache optimization or data partitioning.
package kbuffer

import (
	"unsafe"
)

// Ensure unsafeShardedBuffer implements Sharded interface at compile time.
var _ Sharded = (*unsafeShardedBuffer)(nil)

// unsafeShardedBuffer provides NON-THREAD-SAFE sharded buffer for maximum performance.
//
// ⚠️ WARNING: Only use in single-threaded context or with external synchronization!
//
// This implementation provides sharding benefits without synchronization overhead:
//   - Improved cache locality through data distribution
//   - Foundation for scatter-gather algorithms
//   - Preparation for parallel processing pipelines
//   - Zero synchronization overhead
//
// Use cases:
//   - Single-threaded MapReduce-style algorithms
//   - Cache-optimized sequential processing
//   - Data partitioning for future parallelization
//   - Custom lock-free data structures (with external sync)
//
// Sharding improves performance even in single-threaded contexts by:
//   - Reducing cache misses on large data sets
//   - Enabling better memory prefetching patterns
//   - Allowing incremental processing of partitions
type unsafeShardedBuffer struct {
	// Cache line 1 (64 bytes) - Core configuration
	shards     []*unsafeBufferShard // Array of buffer shards (8 bytes)
	shardCount uint32               // Number of shards (4 bytes)
	shardMask  uint32               // Mask for fast shard selection (4 bytes)
	cap        uint32               // Total capacity across all shards (4 bytes)
	_          [44]byte             // Cache line padding

	// Cache line 2 (64 bytes) - Safety and metadata
	checker goroutineChecker // Goroutine safety checker (16 bytes)
	pooled  bool             // From pool flag (1 byte)
	_       [47]byte         // Cache line padding
}

// unsafeBufferShard represents a single UNSAFE shard.
// Each shard is cache-line aligned to prevent false sharing even in single-threaded use.
// The padding ensures optimal memory layout for sequential access patterns.
type unsafeBufferShard struct {
	// Cache line aligned shard data
	buffer Buffer   // Underlying UNSAFE buffer implementation (8 bytes)
	_      [56]byte // Cache line padding
}

// newUnsafeShardedBuffer creates an UNSAFE sharded buffer.
// WARNING: NOT THREAD-SAFE! Use only in single-threaded context!
// Creates a sharded buffer where each shard is an unsafe buffer for maximum performance.
// Sharding helps with data organization and can improve cache locality for certain access patterns.
//
//go:nosplit
func newUnsafeShardedBuffer(capacity, shardCount int, opts ...Option) Sharded {
	// Validate and normalize parameters
	if capacity <= 0 {
		capacity = defaultBufferSize
	}
	if capacity > maxBufferSize {
		capacity = maxBufferSize
	}

	// Validate shard count (must be power of 2)
	if shardCount <= 0 {
		shardCount = defaultShardCount
	}
	if shardCount > maxShardCount {
		shardCount = maxShardCount
	}

	// Round up to nearest power of 2 for efficient masking
	shardCount = int(nextPowerOf2(uint32(shardCount)))

	// Calculate per-shard capacity
	shardCapacity := capacity / shardCount
	if shardCapacity < minBufferSize {
		shardCapacity = minBufferSize
	}

	// Create sharded buffer
	b := &unsafeShardedBuffer{
		shards:     make([]*unsafeBufferShard, shardCount),
		shardCount: uint32(shardCount),
		shardMask:  uint32(shardCount - 1), // For fast modulo via AND
		cap:        uint32(shardCapacity * shardCount),
		pooled:     false,
	}

	// Initialize shards with UNSAFE buffers for maximum performance
	for i := 0; i < shardCount; i++ {
		shard := &unsafeBufferShard{
			buffer: newUnsafeBuffer(shardCapacity), // Use unsafe buffer for each shard
		}
		b.shards[i] = shard
	}

	// Apply options
	for _, opt := range opts {
		opt(b)
	}

	return b
}

// selectShard chooses optimal shard using simple round-robin.
// No goroutine affinity needed since this is single-threaded.
// Uses a simple static counter for deterministic shard selection.
//
//go:inline
//go:nosplit
func (b *unsafeShardedBuffer) selectShard() *unsafeBufferShard {
	// Simple round-robin for single-threaded use
	// Could also use a counter or hash if needed
	static := uint32(0)
	shardIndex := static & b.shardMask
	static++
	return b.shards[shardIndex]
}

// Write distributes writes across shards.
//
// ⚠️ NOT THREAD-SAFE - no synchronization! Will panic if used concurrently.
//
// Write strategy:
//   - Attempts shards in sequence (not round-robin)
//   - First shard with space accepts the write
//   - Returns errBufferFull only if all shards are full
//   - Maintains data locality within shards
//
// This sequential strategy is optimal for single-threaded use as it
// maximizes cache locality and minimizes shard switching overhead.
func (b *unsafeShardedBuffer) Write(p []byte) (n int, err error) {
	// Check for concurrent access
	b.checker.checkSafety()

	// Try shards in sequence (no concurrency concerns)
	for i := uint32(0); i < b.shardCount; i++ {
		shard := b.shards[i]
		n, err = shard.buffer.Write(p)
		if err == nil {
			return n, nil
		}
	}
	return 0, errBufferFull // All shards full
}

// WriteString performs sharded string write.
// NOT THREAD-SAFE! Will panic if used concurrently.
// Uses zero-copy string writing for maximum performance.
// Returns the number of bytes written and any error.
//
//go:nosplit
func (b *unsafeShardedBuffer) WriteString(s string) (n int, err error) {
	// Check for concurrent access
	b.checker.checkSafety()

	for i := uint32(0); i < b.shardCount; i++ {
		shard := b.shards[i]
		n, err = shard.buffer.WriteString(s)
		if err == nil {
			return n, nil
		}
	}
	return 0, errBufferFull
}

// WriteByte writes single byte to next available shard.
// NOT THREAD-SAFE! Will panic if used concurrently.
// Finds the first shard with available space and writes the byte.
// Returns error only if all shards are full.
//
//go:inline
func (b *unsafeShardedBuffer) WriteByte(c byte) error {
	// Check for concurrent access
	b.checker.checkSafety()

	for i := uint32(0); i < b.shardCount; i++ {
		if err := b.shards[i].buffer.WriteByte(c); err == nil {
			return nil
		}
	}
	return errBufferFull
}

// WriteAt writes at specific global offset across shards.
// NOT THREAD-SAFE! Will panic if used concurrently.
// Calculates which shard contains the offset and performs the write.
// Returns the number of bytes written and any error.
func (b *unsafeShardedBuffer) WriteAt(p []byte, off int64) (n int, err error) {
	// Check for concurrent access
	b.checker.checkSafety()

	// Calculate shard and local offset
	shardCapacity := int64(b.cap) / int64(b.shardCount)
	shardIndex := off / shardCapacity

	if shardIndex >= int64(b.shardCount) {
		return 0, errInvalidOffset
	}

	localOffset := off % shardCapacity
	shard := b.shards[shardIndex]

	return shard.buffer.WriteAt(p, localOffset)
}

// WriteToShard writes directly to specific shard.
// NOT THREAD-SAFE! Will panic if used concurrently.
// Allows manual shard selection for advanced use cases and algorithms.
// Returns the number of bytes written and any error.
func (b *unsafeShardedBuffer) WriteToShard(shardIdx int, p []byte) (int, error) {
	// Check for concurrent access
	b.checker.checkSafety()

	if shardIdx < 0 || shardIdx >= int(b.shardCount) {
		return 0, errShardOutOfBounds
	}

	shard := b.shards[shardIdx]
	return shard.buffer.Write(p)
}

// TryWrite attempts non-blocking write to first available shard.
// NOT THREAD-SAFE! Will panic if used concurrently.
// Since no locks are involved, this behaves identically to Write.
// Returns true if any shard accepted the data, false if all are full.
//
//go:inline
//go:nosplit
func (b *unsafeShardedBuffer) TryWrite(p []byte) bool {
	// Check for concurrent access
	b.checker.checkSafety()

	for i := uint32(0); i < b.shardCount; i++ {
		if b.shards[i].buffer.TryWrite(p) {
			return true
		}
	}
	return false
}

// Bytes collects data from all shards into single slice.
// Allocates a new slice and copies data from all shards in order.
// The returned slice is independent of the buffer's internal memory.
func (b *unsafeShardedBuffer) Bytes() []byte {
	// Calculate total size
	totalSize := 0
	for i := uint32(0); i < b.shardCount; i++ {
		totalSize += b.shards[i].buffer.Len()
	}

	if totalSize == 0 {
		return nil
	}

	// Collect from all shards
	result := make([]byte, 0, totalSize)
	for i := uint32(0); i < b.shardCount; i++ {
		shardData := b.shards[i].buffer.Bytes()
		result = append(result, shardData...)
	}

	return result
}

// String returns consolidated string from all shards.
// Creates a consolidated view of all shard data as a single string.
// Uses unsafe string conversion for performance after collecting the data.
func (b *unsafeShardedBuffer) String() string {
	data := b.Bytes()
	if len(data) == 0 {
		return ""
	}
	return unsafe.String(&data[0], len(data))
}

// BytesUnsafe returns pointer to first shard's data.
// WARNING: Only represents first shard, not all data!
// This is a limitation of the interface - use Bytes() for complete data.
// The pointer is valid until the buffer is modified or freed.
//
//go:inline
//go:nosplit
func (b *unsafeShardedBuffer) BytesUnsafe() (ptr uintptr, len int) {
	if b.shardCount > 0 {
		return b.shards[0].buffer.BytesUnsafe()
	}
	return 0, 0
}

// Len returns total length across all shards.
// Sums the current length of all shards to get total data size.
// This requires iterating through all shards.
//
//go:nosplit
func (b *unsafeShardedBuffer) Len() int {
	total := 0
	for i := uint32(0); i < b.shardCount; i++ {
		total += b.shards[i].buffer.Len()
	}
	return total
}

// Cap returns total capacity across all shards.
// Returns the sum of all shard capacities (precomputed at creation).
// This is a constant value set when the buffer was created.
//
//go:inline
//go:nosplit
func (b *unsafeShardedBuffer) Cap() int {
	return int(b.cap)
}

// Available returns total available space across all shards.
// Sums the available space in each shard to get total free space.
// This requires iterating through all shards to calculate.
//
//go:nosplit
func (b *unsafeShardedBuffer) Available() int {
	total := 0
	for i := uint32(0); i < b.shardCount; i++ {
		total += b.shards[i].buffer.Available()
	}
	return total
}

// Reset resets all shards to empty state.
// NOT THREAD-SAFE! Will panic if used concurrently.
// Clears all shards back to zero length while preserving capacity.
//
//go:nosplit
func (b *unsafeShardedBuffer) Reset() {
	// Check for concurrent access
	b.checker.checkSafety()

	for i := uint32(0); i < b.shardCount; i++ {
		b.shards[i].buffer.Reset()
	}
}

// Clear zeros and resets all shards.
// NOT THREAD-SAFE! Will panic if used concurrently.
// Securely wipes all data across all shards before resetting.
func (b *unsafeShardedBuffer) Clear() {
	// Check for concurrent access
	b.checker.checkSafety()

	for i := uint32(0); i < b.shardCount; i++ {
		b.shards[i].buffer.Clear()
	}
}

// Truncate sets the total buffer length to exactly n bytes.
// NOT THREAD-SAFE! Will panic if used concurrently.
// This is an absolute operation, not relative. Distributes the truncation
// across shards proportionally with remainder handling.
func (b *unsafeShardedBuffer) Truncate(n int) {
	// Check for concurrent access
	b.checker.checkSafety()

	if n <= 0 {
		b.Reset()
		return
	}

	// Distribute truncation across shards
	perShard := n / int(b.shardCount)
	remainder := n % int(b.shardCount)

	for i := uint32(0); i < b.shardCount; i++ {
		truncateSize := perShard
		if i < uint32(remainder) {
			truncateSize++
		}
		b.shards[i].buffer.Truncate(truncateSize)
	}
}

// Grow ensures space available in at least one shard.
// Checks if any shard has at least n bytes of free space.
// Returns nil if space is available, errBufferFull otherwise.
//
//go:inline
func (b *unsafeShardedBuffer) Grow(n int) error {
	// Check if any shard has enough space
	for i := uint32(0); i < b.shardCount; i++ {
		if b.shards[i].buffer.Available() >= n {
			return nil
		}
	}
	return errBufferFull
}

// Extend advances position in first available shard.
// NOT THREAD-SAFE! Will panic if used concurrently.
// Reserves n bytes in the first shard that has sufficient space.
func (b *unsafeShardedBuffer) Extend(n int) error {
	// Check for concurrent access
	b.checker.checkSafety()

	for i := uint32(0); i < b.shardCount; i++ {
		if err := b.shards[i].buffer.Extend(n); err == nil {
			return nil
		}
	}
	return errBufferFull
}

// Clone creates deep copy of sharded buffer.
// Returns a new independent buffer with the same shard structure and data.
// The clone is not pooled even if the original was from a pool.
func (b *unsafeShardedBuffer) Clone() Buffer {
	clone := &unsafeShardedBuffer{
		shards:     make([]*unsafeBufferShard, b.shardCount),
		shardCount: b.shardCount,
		shardMask:  b.shardMask,
		cap:        b.cap,
		pooled:     false,
	}

	// Clone each shard
	for i := uint32(0); i < b.shardCount; i++ {
		clonedShard := &unsafeBufferShard{
			buffer: b.shards[i].buffer.Clone(),
		}
		clone.shards[i] = clonedShard
	}

	return clone
}

// RemainingSlice returns remaining space from first available shard.
// Searches shards in order for the first one with available space.
// Returns nil if no shard has available space.
//
//go:nosplit
func (b *unsafeShardedBuffer) RemainingSlice() []byte {
	for i := uint32(0); i < b.shardCount; i++ {
		if remaining := b.shards[i].buffer.RemainingSlice(); len(remaining) > 0 {
			return remaining
		}
	}
	return nil
}

// AppendBytes appends to first available shard.
// NOT THREAD-SAFE! Will panic if used concurrently.
// Variadic convenience method equivalent to Write(data).
func (b *unsafeShardedBuffer) AppendBytes(data ...byte) error {
	if len(data) == 0 {
		return nil
	}
	_, err := b.Write(data)
	return err
}

// ShardCount returns the number of shards.
// Returns the shard count that was set when the buffer was created.
// This is useful for algorithms that need to know the shard structure.
//
//go:inline
//go:nosplit
func (b *unsafeShardedBuffer) ShardCount() int {
	return int(b.shardCount)
}

// Balance redistributes data across shards for better distribution.
//
// ⚠️ NOT THREAD-SAFE! Will panic if used concurrently.
//
// Rebalancing process:
//  1. Collects all data from all shards
//  2. Resets all shards to empty
//  3. Redistributes data evenly across shards
//  4. Optimizes for sequential access patterns
//
// Use Balance() when:
//   - Write patterns have created uneven distribution
//   - Preparing for parallel processing of shards
//   - Optimizing for sequential read patterns
//   - Before passing shards to different processing stages
//
// Performance note: This operation is O(n) where n is total data size.
// Avoid calling frequently in hot paths.
func (b *unsafeShardedBuffer) Balance() {
	// Check for concurrent access
	b.checker.checkSafety()

	// Collect all data
	allData := b.Bytes()
	if len(allData) == 0 {
		return
	}

	// Reset all shards
	b.Reset()

	// Redistribute evenly
	chunkSize := len(allData) / int(b.shardCount)
	remainder := len(allData) % int(b.shardCount)

	offset := 0
	for i := uint32(0); i < b.shardCount && offset < len(allData); i++ {
		size := chunkSize
		if i < uint32(remainder) {
			size++
		}

		if size > 0 && offset+size <= len(allData) {
			b.shards[i].buffer.Write(allData[offset : offset+size])
			offset += size
		}
	}
}
