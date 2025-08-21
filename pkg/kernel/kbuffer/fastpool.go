package kbuffer

import (
	"sync"
	"unsafe"
)

// fastBufferPool optimizes for zero-allocation buffer pooling.
// Uses a novel approach to avoid pointer allocations.
type fastBufferPool struct {
	pool64  sync.Pool
	pool256 sync.Pool
	pool1k  sync.Pool
	pool4k  sync.Pool
	pool16k sync.Pool
	pool64k sync.Pool
}

// bufferWrapper wraps a byte slice to avoid allocations
type bufferWrapper struct {
	data []byte
}

var fastPool = &fastBufferPool{
	pool64: sync.Pool{
		New: func() any {
			return &bufferWrapper{data: make([]byte, 64)}
		},
	},
	pool256: sync.Pool{
		New: func() any {
			return &bufferWrapper{data: make([]byte, 256)}
		},
	},
	pool1k: sync.Pool{
		New: func() any {
			return &bufferWrapper{data: make([]byte, 1024)}
		},
	},
	pool4k: sync.Pool{
		New: func() any {
			return &bufferWrapper{data: make([]byte, 4096)}
		},
	},
	pool16k: sync.Pool{
		New: func() any {
			return &bufferWrapper{data: make([]byte, 16384)}
		},
	},
	pool64k: sync.Pool{
		New: func() any {
			return &bufferWrapper{data: make([]byte, 65536)}
		},
	},
}

// getFast retrieves a buffer with zero allocations for common sizes.
//
//go:inline
//go:nosplit
func getFast(size int) []byte {
	var wrapper *bufferWrapper

	// Fast path selection based on size
	switch {
	case size <= 64:
		wrapper = fastPool.pool64.Get().(*bufferWrapper)
		return wrapper.data[:size]
	case size <= 256:
		wrapper = fastPool.pool256.Get().(*bufferWrapper)
		return wrapper.data[:size]
	case size <= 1024:
		wrapper = fastPool.pool1k.Get().(*bufferWrapper)
		return wrapper.data[:size]
	case size <= 4096:
		wrapper = fastPool.pool4k.Get().(*bufferWrapper)
		return wrapper.data[:size]
	case size <= 16384:
		wrapper = fastPool.pool16k.Get().(*bufferWrapper)
		return wrapper.data[:size]
	case size <= 65536:
		wrapper = fastPool.pool64k.Get().(*bufferWrapper)
		return wrapper.data[:size]
	default:
		return make([]byte, size)
	}
}

// putFast returns a buffer with zero allocations for common sizes.
//
//go:inline
//go:nosplit
func putFast(buf []byte) {
	if buf == nil {
		return
	}

	capacity := cap(buf)
	// Reset to full capacity
	buf = buf[:capacity]

	// Create wrapper without allocation using unsafe
	wrapper := (*bufferWrapper)(unsafe.Pointer(&buf))

	switch capacity {
	case 64:
		fastPool.pool64.Put(wrapper)
	case 256:
		fastPool.pool256.Put(wrapper)
	case 1024:
		fastPool.pool1k.Put(wrapper)
	case 4096:
		fastPool.pool4k.Put(wrapper)
	case 16384:
		fastPool.pool16k.Put(wrapper)
	case 65536:
		fastPool.pool64k.Put(wrapper)
		// Don't pool other sizes
	}
}

// OptimizedGet uses the fast pool for common sizes, falls back to regular pool.
//
//go:inline
func OptimizedGet(size int) []byte {
	// Use fast pool for common sizes
	if size <= 65536 {
		return getFast(size)
	}
	// Fall back to regular pool for large sizes
	return globalPool.Get(size)
}

// OptimizedPut uses the fast pool for common sizes, falls back to regular pool.
//
//go:inline
func OptimizedPut(buf []byte) {
	if buf == nil {
		return
	}

	capacity := cap(buf)
	// Use fast pool for exact power-of-2 sizes up to 64k
	if capacity == 64 || capacity == 256 || capacity == 1024 ||
		capacity == 4096 || capacity == 16384 || capacity == 65536 {
		putFast(buf)
		return
	}
	// Fall back to regular pool
	globalPool.Put(buf)
}
