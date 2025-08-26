package kcache_test

import (
	"sync"
	"testing"
	"time"

	"github.com/kitsunium/sdk/pkg/kernel/kcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtomicCache_NewAtomicCache(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
	}{
		{"Default capacity", 0},
		{"Custom capacity", 1000},
		{"Small capacity", 10},
		{"Large capacity", 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := kcache.NewAtomicCache[string, int](tt.capacity)
			assert.NotNil(t, c)
			assert.Equal(t, 0, c.Size())
		})
	}
}

func TestAtomicCache_BasicOperations(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](100)

	// Test Set and Get
	c.Set("key1", 100)
	c.Set("key2", 200)

	val, ok := c.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, 100, val)

	val, ok = c.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, 200, val)

	// Test missing key
	_, ok = c.Get("missing")
	assert.False(t, ok)

	// Test Size
	assert.Equal(t, 2, c.Size())

	// Test Has
	assert.True(t, c.Has("key1"))
	assert.False(t, c.Has("missing"))
}

func TestAtomicCache_SetWithTTL(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](100)

	// Set with TTL
	c.SetWithTTL("ttl1", 100, 50*time.Millisecond)
	c.SetWithTTL("ttl2", 200, 200*time.Millisecond)
	c.Set("permanent", 300)

	// Check immediately
	val, ok := c.Get("ttl1")
	assert.True(t, ok)
	assert.Equal(t, 100, val)

	// Wait for first TTL to expire
	time.Sleep(100 * time.Millisecond)

	_, ok = c.Get("ttl1")
	assert.False(t, ok)

	val, ok = c.Get("ttl2")
	assert.True(t, ok)
	assert.Equal(t, 200, val)

	val, ok = c.Get("permanent")
	assert.True(t, ok)
	assert.Equal(t, 300, val)

	// Wait for second TTL to expire
	time.Sleep(150 * time.Millisecond)

	_, ok = c.Get("ttl2")
	assert.False(t, ok)

	// Permanent should still exist
	val, ok = c.Get("permanent")
	assert.True(t, ok)
	assert.Equal(t, 300, val)
}

func TestAtomicCache_Update(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](100)

	c.Set("key", 100)
	assert.Equal(t, 1, c.Size())

	c.Set("key", 200)
	assert.Equal(t, 1, c.Size())

	val, ok := c.Get("key")
	assert.True(t, ok)
	assert.Equal(t, 200, val)
}

func TestAtomicCache_Delete(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](100)

	c.Set("key1", 100)
	c.Set("key2", 200)
	assert.Equal(t, 2, c.Size())

	c.Delete("key1")
	assert.Equal(t, 1, c.Size())
	assert.False(t, c.Has("key1"))
	assert.True(t, c.Has("key2"))

	// Delete non-existent key
	c.Delete("missing")
	assert.Equal(t, 1, c.Size())
}

func TestAtomicCache_Clear(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](100)

	for i := 0; i < 50; i++ {
		c.Set(string(rune(i)), i)
	}

	assert.Equal(t, 50, c.Size())

	c.Clear()
	assert.Equal(t, 0, c.Size())

	for i := 0; i < 50; i++ {
		assert.False(t, c.Has(string(rune(i))))
	}
}

func TestAtomicCache_Eviction(t *testing.T) {
	c := kcache.NewAtomicCache[int, int](10)

	// Fill cache beyond capacity
	for i := 0; i < 20; i++ {
		c.Set(i, i*10)
		// Small delay to ensure different access times
		time.Sleep(1 * time.Millisecond)
	}

	// Should have evicted oldest entries
	assert.LessOrEqual(t, c.Size(), 10)

	// Recent entries should exist
	val, ok := c.Get(19)
	assert.True(t, ok)
	assert.Equal(t, 190, val)
}

func TestAtomicCache_Stats(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](100)

	c.Set("key1", 100)
	c.Set("key2", 200)

	// Generate hits and misses
	c.Get("key1") // hit
	c.Get("key2") // hit
	c.Get("miss") // miss

	// Stats removed for performance optimization
}

func TestAtomicCache_FastGet(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](100)

	c.Set("key1", 100)
	c.SetWithTTL("ttl", 200, 50*time.Millisecond)

	// Test existing key
	valPtr, ok := c.FastGet("key1")
	assert.True(t, ok)
	assert.NotNil(t, valPtr)
	assert.Equal(t, 100, *valPtr)

	// Test missing key
	valPtr, ok = c.FastGet("missing")
	assert.False(t, ok)
	assert.Nil(t, valPtr)

	// Test expired key
	time.Sleep(100 * time.Millisecond)
	valPtr, ok = c.FastGet("ttl")
	assert.False(t, ok)
	assert.Nil(t, valPtr)
}

func TestAtomicCache_BatchGet(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](100)

	c.Set("key1", 100)
	c.Set("key2", 200)
	c.SetWithTTL("ttl", 300, 50*time.Millisecond)

	// Test batch get with existing and missing keys
	keys := []string{"key1", "key2", "missing", "ttl"}
	result := c.BatchGet(keys)

	assert.Len(t, result, 3)
	assert.Equal(t, 100, result["key1"])
	assert.Equal(t, 200, result["key2"])
	assert.Equal(t, 300, result["ttl"])
	_, hasMissing := result["missing"]
	assert.False(t, hasMissing)

	// Test with expired key
	time.Sleep(100 * time.Millisecond)
	result = c.BatchGet([]string{"key1", "ttl"})
	assert.Len(t, result, 1)
	assert.Equal(t, 100, result["key1"])
	_, hasTTL := result["ttl"]
	assert.False(t, hasTTL)
}

func TestAtomicCache_BatchSet(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](100)

	items := map[string]int{
		"key1": 100,
		"key2": 200,
		"key3": 300,
	}

	c.BatchSet(items)

	assert.Equal(t, 3, c.Size())

	for key, expectedVal := range items {
		val, ok := c.Get(key)
		assert.True(t, ok)
		assert.Equal(t, expectedVal, val)
	}
}

func TestAtomicCache_BatchSetWithEviction(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](5)

	// First set some items
	c.Set("old1", 1)
	c.Set("old2", 2)
	time.Sleep(10 * time.Millisecond)

	// Batch set that will trigger eviction
	items := map[string]int{
		"new1": 10,
		"new2": 20,
		"new3": 30,
		"new4": 40,
	}

	c.BatchSet(items)

	// Should have evicted oldest entries
	assert.LessOrEqual(t, c.Size(), 5)

	// New entries should exist
	for key, expectedVal := range items {
		val, ok := c.Get(key)
		assert.True(t, ok, "Key %s should exist", key)
		assert.Equal(t, expectedVal, val)
	}
}

func TestAtomicCache_ConcurrentAccess(t *testing.T) {
	c := kcache.NewAtomicCache[int, int](1000)
	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 1000

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := (id*numOperations + j) % 2000
				c.Set(key, key*2)
				val, ok := c.Get(key)
				if ok {
					assert.Equal(t, key*2, val)
				}
				if j%10 == 0 {
					c.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()
	assert.LessOrEqual(t, c.Size(), 1000)
}

func TestAtomicCache_ConcurrentBatch(t *testing.T) {
	c := kcache.NewAtomicCache[int, int](1000)
	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Batch set
			items := make(map[int]int)
			for j := 0; j < 100; j++ {
				key := id*100 + j
				items[key] = key * 2
			}
			c.BatchSet(items)

			// Batch get
			keys := make([]int, 100)
			for j := 0; j < 100; j++ {
				keys[j] = id*100 + j
			}
			result := c.BatchGet(keys)
			for key, val := range result {
				assert.Equal(t, key*2, val)
			}
		}(i)
	}

	wg.Wait()
}

func TestAtomicCache_HasWithExpiredTTL(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](100)

	c.SetWithTTL("expiring", 100, 50*time.Millisecond)
	assert.True(t, c.Has("expiring"))

	time.Sleep(100 * time.Millisecond)
	assert.False(t, c.Has("expiring"))
}

func TestAtomicCache_Interface(t *testing.T) {
	var c kcache.Cache[string, int] = kcache.NewAtomicCache[string, int](100)
	require.NotNil(t, c)

	c.Set("test", 42)
	val, ok := c.Get("test")
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

func TestAtomicCache_StatsRemoved(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](100)

	// Stats removed for performance
	// Operations should still work
	c.Set("key", 100)
	val, ok := c.Get("key")
	assert.True(t, ok)
	assert.Equal(t, 100, val)
}

func TestAtomicCache_EvictionWithExpired(t *testing.T) {
	c := kcache.NewAtomicCache[string, int](5)

	// Set items with mixed TTLs
	c.SetWithTTL("ttl1", 1, 50*time.Millisecond)
	c.Set("perm1", 10)
	c.Set("perm2", 20)
	c.Set("perm3", 30)

	// Sleep to ensure different access times
	time.Sleep(10 * time.Millisecond)

	// Access some items to update their access time
	c.Get("perm2")
	c.Get("perm3")

	// Add more items to trigger eviction
	c.Set("new1", 40)
	c.Set("new2", 50)

	// Should have evicted LRU items
	assert.LessOrEqual(t, c.Size(), 5)
}
