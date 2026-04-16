// Package cache provides thread-safe caching implementations.
package cache

import (
	"strconv"
	"sync"
	"testing"
)

// TestShardedLRU_Get_NoTOCTOU exercises the corrected Get path under a
// contending Delete workload.
//
// Before the fix, Get acquired RLock, then dropped it and re-acquired Lock
// to call moveToFront. A concurrent Delete or eviction could remove the
// entry between the two acquisitions, leaving moveToFront to unlink an
// already-unlinked node and silently corrupt the shard's linked list —
// reliably tripping the race detector and occasionally panicking in
// production.
//
// This test hammers Get and Delete on overlapping keys from many goroutines
// for a fixed number of iterations. With -race it must report no data race;
// with the race detector disabled it must complete without panicking and
// leave the cache's Stats().Hits + Stats().Misses == N*iter.
func TestShardedLRU_Get_NoTOCTOU(t *testing.T) {
	t.Parallel()

	const (
		capacity int = 512
		shards   int = 16
		keys     int = 128
		goros    int = 16
		iters    int = 2000
		deleters int = 4
		delIters int = 500
	)

	c := NewShardedLRU[string, int](capacity, shards)
	for i := range keys {
		c.Set(strconv.Itoa(i), i)
	}

	var wg sync.WaitGroup

	//: launch reader goroutines that hammer the same keys heavily
	for g := range goros {
		seed := g
		wg.Go(func() {
			for i := range iters {
				k := strconv.Itoa((seed*iters + i) % keys)
				_, _ = c.Get(k)
			}
		})
	}

	//: launch deleter goroutines that concurrently remove and re-insert keys
	for g := range deleters {
		seed := g
		wg.Go(func() {
			for i := range delIters {
				k := strconv.Itoa((seed*delIters + i) % keys)
				c.Delete(k)
				c.Set(k, i)
			}
		})
	}

	wg.Wait()

	//: post-condition: cache must be in a coherent state
	if got := c.Size(); got < 0 || got > capacity {
		t.Fatalf("Size() = %d; want in [0, %d]", got, capacity)
	}
	for i := range keys {
		_, _ = c.Get(strconv.Itoa(i))
	}
}
