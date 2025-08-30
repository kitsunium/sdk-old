# kcache

Cache implementations with explicit thread safety selection for kernel-level
operations.

## Package Design

The kcache package provides cache implementations requiring explicit safety
choice:

- **Unsafe caches**: Single-threaded access only, no synchronization overhead
- **Safe caches**: Thread-safe with mutex protection for concurrent access
- **Sharded variants**: Distribute keys across multiple buckets to reduce
  contention

All caches must be explicitly instantiated. No global instances are provided -
the application or framework decides on instance management.

## Thread Safety Selection

### Explicit Safety Requirement

Every cache creation requires an explicit choice between safe and unsafe
variants:

```go
// Single-threaded context: use unsafe cache
cache := kcache.NewUnsafeCache(10000)

// Multi-threaded context: use safe cache
cache := kcache.NewSafeCache(10000)

// High contention: use sharded safe cache
cache := kcache.NewSafeShardedCache(10000, 32)
```

### Basic Operations

```go
// Set a key-value pair
cache.Set("key", "value")

// Get a value
value, found := cache.Get("key")
if found {
    // Use value
}

// Check if key exists
if cache.Has("key") {
    // Key exists
}

// Delete a key
deleted := cache.Delete("key")

// Clear all entries
cache.Clear()

// Get cache statistics
size := cache.Len()
capacity := cache.Cap()
```

## Implementation Variants

### Unsafe Cache (`NewUnsafeCache`)

- **Usage**: Single-threaded contexts only
- **Synchronization**: None
- **Access**: Will panic if accessed from multiple goroutines (development
  builds)
- **Typical latency**: 20-30 ns/operation

### Safe Cache (`NewSafeCache`)

- **Usage**: Multi-threaded contexts with low to moderate contention
- **Synchronization**: Single RWMutex for entire cache
- **Access**: Thread-safe for concurrent operations
- **Typical latency**: 60-80 ns/operation

### Unsafe Sharded Cache (`NewUnsafeShardedCache`)

- **Usage**: Single-threaded contexts with large datasets
- **Synchronization**: None (sharding for cache locality only)
- **Access**: Will panic if accessed from multiple goroutines (development
  builds)
- **Typical latency**: 25-35 ns/operation

### Safe Sharded Cache (`NewSafeShardedCache`)

- **Usage**: Multi-threaded contexts with high contention
- **Synchronization**: Per-shard RWMutex
- **Access**: Thread-safe with reduced contention through sharding
- **Typical latency**: 40-50 ns/operation

## Build Configuration

### Development Builds (default)

- Goroutine safety checks enabled for unsafe caches
- Detects concurrent access violations
- Panics with diagnostic message on safety violations
- Minimal overhead through sampling (1 in 512 operations)

### Production Builds

- Build tag: `unsafe_no_check`
- All safety checks compiled out
- Zero overhead for safety checking
- Responsibility for correct usage lies with the application

## Instance Management

### No Global State

- All caches require explicit instantiation
- No global pools or caches provided
- Application/framework manages instance lifecycle
- Allows for proper resource control and testing

### Object Pooling

Applications can create pools for reduced allocations:

```go
// Application-managed entry pool
entryPool := kcache.NewEntryPool()

// Application-managed batch pool
batchPool := kcache.NewBatchPool()
```

## Batch Operations

Caches implement batch interfaces for bulk operations:

```go
// Batch set operations
cache.SetBatch(keys, values)

// Batch get operations
values, found := cache.GetBatch(keys)

// Batch delete operations
deleted := cache.DeleteBatch(keys)
```

## Technical Implementation

### Hash Table Design

- Robin Hood hashing for collision resolution
- Power-of-2 sizing for bitwise modulo
- Pre-computed hash storage to avoid recomputation
- Entry states: empty, active, deleted (tombstone)

### Memory Layout

- Cache-line aligned structures (64 bytes)
- Padding to prevent false sharing
- Inline entry storage for cache locality
- Careful field ordering for access patterns

### Sharding Strategy

- Power-of-2 shard count for fast selection
- FNV-1a hash function for distribution
- Independent shard management
- Per-shard size tracking

## Usage Guidelines

### Single-Threaded Contexts

- Use unsafe variants for zero synchronization overhead
- Development builds will detect accidental concurrent access
- Suitable for request-scoped caches in single-threaded handlers

### Multi-Threaded Contexts

- Use safe variants for automatic synchronization
- Consider sharded variants when contention exceeds 10%
- Shard count typically 4x CPU cores for optimal distribution

### Memory Pressure

- Caches retain internal structures after Clear()
- Call runtime.GC() explicitly if immediate reclamation needed
- Use pools for frequently allocated/deallocated caches

## Testing

### Unit Tests

```bash
# Run with safety checks (default)
bazel test //pkg/kernel/kcache:kcache_test

# Run specific test
bazel test //pkg/kernel/kcache:kcache_test --test_filter=TestUnsafeCache
```

### Benchmarks

```bash
# Run benchmarks without safety checks
bazel test //pkg/kernel/kcache:kcache_bench_test --test_arg=-test.bench=. \
  --define gotags=unsafe_no_check

# Run specific benchmark
bazel test //pkg/kernel/kcache:kcache_bench_test \
  --test_arg=-test.bench=BenchmarkUnsafeCache
```

## Build Tags

- `unsafe_no_check`: Removes goroutine safety checks for production builds
