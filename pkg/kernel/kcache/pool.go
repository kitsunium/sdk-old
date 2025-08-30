package kcache

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// entryPool manages a pool of reusable entry objects.
// Reduces allocation pressure in hot paths.
type entryPool struct {
	pool    *sync.Pool   // Underlying sync.Pool
	size    atomic.Int64 // Current pool size estimate
	maxSize int64        // Maximum pool size
	_       [32]byte     // Cache line padding
}

// newEntryPool creates a new entry object pool.
// Pre-allocates entries to avoid startup allocation spike.
func newEntryPool() *entryPool {
	ep := &entryPool{
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
func (ep *entryPool) prewarm(count int) {
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

// get retrieves an entry from the pool or creates a new one.
// Resets the entry to zero values for safety.
//
//go:inline
func (ep *entryPool) get() *entry {
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

// put returns an entry to the pool for reuse.
// Clears sensitive data before pooling.
//
//go:inline
func (ep *entryPool) put(e *entry) {
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

// keyPool manages string key allocations.
// Optimizes for common key patterns.
type keyPool struct {
	small  *sync.Pool // Pool for small strings (<= 64 bytes)
	medium *sync.Pool // Pool for medium strings (<= 256 bytes)
	large  *sync.Pool // Pool for large strings (<= 1024 bytes)
	_      [32]byte   // Cache line padding
}

// newKeyPool creates a pool for string keys.
func newKeyPool() *keyPool {
	return &keyPool{
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

// getBuffer retrieves a buffer of appropriate size.
//
//go:inline
func (kp *keyPool) getBuffer(size int) *[]byte {
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

// putBuffer returns a buffer to the appropriate pool.
//
//go:inline
func (kp *keyPool) putBuffer(buf *[]byte) {
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

// nodePool manages hash table node allocations.
// Used for collision chain nodes in advanced implementations.
type nodePool struct {
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

// newNodePool creates a pool for collision chain nodes.
func newNodePool() *nodePool {
	return &nodePool{
		maxSize: MaxPoolSize,
		pool: &sync.Pool{
			New: func() interface{} {
				return &node{}
			},
		},
	}
}

// get retrieves a node from the pool.
//
//go:inline
func (np *nodePool) get() *node {
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

// put returns a node to the pool.
//
//go:inline
func (np *nodePool) put(n *node) {
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

// globalPools holds globally shared pools.
// Reduces per-cache memory overhead.
var globalPools struct {
	entries *entryPool
	keys    *keyPool
	nodes   *nodePool
	once    sync.Once
}

// initGlobalPools initializes the global pools once.
func initGlobalPools() {
	globalPools.once.Do(func() {
		globalPools.entries = newEntryPool()
		globalPools.keys = newKeyPool()
		globalPools.nodes = newNodePool()

		// Register cleanup on GC
		runtime.SetFinalizer(&globalPools, cleanupPoolsInternal)
	})
}

// cleanupPools releases pool resources.
// This is the internal cleanup function for the finalizer.
func cleanupPoolsInternal(gp *struct {
	entries *entryPool
	keys    *keyPool
	nodes   *nodePool
	once    sync.Once
}) {
	// Pools are automatically cleaned by GC
	// This is just for explicit cleanup if needed
}

// cleanupPools is the exported cleanup function for testing.
func cleanupPools() {
	// This function is primarily for testing purposes
	// to force cleanup of global pools
	if globalPools.entries != nil || globalPools.keys != nil || globalPools.nodes != nil {
		cleanupPoolsInternal(&globalPools)
	}
}

// getGlobalEntryPool returns the global entry pool.
//
//go:inline
func getGlobalEntryPool() *entryPool {
	initGlobalPools()
	return globalPools.entries
}

// getGlobalKeyPool returns the global key pool.
//
//go:inline
func getGlobalKeyPool() *keyPool {
	initGlobalPools()
	return globalPools.keys
}

// getGlobalNodePool returns the global node pool.
//
//go:inline
func getGlobalNodePool() *nodePool {
	initGlobalPools()
	return globalPools.nodes
}

// Memory allocation optimizations

// allocator provides custom memory allocation strategies.
// Bypasses standard allocator for specific patterns.
type allocator struct {
	arena    []byte           // Memory arena
	offset   uintptr          // Current allocation offset
	size     uintptr          // Arena size
	fallback func(int) []byte // Fallback allocator
	_        [32]byte         // Cache line padding
}

// newAllocator creates a custom allocator with arena.
func newAllocator(size int) *allocator {
	return &allocator{
		arena:  make([]byte, size),
		size:   uintptr(size),
		offset: 0,
		fallback: func(n int) []byte {
			return make([]byte, n)
		},
	}
}

// alloc allocates memory from the arena.
// Falls back to standard allocator if arena exhausted.
//
//go:inline
//go:nosplit
func (a *allocator) alloc(size int) []byte {
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

// reset resets the allocator for reuse.
func (a *allocator) reset() {
	atomic.StoreUintptr(&a.offset, 0)
	// Clear arena for security
	for i := range a.arena {
		a.arena[i] = 0
	}
}

// Object pooling for batch operations

// batchPool pools batch operation buffers.
type batchPool struct {
	pool *sync.Pool
}

// newBatchPool creates a pool for batch buffers.
func newBatchPool() *batchPool {
	return &batchPool{
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

// get retrieves a batch buffer from the pool.
//
//go:inline
func (bp *batchPool) get() *batchBuffer {
	buf := bp.pool.Get().(*batchBuffer)
	buf.reset()
	return buf
}

// put returns a batch buffer to the pool.
//
//go:inline
func (bp *batchPool) put(buf *batchBuffer) {
	buf.reset()
	bp.pool.Put(buf)
}

// Global batch pool instance
var globalBatchPool = newBatchPool()

// getBatchBuffer gets a pooled batch buffer.
//
//go:inline
func getBatchBuffer() *batchBuffer {
	return globalBatchPool.get()
}

// putBatchBuffer returns a batch buffer to the pool.
//
//go:inline
func putBatchBuffer(buf *batchBuffer) {
	globalBatchPool.put(buf)
}

// Memory pressure handling

// handleMemoryPressure responds to memory pressure.
// Clears pools to release memory back to OS.
func handleMemoryPressure() {
	// Clear global pools
	if globalPools.entries != nil {
		// Force pool to release objects
		for i := 0; i < 100; i++ {
			if obj := globalPools.entries.pool.Get(); obj != nil {
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

// monitorMemory monitors memory usage and triggers cleanup.
func monitorMemory() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check if memory usage is high
	if m.Alloc > uint64(MaxCapacity*EntrySize) {
		handleMemoryPressure()
	}
}

// init registers memory monitor
func init() {
	// Periodic memory monitoring
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			monitorMemory()
		}
	}()
}
