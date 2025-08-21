package kbuffer

// PoolStats contains pool usage statistics.
// All fields are read-only snapshots of atomic counters.
type PoolStats struct {
	// Gets is the total number of Get operations.
	Gets uint64

	// Puts is the total number of Put operations.
	Puts uint64

	// Allocs is the total number of new allocations.
	Allocs uint64

	// Hits is the number of successful pool retrievals.
	Hits uint64

	// Misses is the number of pool misses requiring allocation.
	Misses uint64
}

// HitRate returns the pool hit rate as a percentage.
func (s PoolStats) HitRate() float64 {
	if s.Gets == 0 {
		return 0
	}
	return float64(s.Hits) / float64(s.Gets) * 100
}

// AllocRate returns the allocation rate as a percentage.
func (s PoolStats) AllocRate() float64 {
	if s.Gets == 0 {
		return 0
	}
	return float64(s.Allocs) / float64(s.Gets) * 100
}
