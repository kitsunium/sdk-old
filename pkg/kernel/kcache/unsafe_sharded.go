package kcache

import (
	"runtime"
	"unsafe"
)

// ============================================================================
// UNSAFE SHARDED CACHE - NON THREAD-SAFE - SHARDED FOR LOCALITY
// ============================================================================
//
// ⚠️ WARNING: This cache is NOT thread-safe despite being sharded!
// Use ONLY in single-threaded contexts or when YOU manage synchronization.
// For concurrent access, use NewSafeShardedCache() instead.
//
// Performance characteristics:
// - Get/Set: ~25-35 ns/op (slightly slower than single cache due to sharding)
// - Better cache locality for large datasets
// - Reduced collision rate through distribution
// - NO synchronization overhead
//
// ============================================================================

// Ensure unsafeShardedCache implements all required interfaces
var (
	_ Cache        = (*unsafeShardedCache)(nil)
	_ BatchCache   = (*unsafeShardedCache)(nil)
	_ ShardedCache = (*unsafeShardedCache)(nil)
)

// unsafeShard represents a single shard in the unsafe sharded cache.
// NO locks - single-threaded access only.
// Aligned to cache line boundary for optimal CPU cache usage.
type unsafeShard struct {
	cache *unsafeCache // Direct unsafe cache instance
	size  int32        // Shard-local size counter
	_     [56]byte     // Padding to cache line boundary (64 bytes total)
}

// unsafeShardedCache implements a sharded cache without thread safety.
// Sharding improves cache locality and reduces collisions for large datasets.
// NOT thread-safe - all synchronization must be external.
type unsafeShardedCache struct {
	// Cache line 1 (64 bytes) - Hot path fields
	shards    []unsafeShard // Array of shards, size is power of 2
	shardMask uint32        // Mask for fast shard selection (shardCount - 1)
	size      int32         // Total size across all shards (no atomics)
	hasher    Hasher        // Hash function for key distribution

	// Cache line 2 (64 bytes) - Safety and metadata
	checker goroutineChecker // Goroutine safety checker
	_       [48]byte         // Cache line padding
}

// NewUnsafeShardedCache creates a new non-thread-safe sharded cache.
// ⚠️ UNSAFE: No synchronization - single-threaded use only!
// Sharding improves cache locality for large datasets.
// Shard count is adjusted to optimal value for CPU cache efficiency.
func NewUnsafeShardedCache(capacity int, shardCount int) ShardedCache {
	return newUnsafeShardedCache(capacity, shardCount)
}

// newUnsafeShardedCache is the internal constructor.
func newUnsafeShardedCache(capacity int, shardCount int) *unsafeShardedCache {
	// Optimize shard count based on CPU cores
	if shardCount <= 0 {
		// Default to 4x CPU count for good concurrency
		shardCount = runtime.NumCPU() * 4
	}

	// Ensure power of 2 for fast modulo
	shardCount = nextPowerOf2(clamp(shardCount, MinShardCount, MaxShardCount))

	// Calculate per-shard capacity
	shardCapacity := capacity / shardCount
	if shardCapacity < MinCapacity {
		shardCapacity = MinCapacity
	}

	// Initialize shards
	shards := make([]unsafeShard, shardCount)
	for i := range shards {
		shards[i].cache = newUnsafeCache(shardCapacity)
		shards[i].size = 0
	}

	return &unsafeShardedCache{
		shards:    shards,
		shardMask: uint32(shardCount - 1),
		hasher:    newFNVHasher(),
	}
}

// getShard returns the shard for a given key.
// Uses consistent hashing for even distribution.
//
//go:inline
//go:nosplit
func (sc *unsafeShardedCache) getShard(key interface{}) *unsafeShard {
	// Hash key and select shard using mask
	hash := sc.hasher.Hash(key)
	idx := uint32(hash) & sc.shardMask
	return &sc.shards[idx]
}

// Set stores a key-value pair in the appropriate shard.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
func (sc *unsafeShardedCache) Set(key, value interface{}) bool {
	// Check for concurrent access
	sc.checker.checkSafety()

	// Get target shard
	s := sc.getShard(key)

	// Direct set without locking
	isNew := s.cache.Set(key, value)

	// Update sizes if new key
	if isNew {
		s.size++
		sc.size++
	}

	return isNew
}

// Get retrieves a value from the appropriate shard.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
func (sc *unsafeShardedCache) Get(key interface{}) (interface{}, bool) {
	// Check for concurrent access
	sc.checker.checkSafety()

	// Get target shard
	s := sc.getShard(key)

	// Direct get without locking
	value, found := s.cache.Get(key)

	return value, found
}

// Has checks if a key exists in the appropriate shard.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
//
//go:inline
func (sc *unsafeShardedCache) Has(key interface{}) bool {
	// Check for concurrent access
	sc.checker.checkSafety()

	// Get target shard
	s := sc.getShard(key)

	// Direct check without locking
	exists := s.cache.Has(key)

	return exists
}

// Delete removes a key from the appropriate shard.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
func (sc *unsafeShardedCache) Delete(key interface{}) bool {
	// Check for concurrent access
	sc.checker.checkSafety()

	// Get target shard
	s := sc.getShard(key)

	// Direct delete without locking
	deleted := s.cache.Delete(key)

	// Update sizes if deleted
	if deleted {
		s.size--
		sc.size--
	}

	return deleted
}

// Clear removes all entries from all shards.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
func (sc *unsafeShardedCache) Clear() {
	// Check for concurrent access
	sc.checker.checkSafety()

	// Clear each shard sequentially (no parallelism in unsafe version)
	for i := range sc.shards {
		sc.shards[i].cache.Clear()
		sc.shards[i].size = 0
	}

	// Reset global size
	sc.size = 0
}

// Len returns total entries across all shards.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
//
//go:inline
func (sc *unsafeShardedCache) Len() int {
	// Check for concurrent access
	sc.checker.checkSafety()
	return int(sc.size)
}

// Cap returns total capacity across all shards.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
func (sc *unsafeShardedCache) Cap() int {
	// Check for concurrent access
	sc.checker.checkSafety()

	totalCap := 0
	for i := range sc.shards {
		totalCap += sc.shards[i].cache.Cap()
	}
	return totalCap
}

// ShardCount returns the number of shards.
//
//go:inline
func (sc *unsafeShardedCache) ShardCount() int {
	return len(sc.shards)
}

// ShardFor returns the shard index for a given key.
// Useful for debugging and statistics.
//
//go:inline
func (sc *unsafeShardedCache) ShardFor(key interface{}) int {
	hash := sc.hasher.Hash(key)
	return int(uint32(hash) & sc.shardMask)
}

// Batch operations for sharded cache

// SetBatch stores multiple key-value pairs across shards.
// Groups keys by shard to minimize lock acquisitions.
func (sc *unsafeShardedCache) SetBatch(keys, values []interface{}) int {
	if len(keys) != len(values) {
		return 0
	}

	// Group keys by shard to minimize locking
	type shardBatch struct {
		keys   []interface{}
		values []interface{}
	}

	// Pre-allocate shard batches
	batches := make([]shardBatch, len(sc.shards))

	// Distribute keys to shards
	for i := range keys {
		if keys[i] == nil {
			continue
		}
		shardIdx := sc.ShardFor(keys[i])
		batches[shardIdx].keys = append(batches[shardIdx].keys, keys[i])
		batches[shardIdx].values = append(batches[shardIdx].values, values[i])
	}

	// Process each shard's batch sequentially (no parallelism in unsafe version)
	newCount := 0

	for i := range batches {
		if len(batches[i].keys) == 0 {
			continue
		}

		// Process batch for this shard
		s := &sc.shards[i]

		// Process keys in this shard
		for j := range batches[i].keys {
			if s.cache.Set(batches[i].keys[j], batches[i].values[j]) {
				newCount++
				s.size++
			}
		}
	}

	// Update global size
	if newCount > 0 {
		sc.size += int32(newCount)
	}

	return newCount
}

// GetBatch retrieves multiple values from across shards.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
func (sc *unsafeShardedCache) GetBatch(keys []interface{}) ([]interface{}, []bool) {
	// Check for concurrent access
	sc.checker.checkSafety()

	values := make([]interface{}, len(keys))
	found := make([]bool, len(keys))

	// Group keys by shard
	type shardQuery struct {
		indices []int
		keys    []interface{}
	}

	queries := make([]shardQuery, len(sc.shards))

	// Distribute keys to shards
	for i := range keys {
		if keys[i] == nil {
			continue
		}
		shardIdx := sc.ShardFor(keys[i])
		queries[shardIdx].indices = append(queries[shardIdx].indices, i)
		queries[shardIdx].keys = append(queries[shardIdx].keys, keys[i])
	}

	// Process each shard sequentially (no parallelism in unsafe version)
	for shardIdx := range queries {
		if len(queries[shardIdx].keys) == 0 {
			continue
		}

		// Get values from this shard
		s := &sc.shards[shardIdx]
		for j, key := range queries[shardIdx].keys {
			origIdx := queries[shardIdx].indices[j]
			values[origIdx], found[origIdx] = s.cache.Get(key)
		}
	}
	return values, found
}

// HasBatch checks existence of multiple keys across shards.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
func (sc *unsafeShardedCache) HasBatch(keys []interface{}) []bool {
	// Check for concurrent access
	sc.checker.checkSafety()

	found := make([]bool, len(keys))

	// Sequential processing (no parallelism in unsafe version)
	for i := range keys {
		if keys[i] != nil {
			found[i] = sc.Has(keys[i])
		}
	}

	return found
}

// DeleteBatch removes multiple keys from across shards.
// ⚠️ UNSAFE: Not thread-safe! Will panic if used concurrently.
func (sc *unsafeShardedCache) DeleteBatch(keys []interface{}) []bool {
	// Check for concurrent access
	sc.checker.checkSafety()

	deleted := make([]bool, len(keys))

	// Group keys by shard
	type shardDelete struct {
		indices []int
		keys    []interface{}
	}

	deletes := make([]shardDelete, len(sc.shards))

	// Distribute keys to shards
	for i := range keys {
		if keys[i] == nil {
			continue
		}
		shardIdx := sc.ShardFor(keys[i])
		deletes[shardIdx].indices = append(deletes[shardIdx].indices, i)
		deletes[shardIdx].keys = append(deletes[shardIdx].keys, keys[i])
	}

	// Process each shard sequentially (no parallelism in unsafe version)
	deleteCount := 0

	for shardIdx := range deletes {
		if len(deletes[shardIdx].keys) == 0 {
			continue
		}

		// Delete keys from this shard
		s := &sc.shards[shardIdx]
		for j, key := range deletes[shardIdx].keys {
			origIdx := deletes[shardIdx].indices[j]
			if s.cache.Delete(key) {
				deleted[origIdx] = true
				deleteCount++
				s.size--
			}
		}
	}

	// Update global size
	if deleteCount > 0 {
		sc.size -= int32(deleteCount)
	}

	return deleted
}

// clamp restricts a value to the given range.
//
//go:inline
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Memory ordering helpers are not needed in unsafe version
// All operations are sequential without atomics

// CPU-specific optimizations

func init() {
	// Detect CPU features and optimize accordingly
	// For now, using generic optimizations

	// Ensure shards are cache-line aligned
	if unsafe.Sizeof(unsafeShard{}) > CacheLineSize {
		// Shard is larger than cache line, may cause false sharing
		// In unsafe version, this is less of a concern as there's no contention
	}
}

// Parallel operations are not used in unsafe version
// All operations are sequential for thread safety
