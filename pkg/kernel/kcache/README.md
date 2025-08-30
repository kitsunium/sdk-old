# kcache

Package kcache provides high-performance cache implementations for the Kitsunium
SDK kernel layer.

## Overview

The kcache package offers multiple cache implementations with different
trade-offs between safety and performance. All implementations support key-value
storage with interface{} types.

## Cache Types

### Basic Caches

- **UnsafeCache**: Single-threaded cache without synchronization. Fastest option
  for single-threaded use.
- **SafeCache**: Thread-safe cache using RWMutex. Suitable for low-to-moderate
  concurrent access.

### Sharded Caches

- **UnsafeShardedCache**: Sharded cache without synchronization. For
  single-threaded use with better cache locality.
- **SafeShardedCache**: Thread-safe sharded cache. Reduces contention through
  sharding for high-concurrency scenarios.

## Features

### Core Operations

- `Set(key, value)`: Store key-value pair
- `Get(key)`: Retrieve value by key
- `Has(key)`: Check key existence
- `Delete(key)`: Remove key
- `Clear()`: Remove all entries
- `Len()`: Get entry count
- `Cap()`: Get capacity

### Batch Operations

All cache types support batch operations for improved throughput:

- `SetBatch(keys, values)`: Store multiple pairs
- `GetBatch(keys)`: Retrieve multiple values
- `HasBatch(keys)`: Check multiple keys
- `DeleteBatch(keys)`: Remove multiple keys

### Global Cache

A global singleton cache instance is available via `Global()` for convenient
access across the application.

## Usage

### Single-Threaded

```go
cache := kcache.NewUnsafeCache(1000)
cache.Set("key", "value")
value, found := cache.Get("key")
```

### Multi-Threaded

```go
cache := kcache.NewSafeCache(1000)
// Safe for concurrent access
go cache.Set("key1", "value1")
go cache.Set("key2", "value2")
```

### High Concurrency

```go
cache := kcache.NewSafeShardedCache(1000, 32) // 32 shards
// Reduced contention through sharding
```

### With Options

```go
cache := kcache.NewCache(
    kcache.WithCapacity(10000),
    kcache.WithShards(64),
    kcache.WithThreadSafe(true),
)
```

## Build Configuration

### Development Mode

In development builds (without `kcache_benchmark` tag), the package includes:

- Goroutine safety checks to detect concurrent access violations
- Runtime validation for unsafe operations
- Additional debugging information

### Benchmark Mode

With `kcache_benchmark` build tag:

- Goroutine checks disabled
- Optimized for maximum performance
- Suitable for production and benchmarking

### Build Examples

```bash
# Development build (with safety checks)
go build ./...

# Benchmark/production build (optimized)
go build -tags kcache_benchmark ./...

# Bazel development build
bazel build //pkg/kernel/kcache:test

# Bazel benchmark build
bazel build --config=benchmark //pkg/kernel/kcache:bench
```

## Performance Characteristics

### Operation Latencies (approximate)

| Operation | UnsafeCache | SafeCache | UnsafeSharded | SafeSharded |
| --------- | ----------- | --------- | ------------- | ----------- |
| Get       | 20-30 ns    | 60-80 ns  | 25-35 ns      | 40-60 ns    |
| Set       | 25-35 ns    | 70-90 ns  | 30-40 ns      | 50-70 ns    |

### Memory Layout

- Cache-line aligned structures to prevent false sharing
- Optimized entry size (48 bytes) for efficient memory usage
- Sharded caches use power-of-2 shard counts for fast modulo operations

## Implementation Details

### Hashing

- Multiple hash algorithms: FNV-1a, xxHash, CityHash, Murmur3
- Automatic algorithm selection based on key type
- Optimized for common types (string, int, etc.)

### Collision Resolution

- Open addressing with quadratic probing
- Dynamic resizing at configurable load factors
- Tombstone entries for deletion handling

### Memory Management

- Object pooling for reduced allocations
- Batch operations for improved locality
- Streaming batch API for large datasets

## Thread Safety

### Safe Variants

- Use RWMutex for reader-writer separation
- Sharded variants reduce lock contention
- Safe for concurrent access from multiple goroutines

### Unsafe Variants

- No synchronization overhead
- Must be externally synchronized or used single-threaded
- Development builds include goroutine checks to detect violations

## Testing

```bash
# Run tests
bazel test //pkg/kernel/kcache:test

# Run benchmarks
bazel test //pkg/kernel/kcache:bench --config=benchmark

# Run with race detector
bazel test //pkg/kernel/kcache:test --features=race
```

## Configuration Constants

Key configuration constants can be found in `constants.go`:

- `DefaultCapacity`: 16 entries
- `DefaultLoadFactor`: 0.75
- `DefaultShardCount`: 32 shards
- `CacheLineSize`: 64 bytes
- `MaxCapacity`: 16 million entries

## Dependencies

The package has minimal dependencies:

- Standard library only
- No external dependencies
- Unsafe package for performance optimizations

## License

Part of the Kitsunium SDK. See repository LICENSE for details.
