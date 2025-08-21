package config_test

import (
	"testing"

	"github.com/kitsunium/sdk/pkg/kernel/config"
	"github.com/stretchr/testify/assert"
)

func TestConfig_TypedGetters(t *testing.T) {
	c := config.New()

	c.Set("string", "hello")
	c.Set("int", 42)
	c.Set("float", 3.14)
	c.Set("bool", true)
	c.Set("duration", "5m")

	assert.Equal(t, "hello", c.GetString("string", ""))
	assert.Equal(t, 42, c.GetInt("int", 0))
	assert.Equal(t, 3.14, c.GetFloat64("float", 0))
	assert.True(t, c.GetBool("bool", false))
	assert.Equal(t, "5m0s", c.GetDuration("duration", 0).String())

	assert.Equal(t, "default", c.GetString("missing", "default"))
	assert.Equal(t, 100, c.GetInt("missing", 100))
	assert.Equal(t, 1.5, c.GetFloat64("missing", 1.5))
	assert.False(t, c.GetBool("missing", false))
}

func TestConfig_StringConversions(t *testing.T) {
	c := config.New()

	c.Set("int_string", "123")
	c.Set("float_string", "3.14")
	c.Set("bool_string", "true")

	assert.Equal(t, 123, c.GetInt("int_string", 0))
	assert.Equal(t, 3.14, c.GetFloat64("float_string", 0))
	assert.True(t, c.GetBool("bool_string", false))
}

func TestConfig_Has(t *testing.T) {
	c := config.New()

	c.Set("exists", "value")

	assert.True(t, c.Has("exists"))
	assert.False(t, c.Has("not_exists"))
}

func TestConfig_Keys(t *testing.T) {
	c := config.New()

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	keys := c.Keys()
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
	assert.Contains(t, keys, "key3")
}

func TestConfig_Clear(t *testing.T) {
	c := config.New()

	c.Set("key1", "value1")
	c.Set("key2", "value2")

	assert.Equal(t, 2, c.Size())

	c.Clear()

	assert.Equal(t, 0, c.Size())
	assert.False(t, c.Has("key1"))
	assert.False(t, c.Has("key2"))
}

func TestConfig_CachePerformance(t *testing.T) {
	c := config.NewWithCache(nil, 100)

	for i := 0; i < 1000; i++ {
		key := "key" + string(rune(i))
		c.Set(key, i)
	}

	for i := 0; i < 100; i++ {
		key := "key" + string(rune(i))
		val := c.Get(key, nil)
		assert.NotNil(t, val)
	}

	for i := 0; i < 100; i++ {
		key := "key" + string(rune(i))
		val := c.Get(key, nil)
		assert.NotNil(t, val)
	}
}

func BenchmarkConfig_GetWithCache(b *testing.B) {
	c := config.NewWithCache(nil, 1000)

	for i := 0; i < 1000; i++ {
		c.Set(string(rune(i)), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(string(rune(i%1000)), nil)
			i++
		}
	})
}

func BenchmarkConfig_GetWithoutCache(b *testing.B) {
	c := config.NewWithCache(nil, 0)

	for i := 0; i < 1000; i++ {
		c.Set(string(rune(i)), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(string(rune(i%1000)), nil)
			i++
		}
	})
}

func BenchmarkConfig_TypedGetters(b *testing.B) {
	c := config.NewWithCache(nil, 1000)

	c.Set("string", "test")
	c.Set("int", 42)
	c.Set("float", 3.14)
	c.Set("bool", true)

	b.Run("GetString", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.GetString("string", "")
			}
		})
	})

	b.Run("GetInt", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.GetInt("int", 0)
			}
		})
	})

	b.Run("GetFloat64", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.GetFloat64("float", 0)
			}
		})
	})

	b.Run("GetBool", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.GetBool("bool", false)
			}
		})
	})
}
