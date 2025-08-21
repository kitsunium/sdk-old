package kbuffer

import (
	"math/bits"
	"runtime"
	"sync"
	"sync/atomic"
)

// Pool size class boundaries
const (
	minPoolSize = 64      // Minimum pooled buffer size
	maxPoolSize = 1 << 20 // Maximum pooled buffer size (1MB)

	// Size classes for efficient pooling
	class64B  = 64
	class256B = 256
	class1K   = 1024
	class4K   = 4096
	class16K  = 16384
	class64K  = 65536
	class256K = 262144
	class1M   = 1048576
)

// BufferPool implements a lock-free, size-classed buffer pool.
// Uses sync.Pool internally with power-of-2 size classes for efficiency.
type BufferPool struct {
	pools [21]*sync.Pool // Power-of-2 pools from 2^6 to 2^20
	stats poolStats      // Atomic statistics

	// Configuration
	clearOnPut atomic.Bool  // Clear buffers on return
	maxSize    atomic.Int64 // Maximum pooled size
}

// poolStats tracks pool usage with atomic counters.
type poolStats struct {
	gets   atomic.Uint64
	puts   atomic.Uint64
	allocs atomic.Uint64
	hits   atomic.Uint64
	misses atomic.Uint64
}

// globalPool is the singleton pool instance.
var globalPool = newPool()

func newPool() *BufferPool {
	p := &BufferPool{}
	p.maxSize.Store(maxPoolSize)

	// Initialize pools for each size class
	for i := range p.pools {
		size := 1 << (i + 6) // 2^6 to 2^20
		p.pools[i] = &sync.Pool{
			New: func(sz int) func() any {
				return func() any {
					p.stats.allocs.Add(1)
					return make([]byte, sz)
				}
			}(size),
		}
	}

	// Pre-warm common sizes
	p.prewarm()

	return p
}

// Get retrieves a buffer of at least the requested size.
// The returned buffer may be larger than requested.
//
//go:nosplit
func (p *BufferPool) Get(size int) []byte {
	p.stats.gets.Add(1)

	if size <= 0 {
		return nil
	}

	// Direct allocation for oversized buffers
	if size > int(p.maxSize.Load()) {
		p.stats.misses.Add(1)
		p.stats.allocs.Add(1)
		return make([]byte, size)
	}

	// Calculate size class
	class := sizeClass(size)
	poolIdx := class - 6

	if poolIdx < 0 || poolIdx >= len(p.pools) {
		p.stats.misses.Add(1)
		p.stats.allocs.Add(1)
		return make([]byte, size)
	}

	// Get from pool
	buf := p.pools[poolIdx].Get().([]byte)
	p.stats.hits.Add(1)

	// Return slice of requested size
	return buf[:size]
}

// Put returns a buffer to the pool for reuse.
// The buffer capacity must be a power of 2.
//
//go:nosplit
func (p *BufferPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	p.stats.puts.Add(1)

	capacity := cap(buf)

	// Don't pool oversized or non-power-of-2 buffers
	if capacity > int(p.maxSize.Load()) || !isPowerOf2(capacity) {
		return
	}

	// Calculate pool index
	class := bits.Len(uint(capacity)) - 1
	poolIdx := class - 6

	if poolIdx < 0 || poolIdx >= len(p.pools) {
		return
	}

	// Clear if configured (for security)
	if p.clearOnPut.Load() {
		clear(buf)
	}

	// Reset to full capacity and return to pool
	buf = buf[:capacity]
	p.pools[poolIdx].Put(buf)
}

// GetBuffer retrieves a Buffer from the pool.
//
//go:inline
func (p *BufferPool) GetBuffer(size int) *Buffer {
	buf := p.Get(size)
	if buf == nil {
		return NewBuffer(size)
	}

	// Create new buffer with pooled backing
	return &Buffer{
		data: buf,
		cap:  int32(cap(buf)),
		pos:  0,
	}
}

// PutBuffer returns a Buffer to the pool.
//
//go:inline
func (p *BufferPool) PutBuffer(b *Buffer) {
	if b == nil {
		return
	}
	b.Reset()
	p.Put(b.data)
}

// Stats returns current pool statistics.
func (p *BufferPool) Stats() PoolStats {
	return PoolStats{
		Gets:   p.stats.gets.Load(),
		Puts:   p.stats.puts.Load(),
		Allocs: p.stats.allocs.Load(),
		Hits:   p.stats.hits.Load(),
		Misses: p.stats.misses.Load(),
	}
}

// ResetStats resets all statistics counters to zero.
func (p *BufferPool) ResetStats() {
	p.stats.gets.Store(0)
	p.stats.puts.Store(0)
	p.stats.allocs.Store(0)
	p.stats.hits.Store(0)
	p.stats.misses.Store(0)
}

// SetClearOnPut configures whether buffers are cleared when returned.
//
//go:inline
func (p *BufferPool) SetClearOnPut(clear bool) {
	p.clearOnPut.Store(clear)
}

// SetMaxSize sets the maximum buffer size that will be pooled.
//
//go:inline
func (p *BufferPool) SetMaxSize(size int64) {
	p.maxSize.Store(size)
}

// prewarm pre-allocates buffers for common sizes.
func (p *BufferPool) prewarm() {
	// Pre-warm with common sizes based on CPU count
	numCPU := runtime.NumCPU()
	sizes := []int{class256B, class1K, class4K, class16K, class64K}

	for _, size := range sizes {
		class := sizeClass(size)
		poolIdx := class - 6
		if poolIdx < 0 || poolIdx >= len(p.pools) {
			continue
		}

		// Pre-allocate buffers
		bufs := make([][]byte, numCPU*2)
		for i := range bufs {
			bufs[i] = make([]byte, 1<<class)
		}

		// Return to pool
		for _, buf := range bufs {
			p.pools[poolIdx].Put(buf)
		}
	}
}

// sizeClass returns the size class (power of 2 exponent) for a given size.
//
//go:inline
//go:nosplit
func sizeClass(size int) int {
	if size <= minPoolSize {
		return 6 // 2^6 = 64
	}
	return bits.Len(uint(size - 1))
}

// isPowerOf2 checks if n is a power of 2.
//
//go:inline
//go:nosplit
func isPowerOf2(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

// nextPowerOf2 returns the next power of 2 >= n.
//
//go:inline
//go:nosplit
func nextPowerOf2(n int) int {
	if n <= 1 {
		return 1
	}
	if n > maxPoolSize {
		return maxPoolSize
	}
	return 1 << bits.Len(uint(n-1))
}

// Global pool functions

// Get retrieves a buffer from the global pool.
//
//go:inline
func Get(size int) []byte {
	return globalPool.Get(size)
}

// Put returns a buffer to the global pool.
//
//go:inline
func Put(buf []byte) {
	globalPool.Put(buf)
}

// GetBuffer retrieves a Buffer from the global pool.
//
//go:inline
func GetBuffer(size int) *Buffer {
	return globalPool.GetBuffer(size)
}

// PutBuffer returns a Buffer to the global pool.
//
//go:inline
func PutBuffer(b *Buffer) {
	globalPool.PutBuffer(b)
}

// Stats returns global pool statistics.
//
//go:inline
func Stats() PoolStats {
	return globalPool.Stats()
}

// ResetStats resets global pool statistics.
//
//go:inline
func ResetStats() {
	globalPool.ResetStats()
}
