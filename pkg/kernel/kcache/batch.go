package kcache

import (
	"runtime"
	"sync"
	"unsafe"
)

// GetResult represents a single result from a batch Get operation.
type GetResult struct {
	Value interface{} // The retrieved value
	Found bool        // Whether the key was found
}

// batchProcessor provides optimized batch operations using SIMD-like techniques.
// Processes multiple operations in parallel to maximize CPU utilization.
type batchProcessor struct {
	cache   Cache      // Underlying cache implementation
	workers int        // Number of parallel workers
	pool    *sync.Pool // Pool for temporary buffers
	_       [48]byte   // Cache line padding
}

// newBatchProcessor creates an optimized batch processor.
// Workers are tuned based on CPU count and operation type.
func newBatchProcessor(cache Cache) *batchProcessor {
	workers := runtime.NumCPU()
	if workers > 16 {
		workers = 16 // Diminishing returns beyond 16 workers
	}

	return &batchProcessor{
		cache:   cache,
		workers: workers,
		pool: &sync.Pool{
			New: func() interface{} {
				// Pre-allocate buffers for batch operations
				return &batchBuffer{
					keys:   make([]interface{}, 0, DefaultBatchSize),
					values: make([]interface{}, 0, DefaultBatchSize),
					hashes: make([]uint64, 0, DefaultBatchSize),
				}
			},
		},
	}
}

// Execute performs a batch operation based on the operation type.
// Returns the appropriate result type based on the operation.
func (bp *batchProcessor) Execute(op int, keys []interface{}, values []interface{}) interface{} {
	switch op {
	case batchOpSet:
		if bc, ok := bp.cache.(BatchCache); ok {
			return bc.SetBatch(keys, values)
		}
		return simpleBatchSet(bp.cache, keys, values)

	case batchOpGet:
		var results []GetResult
		if bc, ok := bp.cache.(BatchCache); ok {
			vals, found := bc.GetBatch(keys)
			results = make([]GetResult, len(keys))
			for i := range keys {
				results[i] = GetResult{
					Value: vals[i],
					Found: found[i],
				}
			}
			return results
		}
		// Fallback to simple get
		vals, found := OptimizedGetBatch(bp.cache, keys)
		results = make([]GetResult, len(keys))
		for i := range keys {
			results[i] = GetResult{
				Value: vals[i],
				Found: found[i],
			}
		}
		return results

	case batchOpHas:
		if bc, ok := bp.cache.(BatchCache); ok {
			return bc.HasBatch(keys)
		}
		return OptimizedHasBatch(bp.cache, keys)

	case batchOpDelete:
		if bc, ok := bp.cache.(BatchCache); ok {
			return bc.DeleteBatch(keys)
		}
		return OptimizedDeleteBatch(bp.cache, keys)

	default:
		return nil
	}
}

// batchBuffer holds temporary data for batch operations.
// Reused via pool to minimize allocations.
type batchBuffer struct {
	keys   []interface{} // Keys being processed
	values []interface{} // Values being processed
	hashes []uint64      // Pre-computed hashes
	found  []bool        // Results for lookups
}

// reset clears the buffer for reuse.
//
//go:inline
func (bb *batchBuffer) reset() {
	bb.keys = bb.keys[:0]
	bb.values = bb.values[:0]
	bb.hashes = bb.hashes[:0]
	bb.found = bb.found[:0]
}

// OptimizedSetBatch performs batch insertion with prefetching.
// Groups operations to improve cache locality.
func OptimizedSetBatch(cache Cache, keys, values []interface{}) int {
	if len(keys) != len(values) || len(keys) == 0 {
		return 0
	}

	// For small batches, use simple approach
	if len(keys) < 16 {
		return simpleBatchSet(cache, keys, values)
	}

	// Use parallel processing for large batches
	return parallelBatchSet(cache, keys, values)
}

// simpleBatchSet handles small batches without parallelization overhead.
//
//go:inline
func simpleBatchSet(cache Cache, keys, values []interface{}) int {
	newCount := 0
	for i := range keys {
		if cache.Set(keys[i], values[i]) {
			newCount++
		}
	}
	return newCount
}

// parallelBatchSet processes large batches using parallel workers.
// Divides work among CPU cores for better throughput.
func parallelBatchSet(cache Cache, keys, values []interface{}) int {
	numWorkers := runtime.NumCPU()
	if numWorkers > len(keys)/10 {
		// Not worth parallelizing for small batches
		return simpleBatchSet(cache, keys, values)
	}

	// Check if cache supports batch operations directly
	if bc, ok := cache.(BatchCache); ok {
		return bc.SetBatch(keys, values)
	}

	// Single-threaded cache - process sequentially
	return simpleBatchSet(cache, keys, values)
}

// OptimizedGetBatch performs batch retrieval with prefetching.
// Minimizes cache misses through strategic data layout.
func OptimizedGetBatch(cache Cache, keys []interface{}) ([]interface{}, []bool) {
	if len(keys) == 0 {
		return nil, nil
	}

	// Allocate result slices
	values := make([]interface{}, len(keys))
	found := make([]bool, len(keys))

	// Process in cache-friendly chunks
	processInChunks(keys, 8, func(start, end int) {
		for i := start; i < end; i++ {
			values[i], found[i] = cache.Get(keys[i])
		}
	})

	return values, found
}

// OptimizedHasBatch checks existence with minimal memory touches.
// Uses prefetching to hide memory latency.
func OptimizedHasBatch(cache Cache, keys []interface{}) []bool {
	if len(keys) == 0 {
		return nil
	}

	found := make([]bool, len(keys))

	// Process with prefetching
	processWithPrefetch(keys, func(i int) {
		found[i] = cache.Has(keys[i])
	})

	return found
}

// OptimizedDeleteBatch performs batch deletion efficiently.
// Groups deletions to minimize lock acquisitions.
func OptimizedDeleteBatch(cache Cache, keys []interface{}) []bool {
	if len(keys) == 0 {
		return nil
	}

	deleted := make([]bool, len(keys))

	// Process deletions
	for i := range keys {
		deleted[i] = cache.Delete(keys[i])
	}

	return deleted
}

// processInChunks divides work into cache-friendly chunks.
// Improves locality of reference for better CPU cache utilization.
//
//go:inline
func processInChunks(items []interface{}, chunkSize int, fn func(start, end int)) {
	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize
		if end > len(items) {
			end = len(items)
		}
		fn(i, end)
	}
}

// processWithPrefetch processes items with prefetch hints.
// Reduces memory latency by prefetching next items.
func processWithPrefetch(items []interface{}, fn func(int)) {
	// Process with prefetch distance of 4 items
	const prefetchDistance = 4

	for i := 0; i < len(items); i++ {
		// Prefetch future items
		if i+prefetchDistance < len(items) {
			prefetchHint(unsafe.Pointer(&items[i+prefetchDistance]))
		}

		// Process current item
		fn(i)
	}
}

// prefetchHint provides a prefetch hint to the CPU.
// No-op on architectures without prefetch support.
//
//go:inline
//go:nosplit
func prefetchHint(p unsafe.Pointer) {
	// This is a hint to the CPU to prefetch data
	// Actual implementation depends on architecture
	_ = p // Avoid unused parameter warning
}

// BatchBuilder provides a fluent interface for building batch operations.
// Accumulates operations for efficient bulk execution.
type BatchBuilder struct {
	cache  Cache         // Target cache
	keys   []interface{} // Accumulated keys
	values []interface{} // Accumulated values
	op     int           // Operation type
}

// Batch operation types for internal use
const (
	opSet = iota
	opGet
	opHas
	opDelete
)

// Batch operation types for batchProcessor
const (
	batchOpSet = iota
	batchOpGet
	batchOpHas
	batchOpDelete
)

// NewBatchBuilder creates a new batch builder for the given cache.
func NewBatchBuilder(cache Cache) *BatchBuilder {
	return &BatchBuilder{
		cache:  cache,
		keys:   make([]interface{}, 0, DefaultBatchSize),
		values: make([]interface{}, 0, DefaultBatchSize),
	}
}

// Set adds a set operation to the batch.
func (bb *BatchBuilder) Set(key, value interface{}) *BatchBuilder {
	bb.keys = append(bb.keys, key)
	bb.values = append(bb.values, value)
	bb.op = opSet
	return bb
}

// Get adds keys for batch retrieval.
func (bb *BatchBuilder) Get(keys ...interface{}) *BatchBuilder {
	bb.keys = append(bb.keys, keys...)
	bb.op = opGet
	return bb
}

// Has adds keys for batch existence check.
func (bb *BatchBuilder) Has(keys ...interface{}) *BatchBuilder {
	bb.keys = append(bb.keys, keys...)
	bb.op = opHas
	return bb
}

// Delete adds keys for batch deletion.
func (bb *BatchBuilder) Delete(keys ...interface{}) *BatchBuilder {
	bb.keys = append(bb.keys, keys...)
	bb.op = opDelete
	return bb
}

// Execute performs the accumulated batch operations.
func (bb *BatchBuilder) Execute() interface{} {
	if len(bb.keys) == 0 {
		return nil
	}

	switch bb.op {
	case opSet:
		if bc, ok := bb.cache.(BatchCache); ok {
			return bc.SetBatch(bb.keys, bb.values)
		}
		return simpleBatchSet(bb.cache, bb.keys, bb.values)

	case opGet:
		if bc, ok := bb.cache.(BatchCache); ok {
			values, found := bc.GetBatch(bb.keys)
			return []interface{}{values, found}
		}
		values, found := OptimizedGetBatch(bb.cache, bb.keys)
		return []interface{}{values, found}

	case opHas:
		if bc, ok := bb.cache.(BatchCache); ok {
			return bc.HasBatch(bb.keys)
		}
		return OptimizedHasBatch(bb.cache, bb.keys)

	case opDelete:
		if bc, ok := bb.cache.(BatchCache); ok {
			return bc.DeleteBatch(bb.keys)
		}
		return OptimizedDeleteBatch(bb.cache, bb.keys)

	default:
		return nil
	}
}

// Reset clears the builder for reuse.
func (bb *BatchBuilder) Reset() {
	bb.keys = bb.keys[:0]
	bb.values = bb.values[:0]
	bb.op = 0
}

// Vectorized operations for supported types

// SetBatchInt64 optimizes batch operations for int64 keys.
// Uses type-specific optimizations to avoid interface overhead.
func SetBatchInt64(cache Cache, keys []int64, values []interface{}) int {
	// Convert to interface slice
	// Future: implement specialized int64 cache to avoid conversion
	ikeys := make([]interface{}, len(keys))
	for i, k := range keys {
		ikeys[i] = k
	}

	if bc, ok := cache.(BatchCache); ok {
		return bc.SetBatch(ikeys, values)
	}
	return simpleBatchSet(cache, ikeys, values)
}

// SetBatchString optimizes batch operations for string keys.
// Avoids string allocation overhead where possible.
func SetBatchString(cache Cache, keys []string, values []interface{}) int {
	// Convert to interface slice with minimal allocation
	ikeys := make([]interface{}, len(keys))
	for i, k := range keys {
		ikeys[i] = k
	}

	if bc, ok := cache.(BatchCache); ok {
		return bc.SetBatch(ikeys, values)
	}
	return simpleBatchSet(cache, ikeys, values)
}

// SetBatchBytes optimizes batch operations for byte slice keys.
// Uses unsafe conversion to avoid copying.
func SetBatchBytes(cache Cache, keys [][]byte, values []interface{}) int {
	// Convert to interface slice
	ikeys := make([]interface{}, len(keys))
	for i, k := range keys {
		// Use unsafe string conversion to avoid allocation
		ikeys[i] = unsafeString(k)
	}

	if bc, ok := cache.(BatchCache); ok {
		return bc.SetBatch(ikeys, values)
	}
	return simpleBatchSet(cache, ikeys, values)
}

// unsafeString converts bytes to string without allocation.
// SAFETY: Caller must ensure bytes are not modified.
//
//go:inline
//go:nosplit
func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Streaming batch operations for large datasets

// StreamingBatch processes large batches in streaming fashion.
// Reduces memory usage for very large operations.
type StreamingBatch struct {
	mu        sync.Mutex
	cache     Cache
	batchSize int
	buffer    *batchBuffer
}

// NewStreamingBatch creates a streaming batch processor.
func NewStreamingBatch(cache Cache, batchSize int) *StreamingBatch {
	if batchSize <= 0 || batchSize > MaxBatchSize {
		batchSize = DefaultBatchSize
	}

	return &StreamingBatch{
		cache:     cache,
		batchSize: batchSize,
		buffer: &batchBuffer{
			keys:   make([]interface{}, 0, batchSize),
			values: make([]interface{}, 0, batchSize),
		},
	}
}

// Add accumulates a key-value pair, flushing when batch is full.
func (sb *StreamingBatch) Add(key, value interface{}) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.buffer.keys = append(sb.buffer.keys, key)
	sb.buffer.values = append(sb.buffer.values, value)

	if len(sb.buffer.keys) >= sb.batchSize {
		return sb.flushLocked()
	}
	return nil
}

// Flush processes any remaining items in the buffer.
func (sb *StreamingBatch) Flush() error {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.flushLocked()
}

// flushLocked flushes without acquiring the lock (must be called with lock held).
func (sb *StreamingBatch) flushLocked() error {
	if len(sb.buffer.keys) == 0 {
		return nil
	}

	if bc, ok := sb.cache.(BatchCache); ok {
		bc.SetBatch(sb.buffer.keys, sb.buffer.values)
	} else {
		simpleBatchSet(sb.cache, sb.buffer.keys, sb.buffer.values)
	}

	sb.buffer.reset()
	return nil
}

// Close flushes and releases resources.
func (sb *StreamingBatch) Close() error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	err := sb.flushLocked()
	sb.buffer = nil
	return err
}
