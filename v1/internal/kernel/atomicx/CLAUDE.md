<!-- updated: 2026-04-16T00:00:00Z -->
# v1/internal/kernel/atomicx

Stateless atomic helpers over `sync/atomic`. Zero-alloc by construction.

## Planned API

```go
type Counter struct{ /* uint64 + padding */ }
func (c *Counter) Inc()        uint64
func (c *Counter) Add(n int64) uint64
func (c *Counter) Load()       uint64
func (c *Counter) Store(v uint64)
func (c *Counter) Swap(v uint64) uint64

type Flag struct{ /* uint32 */ }
func (f *Flag) TrySet()  bool
func (f *Flag) IsSet()   bool
func (f *Flag) Clear()
```

## Rules

1. Every exported type is cache-line aligned (64 bytes) to avoid
   false sharing under contention.
2. No heap allocation on any fast path.
3. No dependency beyond `sync/atomic` and `unsafe`.
4. Benchmarks mandatory: `atomicx_bench_test.go`.
