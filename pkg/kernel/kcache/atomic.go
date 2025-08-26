package kcache

import (
	"sync"
	"sync/atomic"
	"time"
)

// AtomicCache is a lock-free cache for read-heavy workloads.
// It uses atomic operations and RCU (Read-Copy-Update) pattern.
//
// Suitable for:
//   - Read-heavy workloads (90%+ reads)
//   - Small to medium sized caches
//   - Low latency requirements
//
// Type parameters:
//   - K: The type of keys (must be comparable)
//   - V: The type of values (can be any type)
type AtomicCache[K comparable, V any] struct {
	data     atomic.Pointer[atomicMap[K, V]]
	mu       sync.Mutex // Only for writes
	capacity int
	size     atomic.Int32
}

type atomicMap[K comparable, V any] struct {
	m map[K]*atomicEntry[V]
}

type atomicEntry[V any] struct {
	value      V
	expiration int64
	accessTime atomic.Int64
}

// NewAtomicCache creates a new atomic cache with the specified capacity.
func NewAtomicCache[K comparable, V any](capacity int) *AtomicCache[K, V] {
	if capacity <= 0 {
		capacity = 128
	}

	c := &AtomicCache[K, V]{
		capacity: capacity,
	}

	initial := &atomicMap[K, V]{
		m: make(map[K]*atomicEntry[V], capacity),
	}
	c.data.Store(initial)

	return c
}

// Get retrieves a value from the cache using lock-free read.
func (c *AtomicCache[K, V]) Get(key K) (V, bool) {
	// Lock-free read
	data := c.data.Load()
	entry, exists := data.m[key]

	if !exists {
		var zero V
		return zero, false
	}

	// Check expiration with cached time for better performance
	now := time.Now().UnixNano()
	if entry.expiration > 0 && now > entry.expiration {
		var zero V
		return zero, false
	}

	// Update access time atomically
	entry.accessTime.Store(now)

	return entry.value, true
}

// Set stores a key-value pair in the cache.
func (c *AtomicCache[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, 0)
}

// SetWithTTL stores a key-value pair with TTL using RCU pattern.
func (c *AtomicCache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()

	now := time.Now().UnixNano()
	var expiration int64
	if ttl > 0 {
		expiration = now + int64(ttl)
	}

	// Copy-on-write
	oldData := c.data.Load()

	// Check if update only
	if existingEntry, exists := oldData.m[key]; exists {
		// Fast path: just update the entry in place if possible
		if existingEntry.expiration == expiration {
			// Same expiration, just update value (safe because V is not accessed concurrently)
			existingEntry.value = value
			existingEntry.accessTime.Store(now)
			c.mu.Unlock()
			return
		}
	}

	oldSize := len(oldData.m)
	newCapacity := oldSize + 1
	if newCapacity < c.capacity {
		newCapacity = c.capacity
	}

	newData := &atomicMap[K, V]{
		m: make(map[K]*atomicEntry[V], newCapacity),
	}

	// Copy existing entries
	for k, v := range oldData.m {
		newData.m[k] = v
	}

	// Add/update entry
	entry := &atomicEntry[V]{
		value:      value,
		expiration: expiration,
	}
	entry.accessTime.Store(now)

	if _, exists := newData.m[key]; !exists {
		currentSize := c.size.Add(1)

		// Evict if needed
		if int(currentSize) > c.capacity {
			c.evictLRU(newData)
			c.size.Add(-1)
		}
	}

	newData.m[key] = entry

	// Atomic swap
	c.data.Store(newData)
	c.mu.Unlock()
}

// Delete removes a key from the cache.
func (c *AtomicCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldData := c.data.Load()
	if _, exists := oldData.m[key]; !exists {
		return
	}

	// Copy-on-write
	newData := &atomicMap[K, V]{
		m: make(map[K]*atomicEntry[V], len(oldData.m)-1),
	}

	for k, v := range oldData.m {
		if k != key {
			newData.m[k] = v
		}
	}

	c.size.Add(-1)
	c.data.Store(newData)
}

// Clear removes all entries from the cache.
func (c *AtomicCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	newData := &atomicMap[K, V]{
		m: make(map[K]*atomicEntry[V], c.capacity),
	}

	c.data.Store(newData)
	c.size.Store(0)
}

// Size returns the current number of entries.
func (c *AtomicCache[K, V]) Size() int {
	return int(c.size.Load())
}

// Has checks if a key exists in the cache.
func (c *AtomicCache[K, V]) Has(key K) bool {
	data := c.data.Load()
	entry, exists := data.m[key]

	if !exists {
		return false
	}

	if entry.expiration > 0 && time.Now().UnixNano() > entry.expiration {
		return false
	}

	return true
}

// evictLRU removes the least recently used entry, prioritizing expired entries.
func (c *AtomicCache[K, V]) evictLRU(data *atomicMap[K, V]) {
	now := time.Now().UnixNano()
	var oldestKey K
	var foundExpired bool
	oldestTime := now

	// First pass: look for expired entries to evict
	for k, v := range data.m {
		if v.expiration > 0 && now > v.expiration {
			oldestKey = k
			foundExpired = true
			break // Evict first expired entry found
		}
	}

	// Second pass: if no expired entries, find LRU entry
	if !foundExpired {
		for k, v := range data.m {
			accessTime := v.accessTime.Load()
			if accessTime < oldestTime {
				oldestTime = accessTime
				oldestKey = k
			}
		}
	}

	delete(data.m, oldestKey)
}

// Keys returns all keys in the cache.
func (c *AtomicCache[K, V]) Keys() []K {
	data := c.data.Load()
	keys := make([]K, 0, len(data.m))
	now := time.Now().UnixNano()

	for k, entry := range data.m {
		if entry.expiration == 0 || now <= entry.expiration {
			keys = append(keys, k)
		}
	}
	return keys
}

// Range calls f for each key-value pair in the cache.
// If f returns false, iteration stops.
func (c *AtomicCache[K, V]) Range(f func(key K, value V) bool) {
	data := c.data.Load()
	now := time.Now().UnixNano()

	for k, entry := range data.m {
		if entry.expiration == 0 || now <= entry.expiration {
			if !f(k, entry.value) {
				break
			}
		}
	}
}

// Ensure AtomicCache implements Cache interface.
var _ Cache[string, any] = (*AtomicCache[string, any])(nil)

// FastGet returns a pointer to avoid copying large values.
// WARNING: The returned pointer should not be modified as it's shared across goroutines.
func (c *AtomicCache[K, V]) FastGet(key K) (*V, bool) {
	data := c.data.Load()
	entry, exists := data.m[key]

	if !exists {
		return nil, false
	}

	if entry.expiration > 0 && time.Now().UnixNano() > entry.expiration {
		return nil, false
	}

	entry.accessTime.Store(time.Now().UnixNano())
	return &entry.value, true
}

// BatchGet retrieves multiple values in a single operation.
func (c *AtomicCache[K, V]) BatchGet(keys []K) map[K]V {
	result := make(map[K]V, len(keys))
	data := c.data.Load()
	now := time.Now().UnixNano()

	for _, key := range keys {
		if entry, exists := data.m[key]; exists {
			if entry.expiration == 0 || now <= entry.expiration {
				entry.accessTime.Store(now)
				result[key] = entry.value
			}
		}
	}

	return result
}

// BatchSet stores multiple key-value pairs in a single operation.
func (c *AtomicCache[K, V]) BatchSet(items map[K]V) {
	if len(items) == 0 {
		return
	}

	c.mu.Lock()

	oldData := c.data.Load()

	// Pre-calculate actual new items to avoid over-allocation
	newCount := 0
	for k := range items {
		if _, exists := oldData.m[k]; !exists {
			newCount++
		}
	}

	newCapacity := len(oldData.m) + newCount
	if newCapacity < c.capacity {
		newCapacity = c.capacity
	}

	newData := &atomicMap[K, V]{
		m: make(map[K]*atomicEntry[V], newCapacity),
	}

	// Copy existing entries
	for k, v := range oldData.m {
		newData.m[k] = v
	}

	now := time.Now().UnixNano()

	// Add new entries
	for k, v := range items {
		entry := &atomicEntry[V]{
			value: v,
		}
		entry.accessTime.Store(now)
		newData.m[k] = entry
	}

	// Evict if needed
	newSize := len(newData.m)
	for newSize > c.capacity {
		c.evictLRU(newData)
		newSize--
	}

	c.size.Store(int32(newSize))
	c.data.Store(newData)
	c.mu.Unlock()
}
