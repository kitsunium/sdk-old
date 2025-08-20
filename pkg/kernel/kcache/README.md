# KCache Package

Thread-safe caching library for Go with multiple implementations for different use cases.

## Installation

```go
import "github.com/kitsunium/sdk/pkg/kernel/kcache"
```

## Features

- Multiple implementations for different use cases
- Thread-safe with various locking strategies
- TTL support for automatic expiration
- Built-in statistics for monitoring
- Generic types for type safety
- Minimal allocations on read operations

## Cache Implementations

### 1. LRU Cache

Standard Least Recently Used cache implementation.

```go
// Create an LRU cache with capacity of 1000
cache := cache.NewLRU[string, any](1000)

// Basic operations
cache.Set("key", "value")
value, ok := cache.Get("key")
cache.Delete("key")
cache.Clear()

// With TTL
cache.SetWithTTL("session", userData, 30*time.Minute)

// Check statistics
stats := cache.Stats()
fmt.Printf("Hit rate: %.2f%%\n", 
    float64(stats.Hits) / float64(stats.Hits + stats.Misses) * 100)
```

**Best for:**
- General-purpose caching
- Single-threaded or low-concurrency scenarios
- When memory efficiency is critical
- Strict LRU eviction requirements

### 2. Sharded LRU Cache

Cache that reduces lock contention by dividing the cache into independent shards.

```go
// Create a sharded cache with 10000 capacity and 256 shards
cache := cache.NewShardedLRU[string, any](10000, 256)

// Same API as regular LRU
cache.Set("key", "value")
value, ok := cache.Get("key")

// Automatically distributes keys across shards
for i := 0; i < 1000; i++ {
    cache.Set(fmt.Sprintf("key_%d", i), i)
}
```

**Best for:**
- High-concurrency applications
- Multi-core systems
- Balanced read/write workloads
- When you need predictable performance under load

**Characteristics:**
- Reduced lock contention compared to standard LRU
- Scalability with multiple CPU cores
- Distributed key storage across shards

### 3. Atomic Cache

Lock-free cache for read-heavy workloads using RCU (Read-Copy-Update) pattern.

```go
// Create an atomic cache with capacity of 1000
cache := cache.NewAtomicCache[string, any](1000)

// Standard operations
cache.Set("key", "value")
value, ok := cache.Get("key")

// Fast get without value copy (returns pointer)
valuePtr, ok := cache.FastGet("key")
if ok {
    // Use *valuePtr (do not modify!)
}

// Batch operations for efficiency
cache.BatchSet(map[string]any{
    "key1": "value1",
    "key2": "value2",
    "key3": "value3",
})

results := cache.BatchGet([]string{"key1", "key2", "key3"})
```

**Best for:**
- Read-heavy workloads (90%+ reads)
- Small to medium cache sizes
- When you need low read latency
- Scenarios where writes are infrequent

**Features:**
- Lock-free reads
- Batch operations
- FastGet for pointer access

## Common Interface

All cache implementations satisfy the `Cache` interface:

```go
type Cache[K comparable, V any] interface {
    Get(key K) (V, bool)
    Set(key K, value V)
    SetWithTTL(key K, value V, ttl time.Duration)
    Delete(key K)
    Clear()
    Size() int
    Has(key K) bool
}
```

This allows easy switching between implementations:

```go
var c cache.Cache[string, int]

// Choose implementation based on requirements
if highConcurrency {
    c = cache.NewShardedLRU[string, int](10000, 256)
} else if readHeavy {
    c = cache.NewAtomicCache[string, int](10000)
} else {
    c = cache.NewLRU[string, int](10000)
}

// Use the same API regardless of implementation
c.Set("count", 42)
```

## Statistics

All caches provide statistics for monitoring:

```go
type Stats struct {
    Hits      uint64  // Successful cache retrievals
    Misses    uint64  // Failed cache retrievals
    Sets      uint64  // Cache insertions
    Evictions uint64  // Entries removed due to capacity
}

stats := cache.Stats()
```

## Configuration Examples

### Web Session Cache
```go
// High concurrency, medium size, with TTL
sessionCache := cache.NewShardedLRU[string, *Session](10000, 256)
sessionCache.SetWithTTL(sessionID, session, 20*time.Minute)
```

### Database Query Cache
```go
// Read-heavy, frequently accessed queries
queryCache := cache.NewAtomicCache[string, *QueryResult](1000)
queryCache.BatchSet(preloadedQueries)
```

### Application Config Cache
```go
// Small, general purpose
configCache := cache.NewLRU[string, any](100)
configCache.Set("database.host", "localhost")
```

## Usage Guidelines

### Choosing the Right Implementation

| Scenario | Recommended | Capacity | Shards |
|----------|-------------|----------|--------|
| Single-threaded | LRU | As needed | N/A |
| Web server (balanced) | ShardedLRU | 10K-100K | 256 |
| Read-heavy API | AtomicCache | 1K-10K | N/A |
| Microservice cache | ShardedLRU | 1K-10K | 64-128 |
| In-memory DB index | AtomicCache | 100K+ | N/A |

### Shard Count Guidelines (ShardedLRU)

- **Default (256)**: Good for most use cases
- **64**: Low concurrency (2-4 cores)
- **128**: Medium concurrency (4-8 cores)
- **256**: High concurrency (8-16 cores)
- **512-1024**: Very high concurrency (16+ cores)

Note: Shard count is automatically rounded to the nearest power of 2.

## Thread Safety

All implementations are fully thread-safe and tested with Go's race detector:

```bash
go test -race ./pkg/kernel/cache/...
```

## TTL (Time To Live)

All caches support optional TTL for automatic expiration:

```go
// Expires after 5 minutes
cache.SetWithTTL("temporary", data, 5*time.Minute)

// Never expires (same as Set)
cache.SetWithTTL("permanent", data, 0)
```

Expired entries are lazily removed on access or during eviction.

## Memory Management

- **LRU**: Minimal memory overhead, one map + doubly-linked list
- **ShardedLRU**: Memory divided across shards, slightly higher overhead
- **AtomicCache**: Copy-on-write may temporarily use more memory during updates

Implementations use object pooling where applicable.

## Integration with Config Package

The cache is integrated into the config package for automatic caching:

```go
import "github.com/kitsunium/sdk/pkg/kernel/config"

// Create config with cache
cfg := config.NewWithCache(parsers, 1000)

// Subsequent calls are cached
value := cfg.Get("database.host", "localhost")
```

## Testing

Run tests with coverage:

```bash
go test -v -cover ./pkg/kernel/cache/...
```

Run benchmarks:

```bash
go test -bench=. -benchmem ./pkg/kernel/cache/...
```

## License

Part of the Kitsunium SDK.