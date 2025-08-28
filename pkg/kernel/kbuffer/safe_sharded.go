package kbuffer

import (
	"runtime"
	"unsafe"
)

// Ensure safeShardedBuffer implements Sharded interface at compile time.
var _ Sharded = (*safeShardedBuffer)(nil)

// safeShardedBuffer provides THREAD-SAFE concurrent write access through sharding.
// Each shard operates independently to minimize contention.
type safeShardedBuffer struct {
	shards     []*safeBufferShard // Array of buffer shards (slice header: 24 bytes on 64-bit)
	shardCount uint32             // Number of shards
	shardMask  uint32             // Mask for fast shard selection
	cap        uint32             // Total capacity across all shards
	pooled     bool               // From pool flag
	// Natural alignment and field ordering provide sufficient performance
	// without artificial padding. Shards are allocated separately.
}

// safeBufferShard represents a single SAFE shard with its own buffer.
// Allocated separately to avoid false sharing between shards.
type safeBufferShard struct {
	buffer Buffer // Underlying SAFE buffer implementation (interface: 16 bytes on 64-bit)
	// Each shard is independently allocated, naturally avoiding false sharing
}

// newSafeShardedBuffer creates a THREAD-SAFE sharded buffer for concurrent access.
// Shards are distributed across CPU cache lines for optimal performance.
//
//go:nosplit
func newSafeShardedBuffer(capacity, shardCount int, opts ...Option) Sharded {
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
	b := &safeShardedBuffer{
		shards:     make([]*safeBufferShard, shardCount),
		shardCount: uint32(shardCount),
		shardMask:  uint32(shardCount - 1), // For fast modulo via AND
		cap:        uint32(shardCapacity * shardCount),
		pooled:     false,
	}

	// Initialize shards with SAFE buffers for thread-safety
	// Each shard uses a thread-safe buffer to handle concurrent access
	for i := 0; i < shardCount; i++ {
		shard := &safeBufferShard{
			buffer: newSafeBuffer(shardCapacity), // Use safe buffer for each shard
		}
		b.shards[i] = shard
	}

	// Apply options
	for _, opt := range opts {
		opt(b)
	}

	return b
}

// selectShard chooses optimal shard for current goroutine.
// Uses goroutine ID hashing for affinity and load distribution.
//
//go:inline
//go:nosplit
func (b *safeShardedBuffer) selectShard() *safeBufferShard {
	// Get goroutine ID for affinity (reduces contention)
	gid := getGoroutineID()

	// Fast modulo using bit mask (works because shardCount is power of 2)
	shardIndex := gid & b.shardMask

	return b.shards[shardIndex]
}

// Write distributes writes across shards for concurrency.
// Each goroutine typically writes to its affinity shard.
func (b *safeShardedBuffer) Write(p []byte) (n int, err error) {
	// Select shard based on goroutine affinity
	shard := b.selectShard()

	// Try primary shard first
	n, err = shard.buffer.Write(p)
	if err == nil {
		return n, nil
	}

	// If primary shard full, try other shards (work stealing)
	if err == errBufferFull {
		for i := uint32(0); i < b.shardCount; i++ {
			altShard := b.shards[i]
			if altShard == shard {
				continue // Skip primary shard
			}

			n, err = altShard.buffer.Write(p)
			if err == nil {
				return n, nil
			}
		}
	}

	return 0, errBufferFull // All shards full
}

// WriteString performs sharded string write.
//
//go:nosplit
func (b *safeShardedBuffer) WriteString(s string) (n int, err error) {
	shard := b.selectShard()

	// Try primary shard
	n, err = shard.buffer.WriteString(s)
	if err == nil {
		return n, nil
	}

	// Work stealing on failure
	if err == errBufferFull {
		for i := uint32(0); i < b.shardCount; i++ {
			altShard := b.shards[i]
			if altShard == shard {
				continue
			}

			n, err = altShard.buffer.WriteString(s)
			if err == nil {
				return n, nil
			}
		}
	}

	return 0, errBufferFull
}

// WriteByte writes single byte to affinity shard.
//
//go:inline
func (b *safeShardedBuffer) WriteByte(c byte) error {
	shard := b.selectShard()
	return shard.buffer.WriteByte(c)
}

// writeToShardAt performs a single write operation to a specific shard at a local offset
func (b *safeShardedBuffer) writeToShardAt(shardIdx int, data []byte, localOffset int64) (int, error) {
	if shardIdx >= int(b.shardCount) {
		return 0, nil
	}
	shard := b.shards[shardIdx]
	return shard.buffer.WriteAt(data, localOffset)
}

// WriteAt writes at specific global offset, potentially spanning multiple shards.
// Handles writes that cross shard boundaries.
func (b *safeShardedBuffer) WriteAt(p []byte, off int64) (n int, err error) {
	// Validate offset
	if off < 0 || off >= int64(b.cap) {
		return 0, errInvalidOffset
	}

	shardCapacity := int64(b.cap) / int64(b.shardCount)
	bytesWritten := 0
	currentOffset := off

	// Write data across shards
	for bytesWritten < len(p) && currentOffset < int64(b.cap) {
		// Calculate shard parameters inline to avoid struct allocation
		shardIdx := int(currentOffset / shardCapacity)
		if shardIdx >= int(b.shardCount) {
			break
		}

		localOffset := currentOffset % shardCapacity
		spaceInShard := shardCapacity - localOffset
		remaining := len(p) - bytesWritten
		toWrite := int64(remaining)
		if toWrite > spaceInShard {
			toWrite = spaceInShard
		}

		// Write to shard
		written, writeErr := b.writeToShardAt(shardIdx, p[bytesWritten:bytesWritten+int(toWrite)], localOffset)
		bytesWritten += written
		currentOffset += int64(written)

		if writeErr != nil {
			return bytesWritten, writeErr
		}
		if written < int(toWrite) {
			break
		}
	}

	return bytesWritten, nil
}

// WriteToShard writes directly to specific shard.
// Allows manual shard selection for advanced use cases.
func (b *safeShardedBuffer) WriteToShard(shardIdx int, p []byte) (int, error) {
	if shardIdx < 0 || shardIdx >= int(b.shardCount) {
		return 0, errShardOutOfBounds
	}

	shard := b.shards[shardIdx]
	return shard.buffer.Write(p)
}

// TryWrite attempts non-blocking write to affinity shard.
//
//go:inline
//go:nosplit
func (b *safeShardedBuffer) TryWrite(p []byte) bool {
	shard := b.selectShard()
	return shard.buffer.TryWrite(p)
}

// Bytes collects data from all shards into single slice.
// Performs allocation and copy for consolidated view.
//
// Note: This method provides a best-effort snapshot. Since each shard
// has its own lock, the returned data may not represent an atomic
// point-in-time view when concurrent writes are happening.
// For atomic operations, use individual shard methods.
func (b *safeShardedBuffer) Bytes() []byte {
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
func (b *safeShardedBuffer) String() string {
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
func (b *safeShardedBuffer) BytesUnsafe() (ptr uintptr, len int) {
	if b.shardCount > 0 {
		return b.shards[0].buffer.BytesUnsafe()
	}
	return 0, 0
}

// Len returns total length across all shards.
//
//go:nosplit
func (b *safeShardedBuffer) Len() int {
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
func (b *safeShardedBuffer) Cap() int {
	return int(b.cap)
}

// Available returns total available space across all shards.
//
//go:nosplit
func (b *safeShardedBuffer) Available() int {
	total := 0
	for i := uint32(0); i < b.shardCount; i++ {
		total += b.shards[i].buffer.Available()
	}
	return total
}

// Reset resets all shards to empty state.
//
//go:nosplit
func (b *safeShardedBuffer) Reset() {
	for i := uint32(0); i < b.shardCount; i++ {
		b.shards[i].buffer.Reset()
	}
}

// Clear zeros and resets all shards.
func (b *safeShardedBuffer) Clear() {
	for i := uint32(0); i < b.shardCount; i++ {
		b.shards[i].buffer.Clear()
	}
}

// Truncate sets the total buffer length to exactly n bytes.
// The n bytes are distributed proportionally across shards.
// This is an absolute operation, not relative.
func (b *safeShardedBuffer) Truncate(n int) {
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
func (b *safeShardedBuffer) Grow(n int) error {
	// Check if any shard has enough space
	for i := uint32(0); i < b.shardCount; i++ {
		if b.shards[i].buffer.Available() >= n {
			return nil
		}
	}
	return errBufferFull
}

// Extend advances position in affinity shard.
func (b *safeShardedBuffer) Extend(n int) error {
	shard := b.selectShard()
	return shard.buffer.Extend(n)
}

// Clone creates deep copy of sharded buffer.
func (b *safeShardedBuffer) Clone() Buffer {
	clone := &safeShardedBuffer{
		shards:     make([]*safeBufferShard, b.shardCount),
		shardCount: b.shardCount,
		shardMask:  b.shardMask,
		cap:        b.cap,
		pooled:     false,
	}

	// Clone each shard
	for i := uint32(0); i < b.shardCount; i++ {
		clonedShard := &safeBufferShard{
			buffer: b.shards[i].buffer.Clone(),
		}
		clone.shards[i] = clonedShard
	}

	return clone
}

// RemainingSlice returns remaining space from first available shard.
//
//go:nosplit
func (b *safeShardedBuffer) RemainingSlice() []byte {
	for i := uint32(0); i < b.shardCount; i++ {
		if remaining := b.shards[i].buffer.RemainingSlice(); len(remaining) > 0 {
			return remaining
		}
	}
	return nil
}

// AppendBytes appends to affinity shard.
func (b *safeShardedBuffer) AppendBytes(data ...byte) error {
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
func (b *safeShardedBuffer) ShardCount() int {
	return int(b.shardCount)
}

// Balance redistributes data across shards for better distribution.
// Useful after skewed write patterns to rebalance load.
func (b *safeShardedBuffer) Balance() {
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

// getGoroutineID returns a pseudo goroutine ID for sharding.
// Uses runtime internals for best performance.
//
//go:inline
//go:nosplit
func getGoroutineID() uint32 {
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

// nextPowerOf2 rounds up to next power of 2.
//
//go:inline
//go:nosplit
func nextPowerOf2(n uint32) uint32 {
	if n == 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}
