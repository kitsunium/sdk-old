<!-- updated: 2026-04-16T00:00:00Z -->
# pkg/component

Ready-to-use SDK products. Top of the Kitsunium dependency stack. A business project imports from here and gets a batteries-included primitive (logger, metrics, future transports).

## Dependency contract

- Imports allowed: `pkg/core/*`, `pkg/kernel/*`, Go stdlib.
- Imports forbidden: anything above itself (there is no layer above component), non-stdlib deps without explicit PR approval.
- Components MAY re-export small helper types from core/kernel, but MUST NOT leak implementation details (e.g. don't return an internal `*safeBuffer`).

## Packages

| Path | Package | Purpose | Public entry points |
|---|---|---|---|
| [`logger`](./logger) | `logger` | Structured logging with typed fields | `JSON(opts...)`, `Text(opts...)`, `Logger`, `Level`, `Field`, `String/Int/Int64/Float/Bool/...` field constructors |
| [`metrics`](./metrics) | `metrics` | Counters, gauges, health checks | `Counter(name, opts...)`, `Gauge(name, opts...)`, `Health(name, check, opts...)`, `Metric`, `Healthcheck`, `Status`, `OK/Degraded/Down` |

## Planned

- `server` — HTTP / FTP / gRPC transports. Target API: `server.HTTP(...)`, `server.FTP(...)`, `server.gRPC(...)`.

Do not pre-create empty subpackage directories for planned components; they were cleaned up during the kernel-rename refactor and should only appear once the first implementation file exists.

## Conventions

1. **Naming**: component = domain, top-level function = product. Examples above: `logger.JSON(...)`, `metrics.Counter(...)`, `metrics.Health(...)`. No `NewLogger`, no `NewCounter`.
2. **Functional options only** for configuration — do not add `SetXxx` methods on concrete return types.
3. **Interfaces, not structs**, as return types of top-level functions (`logger.JSON(...) Logger`, `metrics.Health(...) Healthcheck`).
4. **Status helpers** (`metrics.OK`, `Degraded`, `Down`) are the only public way to construct a `Status` — don't expose the `StatusCode` enum for direct construction.
5. **No global state** except where explicitly documented (e.g. metrics registry). A component instance must be usable without touching package-level vars.

## Validation

```bash
bazel build //pkg/component/...
bazel test //pkg/component/...
```
