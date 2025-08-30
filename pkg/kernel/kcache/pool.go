// Package kcache provides cache implementations with configurable thread safety.
// This file contains object pooling utilities for reducing allocation overhead.
package kcache

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// EntryPool manages a pool of reusable entry objects.
// Each cache instance can create its own pool to reduce allocation pressure.
type EntryPool struct {
	pool    *sync.Pool   // Underlying sync.Pool
	size    atomic.Int64 // Current pool size estimate
	maxSize int64        // Maximum pool size
	_       [32]byte     // Cache line padding
}

// NewEntryPool creates a new entry object pool.
// The pool pre-allocates entries to avoid startup allocation spike.
func NewEntryPool() *EntryPool {
	ep := &EntryPool{
		maxSize: DefaultPoolSize,
	}

	// Set pool with proper New function
	ep.pool = &sync.Pool{
		New: func() interface{} {
			// Allocate new entry with zero values
			return &entry{}
		},
	}

	// Pre-warm pool with initial entries
	ep.prewarm(MinCapacity)

	return ep
}

// prewarm pre-allocates entries to avoid startup latency.
func (ep *EntryPool) prewarm(count int) {
	entries := make([]*entry, count)
	for i := range entries {
		entries[i] = &entry{}
	}

	// Put entries into pool
	for _, e := range entries {
		ep.pool.Put(e)
	}

	ep.size.Store(int64(count))
}

// Get retrieves an entry from the pool or creates a new one.
// The entry is reset to zero values for safety.
//
//go:inline
func (ep *EntryPool) Get() *entry {
	// Get from pool
	if obj := ep.pool.Get(); obj != nil {
		e := obj.(*entry)
		// Reset to zero values
		e.key = nil
		e.value = nil
		e.hash = 0
		e.state = StateEmpty
		return e
	}

	// Pool was empty, allocate new
	return &entry{}
}

// Put returns an entry to the pool for reuse.
// Sensitive data is cleared before pooling.
//
//go:inline
func (ep *EntryPool) Put(e *entry) {
	// Don't pool if at capacity
	if ep.size.Load() >= ep.maxSize {
		return
	}

	// Clear entry for reuse
	e.key = nil
	e.value = nil
	e.hash = 0
	e.state = StateEmpty

	// Return to pool
	ep.pool.Put(e)
	ep.size.Add(1)
}

// KeyPool manages string key allocations.
// Optimizes for common key size patterns.
type KeyPool struct {
	small  *sync.Pool // Pool for small strings (<= 64 bytes)
	medium *sync.Pool // Pool for medium strings (<= 256 bytes)
	large  *sync.Pool // Pool for large strings (<= 1024 bytes)
	_      [32]byte   // Cache line padding
}

// NewKeyPool creates a pool for string keys.
// Separates pools by size for better memory efficiency.
func NewKeyPool() *KeyPool {
	return &KeyPool{
		small: &sync.Pool{
			New: func() interface{} {
				b := make([]byte, 64)
				return &b
			},
		},
		medium: &sync.Pool{
			New: func() interface{} {
				b := make([]byte, 256)
				return &b
			},
		},
		large: &sync.Pool{
			New: func() interface{} {
				b := make([]byte, 1024)
				return &b
			},
		},
	}
}

// GetBuffer retrieves a buffer of appropriate size.
//
//go:inline
func (kp *KeyPool) GetBuffer(size int) *[]byte {
	switch {
	case size <= 64:
		return kp.small.Get().(*[]byte)
	case size <= 256:
		return kp.medium.Get().(*[]byte)
	case size <= 1024:
		return kp.large.Get().(*[]byte)
	default:
		// Too large for pool, allocate directly
		b := make([]byte, size)
		return &b
	}
}

// PutBuffer returns a buffer to the appropriate pool.
//
//go:inline
func (kp *KeyPool) PutBuffer(buf *[]byte) {
	size := cap(*buf)
	// Reset length but keep capacity
	*buf = (*buf)[:0]

	switch {
	case size <= 64:
		kp.small.Put(buf)
	case size <= 256:
		kp.medium.Put(buf)
	case size <= 1024:
		kp.large.Put(buf)
	default:
		// Too large, let GC handle it
	}
}

// NodePool manages hash table node allocations.
// Used for collision chain nodes in hash table implementations.
type NodePool struct {
	pool    *sync.Pool
	maxSize int64
	size    atomic.Int64
	_       [40]byte // Cache line padding
}

// node represents a collision chain node.
type node struct {
	key   interface{}
	value interface{}
	hash  uint64
	next  *node
}

// NewNodePool creates a pool for collision chain nodes.
func NewNodePool() *NodePool {
	return &NodePool{
		maxSize: MaxPoolSize,
		pool: &sync.Pool{
			New: func() interface{} {
				return &node{}
			},
		},
	}
}

// Get retrieves a node from the pool.
//
//go:inline
func (np *NodePool) Get() *node {
	if obj := np.pool.Get(); obj != nil {
		n := obj.(*node)
		// Reset node
		n.key = nil
		n.value = nil
		n.hash = 0
		n.next = nil
		return n
	}
	return &node{}
}

// Put returns a node to the pool.
//
//go:inline
func (np *NodePool) Put(n *node) {
	if np.size.Load() >= np.maxSize {
		return
	}

	// Clear node
	n.key = nil
	n.value = nil
	n.hash = 0
	n.next = nil

	np.pool.Put(n)
	np.size.Add(1)
}

// Allocator provides custom memory allocation strategies.
// Can be used to implement arena allocation for specific use cases.
type Allocator struct {
	arena    []byte           // Memory arena
	offset   uintptr          // Current allocation offset
	size     uintptr          // Arena size
	fallback func(int) []byte // Fallback allocator
	_        [32]byte         // Cache line padding
}

// NewAllocator creates a custom allocator with arena.
// The arena size determines how much memory is pre-allocated.
func NewAllocator(size int) *Allocator {
	return &Allocator{
		arena:  make([]byte, size),
		size:   uintptr(size),
		offset: 0,
		fallback: func(n int) []byte {
			return make([]byte, n)
		},
	}
}

// Alloc allocates memory from the arena.
// Falls back to standard allocator if arena is exhausted.
//
//go:inline
//go:nosplit
func (a *Allocator) Alloc(size int) []byte {
	// Save original size
	origSize := size
	// Align to 8 bytes for memory alignment
	alignedSize := (size + 7) &^ 7

	// Check if fits in arena
	newOffset := atomic.AddUintptr(&a.offset, uintptr(alignedSize))
	if newOffset > a.size {
		// Arena exhausted, use fallback
		return a.fallback(origSize)
	}

	// Allocate from arena, but return slice with original size
	start := newOffset - uintptr(alignedSize)
	return a.arena[start : start+uintptr(origSize) : start+uintptr(alignedSize)]
}

// Reset resets the allocator for reuse.
// Clears the arena for security.
func (a *Allocator) Reset() {
	atomic.StoreUintptr(&a.offset, 0)
	// Clear arena for security
	for i := range a.arena {
		a.arena[i] = 0
	}
}

// BatchPool pools batch operation buffers.
// Reduces allocation overhead for batch operations.
type BatchPool struct {
	pool *sync.Pool
}

// NewBatchPool creates a pool for batch buffers.
func NewBatchPool() *BatchPool {
	return &BatchPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return &batchBuffer{
					keys:   make([]interface{}, 0, DefaultBatchSize),
					values: make([]interface{}, 0, DefaultBatchSize),
					hashes: make([]uint64, 0, DefaultBatchSize),
					found:  make([]bool, 0, DefaultBatchSize),
				}
			},
		},
	}
}

// Get retrieves a batch buffer from the pool.
//
//go:inline
func (bp *BatchPool) Get() *batchBuffer {
	buf := bp.pool.Get().(*batchBuffer)
	buf.reset()
	return buf
}

// Put returns a batch buffer to the pool.
//
//go:inline
func (bp *BatchPool) Put(buf *batchBuffer) {
	buf.reset()
	bp.pool.Put(buf)
}

// HandleMemoryPressure can be called to release pooled objects.
// This is useful when the application detects high memory usage.
// Each pool instance should implement its own memory pressure handling.
func HandleMemoryPressure(pools ...*sync.Pool) {
	// Clear specified pools
	for _, pool := range pools {
		if pool == nil {
			continue
		}
		// Force pool to release objects
		for i := 0; i < 100; i++ {
			if obj := pool.Get(); obj != nil {
				// Don't put back, let GC reclaim
				_ = obj
			} else {
				break
			}
		}
	}

	// Force GC to reclaim memory
	runtime.GC()
	runtime.GC() // Second GC to handle finalizers
}
