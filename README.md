# Kitsunium SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/kitsunium/sdk/v1.svg)](https://pkg.go.dev/github.com/kitsunium/sdk/v1)
[![codecov](https://codecov.io/gh/kitsunium/sdk/branch/main/graph/badge.svg)](https://codecov.io/gh/kitsunium/sdk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CI](https://github.com/kitsunium/sdk/workflows/CI/badge.svg)](https://github.com/kitsunium/sdk/actions)

High-performance Go SDK for Linux. Import ready-to-use **components**
(logger, config, future: metrics, server), wire in the **adapters**
you need (console, syslog, AWS CloudWatch, Grafana Loki…), and let
the private plumbing (pool, cache, ring, fanout) stay out of your way.

## Design principles

- **Pure performance.** Zero-alloc steady-state on hot paths,
  bench-gated in CI.
- **Simplicity.** `pool.Buffer(1024)` reads as "pool, give me a
  buffer". No ceremony.
- **No reflection** on hot paths. Typed field constructors everywhere.
- **Linux-only.** We target a known kernel surface — io_uring, epoll,
  splice, vDSO-backed time.
- **Ports-and-adapters** per component. Components declare
  interfaces; adapters satisfy them. The two never import each other.
- **Multi-module monorepo.** Each major version is an independent
  Go module under `v1/`, `v2/`, `v3/`… Breaking changes cross the vN
  boundary only.

## Status

**Pre-1.0 — v1 module in active development.**

The code lives under `v1/`. Track the roadmap in
[CLAUDE.md §17](./CLAUDE.md#170-roadmap). Cahiers des charges for the
two in-progress components:

- [`v1/components/logger/CLAUDE.md`](./v1/components/logger/CLAUDE.md)
- [`v1/components/config/CLAUDE.md`](./v1/components/config/CLAUDE.md)

## Installation

```bash
go get github.com/kitsunium/sdk/v1
```

## Architecture

```
                 ┌──────── PUBLIC ────────┐
 adapters/ ──────┤                        │
                 ▼                        ▼
              ports/ ◀─────────── components/
                                        │
              ┌──────── PRIVATE ────────┘
              ▼
      internal/core/ ──▶ internal/kernel/ ──▶ Go stdlib (linux)
```

- `v1/components/` — the stable public surface (what you import)
- `v1/ports/` — interfaces components declare, adapters implement
- `v1/adapters/` — concrete backends (console, syslog, cloud providers)
- `v1/internal/{core,kernel}/` — private plumbing, Go-enforced
  - `kernel/` — syscall-near, stateless, Linux-only
  - `core/` — stateful, generic, component-reusable

Layering is enforced mechanically via `go/ast` in
`v1/internal/core/contract/arch_external_test.go`.

## Quick look — logger (spec target)

```go
import (
    "os"

    "github.com/kitsunium/sdk/v1/components/logger"
    "github.com/kitsunium/sdk/v1/adapters/logger/console"
    plogger "github.com/kitsunium/sdk/v1/ports/logger"
)

lg := logger.JSON(
    logger.WithLevel(logger.LevelInfo),
    logger.WithSink(console.Sink(os.Stderr, plogger.FormatJSON)),
)

lg.Info("server started",
    logger.String("addr", ":8080"),
    logger.Int("pid", os.Getpid()),
)

svc := lg.With(logger.String("service", "api"))
if err := doWork(); err != nil {
    svc.Error(err, "worker failed", logger.String("job", "ingest"))
}
```

## Quick look — config (spec target)

```go
import "github.com/kitsunium/sdk/v1/components/config"

cfg, err := config.FromSources(
    config.File("config.yaml"),            // base
    config.File("config.local.yaml"),      // local overrides
    config.Env("APP_"),                    // APP_DATABASE_URL → "database.url"
    config.Args(true),                     // --database.host=... wins
)
if err != nil { log.Fatal(err) }

host, _ := cfg.String("database.host")
port, _ := cfg.Int("server.port")
tls,  _ := cfg.Bool("tls.enabled")
```

## Development

### Prerequisites

- Go 1.26.1
- Bazel (source of truth for builds)
- Linux/Debian host (WSL, native, or container)

### Build & test

```bash
# Canonical (Bazel)
bazel build //...
bazel test //...

# Dev alternative (Go native inside v1/)
cd v1 && go build ./...
cd v1 && go test ./...

# Architecture enforcement
cd v1 && go test -tags=archcheck ./internal/core/contract/...

# Regenerate BUILD.bazel after moves
bazel run //v1:gazelle
```

### Code quality

- **Linter**: `ktn-linter` only (148 rules, 8 phases,
  `.ktn-linter.yaml`).
- **Coverage**: 90% gate in CI, per-package exemptions in
  `.ktn-linter.yaml`.
- **Review**: human + CodeRabbit (`.coderabbit.yaml`) + Qodo Merge
  (`.pr_agent.toml`) + local `/review` skill.

### Benchmarks

Hot-path code in `v1/internal/kernel/` and `v1/internal/core/`
requires benchmarks (`*_bench_test.go`). Regressions > 5% block the
PR.

```bash
cd v1 && go test -bench=. -benchmem ./internal/core/pool/...
bazel test //v1/internal/core/pool:bench --test_arg=-bench=.
```

## Contributing

- No direct commits to `main` — PRs only, CI green, squash-merge.
- Branch naming: `feat/*`, `fix/*`, `refactor/*`, `docs/*`, `chore/*`,
  `perf/*`.
- Conventional commits enforced.
- No AI references in commit messages (enforced by `git-guard.sh`).
- See [CLAUDE.md](./CLAUDE.md) for project instructions.

## License

Apache 2.0 — see [LICENSE](LICENSE).

## Links

- [v1 API docs](https://pkg.go.dev/github.com/kitsunium/sdk/v1)
- [Issues](https://github.com/kitsunium/sdk/issues)
- [Roadmap](./CLAUDE.md#170-roadmap)
