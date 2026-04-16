# cache

Thread-safe cache implementations for Go.

## Usage

```go
import "github.com/kitsunium/sdk/pkg/kernel/cache"
```

## Implementations

### LRU Cache

Least Recently Used cache with TTL support.

```go
cache := cache.NewLRU[string, any](1000)
cache.Set("key", "value")
value, ok := cache.Get("key")
cache.SetWithTTL("session", data, 30*time.Minute)
```

### Sharded LRU Cache

LRU cache divided into shards to reduce lock contention.

```go
cache := cache.NewShardedLRU[string, any](10000, 256)
cache.Set("key", "value")
value, ok := cache.Get("key")
```

### Atomic Cache

Lock-free cache using RCU pattern for read-heavy workloads.

```go
cache := cache.NewAtomicCache[string, any](1000)
cache.Set("key", "value")
value, ok := cache.Get("key")

// Fast get returns pointer (do not modify)
valuePtr, ok := cache.FastGet("key")

// Batch operations
cache.BatchSet(map[string]any{"k1": "v1", "k2": "v2"})
results := cache.BatchGet([]string{"k1", "k2"})
```

## Interface

All implementations satisfy the Cache interface:

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

## Statistics

```go
stats := cache.Stats()
// stats.Hits, stats.Misses, stats.Sets, stats.Evictions
```

## Testing

```bash
go test -race ./pkg/kernel/cache/...
go test -bench=. ./pkg/kernel/cache/...
```
