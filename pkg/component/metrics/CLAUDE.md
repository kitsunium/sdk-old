<!-- updated: 2026-04-16T00:00:00Z -->
# pkg/component/metrics

Instrumentation primitives: counters, gauges, health checks. Product layer on top of `pkg/kernel/*`.

## Files

| File | Role |
|---|---|
| `metrics.go` | Shared types: `Status`, `StatusCode`, `Descriptor`, `Metric`, `Healthcheck`; constructors `OK`, `Degraded`, `Down`; `Option` + `WithHelp`, `WithLabel`, `WithLabels` |
| `counter.go` | `Counter(name string, opts ...Option) *counter` — monotonically increasing |
| `gauge.go` | `Gauge(name string, opts ...Option) *gauge` — settable up/down |
| `health.go` | `CheckFunc = func(ctx) Status`; `Health(name, check, opts...) Healthcheck` |
| `metrics_test.go` | Unit tests |

## API

```go
type Metric interface {
    Name() string
    Describe() Descriptor
}

type Healthcheck interface {
    Check(ctx context.Context) Status
}

type Status struct { Code StatusCode; Reason string }
const (
    StatusOK StatusCode = iota
    StatusDegraded
    StatusDown
)

// Only constructors for Status:
OK(reason), Degraded(reason), Down(reason)
```

## Rules

1. **Entry points**: `Counter`, `Gauge`, `Health` — no `NewCounter`, no `New*`. Matches roadmap §16 target naming.
2. **`Status` is constructed only via `OK` / `Degraded` / `Down`**. Do not expose `StatusCode` for direct `Status{Code: ...}` construction in new code; keep users on the helpers.
3. Concrete return types (`*counter`, `*gauge`) are **unexported**. External code binds to them via methods, not by name. If you need a richer typed return, promote it to an interface first.
4. Labels (`WithLabel`, `WithLabels`) are **immutable per metric instance** — changing labels means a new metric.
5. `CheckFunc` is a type alias (`=`), not a named type. Intentional — keeps `Health(name, func(ctx) Status { ... })` terse at call sites.
6. No package-level registry here yet. When one lands, gate it behind an explicit opt-in constructor (never auto-register on `Counter(...)`).

## Validation

```bash
bazel test //pkg/component/metrics/...
go test -race ./pkg/component/metrics
```
