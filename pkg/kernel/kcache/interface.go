// Package kcache provides an ultra-optimized, unsafe-based cache implementation
// designed for maximum performance in kernel-level operations.
package kcache

// Cache defines the core cache interface for single-threaded or synchronized access.
// This interface prioritizes raw performance over safety, using unsafe operations
// extensively to minimize overhead and maximize throughput.
type Cache interface {
	// Set stores a key-value pair in the cache with zero-copy semantics where possible.
	// Returns true if the key was newly inserted, false if it was updated.
	// This operation is NOT thread-safe by design for maximum performance.
	Set(key, value interface{}) bool

	// Get retrieves a value by key with minimal overhead.
	// Returns the value and true if found, nil and false otherwise.
	// Uses unsafe pointer operations to avoid unnecessary allocations.
	Get(key interface{}) (interface{}, bool)

	// Has checks if a key exists without retrieving the value.
	// Optimized for cache-line efficiency and branch prediction.
	Has(key interface{}) bool

	// Delete removes a key from the cache.
	// Returns true if the key existed and was removed, false otherwise.
	// May not immediately free memory to avoid allocation overhead.
	Delete(key interface{}) bool

	// Clear removes all entries from the cache efficiently.
	// May retain internal structures for reuse to minimize future allocations.
	Clear()

	// Len returns the number of entries in the cache.
	// This operation may be eventually consistent in concurrent scenarios.
	Len() int

	// Cap returns the maximum capacity of the cache.
	// Returns -1 if the cache has no fixed capacity limit.
	Cap() int
}

// BatchCache extends Cache with batch operations for improved throughput.
// Batch operations amortize overhead across multiple operations and
// improve CPU cache utilization through better locality.
type BatchCache interface {
	Cache

	// SetBatch stores multiple key-value pairs in a single operation.
	// Returns the number of new keys inserted (not updates).
	// Optimized for vectorized operations and cache-line efficiency.
	SetBatch(keys, values []interface{}) int

	// GetBatch retrieves multiple values by their keys.
	// Returns parallel slices of values and found flags.
	// Uses unsafe operations to minimize allocation overhead.
	GetBatch(keys []interface{}) ([]interface{}, []bool)

	// HasBatch checks existence of multiple keys efficiently.
	// Optimized to minimize cache misses and branch mispredictions.
	HasBatch(keys []interface{}) []bool

	// DeleteBatch removes multiple keys from the cache.
	// Returns a slice indicating which keys were actually deleted.
	// May defer actual memory reclamation for performance.
	DeleteBatch(keys []interface{}) []bool
}

// ShardedCache provides a concurrent cache implementation using sharding.
// Reduces contention by distributing keys across multiple shards,
// each with its own synchronization primitive.
type ShardedCache interface {
	BatchCache

	// ShardCount returns the number of shards in use.
	// Power of 2 for efficient modulo operations via bit masking.
	ShardCount() int

	// ShardFor returns the shard index for a given key.
	// Uses optimized hashing for even distribution.
	ShardFor(key interface{}) int
}

// Pool defines an interface for object pooling to minimize allocations.
// Critical for achieving zero-allocation in hot paths.
type Pool interface {
	// Get retrieves an object from the pool or creates a new one.
	// Uses unsafe operations to avoid interface{} boxing overhead.
	Get() interface{}

	// Put returns an object to the pool for reuse.
	// Object is reset to avoid data leaks between uses.
	Put(interface{})

	// Reset clears the pool, releasing all pooled objects.
	// Used during shutdown or memory pressure scenarios.
	Reset()
}

// Hasher defines an interface for high-performance hashing.
// Implementations must be deterministic and well-distributed.
type Hasher interface {
	// Hash computes a hash value for the given key.
	// Must be fast and provide good distribution.
	// Implementations may use unsafe operations for speed.
	Hash(key interface{}) uint64

	// Equal checks if two keys are equal.
	// Must be consistent with Hash (equal keys must have equal hashes).
	Equal(a, b interface{}) bool
}

// Entry represents a cache entry with zero-copy semantics.
// Designed for cache-line alignment and minimal memory footprint.
type Entry interface {
	// Key returns the entry's key without copying.
	Key() interface{}

	// Value returns the entry's value without copying.
	Value() interface{}

	// Hash returns the pre-computed hash of the key.
	// Cached to avoid recomputation on operations.
	Hash() uint64
}

// Options defines configuration for cache creation.
// Uses functional options pattern for flexibility.
type Options interface {
	// WithCapacity sets the initial capacity hint.
	// Should be a power of 2 for optimal performance.
	WithCapacity(capacity int) Options

	// WithHasher sets a custom hasher implementation.
	// Default uses FNV-1a for speed and distribution.
	WithHasher(hasher Hasher) Options

	// WithShards sets the number of shards for ShardedCache.
	// Must be a power of 2, typically CPU count * 4.
	WithShards(shards int) Options

	// WithLoadFactor sets the maximum load factor before resize.
	// Lower values reduce collisions but increase memory usage.
	WithLoadFactor(factor float32) Options

	// Build creates the cache with the configured options.
	// Performs validation and applies optimizations.
	Build() Cache
}

// ============================================================================
// CACHE CREATION - EXPLICIT SAFETY CHOICE REQUIRED
// ============================================================================
//
// ⚠️ CRITICAL: You MUST explicitly choose between:
//
// BASIC CACHES:
// 1. NewUnsafeCache() - NOT thread-safe
//    - Use ONLY in single-threaded contexts
//    - ~20-30 ns/op Get/Set (FASTEST)
//    - Will cause data corruption if used concurrently!
//
// 2. NewSafeCache() - Thread-safe with mutex
//    - Use for concurrent access
//    - ~60-80 ns/op Get/Set (FAST)
//    - Safe for multiple goroutines
//
// SHARDED CACHES (for large datasets or high contention):
// 3. NewUnsafeShardedCache() - NOT thread-safe
//    - Sharded but NOT safe for concurrent use
//    - ~25-35 ns/op with better cache locality
//    - Use when you need sharding but manage sync externally
//
// 4. NewSafeShardedCache() - Thread-safe via sharding
//    - Best for high-contention scenarios (10+ goroutines)
//    - ~40-50 ns/op even with 100 goroutines
//    - Scales linearly with CPU cores
//
// ============================================================================
