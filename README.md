# Kitsunium SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/kitsunium/sdk.svg)](https://pkg.go.dev/github.com/kitsunium/sdk)
[![codecov](https://codecov.io/gh/kitsunium/sdk/branch/main/graph/badge.svg)](https://codecov.io/gh/kitsunium/sdk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CI](https://github.com/kitsunium/sdk/workflows/CI/badge.svg)](https://github.com/kitsunium/sdk/actions)

High-performance Go SDK. Import ready-to-use **components** (logger, metrics,
future: server), wire in the **adapters** you need (console, syslog, AWS
CloudWatch…), and the private plumbing (pool, cache, errs, files) stays out
of your way.

## Status

**Pre-1.0 — API in flux.** The SDK is being restructured into a
ports-and-adapters layout. Track the roadmap in [CLAUDE.md §15](./CLAUDE.md#150-roadmap--deep-restructure-in-progress).

## Architecture

```
                 ┌──────── PUBLIC ────────┐
 adapters/ ──────┤                        │
                 ▼                        ▼
              ports/ ◀─────────── components/
                                        │
              ┌──────── PRIVATE ────────┘
              ▼
      internal/core/ ──▶ internal/kernel/ ──▶ Go stdlib
```

- `components/` — the stable public surface (what you import)
- `ports/` — interfaces components declare, adapters implement
- `adapters/` — concrete backends (console, syslog, cloud providers) — not yet populated
- `internal/{core,kernel}/` — private plumbing, Go-enforced

Layering is enforced mechanically via `go/ast` in
`internal/core/contract/arch_external_test.go`.

## Installation

```bash
go get github.com/kitsunium/sdk
```

## Quick look — current state (legacy API)

The constructors listed here still use the `New*` form; the v1 target (per
roadmap §16) is bare-form with functional options:
`logger.JSON(opts...)`, `pool.Buffer(n, pool.Safe())`, etc.

### Logger

```go
import "github.com/kitsunium/sdk/components/logger"

log := logger.JSON(
    logger.WithLevel(logger.LevelInfo),
    logger.WithOutput(os.Stdout),
)

log.Info("server started", logger.String("addr", ":8080"), logger.Int("pid", os.Getpid()))

svc := log.With(logger.String("service", "api"))
if err := doWork(); err != nil {
    svc.Error(err, "worker failed", logger.String("job", "ingest"))
}
```

### Metrics

```go
import "github.com/kitsunium/sdk/components/metrics"

requests := metrics.Counter("http_requests_total", metrics.WithHelp("Total HTTP requests"))
requests.Inc()

queueDepth := metrics.Gauge("queue_depth")
queueDepth.Set(42)

db := metrics.Health("db", func(ctx context.Context) metrics.Status {
    if err := ping(ctx); err != nil {
        return metrics.Down(err.Error())
    }
    return metrics.OK("reachable")
})
```

## Development

### Prerequisites

- Go 1.26.1
- Bazel (source of truth for builds)

### Build & test

```bash
bazel build //...                                          # full build
bazel test //...                                           # full test
go test -tags=archcheck ./internal/core/contract/...       # architecture enforcement
bazel run //:gazelle                                       # regenerate BUILD.bazel after moves
```

Non-Bazel fallback: `go build ./...`, `go test ./...`.

### Code quality

- **Linter**: `ktn-linter` only (148 rules, 8 phases, `.ktn-linter.yaml`).
- **Coverage**: 90% gate in CI, per-package exemptions in `.ktn-linter.yaml`.
- **Review**: human + CodeRabbit + Qodo Merge + local `/review` skill.

### Benchmarks

Hot-path code in `internal/kernel/` requires benchmarks
(`*_bench_test.go`). Regressions >5% block the PR.

```bash
bazel test //internal/kernel/pool:bench --test_arg=-bench=.
go test -bench=. -benchmem ./internal/kernel/pool
```

## Contributing

- No direct commits to `main` — PRs only, CI green, squash-merge.
- Branch naming: `feat/*`, `fix/*`, `refactor/*`, `docs/*`, `chore/*`, `perf/*`.
- Conventional commits enforced.
- No AI references in commit messages (enforced by `git-guard.sh`).
- See [CLAUDE.md](./CLAUDE.md) for project instructions consumed by Claude Code.

## License

Apache 2.0 — see [LICENSE](LICENSE).

## Links

- [API docs](https://pkg.go.dev/github.com/kitsunium/sdk)
- [Issues](https://github.com/kitsunium/sdk/issues)
- [Roadmap](./CLAUDE.md#150-roadmap--deep-restructure-in-progress)
