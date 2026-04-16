<!-- updated: 2026-04-16T00:00:00Z -->
# v1/adapters/logger

Implementations of `ports/logger.Sink`. Each subdirectory is one sink
backend.

## Planned adapters

| Adapter | Status | Transport |
|---|---|---|
| `console/` | SKELETON | `io.Writer` (stdout, stderr, test buffers) |
| `file/` | TODO | Local filesystem with rotation + fsync policy |
| `syslog/` | TODO | Unix `syslog(3)` / RFC 5424 |
| `cloud/aws/cloudwatch/` | TODO | AWS CloudWatch Logs PutLogEvents |
| `cloud/aws/s3/` | TODO | S3 batched object upload |
| `cloud/grafana/loki/` | TODO | Loki push API |

## Constructor convention

```go
import (
    plogger "github.com/kitsunium/sdk/v1/ports/logger"
    "github.com/kitsunium/sdk/v1/adapters/logger/console"
)

sink := console.Sink(os.Stderr)     // returns plogger.Sink
```

Function name = the backend. Package name already said "logger
adapter"; the caller reads `console.Sink(...)` as "a console sink".

## Rules

1. Each adapter SHOULD implement `Sink` + `Flusher`. `Opener` and
   `Closer` when the backend needs explicit lifecycle (files, network).
2. Each adapter MUST accept a `Format` capability set at constructor
   time; `Write` MUST reject unsupported formats.
3. Adapters MUST NOT retain the `EntryEvent` past `Write` return
   unless they copy its payload — fanout may reuse the event.
4. Per-sink backpressure is the component's job (`internal/core/ring` +
   `fanout`). Adapters see one `Write(ctx, event)` at a time.
