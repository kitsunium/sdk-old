<!-- updated: 2026-04-16T00:00:00Z -->
# v1/adapters/logger/console

Reference implementation of `ports/logger.Sink`. Writes every
EntryEvent to an `io.Writer` — typically `os.Stdout` or `os.Stderr`.

## Planned API

```go
import (
    "github.com/kitsunium/sdk/v1/adapters/logger/console"
    plogger "github.com/kitsunium/sdk/v1/ports/logger"
)

// Build a sink that writes JSON-encoded records to stderr.
sink := console.Sink(os.Stderr, plogger.FormatJSON)

// Register with the logger component.
lg := logger.JSON(logger.WithSink(sink))
```

## Behaviour

| Aspect | Behaviour |
|---|---|
| Format support | `FormatJSON`, `FormatText` (rejects `FormatBinary`) |
| Concurrency | One mutex serialises Write; no goroutines spawned |
| Lifecycle | Implements `Sink` only — no Opener/Closer/Flusher |
| Retention | Does NOT retain `EntryEvent` past Write return |
| Error chain | All errors via `errs.Define` catalog (codes 2001–2003) |

## Rules

1. This adapter MUST stay self-contained — no reuse of
   `internal/core` primitives at this level (it is the simplest
   possible reference; other adapters layer on fanout/ring/retry).
2. Bench must hit ≤ 300 ns / ≤ 1 alloc for JSON, ≤ 200 ns / ≤ 1 alloc
   for Text on a warm pool. Measured in
   `console_bench_test.go`.
3. `FormatBinary` is a compile-time known rejection — returns
   `ErrUnsupportedFormat` wrapped with the numeric format code.

## Status

Skeleton only — see `components/logger/CLAUDE.md` §8 for the migration
path that consumers follow, and the logger cahier des charges that
drives the contract this adapter satisfies.
