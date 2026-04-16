<!-- updated: 2026-04-16T00:00:00Z -->
# v1/components

Public products consumers import. Each component is a self-contained
DDD unit — domain + application + ports (as re-exports).

## Contract

| | Rule |
|---|---|
| **Imports allowed** | `ports/*`, `internal/*`, stdlib, vetted 3rd-party |
| **Imports forbidden** | `adapters/*` — components declare ports, adapters satisfy them |
| **Shape** | `domain/` (entities, VOs) + `application/` (use cases) + `ports/` (re-exports) + facade file |
| **API style** | `package.Thing(opts...)` — reads as "logger, give me JSON" |

## Components

| Component | Status | Purpose |
|---|---|---|
| [`logger`](./logger) | MIGRATED BASE | Structured logger with multi-sink fanout (spec) |
| [`config`](./config) | MIGRATED BASE | Multi-source config with string-path accessor (spec) |

## How a component names itself

Package name = the product. Factory function = the variant.

```go
logger.JSON(...)     // "logger, give me JSON"
logger.Text(...)     // "logger, give me Text"
config.FromSources(...)
config.FromStatic(...)
```

Avoid `NewLogger`, `NewJSONLogger`, etc. — the reader already knows
they asked `logger.<something>` for something.

## Rules

1. Components DECLARE their port needs (via `ports/<component>`);
   consumers wire the adapters.
2. No global state. No package-level `init` touching I/O. No
   singletons.
3. Every component must ship a CLAUDE.md *cahier des charges* before
   implementation.
4. Tests are table-driven where the input space is enumerable. Black-
   box tests in `*_external_test.go` ensure the facade matches the
   spec.

## Validation

```bash
cd v1 && go test -race ./components/...
bazel test //v1/components/...
```
