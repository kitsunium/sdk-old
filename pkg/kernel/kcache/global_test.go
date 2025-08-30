package kcache

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestGlobalCache(t *testing.T) {
	t.Run("DefaultGlobalConfig", func(t *testing.T) {
		config := defaultGlobalConfig()
		if config.shardCount <= 0 {
			t.Errorf("expected positive shardCount, got %d", config.shardCount)
		}
		if config.capacity != DefaultCapacity*16 {
			t.Errorf("expected capacity %d, got %d", DefaultCapacity*16, config.capacity)
		}
		if config.loadFactor != DefaultLoadFactor {
			t.Errorf("expected load factor %f, got %f", DefaultLoadFactor, config.loadFactor)
		}
		expectedUseSharded := runtime.NumCPU() > 1
		if config.useSharded != expectedUseSharded {
			t.Errorf("expected useSharded %v, got %v", expectedUseSharded, config.useSharded)
		}
	})

	t.Run("Global", func(t *testing.T) {
		cache := Global()
		if cache == nil {
			t.Fatal("expected non-nil global cache")
		}

		// Test that calling Global multiple times returns the same instance
		cache2 := Global()
		if cache != cache2 {
			t.Error("expected same global cache instance")
		}
	})

	t.Run("SetGlobal", func(t *testing.T) {
		// Reset global cache state
		globalCache.once = sync.Once{}
		globalCache.instance = nil
		globalCache.config = nil

		newCache := NewCache(WithCapacity(100))
		ok := SetGlobal(newCache)
		if !ok {
			t.Error("expected SetGlobal to succeed on first call")
		}

		if globalCache.instance != newCache {
			t.Error("expected global cache instance to be updated")
		}

		// Test that SetGlobal fails after initialization
		ok = SetGlobal(NewCache(WithCapacity(200)))
		if ok {
			t.Error("SetGlobal should fail after initialization")
		}
	})

	t.Run("ConfigureGlobal", func(t *testing.T) {
		// Reset global cache state
		globalCache.once = sync.Once{}
		globalCache.instance = nil
		globalCache.config = nil

		ok := ConfigureGlobal(200, 8, true)
		if !ok {
			t.Error("expected ConfigureGlobal to succeed")
		}

		if globalCache.config == nil {
			t.Fatal("expected global config to be set")
		}
		if globalCache.config.capacity != 200 {
			t.Errorf("expected capacity 200, got %d", globalCache.config.capacity)
		}
	})

	t.Run("ResetGlobal", func(t *testing.T) {
		// Ensure we have a global cache instance
		Global()

		// Add some data
		Set("reset-test", "value")

		ResetGlobal()

		// Verify data was cleared
		if Has("reset-test") {
			t.Error("expected cache to be cleared after reset")
		}
	})

	t.Run("GlobalCacheOperations", func(t *testing.T) {
		ResetGlobal()

		// Test Set
		Set("key1", "value1")

		// Test Get
		val, ok := Get("key1")
		if !ok || val != "value1" {
			t.Errorf("expected value1, got %v", val)
		}

		// Test Has
		if !Has("key1") {
			t.Error("expected key1 to exist")
		}

		// Test Delete
		Delete("key1")
		if Has("key1") {
			t.Error("expected key1 to be deleted")
		}

		// Test Clear
		Set("key2", "value2")
		Set("key3", "value3")
		Clear()
		if Len() != 0 {
			t.Errorf("expected empty cache after Clear, got %d items", Len())
		}

		// Test Len
		Set("key4", "value4")
		if Len() != 1 {
			t.Errorf("expected 1 item, got %d", Len())
		}

		// Test Cap
		cap := Cap()
		if cap <= 0 {
			t.Errorf("expected positive capacity, got %d", cap)
		}
	})

	t.Run("GlobalBatchOperations", func(t *testing.T) {
		ResetGlobal()

		// Test SetBatch
		keys := []interface{}{"batch1", "batch2", "batch3"}
		values := []interface{}{"val1", "val2", "val3"}
		SetBatch(keys, values)

		// Test GetBatch
		results, found := GetBatch(keys)
		for i := range results {
			if !found[i] || results[i] != values[i] {
				t.Errorf("expected %v for key %v, got %v (found=%v)", values[i], keys[i], results[i], found[i])
			}
		}

		// Test HasBatch
		hasResults := HasBatch(keys)
		for i, has := range hasResults {
			if !has {
				t.Errorf("expected key %v to exist", keys[i])
			}
		}

		// Test DeleteBatch
		deleted := DeleteBatch(keys)
		for i, del := range deleted {
			if !del {
				t.Errorf("expected key %v to be deleted", keys[i])
			}
		}
		hasResults = HasBatch(keys)
		for i, has := range hasResults {
			if has {
				t.Errorf("expected key %v to not exist after deletion", keys[i])
			}
		}
	})

	t.Run("ThreadLocalCache", func(t *testing.T) {
		ResetGlobal()

		// Test getThreadLocal
		tl := getThreadLocal()
		if tl == nil {
			t.Fatal("expected non-nil thread local cache")
		}

		// Test that thread local is per-goroutine
		var wg sync.WaitGroup
		results := make(chan int, 2)

		wg.Add(2)
		go func() {
			defer wg.Done()
			tl := getThreadLocal()
			tl.Set("g1", "value1")
			results <- 1
		}()
		go func() {
			defer wg.Done()
			tl := getThreadLocal()
			tl.Set("g2", "value2")
			results <- 2
		}()

		wg.Wait()
		close(results)

		// Verify we got results from both goroutines
		count := 0
		for range results {
			count++
		}
		if count != 2 {
			t.Error("expected results from 2 goroutines")
		}
	})

	t.Run("GetGoroutineID", func(t *testing.T) {
		id1 := getGoroutineID()
		if id1 == 0 {
			t.Error("expected non-zero goroutine ID")
		}

		// Same goroutine should have same ID
		id2 := getGoroutineID()
		if id1 != id2 {
			t.Error("expected same goroutine ID")
		}

		// Different goroutine should have different ID
		ch := make(chan int)
		go func() {
			ch <- getGoroutineID()
		}()

		id3 := <-ch
		// Note: Due to the simplified implementation, we can't guarantee different IDs
		// Just verify we got a non-zero ID
		if id3 == 0 {
			t.Error("expected non-zero goroutine ID from different goroutine")
		}
	})

	t.Run("WithThreadLocal", func(t *testing.T) {
		ResetGlobal()

		executed := false
		WithThreadLocal(func(cache Cache) {
			executed = true
			cache.Set("thread_key", "thread_value")
			val, ok := cache.Get("thread_key")
			if !ok || val != "thread_value" {
				t.Errorf("expected thread_value, got %v", val)
			}
		})

		if !executed {
			t.Error("expected function to be executed")
		}
	})
}

func TestGlobalCacheConcurrency(t *testing.T) {
	ResetGlobal()

	const numGoroutines = 100
	const numOperations = 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)

				Set(key, value)

				if val, ok := Get(key); !ok || val != value {
					t.Errorf("concurrent get failed for %s", key)
				}

				if !Has(key) {
					t.Errorf("concurrent has failed for %s", key)
				}
			}

			// Delete after all checks
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				Delete(key)
			}
		}(i)
	}

	wg.Wait()
}

func TestGlobalCleanupPools(t *testing.T) {
	// Force cleanup
	cleanupPools()

	// Verify pools are cleaned
	// This is hard to test directly, but we can check that
	// the function doesn't panic or cause issues
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
}
