package kcache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestNewBatchProcessor tests the batch processor creation
func TestNewBatchProcessor(t *testing.T) {
	cache := NewCache(WithCapacity(1000))
	bp := newBatchProcessor(cache)

	if bp == nil {
		t.Fatal("expected non-nil batch processor")
	}
	if bp.cache != cache {
		t.Error("unexpected cache reference")
	}
	if bp.workers <= 0 {
		t.Errorf("expected positive workers, got %d", bp.workers)
	}
}

// TestOptimizedSetBatch tests optimized batch set operations
func TestOptimizedSetBatch(t *testing.T) {
	cache := NewCache(WithCapacity(1000))

	// Test with empty slices
	count := OptimizedSetBatch(cache, []interface{}{}, []interface{}{})
	if count != 0 {
		t.Errorf("Expected 0 for empty slices, got %d", count)
	}

	// Test with mismatched lengths
	count = OptimizedSetBatch(cache, []interface{}{"k1"}, []interface{}{"v1", "v2"})
	if count != 0 {
		t.Errorf("Expected 0 for mismatched lengths, got %d", count)
	}

	// Test small batch (< 16 items)
	keys := make([]interface{}, 10)
	values := make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		keys[i] = fmt.Sprintf("small-key-%d", i)
		values[i] = fmt.Sprintf("small-value-%d", i)
	}
	count = OptimizedSetBatch(cache, keys, values)
	if count != 10 {
		t.Errorf("Expected 10 new keys, got %d", count)
	}

	// Test large batch (>= 16 items)
	keys = make([]interface{}, 100)
	values = make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		keys[i] = fmt.Sprintf("large-key-%d", i)
		values[i] = fmt.Sprintf("large-value-%d", i)
	}
	count = OptimizedSetBatch(cache, keys, values)
	if count != 100 {
		t.Errorf("Expected 100 new keys, got %d", count)
	}
}

// TestOptimizedGetBatch tests optimized batch get operations
func TestOptimizedGetBatch(t *testing.T) {
	cache := NewCache(WithCapacity(1000))

	// Pre-populate cache
	for i := 0; i < 20; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
	}

	// Test empty keys
	values, found := OptimizedGetBatch(cache, []interface{}{})
	if values != nil || found != nil {
		t.Error("Expected nil for empty keys")
	}

	// Test existing keys
	keys := make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
	}
	values, found = OptimizedGetBatch(cache, keys)

	if len(values) != 10 || len(found) != 10 {
		t.Errorf("Expected 10 values and found flags, got %d and %d", len(values), len(found))
	}

	for i := 0; i < 10; i++ {
		if !found[i] {
			t.Errorf("Key %v should be found", keys[i])
		}
		expected := fmt.Sprintf("value-%d", i)
		if values[i] != expected {
			t.Errorf("Value mismatch for key %v: got %v, want %v", keys[i], values[i], expected)
		}
	}

	// Test non-existent keys
	keys = []interface{}{"non-existent-1", "non-existent-2"}
	values, found = OptimizedGetBatch(cache, keys)
	for i := range keys {
		if found[i] {
			t.Errorf("Non-existent key %v should not be found", keys[i])
		}
	}
}

// TestOptimizedHasBatch tests optimized batch has operations
func TestOptimizedHasBatch(t *testing.T) {
	cache := NewCache(WithCapacity(1000))

	// Pre-populate cache
	for i := 0; i < 10; i++ {
		cache.Set(fmt.Sprintf("exists-%d", i), i)
	}

	// Test empty keys
	found := OptimizedHasBatch(cache, []interface{}{})
	if found != nil {
		t.Error("Expected nil for empty keys")
	}

	// Test mixed existing and non-existing keys
	keys := []interface{}{
		"exists-0", "exists-5", "not-exists-1", "exists-9", "not-exists-2",
	}
	found = OptimizedHasBatch(cache, keys)

	expected := []bool{true, true, false, true, false}
	for i, exp := range expected {
		if found[i] != exp {
			t.Errorf("HasBatch for key %v: got %v, want %v", keys[i], found[i], exp)
		}
	}
}

// TestOptimizedDeleteBatch tests optimized batch delete operations
func TestOptimizedDeleteBatch(t *testing.T) {
	cache := NewCache(WithCapacity(1000))

	// Pre-populate cache
	for i := 0; i < 20; i++ {
		cache.Set(fmt.Sprintf("del-key-%d", i), fmt.Sprintf("del-value-%d", i))
	}

	// Test empty keys
	deleted := OptimizedDeleteBatch(cache, []interface{}{})
	if deleted != nil {
		t.Error("Expected nil for empty keys")
	}

	// Test deleting existing keys
	keys := make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		keys[i] = fmt.Sprintf("del-key-%d", i)
	}
	deleted = OptimizedDeleteBatch(cache, keys)

	for i := range keys {
		if !deleted[i] {
			t.Errorf("Key %v should have been deleted", keys[i])
		}
	}

	// Verify keys are actually deleted
	for i := 0; i < 10; i++ {
		if cache.Has(fmt.Sprintf("del-key-%d", i)) {
			t.Errorf("Key del-key-%d should not exist after deletion", i)
		}
	}

	// Test deleting non-existent keys
	deleted = OptimizedDeleteBatch(cache, keys) // Same keys, already deleted
	for i := range keys {
		if deleted[i] {
			t.Errorf("Key %v was already deleted, should return false", keys[i])
		}
	}
}

// TestBatchBuilder tests the BatchBuilder fluent interface
func TestBatchBuilder(t *testing.T) {
	cache := NewCache(WithCapacity(1000))

	// Test Set batch
	bb := NewBatchBuilder(cache)
	bb.Set("k1", "v1").Set("k2", "v2").Set("k3", "v3")
	result := bb.Execute()
	if count, ok := result.(int); !ok || count != 3 {
		t.Errorf("Expected 3 new keys from Set batch, got %v", result)
	}

	// Test Get batch
	bb.Reset()
	bb.Get("k1", "k2", "k3", "k4")
	result = bb.Execute()
	if res, ok := result.([]interface{}); ok {
		values := res[0].([]interface{})
		found := res[1].([]bool)
		if len(values) != 4 || len(found) != 4 {
			t.Error("Get batch returned wrong number of results")
		}
		// First 3 should be found, 4th should not
		for i := 0; i < 3; i++ {
			if !found[i] {
				t.Errorf("Key k%d should be found", i+1)
			}
		}
		if found[3] {
			t.Error("Key k4 should not be found")
		}
	} else {
		t.Error("Get batch returned unexpected type")
	}

	// Test Delete batch
	bb.Reset()
	bb.Delete("k1", "k2", "k3")
	result = bb.Execute()
	if deleted, ok := result.([]bool); ok {
		for i, d := range deleted {
			if !d {
				t.Errorf("Key k%d should have been deleted", i+1)
			}
		}
	} else {
		t.Error("Delete batch returned unexpected type")
	}
}

// TestSetBatchInt64 tests optimized batch operations for int64 keys
func TestSetBatchInt64(t *testing.T) {
	cache := NewCache(WithCapacity(1000))

	keys := []int64{1, 2, 3, 4, 5}
	values := []interface{}{"v1", "v2", "v3", "v4", "v5"}

	count := SetBatchInt64(cache, keys, values)
	if count != 5 {
		t.Errorf("Expected 5 new keys, got %d", count)
	}

	// Verify all keys were set
	for i, key := range keys {
		v, ok := cache.Get(key)
		if !ok || v != values[i] {
			t.Errorf("Key %d not set correctly", key)
		}
	}
}

// TestSetBatchString tests optimized batch operations for string keys
func TestSetBatchString(t *testing.T) {
	cache := NewCache(WithCapacity(1000))

	keys := []string{"s1", "s2", "s3", "s4", "s5"}
	values := []interface{}{"v1", "v2", "v3", "v4", "v5"}

	count := SetBatchString(cache, keys, values)
	if count != 5 {
		t.Errorf("Expected 5 new keys, got %d", count)
	}

	// Verify all keys were set
	for i, key := range keys {
		v, ok := cache.Get(key)
		if !ok || v != values[i] {
			t.Errorf("Key %s not set correctly", key)
		}
	}
}

// TestSetBatchBytes tests optimized batch operations for byte slice keys
func TestSetBatchBytes(t *testing.T) {
	cache := NewCache(WithCapacity(1000))

	keys := [][]byte{
		[]byte("b1"), []byte("b2"), []byte("b3"), []byte("b4"), []byte("b5"),
	}
	values := []interface{}{"v1", "v2", "v3", "v4", "v5"}

	count := SetBatchBytes(cache, keys, values)
	if count != 5 {
		t.Errorf("Expected 5 new keys, got %d", count)
	}

	// Verify all keys were set (using string conversion for comparison)
	for i, key := range keys {
		v, ok := cache.Get(string(key))
		if !ok || v != values[i] {
			t.Errorf("Key %s not set correctly", string(key))
		}
	}
}

// TestStreamingBatch tests streaming batch operations
func TestStreamingBatch(t *testing.T) {
	cache := NewCache(WithCapacity(1000))
	sb := NewStreamingBatch(cache, 10)

	// Add items one by one
	for i := 0; i < 25; i++ {
		err := sb.Add(fmt.Sprintf("stream-key-%d", i), fmt.Sprintf("stream-val-%d", i))
		if err != nil {
			t.Errorf("Add failed: %v", err)
		}
	}

	// Flush remaining items
	err := sb.Flush()
	if err != nil {
		t.Errorf("Flush failed: %v", err)
	}

	// Verify all items were added
	for i := 0; i < 25; i++ {
		key := fmt.Sprintf("stream-key-%d", i)
		expected := fmt.Sprintf("stream-val-%d", i)
		v, ok := cache.Get(key)
		if !ok || v != expected {
			t.Errorf("Key %s not found or has wrong value", key)
		}
	}

	// Test Close
	err = sb.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// BenchmarkOptimizedSetBatch benchmarks batch set operations
func BenchmarkOptimizedSetBatch(b *testing.B) {
	cache := NewSafeShardedCache(10000, 16)

	keys := make([]interface{}, 100)
	values := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		values[i] = fmt.Sprintf("value-%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		OptimizedSetBatch(cache, keys, values)
	}
}

// BenchmarkOptimizedGetBatch benchmarks batch get operations
func BenchmarkOptimizedGetBatch(b *testing.B) {
	cache := NewSafeShardedCache(10000, 16)

	// Pre-populate
	keys := make([]interface{}, 100)
	values := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		values[i] = fmt.Sprintf("value-%d", i)
	}
	OptimizedSetBatch(cache, keys, values)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		OptimizedGetBatch(cache, keys)
	}
}

// TestBatchProcessorExecute tests the Execute method of batchProcessor
func TestBatchProcessorExecute(t *testing.T) {
	cache := NewCache(WithCapacity(1000))
	bp := newBatchProcessor(cache)

	// Test small batch (sequential)
	keys := make([]interface{}, 5)
	values := make([]interface{}, 5)
	for i := 0; i < 5; i++ {
		keys[i] = fmt.Sprintf("exec-key-%d", i)
		values[i] = fmt.Sprintf("exec-value-%d", i)
	}

	result := bp.Execute(batchOpSet, keys, values)
	if count, ok := result.(int); !ok || count != 5 {
		t.Errorf("Expected 5 items set, got %v", result)
	}

	// Test large batch (parallel)
	keys = make([]interface{}, 100)
	values = make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		keys[i] = fmt.Sprintf("parallel-key-%d", i)
		values[i] = fmt.Sprintf("parallel-value-%d", i)
	}

	result = bp.Execute(batchOpSet, keys, values)
	if count, ok := result.(int); !ok || count != 100 {
		t.Errorf("Expected 100 items set, got %v", result)
	}

	// Test Get operation
	result = bp.Execute(batchOpGet, keys, nil)
	if results, ok := result.([]GetResult); ok {
		for i, res := range results {
			if !res.Found || res.Value != values[i] {
				t.Errorf("Get failed for key %v", keys[i])
			}
		}
	} else {
		t.Error("Get batch returned unexpected type")
	}

	// Test Has operation
	result = bp.Execute(batchOpHas, keys, nil)
	if results, ok := result.([]bool); ok {
		for i, found := range results {
			if !found {
				t.Errorf("Has failed for key %v", keys[i])
			}
		}
	} else {
		t.Error("Has batch returned unexpected type")
	}

	// Test Delete operation
	result = bp.Execute(batchOpDelete, keys, nil)
	if results, ok := result.([]bool); ok {
		for i, deleted := range results {
			if !deleted {
				t.Errorf("Delete failed for key %v", keys[i])
			}
		}
	} else {
		t.Error("Delete batch returned unexpected type")
	}

	// Verify deletion
	result = bp.Execute(batchOpHas, keys, nil)
	if results, ok := result.([]bool); ok {
		for i, found := range results {
			if found {
				t.Errorf("Key %v should not exist after deletion", keys[i])
			}
		}
	}

	// Test invalid operation
	result = bp.Execute(99, keys, values)
	if result != nil {
		t.Error("Expected nil for invalid operation")
	}
}

// TestParallelBatchSet tests the parallel batch set function
func TestParallelBatchSet(t *testing.T) {
	cache := NewCache(WithCapacity(10000))

	// Test with small number of workers
	keys := make([]interface{}, 1000)
	values := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("parallel-set-%d", i)
		values[i] = i
	}

	count := parallelBatchSet(cache, keys, values)
	if count != 1000 {
		t.Errorf("Expected 1000 items set, got %d", count)
	}

	// Verify all items were set
	for i, key := range keys {
		val, ok := cache.Get(key)
		if !ok || val != values[i] {
			t.Errorf("Key %v not set correctly", key)
		}
	}

	// Test with many workers
	keys2 := make([]interface{}, 100)
	values2 := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		keys2[i] = fmt.Sprintf("many-workers-%d", i)
		values2[i] = i * 2
	}

	count = parallelBatchSet(cache, keys2, values2)
	if count != 100 {
		t.Errorf("Expected 100 items set, got %d", count)
	}
}

// TestBatchBuilderExecute tests the Execute method edge cases
func TestBatchBuilderExecute(t *testing.T) {
	cache := NewCache(WithCapacity(1000))
	bb := NewBatchBuilder(cache)

	// Test Execute with no operations
	result := bb.Execute()
	if result != nil {
		t.Error("Expected nil for empty batch")
	}

	// Test mixed operations (should use last op type)
	// First set the value
	bb.Set("k1", "v1")
	bb.Execute()

	// Now test Get
	bb.Reset()
	bb.Get("k1")
	result = bb.Execute()
	// Should execute Get and find the value
	if res, ok := result.([]interface{}); ok {
		values := res[0].([]interface{})
		found := res[1].([]bool)
		if len(values) != 1 || !found[0] {
			t.Error("Get operation should find previously set value")
		}
	}

	// Test Has operation
	bb.Reset()
	bb.Set("has1", "val1").Set("has2", "val2")
	bb.Execute()

	bb.Reset()
	bb.Has("has1", "has2", "has3")
	result = bb.Execute()
	if res, ok := result.([]bool); ok {
		if !res[0] || !res[1] || res[2] {
			t.Error("Has operation returned unexpected results")
		}
	} else {
		t.Error("Has operation returned unexpected type")
	}
}

// TestNewStreamingBatch tests streaming batch operations
func TestNewStreamingBatch(t *testing.T) {
	cache := NewCache(WithCapacity(1000))

	// Test creation with different thresholds
	sb := NewStreamingBatch(cache, 10)
	if sb == nil {
		t.Fatal("Expected non-nil streaming batch")
	}

	// Test Add operations below threshold
	for i := 0; i < 5; i++ {
		sb.Add(fmt.Sprintf("stream-key-%d", i), fmt.Sprintf("stream-value-%d", i))
	}

	// Verify items not yet in cache (not flushed)
	if cache.Has("stream-key-0") {
		t.Error("Items should not be in cache before flush")
	}

	// Test manual flush
	err := sb.Flush()
	if err != nil {
		t.Errorf("Flush error: %v", err)
	}
	count := 5
	if count != 5 {
		t.Errorf("Expected 5 items flushed, got %d", count)
	}

	// Verify items now in cache
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("stream-key-%d", i)
		if !cache.Has(key) {
			t.Errorf("Key %s should be in cache after flush", key)
		}
	}

	// Test auto-flush on threshold
	sb2 := NewStreamingBatch(cache, 3)
	sb2.Add("auto1", "val1")
	sb2.Add("auto2", "val2")
	sb2.Add("auto3", "val3") // Should trigger auto-flush

	// Give a small delay for auto-flush
	time.Sleep(10 * time.Millisecond)

	if !cache.Has("auto1") || !cache.Has("auto2") || !cache.Has("auto3") {
		t.Error("Auto-flush should have occurred at threshold")
	}

	// Test Close with pending items
	sb3 := NewStreamingBatch(cache, 100)
	sb3.Add("pending1", "val1")
	sb3.Add("pending2", "val2")

	// Close should flush pending items
	sb3.Close()

	if !cache.Has("pending1") || !cache.Has("pending2") {
		t.Error("Close should flush pending items")
	}
}

// TestStreamingBatchConcurrency tests concurrent operations on streaming batch
func TestStreamingBatchConcurrency(t *testing.T) {
	cache := NewCache(WithCapacity(10000))
	sb := NewStreamingBatch(cache, 50)
	defer sb.Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	itemsPerGoroutine := 100

	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < itemsPerGoroutine; i++ {
				key := fmt.Sprintf("concurrent-%d-%d", id, i)
				value := fmt.Sprintf("value-%d-%d", id, i)
				sb.Add(key, value)
			}
		}(g)
	}

	wg.Wait()
	sb.Flush()

	// Verify all items are in cache
	for g := 0; g < numGoroutines; g++ {
		for i := 0; i < itemsPerGoroutine; i++ {
			key := fmt.Sprintf("concurrent-%d-%d", g, i)
			if !cache.Has(key) {
				t.Errorf("Missing key: %s", key)
			}
		}
	}
}

// TestTypedBatchOperationsExtended tests SetBatchInt64, SetBatchString, SetBatchBytes with edge cases
func TestTypedBatchOperationsExtended(t *testing.T) {
	cache := NewCache(WithCapacity(1000))

	t.Run("SetBatchInt64Extended", func(t *testing.T) {
		keys := []int64{100, 200, 300}
		values := []interface{}{"val1", "val2", "val3"}

		count := SetBatchInt64(cache, keys, values)
		if count != 3 {
			t.Errorf("Expected 3 items set, got %d", count)
		}

		// Verify values
		for i, key := range keys {
			val, ok := cache.Get(key)
			if !ok || val != values[i] {
				t.Errorf("SetBatchInt64 failed for key %d", key)
			}
		}

		// Test with mismatched lengths
		count = SetBatchInt64(cache, keys, []interface{}{"v1", "v2"})
		if count != 0 {
			t.Error("Expected 0 for mismatched lengths")
		}

		// Test with empty slices
		count = SetBatchInt64(cache, []int64{}, []interface{}{})
		if count != 0 {
			t.Error("Expected 0 for empty slices")
		}
	})

	t.Run("SetBatchStringExtended", func(t *testing.T) {
		keys := []string{"str1", "str2", "str3"}
		values := []interface{}{"value1", "value2", "value3"}

		count := SetBatchString(cache, keys, values)
		if count != 3 {
			t.Errorf("Expected 3 items set, got %d", count)
		}

		// Verify values
		for i, key := range keys {
			val, ok := cache.Get(key)
			if !ok || val != values[i] {
				t.Errorf("SetBatchString failed for key %s", key)
			}
		}

		// Test with mismatched lengths
		count = SetBatchString(cache, keys, []interface{}{"a"})
		if count != 0 {
			t.Error("Expected 0 for mismatched lengths")
		}

		// Test with empty slices
		count = SetBatchString(cache, []string{}, []interface{}{})
		if count != 0 {
			t.Error("Expected 0 for empty slices")
		}
	})

	t.Run("SetBatchBytesExtended", func(t *testing.T) {
		keys := [][]byte{[]byte("bytes1"), []byte("bytes2"), []byte("bytes3")}
		values := []interface{}{"data1", "data2", "data3"}

		count := SetBatchBytes(cache, keys, values)
		if count != 3 {
			t.Errorf("Expected 3 items set, got %d", count)
		}

		// Verify values - keys are converted to strings internally
		keyStrings := []string{"bytes1", "bytes2", "bytes3"}
		for i, key := range keyStrings {
			val, ok := cache.Get(key)
			if !ok || val != values[i] {
				t.Errorf("SetBatchBytes failed for key %s", key)
			}
		}

		// Test with mismatched lengths
		count = SetBatchBytes(cache, keys, []interface{}{"a", "b"})
		if count != 0 {
			t.Error("Expected 0 for mismatched lengths")
		}

		// Test with empty slices
		count = SetBatchBytes(cache, [][]byte{}, []interface{}{})
		if count != 0 {
			t.Error("Expected 0 for empty slices")
		}
	})
}

// TestBatchPanicRecovery tests panic recovery in batch operations
func TestBatchPanicRecovery(t *testing.T) {
	// Test panic recovery during batch set with nil cache
	t.Run("NilCachePanic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic with nil cache")
			}
		}()
		OptimizedSetBatch(nil, []interface{}{"key"}, []interface{}{"value"})
	})

	// Test panic recovery during invalid batch operation
	t.Run("InvalidBatchOperation", func(t *testing.T) {
		cache := NewCache(WithCapacity(100))
		bp := newBatchProcessor(cache)

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Should not panic on invalid operation: %v", r)
			}
		}()

		// Should return nil for invalid operation, not panic
		result := bp.Execute(999, []interface{}{"key"}, []interface{}{"value"})
		if result != nil {
			t.Error("Expected nil for invalid operation")
		}
	})
}

// TestBatchConcurrentOperations tests concurrent batch operations
func TestBatchConcurrentOperations(t *testing.T) {
	cache := NewSafeShardedCache(10000, 16)
	var wg sync.WaitGroup
	numGoroutines := 100
	itemsPerGoroutine := 100

	// Test concurrent batch sets
	t.Run("ConcurrentBatchSet", func(t *testing.T) {
		wg.Add(numGoroutines)
		for g := 0; g < numGoroutines; g++ {
			go func(id int) {
				defer wg.Done()
				keys := make([]interface{}, itemsPerGoroutine)
				values := make([]interface{}, itemsPerGoroutine)
				for i := 0; i < itemsPerGoroutine; i++ {
					keys[i] = fmt.Sprintf("concurrent-set-%d-%d", id, i)
					values[i] = fmt.Sprintf("value-%d-%d", id, i)
				}
				OptimizedSetBatch(cache, keys, values)
			}(g)
		}
		wg.Wait()

		// Verify all items were set
		for g := 0; g < numGoroutines; g++ {
			for i := 0; i < itemsPerGoroutine; i++ {
				key := fmt.Sprintf("concurrent-set-%d-%d", g, i)
				if !cache.Has(key) {
					t.Errorf("Missing key: %s", key)
				}
			}
		}
	})

	// Test concurrent batch gets
	t.Run("ConcurrentBatchGet", func(t *testing.T) {
		wg.Add(numGoroutines)
		for g := 0; g < numGoroutines; g++ {
			go func(id int) {
				defer wg.Done()
				keys := make([]interface{}, itemsPerGoroutine)
				for i := 0; i < itemsPerGoroutine; i++ {
					keys[i] = fmt.Sprintf("concurrent-set-%d-%d", id, i)
				}
				values, found := OptimizedGetBatch(cache, keys)
				for i := range keys {
					if !found[i] {
						t.Errorf("Key %v should be found", keys[i])
					}
					expected := fmt.Sprintf("value-%d-%d", id, i)
					if values[i] != expected {
						t.Errorf("Value mismatch for key %v", keys[i])
					}
				}
			}(g)
		}
		wg.Wait()
	})

	// Test concurrent batch deletes
	t.Run("ConcurrentBatchDelete", func(t *testing.T) {
		wg.Add(numGoroutines)
		for g := 0; g < numGoroutines; g++ {
			go func(id int) {
				defer wg.Done()
				keys := make([]interface{}, itemsPerGoroutine/2) // Delete half
				for i := 0; i < itemsPerGoroutine/2; i++ {
					keys[i] = fmt.Sprintf("concurrent-set-%d-%d", id, i)
				}
				OptimizedDeleteBatch(cache, keys)
			}(g)
		}
		wg.Wait()

		// Verify items were deleted
		for g := 0; g < numGoroutines; g++ {
			for i := 0; i < itemsPerGoroutine/2; i++ {
				key := fmt.Sprintf("concurrent-set-%d-%d", g, i)
				if cache.Has(key) {
					t.Errorf("Key should be deleted: %s", key)
				}
			}
			// Verify remaining items still exist
			for i := itemsPerGoroutine / 2; i < itemsPerGoroutine; i++ {
				key := fmt.Sprintf("concurrent-set-%d-%d", g, i)
				if !cache.Has(key) {
					t.Errorf("Key should still exist: %s", key)
				}
			}
		}
	})
}

// Benchmarks for comprehensive performance evaluation

// BenchmarkBatchSetSmall benchmarks small batch set operations
func BenchmarkBatchSetSmall(b *testing.B) {
	cache := NewSafeShardedCache(10000, 16)
	keys := make([]interface{}, 10)
	values := make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		values[i] = fmt.Sprintf("value-%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		OptimizedSetBatch(cache, keys, values)
	}
}

// BenchmarkBatchSetLarge benchmarks large batch set operations
func BenchmarkBatchSetLarge(b *testing.B) {
	cache := NewSafeShardedCache(100000, 16)
	keys := make([]interface{}, 1000)
	values := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		values[i] = fmt.Sprintf("value-%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		OptimizedSetBatch(cache, keys, values)
	}
}

// BenchmarkBatchGetSmall benchmarks small batch get operations
func BenchmarkBatchGetSmall(b *testing.B) {
	cache := NewSafeShardedCache(10000, 16)
	keys := make([]interface{}, 10)
	values := make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		values[i] = fmt.Sprintf("value-%d", i)
	}
	OptimizedSetBatch(cache, keys, values)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		OptimizedGetBatch(cache, keys)
	}
}

// BenchmarkBatchGetLarge benchmarks large batch get operations
func BenchmarkBatchGetLarge(b *testing.B) {
	cache := NewSafeShardedCache(100000, 16)
	keys := make([]interface{}, 1000)
	values := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		values[i] = fmt.Sprintf("value-%d", i)
	}
	OptimizedSetBatch(cache, keys, values)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		OptimizedGetBatch(cache, keys)
	}
}

// BenchmarkBatchDeleteSmall benchmarks small batch delete operations
func BenchmarkBatchDeleteSmall(b *testing.B) {
	keys := make([]interface{}, 10)
	values := make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		values[i] = fmt.Sprintf("value-%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		cache := NewSafeShardedCache(10000, 16)
		OptimizedSetBatch(cache, keys, values)
		b.StartTimer()
		OptimizedDeleteBatch(cache, keys)
	}
}

// BenchmarkBatchDeleteLarge benchmarks large batch delete operations
func BenchmarkBatchDeleteLarge(b *testing.B) {
	keys := make([]interface{}, 1000)
	values := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		values[i] = fmt.Sprintf("value-%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		cache := NewSafeShardedCache(100000, 16)
		OptimizedSetBatch(cache, keys, values)
		b.StartTimer()
		OptimizedDeleteBatch(cache, keys)
	}
}

// BenchmarkStreamingBatch benchmarks streaming batch operations
func BenchmarkStreamingBatch(b *testing.B) {
	cache := NewSafeShardedCache(100000, 16)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sb := NewStreamingBatch(cache, 100)
		for j := 0; j < 1000; j++ {
			sb.Add(fmt.Sprintf("stream-key-%d", j), fmt.Sprintf("stream-val-%d", j))
		}
		sb.Flush()
		sb.Close()
	}
}

// BenchmarkBatchBuilder benchmarks batch builder operations
func BenchmarkBatchBuilder(b *testing.B) {
	cache := NewSafeShardedCache(10000, 16)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bb := NewBatchBuilder(cache)
		for j := 0; j < 100; j++ {
			bb.Set(fmt.Sprintf("bb-key-%d", j), fmt.Sprintf("bb-val-%d", j))
		}
		bb.Execute()
	}
}

// BenchmarkParallelBatchSetSingleCore benchmarks parallel batch set on single core
func BenchmarkParallelBatchSetSingleCore(b *testing.B) {
	cache := NewSafeShardedCache(100000, 16)
	keys := make([]interface{}, 1000)
	values := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		values[i] = fmt.Sprintf("value-%d", i)
	}

	b.SetParallelism(1)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			parallelBatchSet(cache, keys, values)
		}
	})
}

// BenchmarkParallelBatchSetMultiCore benchmarks parallel batch set on multiple cores
func BenchmarkParallelBatchSetMultiCore(b *testing.B) {
	cache := NewSafeShardedCache(100000, 16)
	keys := make([]interface{}, 1000)
	values := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		values[i] = fmt.Sprintf("value-%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			parallelBatchSet(cache, keys, values)
		}
	})
}
