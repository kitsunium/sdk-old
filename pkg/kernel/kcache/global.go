package kcache

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// globalCache holds the global cache instance.
// Initialized lazily on first access.
var globalCache struct {
	instance Cache
	once     sync.Once
	mu       sync.RWMutex
	config   *globalConfig
}

// globalConfig holds global cache configuration.
type globalConfig struct {
	capacity    int
	shardCount  int
	loadFactor  float32
	useSharded  bool
	initialized atomic.Bool
}

// defaultGlobalConfig returns default configuration.
func defaultGlobalConfig() *globalConfig {
	return &globalConfig{
		capacity:   DefaultCapacity * 16, // Larger default for global cache
		shardCount: runtime.NumCPU() * 4,
		loadFactor: DefaultLoadFactor,
		useSharded: runtime.NumCPU() > 1, // Use sharded on multi-core
	}
}

// Global returns the global cache instance.
// Creates it lazily on first access with default configuration.
//
//go:inline
func Global() Cache {
	globalCache.once.Do(func() {
		config := globalCache.config
		if config == nil {
			config = defaultGlobalConfig()
		}

		if config.useSharded {
			// Use safe sharded cache for global instance (thread-safe)
			globalCache.instance = NewSafeShardedCache(config.capacity, config.shardCount)
		} else {
			// Use safe cache for global instance (thread-safe)
			globalCache.instance = NewSafeCache(config.capacity)
		}

		config.initialized.Store(true)
	})

	return globalCache.instance
}

// SetGlobal sets the global cache instance.
// Must be called before first use of Global().
// Returns false if global cache was already initialized.
func SetGlobal(cache Cache) bool {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	// Check if already initialized
	if globalCache.config != nil && globalCache.config.initialized.Load() {
		return false
	}

	// Set the instance
	globalCache.instance = cache
	if globalCache.config == nil {
		globalCache.config = defaultGlobalConfig()
	}
	globalCache.config.initialized.Store(true)

	// Mark once as done to prevent lazy init
	globalCache.once.Do(func() {})

	return true
}

// ConfigureGlobal configures the global cache before initialization.
// Must be called before first use of Global().
// Returns false if global cache was already initialized.
func ConfigureGlobal(capacity, shardCount int, useSharded bool) bool {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	// Check if already initialized
	if globalCache.config != nil && globalCache.config.initialized.Load() {
		return false
	}

	// Set configuration
	if globalCache.config == nil {
		globalCache.config = &globalConfig{}
	}

	globalCache.config.capacity = capacity
	globalCache.config.shardCount = shardCount
	globalCache.config.useSharded = useSharded
	globalCache.config.loadFactor = DefaultLoadFactor

	return true
}

// ResetGlobal resets the global cache instance.
// Clears all entries but retains the instance.
func ResetGlobal() {
	if globalCache.instance != nil {
		globalCache.instance.Clear()
	}
}

// Quick access functions for global cache

// Set stores a key-value pair in the global cache.
//
//go:inline
func Set(key, value interface{}) bool {
	return Global().Set(key, value)
}

// Get retrieves a value from the global cache.
//
//go:inline
func Get(key interface{}) (interface{}, bool) {
	return Global().Get(key)
}

// Has checks if a key exists in the global cache.
//
//go:inline
func Has(key interface{}) bool {
	return Global().Has(key)
}

// Delete removes a key from the global cache.
//
//go:inline
func Delete(key interface{}) bool {
	return Global().Delete(key)
}

// Clear removes all entries from the global cache.
//
//go:inline
func Clear() {
	Global().Clear()
}

// Len returns the number of entries in the global cache.
//
//go:inline
func Len() int {
	return Global().Len()
}

// Cap returns the capacity of the global cache.
//
//go:inline
func Cap() int {
	return Global().Cap()
}

// SetBatch stores multiple key-value pairs in the global cache.
//
//go:inline
func SetBatch(keys, values []interface{}) int {
	if bc, ok := Global().(BatchCache); ok {
		return bc.SetBatch(keys, values)
	}
	// Fallback to simple batch
	return simpleBatchSet(Global(), keys, values)
}

// GetBatch retrieves multiple values from the global cache.
//
//go:inline
func GetBatch(keys []interface{}) ([]interface{}, []bool) {
	if keys == nil {
		return nil, nil
	}
	if bc, ok := Global().(BatchCache); ok {
		return bc.GetBatch(keys)
	}
	// Fallback to simple batch
	return OptimizedGetBatch(Global(), keys)
}

// HasBatch checks existence of multiple keys in the global cache.
//
//go:inline
func HasBatch(keys []interface{}) []bool {
	if keys == nil {
		return nil
	}
	if bc, ok := Global().(BatchCache); ok {
		return bc.HasBatch(keys)
	}
	// Fallback to simple batch
	return OptimizedHasBatch(Global(), keys)
}

// DeleteBatch removes multiple keys from the global cache.
//
//go:inline
func DeleteBatch(keys []interface{}) []bool {
	if keys == nil {
		return nil
	}
	if bc, ok := Global().(BatchCache); ok {
		return bc.DeleteBatch(keys)
	}
	// Fallback to simple batch
	return OptimizedDeleteBatch(Global(), keys)
}

// Thread-local cache support for reduced contention

// threadLocalCache provides thread-local caching.
// Reduces contention on global cache for hot keys.
type threadLocalCache struct {
	cache Cache
	tid   int
}

// Thread-local storage using goroutine ID
// This is a hack but works for reducing contention
var threadLocals sync.Map

// getThreadLocal returns the thread-local cache.
// Creates one if it doesn't exist.
func getThreadLocal() Cache {
	tid := getGoroutineID()

	if v, ok := threadLocals.Load(tid); ok {
		return v.(*threadLocalCache).cache
	}

	// Create new thread-local cache
	tlc := &threadLocalCache{
		// Use safe cache for stats collector (thread-safe)
		cache: NewSafeCache(MinCapacity),
		tid:   tid,
	}

	threadLocals.Store(tid, tlc)
	return tlc.cache
}

// getGoroutineID returns the current goroutine ID.
// This is a simplified version for demonstration.
func getGoroutineID() int {
	// In production, this would use runtime internals
	// For now, return a pseudo-random ID based on memory address
	var x int
	return int(uintptr(unsafe.Pointer(&x)) >> 12)
}

// WithThreadLocal executes a function with thread-local cache.
// Useful for reducing contention in hot paths.
func WithThreadLocal(fn func(Cache)) {
	cache := getThreadLocal()
	fn(cache)
}

// Distributed cache support (preparation for future clustering)

// DistributedCache represents a cache that can be distributed.
// Placeholder for future distributed cache implementation.
type DistributedCache interface {
	Cache

	// Replicate replicates an operation to peers
	Replicate(op string, key, value interface{})

	// Sync synchronizes with peers
	Sync() error

	// Peers returns list of peer nodes
	Peers() []string
}
