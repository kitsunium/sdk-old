package kcache

import (
	"fmt"
	"sync"
	"testing"
)

// TestSafeCache_BasicOperations tests all basic cache operations
func TestSafeCache_BasicOperations(t *testing.T) {
	c := NewSafeCache(100)

	// Test Set - new key
	if !c.Set("key1", "value1") {
		t.Error("Expected true for new key")
	}

	// Test Set - existing key
	if c.Set("key1", "value2") {
		t.Error("Expected false for existing key")
	}

	// Test Get - existing key
	v, ok := c.Get("key1")
	if !ok || v != "value2" {
		t.Errorf("Get failed: got %v, %v", v, ok)
	}

	// Test Get - non-existent key
	v, ok = c.Get("nonexistent")
	if ok {
		t.Error("Get should return false for non-existent key")
	}

	// Test Has - existing key
	if !c.Has("key1") {
		t.Error("Has failed for existing key")
	}

	// Test Has - non-existent key
	if c.Has("nonexistent") {
		t.Error("Has returned true for nonexistent key")
	}

	// Test Delete - existing key
	if !c.Delete("key1") {
		t.Error("Delete failed for existing key")
	}

	// Test Delete - already deleted key
	if c.Delete("key1") {
		t.Error("Delete returned true for already deleted key")
	}

	// Test Len
	c.Set("key2", "value2")
	c.Set("key3", "value3")
	if c.Len() != 2 {
		t.Errorf("Len failed: got %d, want 2", c.Len())
	}

	// Test Clear
	c.Clear()
	if c.Len() != 0 {
		t.Errorf("Clear failed: len = %d", c.Len())
	}

	// Test Cap
	if c.Cap() <= 0 {
		t.Errorf("Cap returned invalid value: %d", c.Cap())
	}
}

// TestSafeCache_ConcurrentAccess tests thread safety
func TestSafeCache_ConcurrentAccess(t *testing.T) {
	c := NewSafeCache(10000)
	const goroutines = 100
	const operations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Run concurrent operations
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)

				// Mix of operations
				c.Set(key, value)
				v, ok := c.Get(key)
				if ok && v != value {
					t.Errorf("Concurrent Get returned wrong value: got %v, want %v", v, value)
				}
				c.Has(key)
				if j%10 == 0 {
					c.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is still functional
	c.Set("final", "test")
	v, ok := c.Get("final")
	if !ok || v != "test" {
		t.Error("Cache corrupted after concurrent access")
	}
}

// TestSafeCache_BatchOperations tests batch operations
func TestSafeCache_BatchOperations(t *testing.T) {
	c := NewSafeCache(100).(BatchCache)

	keys := []interface{}{"k1", "k2", "k3"}
	values := []interface{}{"v1", "v2", "v3"}

	// Test SetBatch
	count := c.SetBatch(keys, values)
	if count != 3 {
		t.Errorf("SetBatch: expected 3 new keys, got %d", count)
	}

	// Test GetBatch
	vals, found := c.GetBatch(keys)
	for i := range keys {
		if !found[i] || vals[i] != values[i] {
			t.Errorf("GetBatch failed for key %v", keys[i])
		}
	}

	// Test HasBatch
	exists := c.HasBatch(keys)
	for i, e := range exists {
		if !e {
			t.Errorf("HasBatch failed for key %v", keys[i])
		}
	}

	// Test DeleteBatch
	deleted := c.DeleteBatch(keys)
	for i, d := range deleted {
		if !d {
			t.Errorf("DeleteBatch failed for key %v", keys[i])
		}
	}

	if c.Len() != 0 {
		t.Errorf("DeleteBatch didn't remove all keys: len = %d", c.Len())
	}
}

// TestSafeCache_ConcurrentBatchOperations tests batch operations under concurrency
func TestSafeCache_ConcurrentBatchOperations(t *testing.T) {
	c := NewSafeCache(10000).(BatchCache)
	const goroutines = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			keys := make([]interface{}, 10)
			values := make([]interface{}, 10)
			for j := 0; j < 10; j++ {
				keys[j] = fmt.Sprintf("batch-%d-key-%d", id, j)
				values[j] = fmt.Sprintf("batch-%d-val-%d", id, j)
			}

			// Batch operations
			c.SetBatch(keys, values)
			vals, found := c.GetBatch(keys)
			for k := range keys {
				if !found[k] || vals[k] != values[k] {
					t.Errorf("Concurrent batch failed for key %v", keys[k])
				}
			}
			c.HasBatch(keys)
			c.DeleteBatch(keys)
		}(i)
	}

	wg.Wait()
}

// BenchmarkSafeCache_Set benchmarks Set operation
func BenchmarkSafeCache_Set(b *testing.B) {
	c := NewSafeCache(10000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Set(i, i)
	}
}

// BenchmarkSafeCache_Get benchmarks Get operation
func BenchmarkSafeCache_Get(b *testing.B) {
	c := NewSafeCache(10000)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		c.Set(i, i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Get(i % 1000)
	}
}

// BenchmarkSafeCache_ConcurrentMixed benchmarks concurrent mixed operations
func BenchmarkSafeCache_ConcurrentMixed(b *testing.B) {
	c := NewSafeCache(10000)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		c.Set(i, i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%3 == 0 {
				c.Set(i%2000, i)
			} else {
				c.Get(i % 1000)
			}
			i++
		}
	})
}
