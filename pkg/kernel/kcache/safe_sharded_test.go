package kcache

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// TestSafeShardedCache_BasicOperations tests all basic cache operations
func TestSafeShardedCache_BasicOperations(t *testing.T) {
	c := NewSafeShardedCache(100, 4)

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

// TestSafeShardedCache_ShardDistribution tests key distribution across shards
func TestSafeShardedCache_ShardDistribution(t *testing.T) {
	shardCount := 8
	c := NewSafeShardedCache(1000, shardCount).(ShardedCache)

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

// TestSafeShardedCache_HighConcurrency tests with many goroutines
func TestSafeShardedCache_HighConcurrency(t *testing.T) {
	c := NewSafeShardedCache(10000, runtime.NumCPU()*4)
	const goroutines = 100
	const operations = 1000

	var wg sync.WaitGroup
	var errors atomic.Int32

	wg.Add(goroutines)
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
					errors.Add(1)
				}
				c.Has(key)
				if j%10 == 0 {
					c.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()

	if errors.Load() > 0 {
		t.Errorf("Found %d errors during concurrent access", errors.Load())
	}

	// Verify cache is still functional
	c.Set("final", "test")
	v, ok := c.Get("final")
	if !ok || v != "test" {
		t.Error("Cache corrupted after concurrent access")
	}
}

// TestSafeShardedCache_BatchOperations tests batch operations
func TestSafeShardedCache_BatchOperations(t *testing.T) {
	c := NewSafeShardedCache(100, 4).(BatchCache)

	keys := []interface{}{"k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8"}
	values := []interface{}{"v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8"}

	// Test SetBatch
	count := c.SetBatch(keys, values)
	if count != 8 {
		t.Errorf("SetBatch: expected 8 new keys, got %d", count)
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

// TestSafeShardedCache_ConcurrentBatchOperations tests batch operations under high concurrency
func TestSafeShardedCache_ConcurrentBatchOperations(t *testing.T) {
	c := NewSafeShardedCache(10000, runtime.NumCPU()*4).(BatchCache)
	const goroutines = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			keys := make([]interface{}, 20)
			values := make([]interface{}, 20)
			for j := 0; j < 20; j++ {
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

// BenchmarkSafeShardedCache_Set benchmarks Set operation
func BenchmarkSafeShardedCache_Set(b *testing.B) {
	c := NewSafeShardedCache(10000, runtime.NumCPU()*4)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Set(i, i)
	}
}

// BenchmarkSafeShardedCache_Get benchmarks Get operation
func BenchmarkSafeShardedCache_Get(b *testing.B) {
	c := NewSafeShardedCache(10000, runtime.NumCPU()*4)

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

// BenchmarkSafeShardedCache_ConcurrentMixed benchmarks concurrent mixed operations
func BenchmarkSafeShardedCache_ConcurrentMixed(b *testing.B) {
	c := NewSafeShardedCache(10000, runtime.NumCPU()*4)

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

// BenchmarkSafeShardedCache_Scalability tests scalability with varying goroutines
func BenchmarkSafeShardedCache_Scalability(b *testing.B) {
	for _, goroutines := range []int{1, 2, 4, 8, 16, 32} {
		b.Run(fmt.Sprintf("%dg", goroutines), func(b *testing.B) {
			c := NewSafeShardedCache(10000, runtime.NumCPU()*4)

			// Pre-populate
			for i := 0; i < 1000; i++ {
				c.Set(i, i)
			}

			b.ResetTimer()
			b.SetParallelism(goroutines)
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					if i%10 < 8 { // 80% reads
						c.Get(i % 1000)
					} else { // 20% writes
						c.Set(i%2000, i)
					}
					i++
				}
			})
		})
	}
}
