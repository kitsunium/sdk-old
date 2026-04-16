// Package cache_test provides black-box tests for the cache package.
package cache_test

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/kitsunium/sdk/internal/kernel/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLRU_BasicOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		key   string
		value int
		want  int
	}{
		{"get a", "a", 1, 1},
		{"get b", "b", 2, 2},
		{"get c", "c", 3, 3},
	}

	c := cache.NewLRU[string, int](3)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := c.Get(tt.key)
			assert.True(t, ok)
			assert.Equal(t, tt.want, val)
		})
	}

	assert.Equal(t, 3, c.Size())
	assert.True(t, c.Has("a"))
	assert.True(t, c.Has("b"))
	assert.True(t, c.Has("c"))
	assert.False(t, c.Has("d"))
}

func TestLRU_Eviction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		present bool
	}{
		{"a evicted", "a", false},
		{"b present", "b", true},
		{"c present", "c", true},
		{"d present", "d", true},
	}

	c := cache.NewLRU[string, int](3)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)
	c.Set("d", 4)

	assert.Equal(t, 3, c.Size())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.present, c.Has(tt.key))
		})
	}
}

func TestLRU_LRUOrder(t *testing.T) {
	t.Parallel()

	c := cache.NewLRU[string, int](3)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)
	c.Get("a") // promote a, b becomes LRU
	c.Set("d", 4)

	tests := []struct {
		name    string
		key     string
		present bool
	}{
		{"b evicted", "b", false},
		{"a present", "a", true},
		{"c present", "c", true},
		{"d present", "d", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.present, c.Has(tt.key))
		})
	}
}

func TestLRU_Update(t *testing.T) {
	t.Parallel()

	c := cache.NewLRU[string, int](3)
	c.Set("a", 1)
	c.Set("a", 2)

	val, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 2, val)
	assert.Equal(t, 1, c.Size())
}

func TestLRU_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		present bool
	}{
		{"a deleted", "a", false},
		{"b still present", "b", true},
	}

	c := cache.NewLRU[string, int](3)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Delete("a")

	assert.Equal(t, 1, c.Size())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.present, c.Has(tt.key))
		})
	}

	c.Delete("non-existent")
	assert.Equal(t, 1, c.Size())
}

func TestLRU_Clear(t *testing.T) {
	t.Parallel()

	c := cache.NewLRU[string, int](3)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)
	c.Clear()

	tests := []struct {
		name string
		key  string
	}{
		{"a cleared", "a"},
		{"b cleared", "b"},
		{"c cleared", "c"},
	}

	assert.Equal(t, 0, c.Size())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.False(t, c.Has(tt.key))
		})
	}
}

func TestLRU_TTL(t *testing.T) {
	t.Parallel()

	c := cache.NewLRU[string, int](3)
	c.SetWithTTL("a", 1, 50*time.Millisecond)
	c.SetWithTTL("b", 2, 200*time.Millisecond)
	c.Set("c", 3)

	val, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, val)

	time.Sleep(100 * time.Millisecond)

	_, ok = c.Get("a")
	assert.False(t, ok)

	val, ok = c.Get("b")
	assert.True(t, ok)
	assert.Equal(t, 2, val)

	val, ok = c.Get("c")
	assert.True(t, ok)
	assert.Equal(t, 3, val)

	time.Sleep(150 * time.Millisecond)

	_, ok = c.Get("b")
	assert.False(t, ok)

	val, ok = c.Get("c")
	assert.True(t, ok)
	assert.Equal(t, 3, val)
}

func TestLRU_ZeroCapacity(t *testing.T) {
	t.Parallel()

	c := cache.NewLRU[string, int](0)
	c.Set("a", 1)

	val, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, val)
}

func TestLRU_Stats(t *testing.T) {
	t.Parallel()

	c := cache.NewLRU[string, int](2)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Get("a")
	c.Get("c")
	c.Set("c", 3)

	stats := c.Stats()
	assert.NotNil(t, stats.Hits, "Stats.Hits should not be nil")
	assert.NotNil(t, stats.Misses, "Stats.Misses should not be nil")
	assert.NotNil(t, stats.Sets, "Stats.Sets should not be nil")
	assert.NotNil(t, stats.Evictions, "Stats.Evictions should not be nil")
	assert.Equal(t, uint64(1), stats.Hits.Load())
	assert.Equal(t, uint64(1), stats.Misses.Load())
	assert.Equal(t, uint64(3), stats.Sets.Load())
	assert.Equal(t, uint64(1), stats.Evictions.Load())
}

func TestLRU_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	c := cache.NewLRU[int, int](100)

	const (
		numGoroutines int = 100
		numOperations int = 1000
	)

	var wg sync.WaitGroup

	for i := range numGoroutines {
		id := i
		wg.Go(func() {
			for j := range numOperations {
				key := (id*numOperations + j) % 200
				c.Set(key, key*2)
				c.Get(key)
				if j%10 == 0 {
					c.Delete(key)
				}
			}
		})
	}

	wg.Wait()
	assert.LessOrEqual(t, c.Size(), 100)
}

func TestLRU_ConcurrentEviction(t *testing.T) {
	t.Parallel()

	c := cache.NewLRU[int, int](10)

	const numGoroutines int = 50

	var wg sync.WaitGroup

	for i := range numGoroutines {
		id := i
		wg.Go(func() {
			for j := range 100 {
				c.Set(id*100+j, id*100+j)
			}
		})
	}

	wg.Wait()
	assert.Equal(t, 10, c.Size())
}

func TestLRU_DifferentTypes(t *testing.T) {
	t.Parallel()

	t.Run("StringToStruct", func(t *testing.T) {
		t.Parallel()

		type Person struct {
			Name string
			Age  int
		}

		c := cache.NewLRU[string, Person](3)
		p1 := Person{Name: "Alice", Age: 30}
		p2 := Person{Name: "Bob", Age: 25}
		c.Set("alice", p1)
		c.Set("bob", p2)

		val, ok := c.Get("alice")
		assert.True(t, ok)
		assert.Equal(t, p1, val)
	})

	t.Run("IntToString", func(t *testing.T) {
		t.Parallel()

		c := cache.NewLRU[int, string](3)
		c.Set(1, "one")
		c.Set(2, "two")

		val, ok := c.Get(1)
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("StructToInterface", func(t *testing.T) {
		t.Parallel()

		type Key struct {
			ID   int
			Type string
		}

		c := cache.NewLRU[Key, any](3)
		k1 := Key{ID: 1, Type: "user"}
		k2 := Key{ID: 2, Type: "admin"}
		c.Set(k1, "user_data")
		c.Set(k2, 123)

		val, ok := c.Get(k1)
		assert.True(t, ok)
		assert.Equal(t, "user_data", val)

		val, ok = c.Get(k2)
		assert.True(t, ok)
		assert.Equal(t, 123, val)
	})
}

func TestLRU_MemoryRecycle(t *testing.T) {
	t.Parallel()

	c := cache.NewLRU[int, []byte](100)
	data := make([]byte, 1024)

	for i := range 1000 {
		c.Set(i, data)
	}

	runtime.GC()
	runtime.GC()

	assert.Equal(t, 100, c.Size())
}

func TestLRU_Interface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		key   string
		value int
	}{
		{"set and get test", "test", 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := cache.NewLRU[string, int](10)
			var iface cache.Cache[string, int] = c
			require.NotNil(t, iface)

			iface.Set(tt.key, tt.value)
			val, ok := iface.Get(tt.key)
			assert.True(t, ok)
			assert.Equal(t, tt.value, val)
		})
	}
}
