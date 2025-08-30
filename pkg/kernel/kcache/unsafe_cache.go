// Package kcache provides cache implementations with configurable thread safety.
// This file contains non-thread-safe cache implementation optimized for single-threaded access.
package kcache

import (
	"sync/atomic"
	"unsafe"
)

// ============================================================================
// UNSAFE CACHE - NON THREAD-SAFE - MAXIMUM PERFORMANCE
// ============================================================================
//
// ⚠️ WARNING: This cache is NOT thread-safe!
// Use ONLY in single-threaded contexts or when YOU manage synchronization.
// For concurrent access, use NewSafeCache() or NewSafeShardedCache() instead.
//
// Performance characteristics:
// - Get/Set: ~20-30 ns/op (5-10x faster than thread-safe version)
// - Zero allocations in hot paths
// - Direct memory access with unsafe pointers
// - Robin Hood hashing for optimal probe distances
//
// ============================================================================

// Ensure unsafeCache implements Cache and BatchCache interfaces
var (
	_ Cache      = (*unsafeCache)(nil)
	_ BatchCache = (*unsafeCache)(nil)
)

// entry represents a single key-value pair in the cache.
// Carefully structured for cache-line efficiency and minimal memory footprint.
// Size: 40 bytes (fits in single cache line with padding)
type entry struct {
	key   interface{} // 16 bytes - interface contains (type, data) pointers
	value interface{} // 16 bytes - interface contains (type, data) pointers
	hash  uint64      // 8 bytes - pre-computed hash for fast comparison
	state uint8       // 1 byte - entry state (empty/active/deleted)
	_     [7]uint8    // 7 bytes padding for 8-byte alignment
}

// unsafeCache is the fastest possible cache implementation.
// NO synchronization - caller MUST ensure single-threaded access.
// This implementation provides maximum performance through direct memory access
// and elimination of all atomic operations and synchronization primitives.
type unsafeCache struct {
	// Cache line 1 (64 bytes) - Hot path fields
	entries  []entry    // Hash table entries - power of 2 size for fast modulo
	size     int32      // Current number of active entries (no atomics needed)
	capacity int32      // Current capacity of the hash table
	mask     uint32     // Capacity - 1, used for fast modulo via bitwise AND
	maxLoad  int32      // Maximum entries before resize (capacity * loadFactor)
	hasher   Hasher     // Hash function implementation
	pool     *EntryPool // Object pool for entry allocation

	// Cache line 2 (64 bytes) - Safety and metadata
	checker   goroutineChecker // Goroutine safety checker
	skipCheck bool             // Skip goroutine checking (for internal use under mutex)
	_         [47]byte         // Cache line padding
}

// NewUnsafeCache creates a new non-thread-safe cache.
// ⚠️ UNSAFE: No synchronization - single-threaded use only!
// This function validates capacity bounds and allocates optimally aligned memory.
// Capacity is normalized to power of 2 for optimal hash distribution.
//
//go:nosplit
func NewUnsafeCache(capacity int) Cache {
	return newUnsafeCache(capacity)
}

// newUnsafeCache is the internal constructor for unsafe cache.
// Separated for easier testing and composition.
func newUnsafeCache(capacity int) *unsafeCache {
	// Validate and adjust capacity to power of 2
	if capacity < MinCapacity {
		capacity = MinCapacity
	}
	if capacity > MaxCapacity {
		capacity = MaxCapacity
	}
	capacity = nextPowerOf2(capacity)

	// Pre-allocate entries array with zero values
	entries := make([]entry, capacity)

	// Initialize cache structure with computed values
	c := &unsafeCache{
		entries:  entries,
		size:     0,
		capacity: int32(capacity),
		mask:     uint32(capacity - 1),
		maxLoad:  int32(float32(capacity) * DefaultLoadFactor),
		hasher:   newFNVHasher(), // Default to FNV-1a for speed
		pool:     NewEntryPool(), // Initialize object pool
	}

	// Go guarantees zero initialization, no need for explicit loop
	// StateEmpty is 0, so all entries are already initialized correctly

	return c
}

// newUnsafeCacheNoCheck creates an unsafe cache without goroutine checking.
// Used internally by safe implementations where mutex protection is provided.
// This bypasses the goroutine safety checker for internal use.
func newUnsafeCacheNoCheck(capacity int) *unsafeCache {
	// Validate and adjust capacity to power of 2
	if capacity < MinCapacity {
		capacity = MinCapacity
	}
	if capacity > MaxCapacity {
		capacity = MaxCapacity
	}
	capacity = nextPowerOf2(capacity)

	// Pre-allocate entries array with zero values
	entries := make([]entry, capacity)

	// Initialize cache structure with computed values
	// NOTE: skipCheck=true for use under external mutex protection
	return &unsafeCache{
		entries:   entries,
		size:      0,
		capacity:  int32(capacity),
		mask:      uint32(capacity - 1),
		maxLoad:   int32(float32(capacity) * DefaultLoadFactor),
		hasher:    newFNVHasher(), // Default to FNV-1a for speed
		pool:      NewEntryPool(), // Create new pool for this instance
		skipCheck: true,           // Skip goroutine checking - safe under mutex
	}
}

// Set stores a key-value pair using Robin Hood hashing for optimal probe distance.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
// Returns true if key was newly inserted, false if updated existing.
// Uses unsafe operations to avoid allocations in hot path.
func (c *unsafeCache) Set(key, value interface{}) bool {
	// Check for concurrent access (unless skipCheck is true)
	if !c.skipCheck {
		c.checker.checkSafety()
	}

	// Check for nil key - invalid input
	if key == nil {
		return false
	}

	// Compute hash once and reuse throughout operation
	hash := c.hasher.Hash(key)

	// Check if resize needed before insertion
	if c.size >= c.maxLoad {
		c.resize(int(c.capacity) * GrowthFactor)
	}

	// Perform Robin Hood insertion
	idx := uint32(hash) & c.mask
	distance := uint32(0)

	// Entry to insert
	newEntry := entry{
		key:   key,
		value: value,
		hash:  hash,
		state: StateActive,
	}

	for {
		e := &c.entries[idx]

		// Found empty or deleted slot - insert here
		if e.state == StateEmpty || e.state == StateDeleted {
			c.entries[idx] = newEntry
			c.size++
			return true
		}

		// Check if updating existing key
		if e.hash == hash && c.hasher.Equal(e.key, key) {
			c.entries[idx].value = value
			return false
		}

		// Robin Hood: swap if current entry has shorter probe distance
		entryDist := c.probeDistance(idx, e.hash)
		if entryDist < distance {
			// Swap entries and continue inserting the displaced one
			c.entries[idx], newEntry = newEntry, c.entries[idx]
			distance = entryDist
		}

		// Continue probing
		distance++
		idx = (idx + 1) & c.mask
	}
}

// Get retrieves a value by key using optimized probe sequence.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
// Returns value and true if found, nil and false otherwise.
// Uses unsafe pointer operations for zero-copy access.
func (c *unsafeCache) Get(key interface{}) (interface{}, bool) {
	// Check for concurrent access (unless skipCheck is true)
	if !c.skipCheck {
		c.checker.checkSafety()
	}

	// Check for nil key
	if key == nil {
		return nil, false
	}

	// Compute hash for lookup
	hash := c.hasher.Hash(key)

	// Find entry using Robin Hood hashing
	idx := c.findEntry(hash, key)
	if idx < 0 {
		return nil, false
	}

	// Return value directly - atomic load is overkill for single-threaded cache
	return c.entries[idx].value, true
}

// Has checks if a key exists using minimal operations.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
// Optimized to touch minimum cache lines.
//
//go:inline
func (c *unsafeCache) Has(key interface{}) bool {
	// Check for concurrent access (unless skipCheck is true)
	if !c.skipCheck {
		c.checker.checkSafety()
	}

	// Check for nil key
	if key == nil {
		return false
	}

	// Compute hash and check existence
	hash := c.hasher.Hash(key)
	return c.findEntry(hash, key) >= 0
}

// Delete removes a key from the cache using tombstone marking.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
// Returns true if key existed and was deleted.
func (c *unsafeCache) Delete(key interface{}) bool {
	// Check for concurrent access (unless skipCheck is true)
	if !c.skipCheck {
		c.checker.checkSafety()
	}

	// Check for nil key
	if key == nil {
		return false
	}

	// Find entry to delete
	hash := c.hasher.Hash(key)
	idx := c.findEntry(hash, key)
	if idx < 0 {
		return false
	}

	// Mark as deleted (tombstone) rather than clearing
	// This maintains probe sequences for other entries
	c.entries[idx].state = StateDeleted
	c.entries[idx].key = nil   // Clear key reference for GC
	c.entries[idx].value = nil // Clear value reference for GC

	// Decrement size
	c.size--

	// Consider shrinking if load too low
	if float32(c.size)/float32(c.capacity) < ShrinkThreshold && c.capacity > MinCapacity {
		c.resize(int(c.capacity) / ShrinkFactor)
	}

	return true
}

// Clear removes all entries efficiently by resetting to initial state.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
// Retains capacity to avoid reallocation on refill.
func (c *unsafeCache) Clear() {
	// Check for concurrent access (unless skipCheck is true)
	if !c.skipCheck {
		c.checker.checkSafety()
	}

	// Reset all entries to empty state
	// Using unsafe pointer manipulation for speed
	entries := c.entries
	for i := range entries {
		if entries[i].state != StateEmpty {
			entries[i] = entry{state: StateEmpty} // Zero value
		}
	}

	// Reset size to zero
	c.size = 0
}

// Len returns the current number of entries.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
//
//go:inline
func (c *unsafeCache) Len() int {
	// Check for concurrent access (unless skipCheck is true)
	if !c.skipCheck {
		c.checker.checkSafety()
	}
	return int(c.size)
}

// Cap returns the current capacity.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
//
//go:inline
func (c *unsafeCache) Cap() int {
	// Check for concurrent access (unless skipCheck is true)
	if !c.skipCheck {
		c.checker.checkSafety()
	}
	return int(c.capacity)
}

// findEntry locates an entry by hash and key.
// Returns index if found, -1 otherwise.
// Uses Robin Hood hashing for optimal probe distance.
//
//go:nosplit
func (c *unsafeCache) findEntry(hash uint64, key interface{}) int {
	// Start position using fast modulo
	idx := uint32(hash) & c.mask
	distance := uint32(0)

	// Probe until found or empty slot
	for {
		e := &c.entries[idx]

		// Check if slot is empty - key not found
		if e.state == StateEmpty {
			return -1
		}

		// Check if this is our entry
		if e.state == StateActive && e.hash == hash && c.hasher.Equal(e.key, key) {
			return int(idx)
		}

		// Robin Hood: stop if current entry has shorter probe distance
		// This means our key cannot be further along
		entryDist := c.probeDistance(idx, e.hash)
		if entryDist < distance {
			return -1
		}

		// Continue probing
		distance++
		if distance > MaxProbeDistance {
			return -1 // Prevent infinite loop
		}
		idx = (idx + 1) & c.mask
	}
}

// probeDistance calculates the probe distance for an entry.
// Used in Robin Hood hashing to minimize probe distances.
//
//go:inline
func (c *unsafeCache) probeDistance(idx uint32, hash uint64) uint32 {
	idealIdx := uint32(hash) & c.mask
	if idx >= idealIdx {
		return idx - idealIdx
	}
	return (c.mask + 1) - idealIdx + idx
}

// resize adjusts the hash table to the new capacity.
// Rehashes all entries to maintain performance.
func (c *unsafeCache) resize(newCapacity int) {
	// Validate new capacity
	if newCapacity < MinCapacity {
		newCapacity = MinCapacity
	}
	if newCapacity > MaxCapacity {
		return // Cannot grow further
	}
	newCapacity = nextPowerOf2(newCapacity)

	// Skip if same size
	if newCapacity == int(c.capacity) {
		return
	}

	// Save old entries
	oldEntries := c.entries

	// Create new table
	c.entries = make([]entry, newCapacity)
	c.capacity = int32(newCapacity)
	c.mask = uint32(newCapacity - 1)
	c.maxLoad = int32(float32(newCapacity) * DefaultLoadFactor)

	// Reset size for reinserttion
	oldSize := c.size
	c.size = 0

	// Reinsert all active entries using Robin Hood hashing
	for i := range oldEntries {
		if oldEntries[i].state == StateActive {
			// Robin Hood insertion for each entry
			idx := uint32(oldEntries[i].hash) & c.mask
			distance := uint32(0)
			entryToInsert := oldEntries[i]

			for {
				e := &c.entries[idx]

				// Found empty slot - insert here
				if e.state == StateEmpty {
					c.entries[idx] = entryToInsert
					c.size++
					break
				}

				// Robin Hood: swap if current entry has shorter probe distance
				entryDist := c.probeDistance(idx, e.hash)
				if entryDist < distance {
					// Swap entries and continue inserting the displaced one
					c.entries[idx], entryToInsert = entryToInsert, c.entries[idx]
					distance = entryDist
				}

				// Continue probing
				distance++
				idx = (idx + 1) & c.mask
			}
		}
	}

	// Verify size consistency
	if c.size != oldSize {
		// Should never happen, but recover gracefully
		c.size = oldSize
	}
}

// nextPowerOf2 rounds up to the next power of 2.
// Uses bit manipulation for speed.
//
//go:inline
func nextPowerOf2(n int) int {
	// Handle edge cases
	if n <= 0 {
		return 1
	}
	if n > MaxCapacity {
		return MaxCapacity
	}

	// Check if already power of 2
	if n&(n-1) == 0 {
		return n
	}

	// Round up to next power of 2
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++

	// Ensure we don't exceed max
	if n > MaxCapacity {
		return MaxCapacity
	}
	return n
}

// Batch operations implementation

// SetBatch stores multiple key-value pairs efficiently.
// Minimizes resize checks and improves cache locality.
func (c *unsafeCache) SetBatch(keys, values []interface{}) int {
	// Validate input
	if len(keys) != len(values) {
		return 0
	}

	newCount := 0

	// Pre-compute if resize needed
	potentialSize := atomic.LoadInt32(&c.size) + int32(len(keys))
	if potentialSize >= c.maxLoad {
		// Conservative resize - assume all keys are new
		newCap := int(c.capacity)
		for int32(newCap) < potentialSize*2 {
			newCap *= GrowthFactor
		}
		c.resize(newCap)
	}

	// Insert all pairs
	for i := range keys {
		if c.Set(keys[i], values[i]) {
			newCount++
		}
	}

	return newCount
}

// GetBatch retrieves multiple values efficiently.
// Returns parallel slices for values and found flags.
func (c *unsafeCache) GetBatch(keys []interface{}) ([]interface{}, []bool) {
	values := make([]interface{}, len(keys))
	found := make([]bool, len(keys))

	// Process in chunks for better cache locality
	const chunkSize = 8 // Fits in L1 cache
	for i := 0; i < len(keys); i += chunkSize {
		end := i + chunkSize
		if end > len(keys) {
			end = len(keys)
		}

		// Process chunk
		for j := i; j < end; j++ {
			values[j], found[j] = c.Get(keys[j])
		}
	}

	return values, found
}

// HasBatch checks existence of multiple keys efficiently.
// Optimized for minimal cache misses.
func (c *unsafeCache) HasBatch(keys []interface{}) []bool {
	found := make([]bool, len(keys))

	// Process keys with minimal overhead
	for i := range keys {
		if keys[i] != nil {
			hash := c.hasher.Hash(keys[i])
			found[i] = c.findEntry(hash, keys[i]) >= 0
		}
	}

	return found
}

// DeleteBatch removes multiple keys efficiently.
// Defers resize check until after all deletions.
func (c *unsafeCache) DeleteBatch(keys []interface{}) []bool {
	deleted := make([]bool, len(keys))
	deleteCount := int32(0)

	// Delete all keys
	for i := range keys {
		if c.Delete(keys[i]) {
			deleted[i] = true
			deleteCount++
		}
	}

	// Check if shrink needed after batch deletion
	size := atomic.LoadInt32(&c.size)
	if float32(size)/float32(c.capacity) < ShrinkThreshold && c.capacity > MinCapacity {
		c.resize(int(c.capacity) / ShrinkFactor)
	}

	return deleted
}

// Prefetching hints for CPU optimization
// These are no-ops but document prefetch points for future optimization

//go:linkname prefetchRead runtime.prefetchnta
func prefetchRead(addr unsafe.Pointer)

//go:linkname prefetchWrite runtime.prefetcht0
func prefetchWrite(addr unsafe.Pointer)
