package kbuffer

import (
	"math/bits"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	// MinBitSize is the minimum buffer size as a power of 2 (2^1 = 2 bytes).
	MinBitSize = 1
	// MaxBitSize is the maximum buffer size as a power of 2 (2^20 = 1MB).
	MaxBitSize = 20
)

// BufferPool is a buffer pool that supports any size.
// It automatically rounds up to the nearest power of 2 for pooling.
// The pool uses sync.Map and includes statistics tracking.
type BufferPool struct {
	pools      sync.Map // Use sync.Map for lock-free reads
	stats      PoolStats
	maxSize    int
	clearOnPut bool // Whether to clear buffers on return
}

// PoolStats tracks pool usage statistics.
// All fields are atomically updated.
type PoolStats struct {
	Gets      int64
	Puts      int64
	Allocs    int64
	Hits      int64
	Misses    int64
	BytesUsed int64
}

// GlobalPool is the default enhanced buffer pool instance.
var GlobalPool = initGlobalPool()

func initGlobalPool() *BufferPool {
	pool := NewBufferPool()
	pool.SetClearOnPut(false)
	// Pre-warm with common sizes
	pool.Prewarm([]int{256, 512, 1024, 4096, 8192, 16384, 65536}, 10)
	return pool
}

// NewBufferPool creates a new buffer pool.
func NewBufferPool() *BufferPool {
	p := &BufferPool{
		maxSize:    1 << MaxBitSize,
		clearOnPut: false,
	}
	// Pre-allocate pools for common sizes
	p.initializePools()
	return p
}

// initializePools pre-allocates pools for common buffer sizes.
func (p *BufferPool) initializePools() {
	// Pre-create pools for common sizes to avoid runtime allocation
	commonSizes := []int{256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536}
	for _, size := range commonSizes {
		pool := &sync.Pool{
			New: nil,
		}
		p.pools.Store(size, pool)
	}
}

// Get retrieves a buffer of at least the requested size.
// It automatically rounds up to the nearest power of 2.
func (p *BufferPool) Get(size int) []byte {
	atomic.AddInt64(&p.stats.Gets, 1)

	if size <= 0 {
		return nil
	}

	// For very large sizes, allocate directly without pooling
	if size > p.maxSize {
		atomic.AddInt64(&p.stats.Allocs, 1)
		atomic.AddInt64(&p.stats.Misses, 1)
		return make([]byte, size)
	}

	// Round up to next power of 2
	poolSize := nextPowerOf2(size)

	// Get pool for this size
	pool := p.getPool(poolSize)

	// Try to get from pool
	if buf := pool.Get(); buf != nil {
		atomic.AddInt64(&p.stats.Hits, 1)
		b := buf.([]byte)
		// Return a slice of the requested size
		if len(b) >= size {
			return b[:size]
		}
		// Buffer too small, return it and allocate new
		pool.Put(b) //nolint:staticcheck // sync.Pool accepts interface{}
	}

	// Allocate new buffer
	atomic.AddInt64(&p.stats.Allocs, 1)
	atomic.AddInt64(&p.stats.Misses, 1)
	return make([]byte, size, poolSize)
}

// GetBuffer retrieves a Buffer object from the pool.
func (p *BufferPool) GetBuffer(size int) *Buffer {
	if size <= 0 {
		size = 1 << MinBitSize
	}

	// Round to power of 2 for compatibility with existing Buffer type
	poolSize := nextPowerOf2(size)

	// Get or create pool for Buffer objects
	if v, ok := p.pools.Load(poolSize); ok {
		pool := v.(*sync.Pool)

		if buf := pool.Get(); buf != nil {
			return buf.(*Buffer)
		}
		return NewBuffer(poolSize)
	}

	// Create new pool for Buffer objects
	pool := &sync.Pool{
		New: func() any {
			return NewBuffer(poolSize)
		},
	}
	actual, _ := p.pools.LoadOrStore(poolSize, pool)
	actualPool := actual.(*sync.Pool)

	if buf := actualPool.Get(); buf != nil {
		return buf.(*Buffer)
	}
	return NewBuffer(poolSize)
}

// Put returns a buffer to the pool for reuse.
func (p *BufferPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	atomic.AddInt64(&p.stats.Puts, 1)

	capacity := cap(buf)

	// Don't pool very large buffers
	if capacity > p.maxSize {
		return
	}

	// Only pool if capacity is a power of 2
	if !isPowerOf2(capacity) {
		return
	}

	// Clear the buffer if enabled (for security)
	if p.clearOnPut {
		clear(buf)
	}

	// Reset slice to full capacity
	buf = buf[:capacity]

	// Get pool and put buffer back
	pool := p.getPool(capacity)
	pool.Put(buf) //nolint:staticcheck // sync.Pool accepts interface{}
}

// PutBuffer returns a Buffer object to the pool.
func (p *BufferPool) PutBuffer(buf *Buffer) {
	if buf == nil {
		return
	}

	capacity := buf.Cap()

	// Validate capacity
	if capacity < (1<<MinBitSize) || capacity > (1<<MaxBitSize) || !isPowerOf2(capacity) {
		return
	}

	// Reset buffer
	buf.Free()

	// Get pool and put buffer back
	if v, ok := p.pools.Load(capacity); ok {
		pool := v.(*sync.Pool)
		pool.Put(buf)
		atomic.AddInt64(&p.stats.Puts, 1)
	}
}

// GetStats returns current pool statistics.
func (p *BufferPool) GetStats() PoolStats {
	return PoolStats{
		Gets:      atomic.LoadInt64(&p.stats.Gets),
		Puts:      atomic.LoadInt64(&p.stats.Puts),
		Allocs:    atomic.LoadInt64(&p.stats.Allocs),
		Hits:      atomic.LoadInt64(&p.stats.Hits),
		Misses:    atomic.LoadInt64(&p.stats.Misses),
		BytesUsed: atomic.LoadInt64(&p.stats.BytesUsed),
	}
}

// ResetStats resets the pool statistics.
func (p *BufferPool) ResetStats() {
	atomic.StoreInt64(&p.stats.Gets, 0)
	atomic.StoreInt64(&p.stats.Puts, 0)
	atomic.StoreInt64(&p.stats.Allocs, 0)
	atomic.StoreInt64(&p.stats.Hits, 0)
	atomic.StoreInt64(&p.stats.Misses, 0)
	atomic.StoreInt64(&p.stats.BytesUsed, 0)
}

// SetMaxSize sets the maximum buffer size that will be pooled.
func (p *BufferPool) SetMaxSize(size int) {
	atomic.StoreInt64((*int64)(unsafe.Pointer(&p.maxSize)), int64(size))
}

// SetClearOnPut sets whether buffers should be cleared when returned to pool.
func (p *BufferPool) SetClearOnPut(clear bool) {
	p.clearOnPut = clear
}

// Prewarm pre-allocates buffers of common sizes.
func (p *BufferPool) Prewarm(sizes []int, count int) {
	for _, size := range sizes {
		poolSize := nextPowerOf2(size)
		pool := p.getPool(poolSize)

		// Pre-allocate buffers
		for i := 0; i < count; i++ {
			buf := make([]byte, poolSize)
			pool.Put(buf) //nolint:staticcheck // sync.Pool accepts interface{}
			atomic.AddInt64(&p.stats.Allocs, 1)
		}
	}
}

// getPool retrieves or creates a pool for the given size.
func (p *BufferPool) getPool(size int) *sync.Pool {
	// Try to load existing pool
	if v, ok := p.pools.Load(size); ok {
		return v.(*sync.Pool)
	}

	// Create new pool if not exists
	pool := &sync.Pool{
		New: nil,
	}

	// Store and return (LoadOrStore for race safety)
	actual, _ := p.pools.LoadOrStore(size, pool)
	return actual.(*sync.Pool)
}

// nextPowerOf2 returns the next power of 2 greater than or equal to n.
func nextPowerOf2(n int) int {
	if n <= 1 {
		return 1
	}
	if n > (1 << 30) {
		return 1 << 30
	}
	// Fast path for already power of 2
	if n&(n-1) == 0 {
		return n
	}
	return 1 << bits.Len(uint(n-1))
}

// isPowerOf2 checks if n is a power of 2.
func isPowerOf2(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

// Convenience functions for common buffer sizes

// Get64K gets a 64KB 
func (p *BufferPool) Get64K() []byte {
	return p.Get(64 * 1024)
}

// Get4K gets a 4KB 
func (p *BufferPool) Get4K() []byte {
	return p.Get(4 * 1024)
}

// Get1K gets a 1KB 
func (p *BufferPool) Get1K() []byte {
	return p.Get(1024)
}

// Global convenience functions

// Get retrieves a buffer from the global pool.
func Get(size int) []byte {
	return GlobalPool.Get(size)
}

// Put returns a buffer to the global pool.
func Put(buf []byte) {
	GlobalPool.Put(buf)
}

// GetBuffer retrieves a Buffer object from the global pool.
func GetBuffer(size int) *Buffer {
	return GlobalPool.GetBuffer(size)
}

// PutBuffer returns a Buffer object to the global pool.
func PutBuffer(buf *Buffer) {
	GlobalPool.PutBuffer(buf)
}
