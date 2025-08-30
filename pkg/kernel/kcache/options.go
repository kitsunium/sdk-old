package kcache

// Option is a functional option for configuring cache creation.
// Uses the functional options pattern for flexibility and extensibility.
type Option func(*cacheOptions)

// cacheOptions holds all configuration options for cache creation.
type cacheOptions struct {
	capacity   int     // Initial capacity
	shards     int     // Number of shards for sharded cache
	loadFactor float32 // Maximum load factor before resize
	hasher     Hasher  // Custom hasher implementation
	safe       bool    // Whether to create thread-safe cache
	sharded    bool    // Whether to create sharded cache
}

// defaultOptions returns default cache options.
func defaultOptions() *cacheOptions {
	return &cacheOptions{
		capacity:   DefaultCapacity,
		shards:     DefaultShardCount,
		loadFactor: DefaultLoadFactor,
		safe:       true, // Default to safe for general use
		sharded:    false,
	}
}

// NewCache creates a new cache with the given options.
// By default, creates a thread-safe, non-sharded cache.
// Use options to customize behavior.
func NewCache(opts ...Option) Cache {
	options := defaultOptions()

	// Apply all options
	for _, opt := range opts {
		opt(options)
	}

	// Validate options
	if options.capacity < MinCapacity {
		options.capacity = MinCapacity
	}
	if options.capacity > MaxCapacity {
		options.capacity = MaxCapacity
	}
	if options.shards < MinShardCount {
		options.shards = MinShardCount
	}
	if options.shards > MaxShardCount {
		options.shards = MaxShardCount
	}

	// Create appropriate cache type based on options
	switch {
	case options.sharded && options.safe:
		return NewSafeShardedCache(options.capacity, options.shards)
	case options.sharded && !options.safe:
		return NewUnsafeShardedCache(options.capacity, options.shards)
	case !options.sharded && options.safe:
		return NewSafeCache(options.capacity)
	default:
		return NewUnsafeCache(options.capacity)
	}
}

// WithCapacity sets the initial capacity of the cache.
// The capacity will be rounded up to the nearest power of 2.
func WithCapacity(capacity int) Option {
	return func(o *cacheOptions) {
		o.capacity = capacity
	}
}

// WithShards sets the number of shards for a sharded cache.
// Must be a power of 2. Will automatically enable sharding.
func WithShards(shards int) Option {
	return func(o *cacheOptions) {
		o.shards = shards
		o.sharded = true // Enable sharding
	}
}

// WithLoadFactor sets the maximum load factor before resize.
// Lower values reduce collisions but increase memory usage.
func WithLoadFactor(factor float32) Option {
	return func(o *cacheOptions) {
		if factor < MinLoadFactor {
			factor = MinLoadFactor
		}
		if factor > MaxLoadFactor {
			factor = MaxLoadFactor
		}
		o.loadFactor = factor
	}
}

// WithHasher sets a custom hasher implementation.
// The hasher must be thread-safe if used with a safe cache.
func WithHasher(hasher Hasher) Option {
	return func(o *cacheOptions) {
		o.hasher = hasher
	}
}

// WithSafe explicitly sets whether the cache should be thread-safe.
// Default is true for safety.
func WithSafe(safe bool) Option {
	return func(o *cacheOptions) {
		o.safe = safe
	}
}

// WithSharded explicitly sets whether to use sharding.
// Sharding improves performance under high contention.
func WithSharded(sharded bool) Option {
	return func(o *cacheOptions) {
		o.sharded = sharded
	}
}

// WithUnsafe creates an unsafe (non-thread-safe) cache.
// Use only when you're certain there's no concurrent access.
func WithUnsafe() Option {
	return func(o *cacheOptions) {
		o.safe = false
	}
}
