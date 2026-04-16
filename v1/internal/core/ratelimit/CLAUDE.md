<!-- updated: 2026-04-16T00:00:00Z -->
# v1/internal/core/ratelimit

Rate-control primitives. First consumer: the logger (threshold-based
sync → batched mode switch). Generic — reusable by any component.

## Planned API

```go
// Tokens returns a token-bucket limiter that refills at `rate`
// tokens per second up to `burst`. Take is non-blocking.
type Tokens struct{ /* ... */ }
func TokensBucket(rate float64, burst int) *Tokens
func (t *Tokens) Take(n int) bool      // true iff allowed now
func (t *Tokens) TakeAt(n int, at time.Time) bool

// Leaky returns a leaky-bucket limiter that smooths outbound rate.
type Leaky struct{ /* ... */ }
func LeakyBucket(rate float64, burst int) *Leaky
func (l *Leaky) Wait(ctx context.Context, n int) error

// Gauge exposes an observed rate without limiting — used by the
// logger to decide sync vs batched mode without a feedback loop.
type Gauge struct{ /* ... */ }
func NewGauge(window time.Duration) *Gauge
func (g *Gauge) Observe(n int)
func (g *Gauge) Rate() float64
```

## Use cases

1. **Logger adaptive mode**: `Gauge.Rate()` crosses N/s → emitter
   flips from per-record Write to batched Write.
2. **Cloud adapters**: `Leaky` smooths outbound CloudWatch/Loki
   pushes to stay under the provider's QPS ceiling.
3. **Graceful degradation**: `Tokens.Take` returns false → adapter
   drops or queues, emitter logs the drop count.

## Rules

1. `Take*` calls MUST be O(1) and allocation-free.
2. Time source = `clock.Now()` (vDSO-backed).
3. Benchmarks mandatory: `ratelimit_bench_test.go`.
4. Concurrency-safe by default; pad hot counters for contention.
