package kcache

import (
	"fmt"
	"testing"
)

// TestUnsafeCache_BasicOperations tests all basic cache operations
func TestUnsafeCache_BasicOperations(t *testing.T) {
	c := NewUnsafeCache(100)

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

// TestUnsafeCache_NilKey tests handling of nil keys
func TestUnsafeCache_NilKey(t *testing.T) {
	c := NewUnsafeCache(100)

	// Set with nil key
	if c.Set(nil, "value") {
		t.Error("Set should return false for nil key")
	}

	// Get with nil key
	v, ok := c.Get(nil)
	if ok {
		t.Error("Get should return false for nil key")
	}
	if v != nil {
		t.Error("Get should return nil value for nil key")
	}

	// Has with nil key
	if c.Has(nil) {
		t.Error("Has should return false for nil key")
	}

	// Delete with nil key
	if c.Delete(nil) {
		t.Error("Delete should return false for nil key")
	}
}

// TestUnsafeCache_Resize tests automatic resizing
func TestUnsafeCache_Resize(t *testing.T) {
	c := NewUnsafeCache(16) // Small initial capacity

	// Add more items than initial capacity
	for i := 0; i < 32; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		c.Set(key, value)
	}

	// Verify all items are still accessible
	for i := 0; i < 32; i++ {
		key := fmt.Sprintf("key%d", i)
		expected := fmt.Sprintf("value%d", i)
		v, ok := c.Get(key)
		if !ok || v != expected {
			t.Errorf("After resize, Get(%s) = %v, %v; want %s, true", key, v, ok, expected)
		}
	}

	// Check capacity increased
	if c.Cap() <= 16 {
		t.Errorf("Capacity should have increased from 16, got %d", c.Cap())
	}
}

// TestUnsafeCache_BatchOperations tests batch operations
func TestUnsafeCache_BatchOperations(t *testing.T) {
	c := NewUnsafeCache(100).(BatchCache)

	keys := []interface{}{"k1", "k2", "k3"}
	values := []interface{}{"v1", "v2", "v3"}

	// Test SetBatch
	count := c.SetBatch(keys, values)
	if count != 3 {
		t.Errorf("SetBatch: expected 3 new keys, got %d", count)
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

// TestUnsafeCache_PanicOnConcurrentAccess verifies concurrent access detection
func TestUnsafeCache_PanicOnConcurrentAccess(t *testing.T) {
	// Skip in production mode where checks are disabled
	if testingSkipSafetyCheck {
		t.Skip("Safety checks disabled in production mode")
	}

	c := NewUnsafeCache(100)

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
		t.Error("Expected panic for concurrent access to unsafe cache")
	}
}

// BenchmarkUnsafeCache_Set benchmarks Set operation
func BenchmarkUnsafeCache_Set(b *testing.B) {
	c := NewUnsafeCache(10000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Set(i, i)
	}
}

// BenchmarkUnsafeCache_Get benchmarks Get operation
func BenchmarkUnsafeCache_Get(b *testing.B) {
	c := NewUnsafeCache(10000)

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

// BenchmarkUnsafeCache_Mixed benchmarks mixed operations
func BenchmarkUnsafeCache_Mixed(b *testing.B) {
	c := NewUnsafeCache(10000)

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
