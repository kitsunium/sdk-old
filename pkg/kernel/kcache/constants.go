// Package kcache provides cache implementations with configurable thread safety.
// This file contains configuration constants for cache sizing, performance tuning, and memory optimization.
package kcache

import "unsafe"

// Cache size and capacity constants optimized for CPU cache lines
const (
	// DefaultCapacity is the default initial capacity for new caches.
	// Set to 16 to fit initial allocations in a single cache line cluster.
	DefaultCapacity = 16

	// MinCapacity is the minimum allowed cache capacity.
	// Must be at least 8 to ensure efficient hash distribution.
	MinCapacity = 8

	// MaxCapacity is the maximum allowed cache capacity.
	// Limited to prevent excessive memory usage and allocation time.
	MaxCapacity = 1 << 24 // 16 million entries (~768MB with 48-byte entries)

	// DefaultLoadFactor is the default maximum load factor before resizing.
	// 0.75 provides good balance between memory usage and collision rate.
	DefaultLoadFactor = 0.75

	// MinLoadFactor is the minimum allowed load factor.
	// Lower values waste memory but reduce collisions.
	MinLoadFactor = 0.25

	// MaxLoadFactor is the maximum allowed load factor.
	// Higher values increase collisions and degrade performance.
	MaxLoadFactor = 0.95
)

// Sharding constants for concurrent access optimization
const (
	// DefaultShardCount is the default number of shards for ShardedCache.
	// Set to 32 for good concurrency on modern multi-core systems.
	DefaultShardCount = 32

	// MinShardCount is the minimum number of shards.
	// At least 2 shards needed to reduce contention.
	MinShardCount = 2

	// MaxShardCount is the maximum number of shards.
	// Beyond 256 shards, overhead outweighs benefits.
	MaxShardCount = 256
)

// Memory and cache-line optimization constants
const (
	// CacheLineSize is the typical CPU cache line size in bytes.
	// Used for padding and alignment optimizations.
	CacheLineSize = 64

	// PointerSize is the size of a pointer on this architecture.
	// Used for memory layout calculations.
	PointerSize = int(unsafe.Sizeof(uintptr(0)))

	// InterfaceSize is the size of an interface{} value.
	// Two pointers: type and data.
	InterfaceSize = PointerSize * 2

	// EntrySize is the approximate size of a cache entry.
	// Used for memory estimation and pool sizing.
	EntrySize = InterfaceSize*2 + 8 // key + value + hash
)

// Hash function constants for FNV-1a algorithm
const (
	// FNVOffsetBasis is the FNV-1a 64-bit offset basis.
	// Starting value for FNV hash computation.
	FNVOffsetBasis uint64 = 14695981039346656037

	// FNVPrime is the FNV-1a 64-bit prime multiplier.
	// Used in FNV hash computation for good distribution.
	FNVPrime uint64 = 1099511628211
)

// Probe sequence constants for open addressing
const (
	// LinearProbeStep is the step size for linear probing.
	// Set to 1 for sequential memory access (cache-friendly).
	LinearProbeStep = 1

	// QuadraticProbeC1 is the first coefficient for quadratic probing.
	// Carefully chosen to guarantee full table coverage.
	QuadraticProbeC1 = 1

	// QuadraticProbeC2 is the second coefficient for quadratic probing.
	// Works with C1 to ensure all slots are reachable.
	QuadraticProbeC2 = 3

	// MaxProbeDistance is the maximum distance to probe before resize.
	// Limits worst-case lookup time.
	MaxProbeDistance = 128

	// RobinHoodMaxDistance is the maximum distance for Robin Hood hashing.
	// Ensures bounded worst-case performance.
	RobinHoodMaxDistance = 32
)

// Pool configuration constants
const (
	// DefaultPoolSize is the default size for object pools.
	// Sized to handle typical burst patterns.
	DefaultPoolSize = 128

	// MaxPoolSize is the maximum size for object pools.
	// Prevents excessive memory retention.
	MaxPoolSize = 4096

	// PoolGCInterval is the number of operations between pool GC checks.
	// Balances memory usage with GC overhead.
	PoolGCInterval = 10000
)

// Batch operation constants
const (
	// DefaultBatchSize is the default size for batch operations.
	// Optimized for L1 cache utilization.
	DefaultBatchSize = 64

	// MaxBatchSize is the maximum size for a single batch operation.
	// Prevents stack overflow and ensures predictable latency.
	MaxBatchSize = 10000

	// BatchAllocThreshold is the threshold for switching to heap allocation.
	// Below this, use stack allocation for better performance.
	BatchAllocThreshold = 256
)

// Debug and build mode constants
const (
	// DebugMode indicates if debug checks are enabled.
	// Set via build tags, zero overhead in production.
	// Build with -tags debug to enable
	DebugMode = false // Will be true when debug tag is set

	// RaceMode indicates if race detector is enabled.
	// Used to enable additional synchronization in tests.
	// Automatically set when running with -race
	RaceMode = false // Will be true when race tag is set
)

// State constants for entry and cache lifecycle
const (
	// StateEmpty indicates an empty slot in the hash table.
	// Zero value for efficient initialization.
	StateEmpty uint8 = iota

	// StateActive indicates an active entry with valid data.
	// Most common state during normal operation.
	StateActive

	// StateDeleted indicates a deleted entry (tombstone).
	// Used in open addressing to maintain probe sequences.
	StateDeleted

	// StateMoved indicates an entry moved during resize.
	// Temporary state during table migration.
	StateMoved
)

// Error codes for internal use (not exposed as errors)
const (
	// Success indicates successful operation.
	Success int = iota

	// ErrKeyNotFound indicates key was not found.
	ErrKeyNotFound

	// ErrCapacityExceeded indicates capacity limit reached.
	ErrCapacityExceeded

	// ErrInvalidKey indicates nil or invalid key.
	ErrInvalidKey

	// ErrConcurrentModification indicates unsafe concurrent access.
	ErrConcurrentModification

	// ErrPoolExhausted indicates object pool is exhausted.
	ErrPoolExhausted
)

// Hash table growth factors
const (
	// GrowthFactor is the multiplier for table size on resize.
	// 2x growth balances memory usage with resize frequency.
	GrowthFactor = 2

	// ShrinkFactor is the divisor for table size on shrink.
	// 2x shrink maintains power-of-2 sizes.
	ShrinkFactor = 2

	// ShrinkThreshold is the load factor triggering shrink.
	// Shrink when less than 25% full to reclaim memory.
	ShrinkThreshold = 0.25
)

// Timing constants for spin locks and retries
const (
	// SpinLimit is the number of spin iterations before yielding.
	// Tuned for modern CPU speeds and typical contention patterns.
	SpinLimit = 100

	// BackoffInitial is the initial backoff delay in nanoseconds.
	// Short enough to catch quick releases.
	BackoffInitial = 1

	// BackoffMax is the maximum backoff delay in nanoseconds.
	// Prevents excessive spinning under high contention.
	BackoffMax = 1000

	// RetryLimit is the maximum number of retries for operations.
	// Prevents infinite loops on persistent conflicts.
	RetryLimit = 10
)
