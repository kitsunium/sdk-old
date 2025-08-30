package kcache

import (
	"sync"
	"testing"
)

// TestDefaultOptions tests the default cache options
func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()

	if opts.capacity != DefaultCapacity {
		t.Errorf("Expected default capacity %d, got %d", DefaultCapacity, opts.capacity)
	}
	if opts.shards != DefaultShardCount {
		t.Errorf("Expected default shards %d, got %d", DefaultShardCount, opts.shards)
	}
	if opts.loadFactor != DefaultLoadFactor {
		t.Errorf("Expected default load factor %f, got %f", DefaultLoadFactor, opts.loadFactor)
	}
	if !opts.safe {
		t.Error("Expected default safe to be true")
	}
	if opts.sharded {
		t.Error("Expected default sharded to be false")
	}
}

// TestNewCacheWithOptions tests cache creation with various options
func TestNewCacheWithOptions(t *testing.T) {
	// Test default cache creation
	t.Run("DefaultCache", func(t *testing.T) {
		cache := NewCache()
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		// Should create a safe, non-sharded cache by default
		if _, ok := cache.(*safeCache); !ok {
			t.Error("Expected safeCache type by default")
		}
	})

	// Test with capacity option
	t.Run("WithCapacity", func(t *testing.T) {
		cache := NewCache(WithCapacity(1000))
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		// Test that cache was created successfully with capacity
		// We can't directly check capacity from the interface
		// but we can verify the cache works
		cache.Set("test", "value")
		if val, ok := cache.Get("test"); !ok || val != "value" {
			t.Error("Cache should work with specified capacity")
		}
	})

	// Test with shards option
	t.Run("WithShards", func(t *testing.T) {
		cache := NewCache(WithShards(32))
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		// Should create a sharded cache
		if _, ok := cache.(*safeShardedCache); !ok {
			t.Error("Expected safeShardedCache type with WithShards")
		}
	})

	// Test with unsafe option
	t.Run("WithUnsafe", func(t *testing.T) {
		cache := NewCache(WithUnsafe())
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		// Should create an unsafe cache
		if _, ok := cache.(*unsafeCache); !ok {
			t.Error("Expected unsafeCache type with WithUnsafe")
		}
	})

	// Test with unsafe and sharded options
	t.Run("UnsafeSharded", func(t *testing.T) {
		cache := NewCache(WithUnsafe(), WithSharded(true))
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		// Should create an unsafe sharded cache
		if _, ok := cache.(*unsafeShardedCache); !ok {
			t.Error("Expected unsafeShardedCache type")
		}
	})

	// Test with load factor option
	t.Run("WithLoadFactor", func(t *testing.T) {
		cache := NewCache(WithLoadFactor(0.5))
		if cache != nil {
			// Load factor should be applied (internal to cache implementation)
			// Just verify cache was created successfully
			t.Log("Cache created with custom load factor")
		}
	})

	// Test with nil hasher (should use default)
	t.Run("WithNilHasher", func(t *testing.T) {
		cache := NewCache(WithHasher(nil))
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		// Test that cache still works
		cache.Set("test", "value")
		if val, ok := cache.Get("test"); !ok || val != "value" {
			t.Error("Cache should work even with nil hasher")
		}
	})
}

// TestOptionBoundaryValidation tests boundary conditions for options
func TestOptionBoundaryValidation(t *testing.T) {
	// Test minimum capacity boundary
	t.Run("MinCapacity", func(t *testing.T) {
		cache := NewCache(WithCapacity(1)) // Below minimum
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		// Should be clamped to MinCapacity
		// We can't directly check capacity but verify cache works
		cache.Set("test", "value")
		if val, ok := cache.Get("test"); !ok || val != "value" {
			t.Error("Cache should work with min capacity")
		}
	})

	// Test maximum capacity boundary
	t.Run("MaxCapacity", func(t *testing.T) {
		cache := NewCache(WithCapacity(MaxCapacity + 1000))
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		// Should be clamped to MaxCapacity
		// We can't directly check capacity but verify cache works
		cache.Set("test", "value")
		if val, ok := cache.Get("test"); !ok || val != "value" {
			t.Error("Cache should work with max capacity")
		}
	})

	// Test minimum shard count boundary
	t.Run("MinShardCount", func(t *testing.T) {
		cache := NewCache(WithShards(1)) // Below minimum
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		// Should be clamped to MinShardCount
		// Note: We can't directly check shard count from the interface
	})

	// Test maximum shard count boundary
	t.Run("MaxShardCount", func(t *testing.T) {
		cache := NewCache(WithShards(MaxShardCount + 100))
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		// Should be clamped to MaxShardCount
	})

	// Test minimum load factor boundary
	t.Run("MinLoadFactor", func(t *testing.T) {
		cache := NewCache(WithLoadFactor(0.1)) // Below minimum
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		// Load factor should be clamped to MinLoadFactor
	})

	// Test maximum load factor boundary
	t.Run("MaxLoadFactor", func(t *testing.T) {
		cache := NewCache(WithLoadFactor(1.5)) // Above maximum
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		// Load factor should be clamped to MaxLoadFactor
	})
}

// TestOptionCombinations tests various combinations of options
func TestOptionCombinations(t *testing.T) {
	// Test all safe cache combinations
	t.Run("SafeNonSharded", func(t *testing.T) {
		cache := NewCache(WithSafe(true), WithSharded(false))
		if _, ok := cache.(*safeCache); !ok {
			t.Error("Expected safeCache type")
		}
	})

	t.Run("SafeSharded", func(t *testing.T) {
		cache := NewCache(WithSafe(true), WithSharded(true))
		if _, ok := cache.(*safeShardedCache); !ok {
			t.Error("Expected safeShardedCache type")
		}
	})

	// Test all unsafe cache combinations
	t.Run("UnsafeNonSharded", func(t *testing.T) {
		cache := NewCache(WithSafe(false), WithSharded(false))
		if _, ok := cache.(*unsafeCache); !ok {
			t.Error("Expected unsafeCache type")
		}
	})

	t.Run("UnsafeSharded", func(t *testing.T) {
		cache := NewCache(WithSafe(false), WithSharded(true))
		if _, ok := cache.(*unsafeShardedCache); !ok {
			t.Error("Expected unsafeShardedCache type")
		}
	})

	// Test multiple options applied together
	t.Run("MultipleOptions", func(t *testing.T) {
		cache := NewCache(
			WithCapacity(5000),
			WithShards(16),
			WithLoadFactor(0.75),
			WithSafe(true),
		)
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		if _, ok := cache.(*safeShardedCache); !ok {
			t.Error("Expected safeShardedCache with multiple options")
		}
	})
}

// TestOptionsConcurrency tests that options work correctly under concurrent access
func TestOptionsConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	numGoroutines := 100

	// Test concurrent cache creation with different options
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			var cache Cache
			switch id % 4 {
			case 0:
				cache = NewCache(WithCapacity(1000 + id))
			case 1:
				cache = NewCache(WithShards(8 << (id % 3)))
			case 2:
				cache = NewCache(WithUnsafe())
			case 3:
				cache = NewCache(WithSafe(true), WithSharded(true))
			}

			if cache == nil {
				t.Error("Cache creation failed concurrently")
			}

			// Test basic operations
			cache.Set(id, id*2)
			val, ok := cache.Get(id)
			if !ok || val != id*2 {
				t.Errorf("Concurrent cache operation failed for id %d", id)
			}
		}(i)
	}
	wg.Wait()
}

// TestOptionsPanicRecovery tests panic recovery in options
func TestOptionsPanicRecovery(t *testing.T) {
	// Test with nil hasher (should not panic)
	t.Run("NilHasher", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Should not panic with nil hasher: %v", r)
			}
		}()

		cache := NewCache(WithHasher(nil))
		if cache == nil {
			t.Fatal("Expected non-nil cache even with nil hasher")
		}
	})

	// Test with extreme values
	t.Run("ExtremeValues", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Should not panic with extreme values: %v", r)
			}
		}()

		cache := NewCache(
			WithCapacity(-1),
			WithShards(-1),
			WithLoadFactor(-1),
		)
		if cache == nil {
			t.Fatal("Expected non-nil cache with extreme values")
		}
	})
}

// Benchmarks for options and cache creation

// BenchmarkNewCacheDefault benchmarks default cache creation
func BenchmarkNewCacheDefault(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewCache()
	}
}

// BenchmarkNewCacheWithOptions benchmarks cache creation with options
func BenchmarkNewCacheWithOptions(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewCache(
			WithCapacity(10000),
			WithShards(16),
			WithLoadFactor(0.75),
		)
	}
}

// BenchmarkNewCacheSafe benchmarks safe cache creation
func BenchmarkNewCacheSafe(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewCache(WithSafe(true))
	}
}

// BenchmarkNewCacheUnsafe benchmarks unsafe cache creation
func BenchmarkNewCacheUnsafe(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewCache(WithUnsafe())
	}
}

// BenchmarkNewCacheSharded benchmarks sharded cache creation
func BenchmarkNewCacheSharded(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewCache(WithSharded(true))
	}
}

// BenchmarkNewCacheUnsafeSharded benchmarks unsafe sharded cache creation
func BenchmarkNewCacheUnsafeSharded(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewCache(WithUnsafe(), WithSharded(true))
	}
}

// BenchmarkOptionsApply benchmarks applying multiple options
func BenchmarkOptionsApply(b *testing.B) {
	opts := []Option{
		WithCapacity(10000),
		WithShards(16),
		WithLoadFactor(0.75),
		WithSafe(true),
		WithSharded(true),
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		options := defaultOptions()
		for _, opt := range opts {
			opt(options)
		}
	}
}

// BenchmarkOptionValidation benchmarks option validation
func BenchmarkOptionValidation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		options := &cacheOptions{
			capacity:   -1,
			shards:     999999,
			loadFactor: 2.0,
		}

		// Validate capacity
		if options.capacity < MinCapacity {
			options.capacity = MinCapacity
		}
		if options.capacity > MaxCapacity {
			options.capacity = MaxCapacity
		}

		// Validate shards
		if options.shards < MinShardCount {
			options.shards = MinShardCount
		}
		if options.shards > MaxShardCount {
			options.shards = MaxShardCount
		}

		// Validate load factor
		if options.loadFactor < MinLoadFactor {
			options.loadFactor = MinLoadFactor
		}
		if options.loadFactor > MaxLoadFactor {
			options.loadFactor = MaxLoadFactor
		}
	}
}

// BenchmarkCacheCreationParallel benchmarks parallel cache creation
func BenchmarkCacheCreationParallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewCache(
				WithCapacity(10000),
				WithShards(16),
			)
		}
	})
}
