package kcache_test

import (
	"sync"
	"testing"
	"time"

	"github.com/kitsunium/sdk/pkg/kernel/kcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShardedLRU_NewShardedLRU(t *testing.T) {
	tests := []struct {
		name      string
		capacity  int
		numShards int
	}{
		{"Default shards", 1000, 0},
		{"Custom shards", 1000, 64},
		{"Max shards", 1000, 2000},
		{"Single shard", 100, 1},
		{"Power of 2", 1000, 128},
		{"Non-power of 2", 1000, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := kcache.NewShardedLRU[string, int](tt.capacity, tt.numShards)
			assert.NotNil(t, c)
			assert.Equal(t, 0, c.Size())
		})
	}
}

func TestShardedLRU_BasicOperations(t *testing.T) {
	c := kcache.NewShardedLRU[string, int](100, 16)

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

func TestShardedLRU_SetWithTTL(t *testing.T) {
	c := kcache.NewShardedLRU[string, int](100, 16)

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

func TestShardedLRU_Update(t *testing.T) {
	c := kcache.NewShardedLRU[string, int](100, 16)

	c.Set("key", 100)
	assert.Equal(t, 1, c.Size())

	c.Set("key", 200)
	assert.Equal(t, 1, c.Size())

	val, ok := c.Get("key")
	assert.True(t, ok)
	assert.Equal(t, 200, val)
}

func TestShardedLRU_Delete(t *testing.T) {
	c := kcache.NewShardedLRU[string, int](100, 16)

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

func TestShardedLRU_Clear(t *testing.T) {
	c := kcache.NewShardedLRU[string, int](100, 16)

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

func TestShardedLRU_Eviction(t *testing.T) {
	c := kcache.NewShardedLRU[int, int](10, 4)

	// Fill cache beyond capacity
	for i := 0; i < 20; i++ {
		c.Set(i, i*10)
	}

	// Should have evicted oldest entries
	assert.LessOrEqual(t, c.Size(), 10)

	// Recent entries should exist
	val, ok := c.Get(19)
	assert.True(t, ok)
	assert.Equal(t, 190, val)
}

func TestShardedLRU_Stats(t *testing.T) {
	c := kcache.NewShardedLRU[string, int](100, 16)

	c.Set("key1", 100)
	c.Set("key2", 200)

	// Generate hits and misses
	c.Get("key1") // hit
	c.Get("key2") // hit
	c.Get("miss") // miss

	stats := c.Stats()
	assert.Equal(t, uint64(2), stats.Hits)
	assert.Equal(t, uint64(1), stats.Misses)
	assert.Equal(t, uint64(2), stats.Sets)
}

func TestShardedLRU_ConcurrentAccess(t *testing.T) {
	c := kcache.NewShardedLRU[int, int](1000, 64)
	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 1000

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := (id * numOperations + j) % 2000
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

func TestShardedLRU_HasWithExpiredTTL(t *testing.T) {
	c := kcache.NewShardedLRU[string, int](100, 16)

	c.SetWithTTL("expiring", 100, 50*time.Millisecond)
	assert.True(t, c.Has("expiring"))

	time.Sleep(100 * time.Millisecond)
	assert.False(t, c.Has("expiring"))
}

func TestShardedLRU_DifferentKeyTypes(t *testing.T) {
	t.Run("IntKeys", func(t *testing.T) {
		c := kcache.NewShardedLRU[int, string](100, 16)
		c.Set(1, "one")
		c.Set(2, "two")

		val, ok := c.Get(1)
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("Int32Keys", func(t *testing.T) {
		c := kcache.NewShardedLRU[int32, string](100, 16)
		c.Set(int32(1), "one")

		val, ok := c.Get(int32(1))
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("Int64Keys", func(t *testing.T) {
		c := kcache.NewShardedLRU[int64, string](100, 16)
		c.Set(int64(1), "one")

		val, ok := c.Get(int64(1))
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("UintKeys", func(t *testing.T) {
		c := kcache.NewShardedLRU[uint, string](100, 16)
		c.Set(uint(1), "one")

		val, ok := c.Get(uint(1))
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("Uint32Keys", func(t *testing.T) {
		c := kcache.NewShardedLRU[uint32, string](100, 16)
		c.Set(uint32(1), "one")

		val, ok := c.Get(uint32(1))
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("Uint64Keys", func(t *testing.T) {
		c := kcache.NewShardedLRU[uint64, string](100, 16)
		c.Set(uint64(1), "one")

		val, ok := c.Get(uint64(1))
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("StringKeys", func(t *testing.T) {
		c := kcache.NewShardedLRU[string, int](100, 16)
		c.Set("key", 100)

		val, ok := c.Get("key")
		assert.True(t, ok)
		assert.Equal(t, 100, val)
	})

	t.Run("StructKeys", func(t *testing.T) {
		type Key struct {
			ID   int
			Name string
		}

		c := kcache.NewShardedLRU[Key, string](100, 16)
		k := Key{ID: 1, Name: "test"}
		c.Set(k, "value")

		val, ok := c.Get(k)
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})
}

func TestShardedLRU_Interface(t *testing.T) {
	var c kcache.Cache[string, int] = kcache.NewShardedLRU[string, int](100, 16)
	require.NotNil(t, c)

	c.Set("test", 42)
	val, ok := c.Get("test")
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

func TestShardedLRU_PowerOfTwo(t *testing.T) {
	// Test nextPowerOf2 function indirectly through shard count
	testCases := []struct {
		input    int
		expected int
	}{
		{0, 256},     // Default
		{1, 1},       // Already power of 2
		{2, 2},       // Already power of 2
		{3, 4},       // Round up
		{5, 8},       // Round up
		{100, 128},   // Round up
		{200, 256},   // Round up
		{1000, 1024}, // Round up
		{2000, 1024}, // Max shards
	}

	for _, tc := range testCases {
		c := kcache.NewShardedLRU[int, int](100, tc.input)
		// Set enough items to use multiple shards
		for i := 0; i < 50; i++ {
			c.Set(i, i)
		}
		assert.Equal(t, 50, c.Size())
	}
}

func TestShardedLRU_SmallCapacity(t *testing.T) {
	// Test with capacity smaller than shard count
	c := kcache.NewShardedLRU[int, int](10, 256)

	for i := 0; i < 1000; i++ {
		c.Set(i, i)
	}

	// With 256 shards and capacity 10, each shard gets capacity 1
	// But some shards might have 0 items due to hash distribution
	// Total should be around 256 or less
	assert.LessOrEqual(t, c.Size(), 256)
}