package kbuffer

import (
	"math/bits"
	"runtime"
	"sync"
	"sync/atomic"
)

// Pool size class boundaries for efficient memory allocation.
// Uses power-of-2 sizes to minimize fragmentation and improve cache locality.
const (
	minPoolSize = 64      // Minimum pooled buffer size
	maxPoolSize = 1 << 20 // Maximum pooled buffer size (1MB)
)

// BufferPool implements a lock-free, size-classed buffer pool for high-performance buffer reuse.
//
// The pool uses sync.Pool internally with power-of-2 size classes for efficiency:
//   - Reduces GC pressure by reusing buffers
//   - Minimizes memory fragmentation with size classes
//   - Provides lock-free access to pooled buffers
//   - Supports configuration for security-sensitive use cases
//
// Size classes range from 64 bytes (2^6) to 1MB (2^20), providing efficient
// allocation for a wide range of buffer sizes commonly used in kernel operations.
type BufferPool struct {
	pools [21]*sync.Pool // Power-of-2 pools from 2^6 to 2^20

	// Configuration
	clearOnPut atomic.Bool  // Clear buffers on return
	maxSize    atomic.Int64 // Maximum pooled size
}

// globalPool is the singleton pool instance used by package-level functions.
// Initialized once at package load time and shared across all users of the package.
var globalPool = newPool()

// newPool creates and initializes a new BufferPool with default configuration.
//
// The pool is initialized with:
//   - 21 size classes from 64 bytes to 1MB
//   - Pre-warmed pools for common sizes
//   - Maximum size limit of 1MB
//   - Buffer clearing disabled by default for performance
//
// Returns a fully initialized BufferPool ready for use.
func newPool() *BufferPool {
	p := &BufferPool{}
	p.maxSize.Store(maxPoolSize)

	// Initialize pools for each size class
	for i := range p.pools {
		size := 1 << (i + 6) // 2^6 to 2^20
		p.pools[i] = &sync.Pool{
			New: func(sz int) func() any {
				return func() any {
					buf := make([]byte, sz)
					return &buf
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
func (p *BufferPool) Get(size int) []byte {
	if size <= 0 {
		return nil
	}

	// Fast path for common sizes
	if size <= 256 {
		var bufPtr *[]byte
		if size <= 64 {
			bufPtr = p.pools[0].Get().(*[]byte)
		} else if size <= 128 {
			bufPtr = p.pools[1].Get().(*[]byte)
		} else {
			bufPtr = p.pools[2].Get().(*[]byte)
		}
		return (*bufPtr)[:size]
	}

	// Direct allocation for oversized buffers
	if size > int(p.maxSize.Load()) {
		return make([]byte, size)
	}

	// Calculate size class
	class := sizeClass(size)
	poolIdx := class - 6

	if poolIdx < 0 || poolIdx >= len(p.pools) {
		return make([]byte, size)
	}

	// Get from pool
	bufPtr := p.pools[poolIdx].Get().(*[]byte)
	return (*bufPtr)[:size]
}

// Put returns a buffer to the pool for reuse.
// The buffer capacity must be a power of 2.
func (p *BufferPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	capacity := cap(buf)

	// Clear if configured (for security)
	if p.clearOnPut.Load() {
		clear(buf[:capacity])
	}

	// Try fast path first
	if poolIdx := p.tryFastPath(buf, capacity); poolIdx >= 0 {
		return
	}

	// Slow path for other sizes
	p.putSlowPath(buf, capacity)
}

func (p *BufferPool) tryFastPath(buf []byte, capacity int) int {
	// Fast lookup table for common sizes
	switch capacity {
	case 64:
		buf = buf[:64]
		p.pools[0].Put(&buf)
		return 0
	case 128:
		buf = buf[:128]
		p.pools[1].Put(&buf)
		return 1
	case 256:
		buf = buf[:256]
		p.pools[2].Put(&buf)
		return 2
	case 512:
		buf = buf[:512]
		p.pools[3].Put(&buf)
		return 3
	case 1024:
		buf = buf[:1024]
		p.pools[4].Put(&buf)
		return 4
	case 2048:
		buf = buf[:2048]
		p.pools[5].Put(&buf)
		return 5
	case 4096:
		buf = buf[:4096]
		p.pools[6].Put(&buf)
		return 6
	default:
		return -1
	}
}

func (p *BufferPool) putSlowPath(buf []byte, capacity int) {
	// Check size constraints
	if !p.isPoolable(capacity) {
		return
	}

	// Calculate pool index
	poolIdx := p.getPoolIndex(capacity)
	if poolIdx < 0 || poolIdx >= len(p.pools) {
		return
	}

	// Reset to full capacity and return to pool
	buf = buf[:capacity]
	p.pools[poolIdx].Put(&buf)
}

func (p *BufferPool) isPoolable(capacity int) bool {
	return capacity <= int(p.maxSize.Load()) && isPowerOf2(capacity)
}

func (p *BufferPool) getPoolIndex(capacity int) int {
	return bits.Len(uint(capacity)) - 1 - 6
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
	sizes := []int{256, 1024, 4096, 16384, 65536}

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
			bufCopy := buf
			p.pools[poolIdx].Put(&bufCopy)
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
