<!-- updated: 2026-04-16T00:00:00Z -->
# v1/internal/kernel/clock

Monotonic time helpers. `CLOCK_MONOTONIC`-backed, zero-alloc `Now()`.

## Planned API

```go
// Now returns the current monotonic time. On Linux amd64/arm64 this
// resolves via vDSO in ~15 ns with zero allocation.
func Now() time.Time

// Since is a zero-alloc replacement for time.Since.
func Since(t time.Time) time.Duration

// Ticker is a minimal Ticker that emits on a channel; reset without
// reallocating the underlying goroutine.
type Ticker struct{ /* ... */ }
func NewTicker(d time.Duration) *Ticker
func (t *Ticker) C() <-chan time.Time
func (t *Ticker) Reset(d time.Duration)
func (t *Ticker) Stop()
```

## Rules

1. `Now()` must delegate to `time.Now()` (vDSO path). Never sample
   wall clock (`CLOCK_REALTIME`) on hot paths.
2. No 3rd-party imports.
3. Benchmarks mandatory: `clock_bench_test.go`.
