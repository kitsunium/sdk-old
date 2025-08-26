package kcache

import (
	"fmt"
	"sync"
	"time"
	"unsafe"
)

const (
	// DefaultShards is the default number of shards (should be power of 2).
	// Increased for better concurrency on multi-core systems
	DefaultShards = 512
	// MaxShards limits the maximum number of shards.
	MaxShards = 2048
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

	return lru
}

// Get retrieves a value from the sharded cache.
func (c *ShardedLRU[K, V]) Get(key K) (V, bool) {
	hash := c.hashFn(key)
	shard := c.shards[hash&c.shardMask]

	shard.mu.Lock()

	entry, exists := shard.cache.items[key]
	if !exists {
		shard.mu.Unlock()
		var zero V
		return zero, false
	}

	// Inline expiration check
	if entry.expiration > 0 {
		now := time.Now().UnixNano()
		if now > entry.expiration {
			// Inline removal
			entry.prev.next = entry.next
			entry.next.prev = entry.prev
			delete(shard.cache.items, key)
			shard.mu.Unlock()
			var zero V
			return zero, false
		}
	}

	// Inline moveToFront
	if entry.prev != shard.cache.head {
		entry.prev.next = entry.next
		entry.next.prev = entry.prev
		entry.next = shard.cache.head.next
		entry.prev = shard.cache.head
		shard.cache.head.next.prev = entry
		shard.cache.head.next = entry
	}

	value := entry.value
	shard.mu.Unlock()
	return value, true
}

// Set stores a key-value pair in the sharded cache.
func (c *ShardedLRU[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, 0)
}

// SetWithTTL stores a key-value pair with TTL in the sharded cache.
func (c *ShardedLRU[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	hash := c.hashFn(key)
	shard := c.shards[hash&c.shardMask]
	shard.mu.Lock()

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().UnixNano() + int64(ttl)
	}

	if entry, exists := shard.cache.items[key]; exists {
		entry.value = value
		entry.expiration = expiration
		// Inline moveToFront
		if entry.prev != shard.cache.head {
			entry.prev.next = entry.next
			entry.next.prev = entry.prev
			entry.next = shard.cache.head.next
			entry.prev = shard.cache.head
			shard.cache.head.next.prev = entry
			shard.cache.head.next = entry
		}
		shard.mu.Unlock()
		return
	}

	entry := &fastEntry[V]{
		value:      value,
		expiration: expiration,
		key:        unsafe.Pointer(&key),
	}

	shard.cache.items[key] = entry

	// Inline addToFront
	entry.next = shard.cache.head.next
	entry.prev = shard.cache.head
	shard.cache.head.next.prev = entry
	shard.cache.head.next = entry

	if len(shard.cache.items) > shard.cache.capacity {
		oldest := shard.cache.tail.prev
		if oldest != nil && oldest != shard.cache.head {
			// Inline removeEntry
			oldest.prev.next = oldest.next
			oldest.next.prev = oldest.prev

			if oldest.key != nil {
				oldKey := *(*K)(oldest.key)
				delete(shard.cache.items, oldKey)
			}
		}
	}

	shard.mu.Unlock()
}

// Delete removes a key from the sharded cache.
func (c *ShardedLRU[K, V]) Delete(key K) {
	hash := c.hashFn(key)
	shard := c.shards[hash&c.shardMask]
	shard.mu.Lock()

	entry, exists := shard.cache.items[key]
	if exists {
		// Inline removeEntry
		entry.prev.next = entry.next
		entry.next.prev = entry.prev
		delete(shard.cache.items, key)
	}

	shard.mu.Unlock()
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
	hash := c.hashFn(key)
	shard := c.shards[hash&c.shardMask]
	shard.mu.RLock()

	entry, exists := shard.cache.items[key]
	if !exists {
		shard.mu.RUnlock()
		return false
	}

	result := true
	if entry.expiration > 0 {
		result = time.Now().UnixNano() <= entry.expiration
	}

	shard.mu.RUnlock()
	return result
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

// hashKey computes a hash for different key types.
// Optimized for better distribution and performance.
func hashKey[K comparable](key K) uint32 {
	switch k := any(key).(type) {
	case string:
		return xxHash32([]byte(k))
	case int:
		return mix32(uint32(k))
	case int8:
		return mix32(uint32(k))
	case int16:
		return mix32(uint32(k))
	case int32:
		return mix32(uint32(k))
	case int64:
		return mix32(uint32(k ^ (k >> 32)))
	case uint:
		return mix32(uint32(k))
	case uint8:
		return mix32(uint32(k))
	case uint16:
		return mix32(uint32(k))
	case uint32:
		return mix32(k)
	case uint64:
		return mix32(uint32(k ^ (k >> 32)))
	case float32:
		return mix32(*(*uint32)(unsafe.Pointer(&k)))
	case float64:
		bits := *(*uint64)(unsafe.Pointer(&k))
		return mix32(uint32(bits ^ (bits >> 32)))
	case bool:
		if k {
			return 1
		}
		return 0
	default:
		// For structs and other complex types
		data := fmt.Sprintf("%v", key)
		return xxHash32([]byte(data))
	}
}

// xxHash32 is a faster hash function with better distribution
func xxHash32(data []byte) uint32 {
	const (
		prime1 = 0x9E3779B1
		prime2 = 0x85EBCA77
		prime3 = 0xC2B2AE3D
		prime4 = 0x27D4EB2F
		prime5 = 0x165667B1
	)

	h := uint32(len(data)) + prime5
	i := 0

	for len(data)-i >= 4 {
		k1 := uint32(data[i]) | uint32(data[i+1])<<8 | uint32(data[i+2])<<16 | uint32(data[i+3])<<24
		k1 *= prime2
		k1 = (k1 << 13) | (k1 >> 19)
		k1 *= prime1
		h ^= k1
		h = (h << 17) | (h >> 15)
		h = h*prime3 + prime4
		i += 4
	}

	for i < len(data) {
		h ^= uint32(data[i]) * prime5
		h = (h << 11) | (h >> 21)
		h *= prime1
		i++
	}

	h ^= h >> 15
	h *= prime2
	h ^= h >> 13
	h *= prime3
	h ^= h >> 16

	return h
}

// mix32 improves distribution of integer hashes
func mix32(h uint32) uint32 {
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
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
