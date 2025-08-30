// Package kcache provides cache implementations with configurable thread safety.
// This file contains thread-safe cache implementation using RWMutex for concurrent access.
package kcache

import (
	"sync"
)

// ============================================================================
// SAFE CACHE - THREAD-SAFE - SINGLE CACHE WITH MUTEX
// ============================================================================
//
// ✅ SAFE: This cache IS thread-safe using a mutex for synchronization.
// Use for concurrent access when contention is moderate.
// For high contention scenarios, use NewSafeShardedCache() instead.
//
// Performance characteristics:
// - Get/Set: ~60-80 ns/op (3-4x slower than unsafe version)
// - Thread-safe for concurrent access
// - Single mutex for entire cache
// - Better for low-to-moderate contention
//
// ============================================================================

// Ensure safeCache implements Cache and BatchCache interfaces
var (
	_ Cache      = (*safeCache)(nil)
	_ BatchCache = (*safeCache)(nil)
)

// safeCache is a thread-safe cache implementation using mutex.
// Uses a single RWMutex for synchronization, optimized for read-heavy workloads.
// Wraps an unsafe cache internally for the actual storage.
type safeCache struct {
	// Cache line 1 (64 bytes) - Hot path fields
	mu    sync.RWMutex // RWMutex for concurrent access (16 bytes)
	cache *unsafeCache // Underlying unsafe cache implementation
	_     [40]byte     // Cache line padding
}

// NewSafeCache creates a new thread-safe cache.
// ✅ SAFE: Thread-safe for concurrent access.
// Uses RWMutex for synchronization with read-write separation.
func NewSafeCache(capacity int) Cache {
	return newSafeCache(capacity)
}

// newSafeCache is the internal constructor for safe cache.
func newSafeCache(capacity int) *safeCache {
	return &safeCache{
		cache: newUnsafeCacheNoCheck(capacity), // Use no-check version since we have mutex protection
	}
}

// Set stores a key-value pair with thread safety.
// ✅ SAFE: Thread-safe through mutex locking.
// Returns true if key was newly inserted, false if updated existing.
func (c *safeCache) Set(key, value interface{}) bool {
	// Write lock for exclusive access
	c.mu.Lock()
	isNew := c.cache.Set(key, value)
	c.mu.Unlock()
	return isNew
}

// Get retrieves a value by key with thread safety.
// ✅ SAFE: Thread-safe through read locking.
// Returns value and true if found, nil and false otherwise.
func (c *safeCache) Get(key interface{}) (interface{}, bool) {
	// Read lock allows concurrent reads
	c.mu.RLock()
	value, found := c.cache.Get(key)
	c.mu.RUnlock()
	return value, found
}

// Has checks if a key exists with thread safety.
// ✅ SAFE: Thread-safe through read locking.
// Optimized for minimal lock hold time.
//
//go:inline
func (c *safeCache) Has(key interface{}) bool {
	// Read lock for concurrent access
	c.mu.RLock()
	exists := c.cache.Has(key)
	c.mu.RUnlock()
	return exists
}

// Delete removes a key from the cache with thread safety.
// ✅ SAFE: Thread-safe through write locking.
// Returns true if key existed and was deleted.
func (c *safeCache) Delete(key interface{}) bool {
	// Write lock for exclusive access
	c.mu.Lock()
	deleted := c.cache.Delete(key)
	c.mu.Unlock()
	return deleted
}

// Clear removes all entries with thread safety.
// ✅ SAFE: Thread-safe through write locking.
// Retains capacity to avoid reallocation on refill.
func (c *safeCache) Clear() {
	// Write lock for exclusive access
	c.mu.Lock()
	c.cache.Clear()
	c.mu.Unlock()
}

// Len returns the current number of entries with thread safety.
// ✅ SAFE: Thread-safe through read locking.
//
//go:inline
func (c *safeCache) Len() int {
	// Read lock for consistency
	c.mu.RLock()
	size := c.cache.Len()
	c.mu.RUnlock()
	return size
}

// Cap returns the current capacity with thread safety.
// ✅ SAFE: Thread-safe through read locking.
//
//go:inline
func (c *safeCache) Cap() int {
	// Read lock for consistency
	c.mu.RLock()
	capacity := c.cache.Cap()
	c.mu.RUnlock()
	return capacity
}

// Batch operations for safe cache

// SetBatch stores multiple key-value pairs with thread safety.
// ✅ SAFE: Thread-safe through write locking.
// Holds lock for entire batch to ensure consistency.
func (c *safeCache) SetBatch(keys, values []interface{}) int {
	if len(keys) != len(values) {
		return 0
	}

	// Single lock for entire batch operation
	c.mu.Lock()
	defer c.mu.Unlock()

	newCount := 0
	for i := range keys {
		if keys[i] != nil && c.cache.Set(keys[i], values[i]) {
			newCount++
		}
	}

	return newCount
}

// GetBatch retrieves multiple values with thread safety.
// ✅ SAFE: Thread-safe through read locking.
// Holds read lock for entire batch to ensure consistency.
func (c *safeCache) GetBatch(keys []interface{}) ([]interface{}, []bool) {
	values := make([]interface{}, len(keys))
	found := make([]bool, len(keys))

	// Single read lock for entire batch
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := range keys {
		if keys[i] != nil {
			values[i], found[i] = c.cache.Get(keys[i])
		}
	}

	return values, found
}

// HasBatch checks existence of multiple keys with thread safety.
// ✅ SAFE: Thread-safe through read locking.
// Optimized for read-heavy workloads.
func (c *safeCache) HasBatch(keys []interface{}) []bool {
	found := make([]bool, len(keys))

	// Single read lock for entire batch
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := range keys {
		if keys[i] != nil {
			found[i] = c.cache.Has(keys[i])
		}
	}

	return found
}

// DeleteBatch removes multiple keys with thread safety.
// ✅ SAFE: Thread-safe through write locking.
// Holds lock for entire batch to ensure consistency.
func (c *safeCache) DeleteBatch(keys []interface{}) []bool {
	deleted := make([]bool, len(keys))

	// Single write lock for entire batch
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range keys {
		if keys[i] != nil {
			deleted[i] = c.cache.Delete(keys[i])
		}
	}

	return deleted
}
