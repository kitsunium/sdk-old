package kcache

import (
	"sync"
	"testing"
)

// TestSafeShardedCacheRaceOnShardAccess specifically tests for race conditions
// when accessing the shards array itself (not the data in shards)
func TestSafeShardedCacheRaceOnShardAccess(t *testing.T) {
	// Create cache with many shards to increase chance of detecting issues
	cache := NewSafeShardedCache(10000, 32)

	// Launch many goroutines that will access different shards simultaneously
	var wg sync.WaitGroup
	numGoroutines := 1000
	opsPerGoroutine := 10000

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Each goroutine accesses many different shards
			for j := 0; j < opsPerGoroutine; j++ {
				// Different keys will map to different shards
				key := id*opsPerGoroutine + j

				// These operations all call getShard() internally
				// If there was a race on shard array access, it would be detected
				cache.Set(key, key*2)
				cache.Get(key)
				cache.Has(key)
				if j%100 == 0 {
					cache.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()

	// If we get here without race detector complaining, the shard access is safe
	t.Logf("Successfully completed %d concurrent operations across shards",
		numGoroutines*opsPerGoroutine*3)
}

// TestSafeShardedCacheNoMutationOfShards verifies that shards array is never mutated
func TestSafeShardedCacheNoMutationOfShards(t *testing.T) {
	cache := newSafeShardedCache(1000, 16)

	// Capture initial shard count
	initialShardCount := len(cache.shards)
	initialShardMask := cache.shardMask

	// Capture pointers to all shards
	shardPointers := make([]*safeShard, len(cache.shards))
	for i := range cache.shards {
		shardPointers[i] = &cache.shards[i]
	}

	// Perform many operations
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				cache.Set(id*1000+j, j)
				cache.Get(id*1000 + j)
			}
		}(i)
	}
	wg.Wait()

	// Verify shards array was never modified
	if len(cache.shards) != initialShardCount {
		t.Errorf("Shard count changed from %d to %d", initialShardCount, len(cache.shards))
	}

	if cache.shardMask != initialShardMask {
		t.Errorf("Shard mask changed from %d to %d", initialShardMask, cache.shardMask)
	}

	// Verify shard pointers are still the same (no reallocation)
	for i := range cache.shards {
		if &cache.shards[i] != shardPointers[i] {
			t.Errorf("Shard %d pointer changed, array was reallocated!", i)
		}
	}

	t.Log("✅ Confirmed: shards array is immutable after creation")
}
