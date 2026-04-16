<!-- updated: 2026-04-16T00:00:00Z -->
# pkg/kernel/cache

Generic in-memory caches parametrized on `[K comparable, V any]`. TTL-aware.

## Files

| File | Role |
|---|---|
| `cache.go` | `Cache[K,V]` interface + classic mutex-guarded `LRU[K,V]` + `Stats` (atomic counters) |
| `sharded.go` | `ShardedLRU[K,V]` — partitioned LRU to reduce contention; `FastLRU` per-shard type |
| `cache_bench_test.go` | Benchmarks for LRU / sharded throughput |

## Interface surface

```go
type Cache[K comparable, V any] interface {
    Get(key K) (V, bool)
    Set(key K, value V)
    SetWithTTL(key K, value V, ttl time.Duration)
    Delete(key K)
    Clear()
    Has(key K) bool
    // + Len(), Stats() — see cache.go
}
```

Constructors:
- `NewLRU[K,V](capacity int) *LRU[K,V]` — single-mutex, TTL-aware
- `NewShardedLRU[K,V](capacity, numShards int) *ShardedLRU[K,V]` — power-of-two shards

## Rules

1. **Shard count** of `ShardedLRU` should be a power of two for the `hash & (n-1)` path.
2. **TTL**: zero duration means "never expires". `SetWithTTL` is still on the `Cache` interface but no production caller uses it with non-zero TTL today.
3. `Stats` fields are atomic counters — read via `.Load()` not `==`.
4. **Concurrency invariant**: `ShardedLRU.Get` takes a single write lock for the full operation (look-up → expiry check → `moveToFront`). The previous RLock→Lock upgrade had a TOCTOU hole that could corrupt the shard's linked list under concurrent Delete/evict. Do not re-introduce the split-lock pattern.
5. Target naming migration (roadmap §16): `NewLRU[K,V](n)` → `cache.LRU[K,V](n)`, `NewShardedLRU(n, s)` → `cache.LRU(n, cache.Sharded(s))`.

## Deliberately absent

Removed during the 2026-04 rebuild (see `/workspace/.claude/contexts/kernel-cache-audit.md`):
- `AtomicCache[K,V]` / `NewAtomicCache` — marketed as lock-free-read RCU; actual implementation did `maps.Copy` on every `Set` (O(n) per write). Its "random eviction" was not LRU. The `errs` registry used it only to abuse `Range` via type assertion through the `Cache[K,V]` interface.
- Exported `FastLRU` as a standalone type — kept as unexported shard-local LRU inside `ShardedLRU`.

## Validation

```bash
bazel test //pkg/kernel/cache/...
go test -race -bench=. -benchmem ./pkg/kernel/cache
```
