<!-- updated: 2026-04-16T00:00:00Z -->
# v1/internal/core/fanout

One event → N consumers, each with its own ring buffer + worker.

## Planned API

```go
// Fanout[T] holds N consumer lanes. Each lane has its own ring and
// worker goroutine.
type Fanout[T any] struct{ /* ... */ }

// Config per lane: ring depth, handler, drop policy.
type LaneConfig[T any] struct {
    ID      string
    Depth   int
    Handler func(ctx context.Context, v T) error
}

func New[T any](lanes []LaneConfig[T]) *Fanout[T]

// Publish pushes v into every lane. Returns a per-lane map of dropped
// counts for this call (empty when no lane was at capacity).
func (f *Fanout[T]) Publish(v T) map[string]int

// Flush drains every lane or returns DeadlineExceeded.
func (f *Fanout[T]) Flush(ctx context.Context) error

// Close stops every worker after a Flush.
func (f *Fanout[T]) Close(ctx context.Context) error

// Drops returns the cumulative per-lane drop count since boot.
func (f *Fanout[T]) Drops() map[string]int
```

## Design

- One ring per lane → a slow lane cannot starve other lanes.
- Workers exit on `ctx.Done()` (passed to `Close`).
- `Publish` is non-blocking — full ring = drop counted, never block.
- Backpressure hook: a lane can plug in a `backpressure.Credit` that
  narrows its effective depth under sustained pressure.

## Rules

1. Lanes are immutable once the Fanout is created — no Add/Remove at
   runtime. Rebuild the Fanout when topology changes.
2. Generic over `T` — no interface boxing on the hot path.
3. Bench mandatory: `fanout_bench_test.go` (1→N scaling).
