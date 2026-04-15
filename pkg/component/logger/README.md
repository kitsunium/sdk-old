# logger

Structured, leveled logger for the Kitsunium SDK. Stdlib-only, zero
external dependencies. Two interchangeable formats:

- `logger.JSON(opts...)` — one JSON object per line, machine-friendly.
- `logger.Text(opts...)` — `time LEVEL msg key=value ...`, human-friendly.

Both implement the `Logger` interface and are safe for concurrent use.

## Quick start

```go
import "github.com/kitsunium/sdk/pkg/component/logger"

log := logger.JSON(
    logger.WithLevel(logger.LevelInfo),
    logger.WithOutput(os.Stdout),
)

log.Info("server started", logger.String("addr", ":8080"), logger.Int("pid", os.Getpid()))

svc := log.With(logger.String("service", "api"))
svc.Warn("slow query", logger.Float("ms", 412.5))

if err := doWork(); err != nil {
    svc.Error(err, "worker failed", logger.String("job", "ingest"))
}
```

## Options

| Option | Description |
|--------|-------------|
| `WithLevel(Level)` | Minimum level emitted. `LevelOff` silences everything. |
| `WithOutput(io.Writer)` | Destination writer (default: `os.Stdout`). |
| `WithTimeFormat(string)` | Go time layout (default: `time.RFC3339`). |

## Field constructors

`String`, `Int`, `Int64`, `Float`, `Bool`, `Err`, `NamedErr`, `Any`.

## Levels

`LevelDebug` < `LevelInfo` < `LevelWarn` < `LevelError` < `LevelOff`.
