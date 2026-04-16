<!-- updated: 2026-04-16T00:00:00Z -->
# v1/internal/core/backpressure

Flow-control primitives for adapters that must respect a downstream
throughput cap (CloudWatch, Loki, managed queues).

## Planned API

```go
// Credit is a semaphore that tracks in-flight batches.
// Acquire blocks until a credit is available or ctx is done.
type Credit struct{ /* ... */ }
func NewCredit(total int) *Credit
func (c *Credit) Acquire(ctx context.Context) error
func (c *Credit) Release(n int)

// AIMD adjusts a capacity target on ACK/NACK signals. It shrinks
// multiplicatively on failure and grows additively on sustained
// success — classic TCP congestion control shape.
type AIMD struct{ /* ... */ }
func NewAIMD(initial, min, max int) *AIMD
func (a *AIMD) OnAck()
func (a *AIMD) OnNack()
func (a *AIMD) Capacity() int
```

## Rules

1. No allocation per Acquire/Release on the fast path.
2. `Credit.Acquire` respects ctx cancellation — no spinloops.
3. Bench mandatory: `backpressure_bench_test.go`.
