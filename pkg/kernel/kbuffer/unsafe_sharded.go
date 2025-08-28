package kbuffer

import (
	"unsafe"
)

// Ensure unsafeShardedBuffer implements Sharded interface at compile time.
var _ Sharded = (*unsafeShardedBuffer)(nil)

// unsafeShardedBuffer provides NON-THREAD-SAFE sharded buffer for maximum performance.
// WARNING: Only use in single-threaded context or with external synchronization!
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
// Each shard is cache-line aligned to prevent false sharing.
type unsafeBufferShard struct {
	// Cache line aligned shard data
	buffer Buffer   // Underlying UNSAFE buffer implementation (8 bytes)
	_      [56]byte // Cache line padding
}

// newUnsafeShardedBuffer creates an UNSAFE sharded buffer.
// WARNING: NOT THREAD-SAFE! Use only in single-threaded context!
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
// NOT THREAD-SAFE - no synchronization!
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
// NOT THREAD-SAFE!
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
// NOT THREAD-SAFE!
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
func (b *unsafeShardedBuffer) WriteToShard(shardIdx int, p []byte) (int, error) {
	if shardIdx < 0 || shardIdx >= int(b.shardCount) {
		return 0, errShardOutOfBounds
	}

	shard := b.shards[shardIdx]
	return shard.buffer.Write(p)
}

// TryWrite attempts non-blocking write to first available shard.
//
//go:inline
//go:nosplit
func (b *unsafeShardedBuffer) TryWrite(p []byte) bool {
	for i := uint32(0); i < b.shardCount; i++ {
		if b.shards[i].buffer.TryWrite(p) {
			return true
		}
	}
	return false
}

// Bytes collects data from all shards into single slice.
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
func (b *unsafeShardedBuffer) String() string {
	data := b.Bytes()
	if len(data) == 0 {
		return ""
	}
	return unsafe.String(&data[0], len(data))
}

// BytesUnsafe returns pointer to first shard's data.
// WARNING: Only represents first shard, not all data.
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
//
//go:inline
//go:nosplit
func (b *unsafeShardedBuffer) Cap() int {
	return int(b.cap)
}

// Available returns total available space across all shards.
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
//
//go:nosplit
func (b *unsafeShardedBuffer) Reset() {
	for i := uint32(0); i < b.shardCount; i++ {
		b.shards[i].buffer.Reset()
	}
}

// Clear zeros and resets all shards.
func (b *unsafeShardedBuffer) Clear() {
	for i := uint32(0); i < b.shardCount; i++ {
		b.shards[i].buffer.Clear()
	}
}

// Truncate reduces all shards proportionally.
func (b *unsafeShardedBuffer) Truncate(n int) {
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
func (b *unsafeShardedBuffer) Extend(n int) error {
	for i := uint32(0); i < b.shardCount; i++ {
		if err := b.shards[i].buffer.Extend(n); err == nil {
			return nil
		}
	}
	return errBufferFull
}

// Clone creates deep copy of sharded buffer.
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
func (b *unsafeShardedBuffer) AppendBytes(data ...byte) error {
	if len(data) == 0 {
		return nil
	}
	_, err := b.Write(data)
	return err
}

// ShardCount returns the number of shards.
//
//go:inline
//go:nosplit
func (b *unsafeShardedBuffer) ShardCount() int {
	return int(b.shardCount)
}

// Balance redistributes data across shards for better distribution.
func (b *unsafeShardedBuffer) Balance() {
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
