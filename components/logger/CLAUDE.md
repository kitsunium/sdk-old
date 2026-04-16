<!-- updated: 2026-04-16T00:00:00Z -->
# pkg/component/logger

Structured logger with typed fields. Two flavors: `logger.JSON(...)` (machine-readable) and `logger.Text(...)` (human-readable).

## Files

| File | Role |
|---|---|
| `logger.go` | Public API: `Logger` interface, `Level`, `Field`, typed field constructors, `Option` + `WithLevel/WithOutput/WithTimeFormat` |
| `json.go` | `JSON(opts ...Option) Logger` — JSON line output |
| `text.go` | `Text(opts ...Option) Logger` — human-readable output |
| `logger_test.go` | Unit tests (JSON + Text + field kinds) |

## API

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(err error, msg string, fields ...Field)
    With(fields ...Field) Logger
}

// Levels
const (
    LevelDebug Level = iota
    LevelInfo
    LevelWarn
    LevelError
)

// Field constructors (typed, no reflection on hot path)
String, Int, Int64, Float, Bool, Err, NamedErr, Any

// Options
WithLevel(Level)
WithOutput(io.Writer)
WithTimeFormat(layout string)
```

## Rules

1. **Top-level entry points are `JSON` and `Text`** — not `NewLogger`. This is the target naming from roadmap §16 and must stay that way.
2. **Typed fields are the default**. `Any(k, v)` is an escape hatch — discourage it on the hot path (it boxes into `any`).
3. `Error(err, msg, fields...)` takes the error **first** deliberately — it enforces at the type system that an error log carries one. Do not reorder.
4. `With(fields...) Logger` returns a child logger with pre-bound fields; it must not share mutable state with the parent beyond the underlying writer.
5. Default level is `LevelInfo` (see `logger.go`); default output is `os.Stderr` unless overridden.
6. No global logger in this package. Callers own the instance.

## Validation

```bash
bazel test //pkg/component/logger/...
go test -race ./pkg/component/logger
```
