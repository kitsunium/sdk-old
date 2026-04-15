package cache

import (
	"maps"
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
	stats    atomic.Pointer[Stats]
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

	c.stats.Store(NewStats())

	return c
}

// Get retrieves a value from the cache using lock-free read.
func (c *AtomicCache[K, V]) Get(key K) (V, bool) {
	// Lock-free read
	data := c.data.Load()
	entry, exists := data.m[key]

	if !exists {
		c.incrementMisses()
		var zero V
		return zero, false
	}

	// Check expiration
	if entry.expiration > 0 && time.Now().UnixNano() > entry.expiration {
		c.incrementMisses()
		var zero V
		return zero, false
	}

	// Update access time atomically
	entry.accessTime.Store(time.Now().UnixNano())
	c.incrementHits()

	return entry.value, true
}

// Set stores a key-value pair in the cache.
func (c *AtomicCache[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, 0)
}

// SetWithTTL stores a key-value pair with TTL using RCU pattern.
func (c *AtomicCache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.incrementSets()

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	// Copy-on-write
	oldData := c.data.Load()
	newData := &atomicMap[K, V]{
		m: make(map[K]*atomicEntry[V], len(oldData.m)+1),
	}

	// Copy existing entries
	maps.Copy(newData.m, oldData.m)

	// Add/update entry
	entry := &atomicEntry[V]{
		value:      value,
		expiration: expiration,
	}
	entry.accessTime.Store(time.Now().UnixNano())

	if _, exists := newData.m[key]; !exists {
		currentSize := c.size.Add(1)

		// Evict if needed
		if int(currentSize) > c.capacity {
			c.evictLRU(newData)
			c.size.Add(-1)
			c.incrementEvictions()
		}
	}

	newData.m[key] = entry

	// Atomic swap
	c.data.Store(newData)
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

// Stats returns cache statistics.
func (c *AtomicCache[K, V]) Stats() *Stats {
	stats := c.stats.Load()
	if stats == nil {
		// Lazy initialization with CAS to ensure we always return the live stats
		newStats := NewStats()
		if c.stats.CompareAndSwap(nil, newStats) {
			return newStats
		}
		// Another goroutine initialized it, load again
		stats = c.stats.Load()
	}
	return stats
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

// incrementHits atomically increments the hit counter.
func (c *AtomicCache[K, V]) incrementHits() {
	// Ensure stats are initialized
	stats := c.Stats()
	stats.Hits.Add(1)
}

// incrementMisses atomically increments the miss counter.
func (c *AtomicCache[K, V]) incrementMisses() {
	// Ensure stats are initialized
	stats := c.Stats()
	stats.Misses.Add(1)
}

// incrementSets atomically increments the set counter.
func (c *AtomicCache[K, V]) incrementSets() {
	// Ensure stats are initialized
	stats := c.Stats()
	stats.Sets.Add(1)
}

// incrementEvictions atomically increments the eviction counter.
func (c *AtomicCache[K, V]) incrementEvictions() {
	// Ensure stats are initialized
	stats := c.Stats()
	stats.Evictions.Add(1)
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
				c.incrementHits()
			} else {
				c.incrementMisses()
			}
		} else {
			c.incrementMisses()
		}
	}

	return result
}

// BatchSet stores multiple key-value pairs in a single operation.
func (c *AtomicCache[K, V]) BatchSet(items map[K]V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldData := c.data.Load()
	newData := &atomicMap[K, V]{
		m: make(map[K]*atomicEntry[V], len(oldData.m)+len(items)),
	}

	// Copy existing entries
	maps.Copy(newData.m, oldData.m)

	now := time.Now().UnixNano()

	// Add new entries
	for k, v := range items {
		entry := &atomicEntry[V]{
			value: v,
		}
		entry.accessTime.Store(now)
		newData.m[k] = entry
		c.incrementSets()
	}

	// Evict if needed
	newSize := len(newData.m)
	for newSize > c.capacity {
		c.evictLRU(newData)
		newSize--
		c.incrementEvictions()
	}

	c.size.Store(int32(newSize))
	c.data.Store(newData)
}
