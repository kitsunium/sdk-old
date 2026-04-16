# pkg/component

Ready-to-use SDK products. The component layer is the top of the
Kitsunium SDK dependency stack.

## Dependency rule

```
pkg/kernel/*   <- stdlib only
pkg/core/*     <- may depend on pkg/kernel/*
pkg/component/* <- may depend on pkg/core/* and pkg/kernel/*
```

Components MUST NOT import anything above themselves, and MUST NOT
pull in external (non-stdlib) dependencies unless explicitly approved.

## Naming convention

Every component follows the rule `package = domain, function = specialization`:

```go
logger.JSON(opts...)       // JSON-formatted Logger
logger.Text(opts...)       // text-formatted Logger

metrics.Counter(name, ...) // thread-safe counter
metrics.Gauge(name, ...)   // thread-safe gauge
metrics.Health(name, ...)  // named health check
```

## Current components

| Package | Products |
|---------|----------|
| [logger](./logger) | `logger.JSON`, `logger.Text` |
| [metrics](./metrics) | `metrics.Counter`, `metrics.Gauge`, `metrics.Health` |

## Planned components

- `server` — HTTP / FTP / gRPC transports.
- Additional future products will be listed here as they land.
