# metrics

Stdlib-only runtime instrumentation primitives for the Kitsunium SDK.

## Products

| Constructor | Returns | Purpose |
|-------------|---------|---------|
| `metrics.Counter(name, opts...)` | `Counter` | Monotonic counter. |
| `metrics.Gauge(name, opts...)`   | `Gauge`   | Value that moves up/down. |
| `metrics.Health(name, fn, opts...)` | `Healthcheck` | Named liveness/readiness probe. |

All three are concurrent-safe. Counter and Gauge use atomic CAS on a
`uint64` holding the bits of a `float64`.

## Quick start

```go
import (
    "context"
    "github.com/kitsunium/sdk/pkg/component/metrics"
)

requests := metrics.Counter("http_requests_total",
    metrics.WithHelp("Total HTTP requests"),
    metrics.WithLabel("service", "api"),
)
requests.Inc()
requests.Add(5)

queueDepth := metrics.Gauge("queue_depth")
queueDepth.Set(42)
queueDepth.Add(-3)

db := metrics.Health("db",
    func(ctx context.Context) metrics.Status {
        if err := ping(ctx); err != nil {
            return metrics.Down(err.Error())
        }
        return metrics.OK("reachable")
    },
)
status := db.Check(context.Background())
```

## Options

| Option | Applies to | Description |
|--------|-----------|-------------|
| `WithHelp(string)` | all | Descriptor help text. |
| `WithLabel(k, v)`  | all | Adds a single label. |
| `WithLabels(map)`  | all | Merges multiple labels. |

## Status

| Helper | Code |
|--------|------|
| `metrics.OK(reason)` | `StatusOK` |
| `metrics.Degraded(reason)` | `StatusDegraded` |
| `metrics.Down(reason)` | `StatusDown` |
