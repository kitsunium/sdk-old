<!-- updated: 2026-04-16T00:00:00Z -->
# v1/internal/core/ring

Lock-free ring buffer. MPSC by default, SPMC variant planned.

## Planned API

```go
type Ring[T any] struct{ /* cache-padded head/tail + slots */ }

// New allocates a ring with capacity rounded up to the next power of 2.
func New[T any](capacity int) *Ring[T]

// Push publishes an element; returns false when the ring is full.
func (r *Ring[T]) Push(v T) bool

// Pop consumes one element; returns zero + false when empty.
func (r *Ring[T]) Pop() (v T, ok bool)

// Len reports the current count (approximate under contention).
func (r *Ring[T]) Len() int

// Cap returns the fixed capacity.
func (r *Ring[T]) Cap() int
```

## Design

- Power-of-two capacity → mask instead of modulo.
- Cache-line padded head/tail counters → no false sharing.
- `Push` failure is a deliberate signal: the caller decides to drop,
  block, or spill. The ring itself never blocks.

## Rules

1. Generic over `T` — typed consumers get zero-alloc Push/Pop.
2. Benchmarks mandatory: `ring_bench_test.go`. Baseline: single-core
   P+C ≤ 30 ns per op.
3. Must hold up under `-race` on multi-CPU hosts.
