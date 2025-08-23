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
)

// BufferPool implements a lock-free, size-classed buffer pool.
// Uses sync.Pool internally with power-of-2 size classes for efficiency.
type BufferPool struct {
	pools [21]*sync.Pool // Power-of-2 pools from 2^6 to 2^20

	// Configuration
	clearOnPut atomic.Bool  // Clear buffers on return
	maxSize    atomic.Int64 // Maximum pooled size
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
//
//go:nosplit
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
//
//go:nosplit
func (p *BufferPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	capacity := cap(buf)

	// Clear if configured (for security) - do this before fast path
	if p.clearOnPut.Load() {
		clear(buf[:cap(buf)])
	}

	// Fast path for common sizes
	switch capacity {
	case 64:
		buf = buf[:64]
		p.pools[0].Put(&buf)
		return
	case 128:
		buf = buf[:128]
		p.pools[1].Put(&buf)
		return
	case 256:
		buf = buf[:256]
		p.pools[2].Put(&buf)
		return
	case 512:
		buf = buf[:512]
		p.pools[3].Put(&buf)
		return
	case 1024:
		buf = buf[:1024]
		p.pools[4].Put(&buf)
		return
	case 2048:
		buf = buf[:2048]
		p.pools[5].Put(&buf)
		return
	case 4096:
		buf = buf[:4096]
		p.pools[6].Put(&buf)
		return
	}

	// Don't pool oversized or non-power-of-2 buffers
	if capacity > int(p.maxSize.Load()) || !isPowerOf2(capacity) {
		return
	}

	// Calculate pool index for larger sizes
	class := bits.Len(uint(capacity)) - 1
	poolIdx := class - 6

	if poolIdx < 0 || poolIdx >= len(p.pools) {
		return
	}

	// Reset to full capacity and return to pool
	buf = buf[:capacity]
	p.pools[poolIdx].Put(&buf)
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
