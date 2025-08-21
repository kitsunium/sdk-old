package kcache

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	// DefaultShards is the default number of shards (should be power of 2).
	DefaultShards = 256
	// MaxShards limits the maximum number of shards.
	MaxShards = 1024
)

// ShardedLRU is a sharded LRU cache that reduces lock contention
// by distributing entries across multiple independent LRU caches.
//
// Type parameters:
//   - K: The type of keys (must be comparable)
//   - V: The type of values (can be any type)
type ShardedLRU[K comparable, V any] struct {
	shards    []*shard[K, V]
	shardMask uint32
	hashFn    func(K) uint32
}

type shard[K comparable, V any] struct {
	cache *FastLRU[K, V]
	mu    sync.RWMutex
}

// FastLRU is an LRU implementation with reduced locking.
type FastLRU[K comparable, V any] struct {
	capacity int
	items    map[K]*fastEntry[V]
	head     *fastEntry[V]
	tail     *fastEntry[V]
	stats    atomic.Pointer[Stats]
}

type fastEntry[V any] struct {
	value      V
	expiration int64
	next       *fastEntry[V]
	prev       *fastEntry[V]
	key        unsafe.Pointer
}

// NewShardedLRU creates a new sharded LRU cache.
func NewShardedLRU[K comparable, V any](capacity int, numShards int) *ShardedLRU[K, V] {
	if numShards <= 0 {
		numShards = DefaultShards
	}
	if numShards > MaxShards {
		numShards = MaxShards
	}

	// Round up to power of 2 for fast modulo
	numShards = nextPowerOf2(numShards)

	perShardCapacity := capacity / numShards
	if perShardCapacity < 1 {
		perShardCapacity = 1
	}

	c := &ShardedLRU[K, V]{
		shards:    make([]*shard[K, V], numShards),
		shardMask: uint32(numShards - 1),
		hashFn:    hashKey[K],
	}

	for i := 0; i < numShards; i++ {
		c.shards[i] = &shard[K, V]{
			cache: newFastLRU[K, V](perShardCapacity),
		}
	}

	return c
}

func newFastLRU[K comparable, V any](capacity int) *FastLRU[K, V] {
	lru := &FastLRU[K, V]{
		capacity: capacity,
		items:    make(map[K]*fastEntry[V], capacity),
	}

	// Initialize sentinel nodes
	lru.head = &fastEntry[V]{}
	lru.tail = &fastEntry[V]{}
	lru.head.next = lru.tail
	lru.tail.prev = lru.head

	stats := &Stats{}
	lru.stats.Store(stats)

	return lru
}

// Get retrieves a value from the sharded cache.
func (c *ShardedLRU[K, V]) Get(key K) (V, bool) {
	shard := c.getShard(key)

	// Try read lock first for better performance
	shard.mu.RLock()
	entry, exists := shard.cache.items[key]
	if !exists {
		shard.mu.RUnlock()
		c.incrementMisses(shard)
		var zero V
		return zero, false
	}

	// Check expiration without upgrading lock
	if entry.expiration > 0 && time.Now().UnixNano() > entry.expiration {
		shard.mu.RUnlock()
		// Need write lock to remove expired entry
		shard.mu.Lock()
		shard.cache.removeEntry(entry)
		delete(shard.cache.items, key)
		shard.mu.Unlock()
		c.incrementMisses(shard)
		var zero V
		return zero, false
	}

	value := entry.value
	shard.mu.RUnlock()

	// Move to front with write lock
	shard.mu.Lock()
	shard.cache.moveToFront(entry)
	shard.mu.Unlock()

	c.incrementHits(shard)
	return value, true
}

// Set stores a key-value pair in the sharded cache.
func (c *ShardedLRU[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, 0)
}

// SetWithTTL stores a key-value pair with TTL in the sharded cache.
func (c *ShardedLRU[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	c.incrementSets(shard)

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	if entry, exists := shard.cache.items[key]; exists {
		entry.value = value
		entry.expiration = expiration
		shard.cache.moveToFront(entry)
		return
	}

	entry := &fastEntry[V]{
		value:      value,
		expiration: expiration,
	}

	keyPtr := unsafe.Pointer(&key)
	entry.key = keyPtr

	shard.cache.items[key] = entry
	shard.cache.addToFront(entry)

	if len(shard.cache.items) > shard.cache.capacity {
		oldest := shard.cache.tail.prev
		if oldest != nil && oldest != shard.cache.head {
			shard.cache.removeEntry(oldest)

			// Reconstruct key from unsafe pointer
			if oldest.key != nil {
				oldKey := *(*K)(oldest.key)
				delete(shard.cache.items, oldKey)
				c.incrementEvictions(shard)
			}
		}
	}
}

// Delete removes a key from the sharded cache.
func (c *ShardedLRU[K, V]) Delete(key K) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if entry, exists := shard.cache.items[key]; exists {
		shard.cache.removeEntry(entry)
		delete(shard.cache.items, key)
	}
}

// Clear removes all entries from the sharded cache.
func (c *ShardedLRU[K, V]) Clear() {
	for _, shard := range c.shards {
		shard.mu.Lock()
		shard.cache.items = make(map[K]*fastEntry[V], shard.cache.capacity)
		shard.cache.head.next = shard.cache.tail
		shard.cache.tail.prev = shard.cache.head
		shard.mu.Unlock()
	}
}

// Size returns the total number of entries across all shards.
func (c *ShardedLRU[K, V]) Size() int {
	size := 0
	for _, shard := range c.shards {
		shard.mu.RLock()
		size += len(shard.cache.items)
		shard.mu.RUnlock()
	}
	return size
}

// Has checks if a key exists in the sharded cache.
func (c *ShardedLRU[K, V]) Has(key K) bool {
	shard := c.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	entry, exists := shard.cache.items[key]
	if !exists {
		return false
	}

	if entry.expiration > 0 && time.Now().UnixNano() > entry.expiration {
		return false
	}

	return true
}

// Stats returns aggregated statistics from all shards.
func (c *ShardedLRU[K, V]) Stats() Stats {
	aggregated := Stats{}
	for _, shard := range c.shards {
		stats := shard.cache.stats.Load()
		if stats != nil {
			aggregated.Hits.Add(stats.Hits.Load())
			aggregated.Misses.Add(stats.Misses.Load())
			aggregated.Sets.Add(stats.Sets.Load())
			aggregated.Evictions.Add(stats.Evictions.Load())
		}
	}
	return aggregated
}

// getShard returns the shard responsible for the given key.
// It uses bit masking for modulo operation.
func (c *ShardedLRU[K, V]) getShard(key K) *shard[K, V] {
	hash := c.hashFn(key)
	return c.shards[hash&c.shardMask]
}

// addToFront adds an entry to the front of the LRU list.
func (c *FastLRU[K, V]) addToFront(e *fastEntry[V]) {
	e.next = c.head.next
	e.prev = c.head
	c.head.next.prev = e
	c.head.next = e
}

// removeEntry removes an entry from the LRU list.
func (c *FastLRU[K, V]) removeEntry(e *fastEntry[V]) {
	e.prev.next = e.next
	e.next.prev = e.prev
}

// moveToFront moves an entry to the front of the LRU list.
func (c *FastLRU[K, V]) moveToFront(e *fastEntry[V]) {
	c.removeEntry(e)
	c.addToFront(e)
}

// incrementHits atomically increments the hit counter for a shard.
func (c *ShardedLRU[K, V]) incrementHits(s *shard[K, V]) {
	stats := s.cache.stats.Load()
	if stats != nil {
		stats.Hits.Add(1)
	}
}

// incrementMisses atomically increments the miss counter for a shard.
func (c *ShardedLRU[K, V]) incrementMisses(s *shard[K, V]) {
	stats := s.cache.stats.Load()
	if stats != nil {
		stats.Misses.Add(1)
	}
}

// incrementSets atomically increments the set counter for a shard.
func (c *ShardedLRU[K, V]) incrementSets(s *shard[K, V]) {
	stats := s.cache.stats.Load()
	if stats != nil {
		stats.Sets.Add(1)
	}
}

// incrementEvictions atomically increments the eviction counter for a shard.
func (c *ShardedLRU[K, V]) incrementEvictions(s *shard[K, V]) {
	stats := s.cache.stats.Load()
	if stats != nil {
		stats.Evictions.Add(1)
	}
}

// hashKey computes a hash for different key types.
// It uses specific hash functions for common types and falls back
// to string representation for complex types.
func hashKey[K comparable](key K) uint32 {
	switch k := any(key).(type) {
	case string:
		return fnvHash([]byte(k))
	case int:
		return uint32(k)
	case int8:
		return uint32(k)
	case int16:
		return uint32(k)
	case int32:
		return uint32(k)
	case int64:
		return uint32(k ^ (k >> 32))
	case uint:
		return uint32(k)
	case uint8:
		return uint32(k)
	case uint16:
		return uint32(k)
	case uint32:
		return k
	case uint64:
		return uint32(k ^ (k >> 32))
	case float32:
		// Convert float32 bits to uint32 for hashing
		return *(*uint32)(unsafe.Pointer(&k))
	case float64:
		// Convert float64 bits to uint64 then to uint32
		bits := *(*uint64)(unsafe.Pointer(&k))
		return uint32(bits ^ (bits >> 32))
	case bool:
		if k {
			return 1
		}
		return 0
	default:
		// For structs and other complex types, use fmt.Sprintf
		// This ensures consistent hashing based on content, not memory address
		data := fmt.Sprintf("%v", key)
		return fnvHash([]byte(data))
	}
}

// fnvHash implements FNV-1a hash algorithm for string hashing.
// FNV-1a provides good distribution.
func fnvHash(data []byte) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	hash := uint32(offset32)
	for _, b := range data {
		hash ^= uint32(b)
		hash *= prime32
	}
	return hash
}

// nextPowerOf2 rounds up to the next power of 2.
// For example: 3 -> 4, 5 -> 8, 100 -> 128.
// This is used for modulo operations using bit masking.
func nextPowerOf2(n int) int {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

// Keys returns all keys in the cache across all shards.
func (c *ShardedLRU[K, V]) Keys() []K {
	var keys []K
	for _, shard := range c.shards {
		shard.mu.RLock()
		for k := range shard.cache.items {
			keys = append(keys, k)
		}
		shard.mu.RUnlock()
	}
	return keys
}

// Range calls f for each key-value pair in the cache.
// If f returns false, iteration stops.
func (c *ShardedLRU[K, V]) Range(f func(key K, value V) bool) {
	for _, shard := range c.shards {
		shard.mu.RLock()
		shouldContinue := true
		for k, entry := range shard.cache.items {
			if entry.expiration == 0 || time.Now().UnixNano() <= entry.expiration {
				if !f(k, entry.value) {
					shouldContinue = false
					break
				}
			}
		}
		shard.mu.RUnlock()
		if !shouldContinue {
			break
		}
	}
}

// Ensure ShardedLRU implements Cache interface.
var _ Cache[string, any] = (*ShardedLRU[string, any])(nil)
