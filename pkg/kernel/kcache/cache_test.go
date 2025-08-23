package kcache_test

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/kitsunium/sdk/pkg/kernel/kcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLRU_BasicOperations(t *testing.T) {
	c := kcache.NewLRU[string, int](3)

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	val, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, val)

	val, ok = c.Get("b")
	assert.True(t, ok)
	assert.Equal(t, 2, val)

	assert.Equal(t, 3, c.Size())
	assert.True(t, c.Has("a"))
	assert.True(t, c.Has("b"))
	assert.True(t, c.Has("c"))
	assert.False(t, c.Has("d"))
}

func TestLRU_Eviction(t *testing.T) {
	c := kcache.NewLRU[string, int](3)

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)
	c.Set("d", 4)

	assert.Equal(t, 3, c.Size())
	assert.False(t, c.Has("a"))
	assert.True(t, c.Has("b"))
	assert.True(t, c.Has("c"))
	assert.True(t, c.Has("d"))
}

func TestLRU_LRUOrder(t *testing.T) {
	c := kcache.NewLRU[string, int](3)

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	c.Get("a")

	c.Set("d", 4)

	assert.False(t, c.Has("b"))
	assert.True(t, c.Has("a"))
	assert.True(t, c.Has("c"))
	assert.True(t, c.Has("d"))
}

func TestLRU_Update(t *testing.T) {
	c := kcache.NewLRU[string, int](3)

	c.Set("a", 1)
	c.Set("a", 2)

	val, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 2, val)
	assert.Equal(t, 1, c.Size())
}

func TestLRU_Delete(t *testing.T) {
	c := kcache.NewLRU[string, int](3)

	c.Set("a", 1)
	c.Set("b", 2)

	c.Delete("a")

	assert.False(t, c.Has("a"))
	assert.True(t, c.Has("b"))
	assert.Equal(t, 1, c.Size())

	c.Delete("non-existent")
	assert.Equal(t, 1, c.Size())
}

func TestLRU_Clear(t *testing.T) {
	c := kcache.NewLRU[string, int](3)

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	c.Clear()

	assert.Equal(t, 0, c.Size())
	assert.False(t, c.Has("a"))
	assert.False(t, c.Has("b"))
	assert.False(t, c.Has("c"))
}

func TestLRU_TTL(t *testing.T) {
	c := kcache.NewLRU[string, int](3)

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
	c := kcache.NewLRU[string, int](0)

	c.Set("a", 1)

	val, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, val)
}

func TestLRU_Stats(t *testing.T) {
	c := kcache.NewLRU[string, int](2)

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
	c := kcache.NewLRU[int, int](100)
	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 1000

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := (id*numOperations + j) % 200
				c.Set(key, key*2)
				c.Get(key)
				if j%10 == 0 {
					c.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()
	assert.LessOrEqual(t, c.Size(), 100)
}

func TestLRU_ConcurrentEviction(t *testing.T) {
	c := kcache.NewLRU[int, int](10)
	var wg sync.WaitGroup
	numGoroutines := 50

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Set(id*100+j, id*100+j)
			}
		}(i)
	}

	wg.Wait()
	assert.Equal(t, 10, c.Size())
}

func TestLRU_DifferentTypes(t *testing.T) {
	t.Run("StringToStruct", func(t *testing.T) {
		type Person struct {
			Name string
			Age  int
		}

		c := kcache.NewLRU[string, Person](3)

		p1 := Person{Name: "Alice", Age: 30}
		p2 := Person{Name: "Bob", Age: 25}

		c.Set("alice", p1)
		c.Set("bob", p2)

		val, ok := c.Get("alice")
		assert.True(t, ok)
		assert.Equal(t, p1, val)
	})

	t.Run("IntToString", func(t *testing.T) {
		c := kcache.NewLRU[int, string](3)

		c.Set(1, "one")
		c.Set(2, "two")

		val, ok := c.Get(1)
		assert.True(t, ok)
		assert.Equal(t, "one", val)
	})

	t.Run("StructToInterface", func(t *testing.T) {
		type Key struct {
			ID   int
			Type string
		}

		c := kcache.NewLRU[Key, interface{}](3)

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
	c := kcache.NewLRU[int, []byte](100)

	data := make([]byte, 1024)

	for i := 0; i < 1000; i++ {
		c.Set(i, data)
	}

	runtime.GC()
	runtime.GC()

	assert.Equal(t, 100, c.Size())
}

func BenchmarkLRU_Set(b *testing.B) {
	c := kcache.NewLRU[int, int](1000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Set(i, i)
			i++
		}
	})
}

func BenchmarkLRU_Get(b *testing.B) {
	c := kcache.NewLRU[int, int](1000)
	for i := 0; i < 1000; i++ {
		c.Set(i, i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(i % 1000)
			i++
		}
	})
}

func BenchmarkLRU_SetWithEviction(b *testing.B) {
	c := kcache.NewLRU[int, int](100)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Set(i, i)
			i++
		}
	})
}

func BenchmarkLRU_GetMiss(b *testing.B) {
	c := kcache.NewLRU[int, int](1000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(i)
			i++
		}
	})
}

func BenchmarkLRU_SetTTL(b *testing.B) {
	c := kcache.NewLRU[int, int](1000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.SetWithTTL(i%1000, i, time.Minute)
			i++
		}
	})
}

func BenchmarkLRU_Delete(b *testing.B) {
	c := kcache.NewLRU[int, int](1000)
	for i := 0; i < 1000; i++ {
		c.Set(i, i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Delete(i % 1000)
			c.Set(i%1000, i)
			i++
		}
	})
}

func BenchmarkLRU_ConcurrentMixed(b *testing.B) {
	c := kcache.NewLRU[int, int](1000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 3 {
			case 0:
				c.Set(i%1000, i)
			case 1:
				c.Get(i % 1000)
			case 2:
				c.Delete(i % 1000)
			}
			i++
		}
	})
}

func TestLRU_Interface(t *testing.T) {
	var c kcache.Cache[string, int] = kcache.NewLRU[string, int](10)
	require.NotNil(t, c)

	c.Set("test", 42)
	val, ok := c.Get("test")
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}
