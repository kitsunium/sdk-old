package kcache

import (
	"fmt"
	"testing"
)

// TestUnsafeShardedCache_BasicOperations tests all basic cache operations
func TestUnsafeShardedCache_BasicOperations(t *testing.T) {
	c := NewUnsafeShardedCache(100, 4)

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

// TestUnsafeShardedCache_ShardDistribution tests key distribution across shards
func TestUnsafeShardedCache_ShardDistribution(t *testing.T) {
	shardCount := 8
	c := NewUnsafeShardedCache(1000, shardCount).(ShardedCache)

	// Verify shard count
	if c.ShardCount() != shardCount {
		t.Errorf("ShardCount: got %d, want %d", c.ShardCount(), shardCount)
	}

	// Test that keys distribute across shards
	shardMap := make(map[int]int)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		c.Set(key, i)
		shard := c.ShardFor(key)
		shardMap[shard]++
	}

	// Verify keys are distributed (not all in one shard)
	if len(shardMap) < 2 {
		t.Error("Keys not distributed across shards")
	}

	// Verify shard indices are valid
	for shard := range shardMap {
		if shard < 0 || shard >= shardCount {
			t.Errorf("Invalid shard index: %d", shard)
		}
	}
}

// TestUnsafeShardedCache_BatchOperations tests batch operations
func TestUnsafeShardedCache_BatchOperations(t *testing.T) {
	c := NewUnsafeShardedCache(100, 4).(BatchCache)

	keys := []interface{}{"k1", "k2", "k3", "k4", "k5"}
	values := []interface{}{"v1", "v2", "v3", "v4", "v5"}

	// Test SetBatch
	count := c.SetBatch(keys, values)
	if count != 5 {
		t.Errorf("SetBatch: expected 5 new keys, got %d", count)
	}

	// Test SetBatch with existing keys
	count = c.SetBatch(keys, values)
	if count != 0 {
		t.Errorf("SetBatch: expected 0 new keys for existing keys, got %d", count)
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

// TestUnsafeShardedCache_PanicOnConcurrentAccess verifies concurrent access detection
func TestUnsafeShardedCache_PanicOnConcurrentAccess(t *testing.T) {
	// Skip in production mode where checks are disabled
	if testingSkipSafetyCheck {
		t.Skip("Safety checks disabled in production mode")
	}

	c := NewUnsafeShardedCache(100, 4)

	// Set from main goroutine
	c.Set("key1", "value1")

	// Try to access from another goroutine - should panic
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic
				done <- true
			} else {
				done <- false
			}
		}()
		c.Set("key2", "value2") // This should panic
	}()

	panicked := <-done
	if !panicked {
		t.Error("Expected panic for concurrent access to unsafe sharded cache")
	}
}

// BenchmarkUnsafeShardedCache_Set benchmarks Set operation
func BenchmarkUnsafeShardedCache_Set(b *testing.B) {
	c := NewUnsafeShardedCache(10000, 16)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Set(i, i)
	}
}

// BenchmarkUnsafeShardedCache_Get benchmarks Get operation
func BenchmarkUnsafeShardedCache_Get(b *testing.B) {
	c := NewUnsafeShardedCache(10000, 16)

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

// BenchmarkUnsafeShardedCache_Mixed benchmarks mixed operations
func BenchmarkUnsafeShardedCache_Mixed(b *testing.B) {
	c := NewUnsafeShardedCache(10000, 16)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		c.Set(i, i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if i%3 == 0 {
			c.Set(i%2000, i)
		} else {
			c.Get(i % 1000)
		}
	}
}
