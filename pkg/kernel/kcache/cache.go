// Package kcache provides thread-safe caching implementations.
// It includes a generic LRU (Least Recently Used) cache with TTL support.
package kcache

import (
	"sync"
	"time"
)

// Cache defines the interface for cache implementations.
// It provides a generic interface for storing and retrieving
// key-value pairs with optional TTL support.
//
// Type parameters:
//   - K: The type of keys (must be comparable)
//   - V: The type of values (can be any type)
type Cache[K comparable, V any] interface {
	// Get retrieves a value from the cache by its key.
	// Returns the value and true if found, or zero value and false if not found or expired.
	Get(key K) (V, bool)

	// Set stores a key-value pair in the cache without expiration.
	Set(key K, value V)

	// SetWithTTL stores a key-value pair in the cache with a TTL (Time To Live).
	// The entry will be automatically removed after the specified duration.
	SetWithTTL(key K, value V, ttl time.Duration)

	// Delete removes a key-value pair from the cache.
	Delete(key K)

	// Clear removes all entries from the cache.
	Clear()

	// Size returns the current number of entries in the cache.
	Size() int

	// Has checks if a key exists in the cache without retrieving its value.
	// Returns false if the key doesn't exist or if the entry has expired.
	Has(key K) bool
}

// Stats holds cache statistics for monitoring.
type Stats struct {
	// Hits is the number of successful cache retrievals.
	Hits uint64
	// Misses is the number of failed cache retrievals.
	Misses uint64
	// Sets is the number of cache insertions.
	Sets uint64
	// Evictions is the number of entries removed due to capacity limits.
	Evictions uint64
}

type entry[V any] struct {
	value      V
	expiration int64
	next       *entry[V]
	prev       *entry[V]
	key        any
}

// LRU implements a Least Recently Used cache with optional TTL support.
// It is thread-safe and uses a doubly-linked list.
// The cache automatically evicts the least recently used items when capacity is reached.
//
// Type parameters:
//   - K: The type of keys (must be comparable)
//   - V: The type of values (can be any type)
//
// Example:
//
//	cache := NewLRU[string, int](100)
//	cache.Set("key", 42)
//	if val, ok := cache.Get("key"); ok {
//	    fmt.Println(val) // Output: 42
//	}
type LRU[K comparable, V any] struct {
	capacity int
	size     int
	items    map[K]*entry[V]
	head     *entry[V]
	tail     *entry[V]
	mu       sync.RWMutex
	stats    Stats
	pool     *sync.Pool
}

// NewLRU creates a new LRU cache with the specified capacity.
// If capacity is <= 0, it defaults to 128.
//
// Parameters:
//   - capacity: Maximum number of entries the cache can hold
//
// Returns:
//   - *LRU[K, V]: A new LRU cache instance
//
// Example:
//
//	cache := NewLRU[string, interface{}](1000)
func NewLRU[K comparable, V any](capacity int) *LRU[K, V] {
	if capacity <= 0 {
		capacity = 128
	}

	lru := &LRU[K, V]{
		capacity: capacity,
		items:    make(map[K]*entry[V], capacity),
		pool: &sync.Pool{
			New: func() any {
				return &entry[V]{}
			},
		},
	}

	lru.head = &entry[V]{}
	lru.tail = &entry[V]{}
	lru.head.next = lru.tail
	lru.tail.prev = lru.head

	return lru
}

// Get retrieves a value from the cache by its key.
// If the key exists and hasn't expired, it moves the entry to the front (most recently used)
// and returns the value with true. Otherwise, returns zero value with false.
//
// Parameters:
//   - key: The key to retrieve
//
// Returns:
//   - V: The value associated with the key (or zero value if not found)
//   - bool: true if the key was found and valid, false otherwise
//
// This operation is O(1) in time complexity.
func (c *LRU[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, exists := c.items[key]
	if !exists {
		c.stats.Misses++
		var zero V
		return zero, false
	}

	if e.expiration > 0 && time.Now().UnixNano() > e.expiration {
		c.removeEntry(e)
		delete(c.items, key)
		c.size--
		c.stats.Misses++
		var zero V
		return zero, false
	}

	c.moveToFront(e)
	c.stats.Hits++
	return e.value, true
}

// Set stores a key-value pair in the cache without expiration.
// If the key already exists, it updates the value and moves it to the front.
// If the cache is at capacity, it evicts the least recently used entry.
//
// Parameters:
//   - key: The key to store
//   - value: The value to associate with the key
//
// This operation is O(1) in time complexity.
func (c *LRU[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, 0)
}

// SetWithTTL stores a key-value pair in the cache with a Time To Live.
// After the TTL expires, the entry will be considered invalid and removed on next access.
// If ttl is 0 or negative, the entry will not expire.
//
// Parameters:
//   - key: The key to store
//   - value: The value to associate with the key
//   - ttl: Time To Live duration for the entry
//
// Example:
//
//	cache.SetWithTTL("session", userData, 30*time.Minute)
//
// This operation is O(1) in time complexity.
func (c *LRU[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.Sets++

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	if e, exists := c.items[key]; exists {
		e.value = value
		e.expiration = expiration
		c.moveToFront(e)
		return
	}

	e := c.pool.Get().(*entry[V])
	e.value = value
	e.expiration = expiration
	e.key = key
	e.next = nil
	e.prev = nil

	c.items[key] = e
	c.addToFront(e)
	c.size++

	if c.size > c.capacity {
		oldest := c.tail.prev
		c.removeEntry(oldest)
		delete(c.items, oldest.key.(K))
		c.size--
		c.stats.Evictions++

		oldest.value = *new(V)
		oldest.expiration = 0
		oldest.key = nil
		oldest.next = nil
		oldest.prev = nil
		c.pool.Put(oldest)
	}
}

// Delete removes a key-value pair from the cache.
// If the key doesn't exist, this operation is a no-op.
//
// Parameters:
//   - key: The key to delete
//
// This operation is O(1) in time complexity.
func (c *LRU[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, exists := c.items[key]; exists {
		c.removeEntry(e)
		delete(c.items, key)
		c.size--

		e.value = *new(V)
		e.expiration = 0
		e.key = nil
		e.next = nil
		e.prev = nil
		c.pool.Put(e)
	}
}

// Clear removes all entries from the cache and resets it to an empty state.
// All cached entries are returned to the pool for reuse.
//
// This operation is O(n) where n is the number of entries.
func (c *LRU[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, e := range c.items {
		e.value = *new(V)
		e.expiration = 0
		e.key = nil
		e.next = nil
		e.prev = nil
		c.pool.Put(e)
		delete(c.items, k)
	}

	c.head.next = c.tail
	c.tail.prev = c.head
	c.size = 0
}

// Size returns the current number of entries in the cache.
// This count includes only valid (non-expired) entries.
//
// Returns:
//   - int: The number of entries currently in the cache
//
// This operation is O(1) in time complexity.
func (c *LRU[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

// Has checks if a key exists in the cache without retrieving its value.
// This method doesn't affect the LRU ordering.
// Returns false if the key doesn't exist or if the entry has expired.
//
// Parameters:
//   - key: The key to check
//
// Returns:
//   - bool: true if the key exists and is valid, false otherwise
//
// This operation is O(1) in time complexity.
func (c *LRU[K, V]) Has(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, exists := c.items[key]
	if !exists {
		return false
	}

	if e.expiration > 0 && time.Now().UnixNano() > e.expiration {
		return false
	}

	return true
}

// Stats returns the cache performance statistics.
// The statistics are not reset by this call and accumulate over the cache lifetime.
//
// Returns:
//   - Stats: Current cache statistics including hits, misses, sets, and evictions
//
// Example:
//
//	stats := cache.Stats()
//	hitRate := float64(stats.Hits) / float64(stats.Hits + stats.Misses)
//	fmt.Printf("Cache hit rate: %.2f%%\n", hitRate * 100)
func (c *LRU[K, V]) Stats() Stats {
	return c.stats
}

func (c *LRU[K, V]) addToFront(e *entry[V]) {
	e.next = c.head.next
	e.prev = c.head
	c.head.next.prev = e
	c.head.next = e
}

func (c *LRU[K, V]) removeEntry(e *entry[V]) {
	e.prev.next = e.next
	e.next.prev = e.prev
}

func (c *LRU[K, V]) moveToFront(e *entry[V]) {
	c.removeEntry(e)
	c.addToFront(e)
}
