// Package cache_test provides black-box tests for the sharded cache implementation.
package cache_test

import (
	"sync"
	"testing"
	"time"

	"github.com/kitsunium/sdk/v1/internal/core/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShardedLRU_NewShardedLRU(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
			c := cache.NewShardedLRU[string, int](tt.capacity, tt.numShards)
			assert.NotNil(t, c)
			assert.Equal(t, 0, c.Size())
		})
	}
}

func TestShardedLRU_BasicOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		value   int
		present bool
	}{
		{"key1 present", "key1", 100, true},
		{"key2 present", "key2", 200, true},
		{"missing absent", "missing", 0, false},
	}

	c := cache.NewShardedLRU[string, int](100, 16)
	c.Set("key1", 100)
	c.Set("key2", 200)

	assert.Equal(t, 2, c.Size())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := c.Get(tt.key)
			assert.Equal(t, tt.present, ok)
			if tt.present {
				assert.Equal(t, tt.value, val)
			}
		})
	}

	assert.True(t, c.Has("key1"))
	assert.False(t, c.Has("missing"))
}

func TestShardedLRU_SetWithTTL(t *testing.T) {
	t.Parallel()

	c := cache.NewShardedLRU[string, int](100, 16)
	c.SetWithTTL("ttl1", 100, 50*time.Millisecond)
	c.SetWithTTL("ttl2", 200, 200*time.Millisecond)
	c.Set("permanent", 300)

	val, ok := c.Get("ttl1")
	assert.True(t, ok)
	assert.Equal(t, 100, val)

	time.Sleep(100 * time.Millisecond)

	_, ok = c.Get("ttl1")
	assert.False(t, ok)

	val, ok = c.Get("ttl2")
	assert.True(t, ok)
	assert.Equal(t, 200, val)

	val, ok = c.Get("permanent")
	assert.True(t, ok)
	assert.Equal(t, 300, val)

	time.Sleep(150 * time.Millisecond)

	_, ok = c.Get("ttl2")
	assert.False(t, ok)

	val, ok = c.Get("permanent")
	assert.True(t, ok)
	assert.Equal(t, 300, val)
}

func TestShardedLRU_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value int
		want  int
	}{
		{"update key", 200, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := cache.NewShardedLRU[string, int](100, 16)
			c.Set("key", 100)
			assert.Equal(t, 1, c.Size())
			c.Set("key", tt.value)
			assert.Equal(t, 1, c.Size())

			val, ok := c.Get("key")
			assert.True(t, ok)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestShardedLRU_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		present bool
	}{
		{"key1 deleted", "key1", false},
		{"key2 still present", "key2", true},
	}

	c := cache.NewShardedLRU[string, int](100, 16)
	c.Set("key1", 100)
	c.Set("key2", 200)
	assert.Equal(t, 2, c.Size())
	c.Delete("key1")
	assert.Equal(t, 1, c.Size())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.present, c.Has(tt.key))
		})
	}

	c.Delete("missing")
	assert.Equal(t, 1, c.Size())
}

func TestShardedLRU_Clear(t *testing.T) {
	t.Parallel()

	c := cache.NewShardedLRU[string, int](100, 16)

	for i := range 50 {
		c.Set(string(rune(i)), i)
	}
	assert.Equal(t, 50, c.Size())
	c.Clear()
	assert.Equal(t, 0, c.Size())

	for i := range 50 {
		assert.False(t, c.Has(string(rune(i))))
	}
}

func TestShardedLRU_Eviction(t *testing.T) {
	t.Parallel()

	c := cache.NewShardedLRU[int, int](10, 4)

	for i := range 20 {
		c.Set(i, i*10)
	}

	assert.LessOrEqual(t, c.Size(), 10)

	val, ok := c.Get(19)
	assert.True(t, ok)
	assert.Equal(t, 190, val)
}

func TestShardedLRU_Stats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantHits uint64
		wantMiss uint64
		wantSets uint64
	}{
		{"stats after ops", 2, 1, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := cache.NewShardedLRU[string, int](100, 16)
			c.Set("key1", 100)
			c.Set("key2", 200)
			c.Get("key1") // hit
			c.Get("key2") // hit
			c.Get("miss") // miss

			stats := c.Stats()
			assert.NotNil(t, stats.Hits, "Stats.Hits should not be nil")
			assert.NotNil(t, stats.Misses, "Stats.Misses should not be nil")
			assert.NotNil(t, stats.Sets, "Stats.Sets should not be nil")
			assert.NotNil(t, stats.Evictions, "Stats.Evictions should not be nil")
			assert.Equal(t, tt.wantHits, stats.Hits.Load())
			assert.Equal(t, tt.wantMiss, stats.Misses.Load())
			assert.Equal(t, tt.wantSets, stats.Sets.Load())
		})
	}
}

func TestShardedLRU_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	c := cache.NewShardedLRU[int, int](1000, 64)

	const (
		numGoroutines int = 100
		numOperations int = 1000
	)

	var wg sync.WaitGroup

	for i := range numGoroutines {
		id := i
		wg.Go(func() {
			for j := range numOperations {
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
		})
	}

	wg.Wait()
	assert.LessOrEqual(t, c.Size(), 1000)
}

func TestShardedLRU_HasWithExpiredTTL(t *testing.T) {
	t.Parallel()

	c := cache.NewShardedLRU[string, int](100, 16)
	c.SetWithTTL("expiring", 100, 50*time.Millisecond)
	assert.True(t, c.Has("expiring"))
	time.Sleep(100 * time.Millisecond)
	assert.False(t, c.Has("expiring"))
}

func TestShardedLRU_DifferentKeyTypes(t *testing.T) {
	t.Parallel()

	t.Run("IntKeys", func(t *testing.T) {
		t.Parallel()
		c := cache.NewShardedLRU[int, string](100, 16)
		c.Set(1, "one")
		c.Set(2, "two")
		val, ok := c.Get(1)
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("Int32Keys", func(t *testing.T) {
		t.Parallel()
		c := cache.NewShardedLRU[int32, string](100, 16)
		c.Set(int32(1), "one")
		val, ok := c.Get(int32(1))
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("Int64Keys", func(t *testing.T) {
		t.Parallel()
		c := cache.NewShardedLRU[int64, string](100, 16)
		c.Set(int64(1), "one")
		val, ok := c.Get(int64(1))
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("UintKeys", func(t *testing.T) {
		t.Parallel()
		c := cache.NewShardedLRU[uint, string](100, 16)
		c.Set(uint(1), "one")
		val, ok := c.Get(uint(1))
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("Uint32Keys", func(t *testing.T) {
		t.Parallel()
		c := cache.NewShardedLRU[uint32, string](100, 16)
		c.Set(uint32(1), "one")
		val, ok := c.Get(uint32(1))
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("Uint64Keys", func(t *testing.T) {
		t.Parallel()
		c := cache.NewShardedLRU[uint64, string](100, 16)
		c.Set(uint64(1), "one")
		val, ok := c.Get(uint64(1))
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("StringKeys", func(t *testing.T) {
		t.Parallel()
		c := cache.NewShardedLRU[string, int](100, 16)
		c.Set("key", 100)
		val, ok := c.Get("key")
		assert.True(t, ok)
		assert.Equal(t, 100, val)
	})

	t.Run("StructKeys", func(t *testing.T) {
		t.Parallel()

		type Key struct {
			ID   int
			Name string
		}

		c := cache.NewShardedLRU[Key, string](100, 16)
		k := Key{ID: 1, Name: "test"}
		c.Set(k, "value")
		val, ok := c.Get(k)
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})
}

func TestShardedLRU_Interface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		key   string
		value int
	}{
		{"interface set and get", "test", 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := cache.NewShardedLRU[string, int](100, 16)
			var iface cache.Cache[string, int] = c
			require.NotNil(t, iface)

			iface.Set(tt.key, tt.value)
			val, ok := iface.Get(tt.key)
			assert.True(t, ok)
			assert.Equal(t, tt.value, val)
		})
	}
}

func TestShardedLRU_PowerOfTwo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input int
	}{
		{"zero default", 0},
		{"already power of 2 one", 1},
		{"already power of 2 two", 2},
		{"round up three", 3},
		{"round up five", 5},
		{"round up hundred", 100},
		{"round up two hundred", 200},
		{"round up thousand", 1000},
		{"max shards", 2000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := cache.NewShardedLRU[int, int](100, tt.input)
			for i := range 50 {
				c.Set(i, i)
			}
			assert.Equal(t, 50, c.Size())
		})
	}
}

func TestShardedLRU_SmallCapacity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		capacity int
		shards   int
		inserts  int
		maxSize  int
	}{
		{"10 cap 256 shards", 10, 256, 1000, 256},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := cache.NewShardedLRU[int, int](tt.capacity, tt.shards)
			for i := range tt.inserts {
				c.Set(i, i)
			}
			assert.LessOrEqual(t, c.Size(), tt.maxSize)
		})
	}
}
