// Put returns a Buffer to the appropriate pool based on its capacity.
package buffer

import (
	"sync"
)

const (
	MinBitSize = 1  // Smallest size is 2**1 = 2 bytes
	MaxBitSize = 20 // Largest size is 2**20 = 1 MB
)

// BufferPool manages a pool of Buffers of different sizes.
type BufferPool struct {
	pools map[int]*sync.Pool
	mu    sync.Mutex // Protects pools for lazy initialization
}

// NewBufferPool creates a new BufferPool.
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pools: make(map[int]*sync.Pool),
	}
}

// Get retrieves a Buffer from the pool that best fits the requested size.
func (p *BufferPool) Get(size int) *Buffer {
	if size < (1<<MinBitSize) || size > (1<<MaxBitSize) || (size&(size-1)) != 0 {
		panic("requested size must be a power of 2 within valid bounds")
	}

	// Protect access to the pools map
	p.mu.Lock()
	pool, exists := p.pools[size]
	if !exists {
		pool = &sync.Pool{
			New: func() any {
				return NewBuffer(size)
			},
		}
		p.pools[size] = pool
	}
	p.mu.Unlock()

	return pool.Get().(*Buffer)
}

// Put returns a Buffer to the appropriate pool based on its capacity.
func (p *BufferPool) Put(buf *Buffer) {
	if buf.Cap() < (1<<MinBitSize) || buf.Cap() > (1<<MaxBitSize) || (buf.Cap()&(buf.Cap()-1)) != 0 {
		panic("buffer capacity must be a power of 2 within valid bounds")
	}

	// Protect access to the pools map
	p.mu.Lock()
	pool := p.pools[buf.Cap()]
	p.mu.Unlock()

	buf.Free()
	pool.Put(buf)
}
