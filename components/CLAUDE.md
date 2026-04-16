<!-- updated: 2026-04-16T00:00:00Z -->
# components

**Public SDK surface.** End users import from here.

## Dependency contract

- Imports allowed: `ports/`, `internal/kernel/*`, `internal/core/*`, Go stdlib.
- Imports forbidden: `adapters/` (inversion of control — components declare ports, adapters implement them).
- Enforced by `internal/core/contract/arch_test.go`.

## Packages

| Path | Purpose |
|---|---|
| [`logger`](./logger) | Structured logging — Repository + Instance (multiton) |
| [`metrics`](./metrics) | Counters, gauges, health checks |

## Component contract (v1)

Every component exposes a uniform API:

```go
type Config struct { /* declarative fields */ }
type Instance interface { /* domain methods */ }
type Repository interface {
    Instance(name string, sink SinkID, sinks ...SinkID) (Instance, error)
    Get(name string) (Instance, bool)
    Delete(name string) error
    Close(ctx context.Context) error
}

func FromConfig(cfg Config) (Repository, error)    // structural validation + sink Open
func MustFromConfig(cfg Config) Repository         // panic-on-error variant
```

The `Repository.Output` namespace exposes sinks as struct fields (`repo.Output.Console`, `repo.Output.AWS.S3`).

## Planned

- `server` — HTTP / gRPC transports (future phase)
- `tracer` — OTEL-compatible spans (future phase)

## Validation

```bash
bazel build //components/...
bazel test //components/...
```
