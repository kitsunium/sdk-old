<!-- updated: 2026-04-15 -->
# Kitsunium SDK — Project Instructions

## 1.0 Vision

High-performance Go SDK providing **low-level wrappers** around the standard
library, organized in **strict unidirectional layers**, so that a business
project can import **ready-to-use components** (HTTP, logger, metrics…) tested,
validated, and just focus on its domain.

## 2.0 Architecture — Strict Layers

```
pkg/component/ → core/ + kernel/ + stdlib       [ready-to-use products]
      ↑
pkg/core/      → kernel/ only                   [composable bricks]
      ↑
pkg/kernel/    → stdlib only                    [optimized stdlib wrappers]
      ↑
      Go stdlib
```

### Dependency rules (ENFORCED)

| Layer | May depend on | MUST NOT depend on |
|---|---|---|
| `pkg/kernel/*` | Go stdlib only | Anything else in the repo |
| `pkg/core/*` | `pkg/kernel/*` | stdlib directly (use kernel wrappers), component |
| `pkg/component/*` | `pkg/core/*`, `pkg/kernel/*` | Anything above itself |

**Forbidden**: reverse dependencies, layer skipping, external deps in `kernel`.

## 3.0 Naming Convention — UNIVERSAL

> **Package = domain. Function = specialization within that domain.**

Reading an import + call must form a sentence: `pool.Buffer(1024)` = "pool, give me a buffer of 1024".

### Kernel layer

| Path | Package | API examples |
|---|---|---|
| `pkg/kernel/pool/` | `pool` | `pool.Buffer(1024)`, `pool.Worker(n)` (future), `pool.Connection(...)` (future) |
| `pkg/kernel/cache/` | `cache` | `cache.LRU(1000)`, `cache.Atomic(...)`, `cache.Sharded(...)` |
| `pkg/kernel/errs/` | `errs` | `errs.New(code, name, msg)`, `errs.Register(...)`, `errs.Wrap(...)` |
| `pkg/kernel/files/` | `files` | `files.Stats(path)`, `files.Archive(...)`, `files.Walk(...)` |

### Core layer

| Path | Package | API examples |
|---|---|---|
| `pkg/core/config/parser/` | `parser` | `parser.YAML.Load(...)`, `parser.JSON.Load(...)`, `parser.TOML.Load(...)` |
| `pkg/core/config/normalize/` | `normalize` | `normalize.Keys(...)` |

### Component layer

| Path | Package | API examples |
|---|---|---|
| `pkg/component/logger/` | `logger` | `logger.JSON(...)`, `logger.Text(...)`, `logger.Structured(...)` |
| `pkg/component/metrics/` | `metrics` | `metrics.Health(...)`, `metrics.Counter(...)`, `metrics.Gauge(...)` |
| `pkg/component/server/` *(planned)* | `server` | `server.HTTP(...)`, `server.FTP(...)`, `server.gRPC(...)` |

### Rules

1. **No `New` prefix on constructors** — `pool.Buffer(1024)`, not `pool.NewBuffer(1024)`.
2. **Variants via functional options** — `pool.Buffer(1024, pool.Unsafe(), pool.Sharded(16))` rather than `NewUnsafeShardedBuffer`.
3. **No abbreviations** in package names unless resolving a collision (`errs` exists because `error` is a Go built-in AND `errors` is stdlib).
4. **Plural OK** when it reads better (`files` > `file`).
5. **No underscores or hyphens** in package names (Go convention).

## 4.0 Build System

**Bazel is the source of truth.** Go native tooling is for day-to-day dev convenience.

| Task | Canonical | Dev alternative |
|---|---|---|
| Build | `bazel build //...` | `go build ./...` |
| Test | `bazel test //...` | `go test ./...` |
| Coverage | `bazel coverage --combined_report=lcov //...` | `go test -cover ./...` |
| Regenerate BUILD | `bazel run //:gazelle` | — |

Always run **gazelle** after adding/removing Go files.

## 5.0 Go Version

Pinned to **Go 1.26.1** (`go.mod`, `MODULE.bazel`, `.github/workflows/ci.yml`).
Bumps require updating all three.

## 6.0 Quality Gates

### Linting
**Single linter: `ktn-linter`** (`github.com/kodflow/ktn-linter`).
Config: `.ktn-linter.yaml` at root. No golangci-lint, no prettier, no go vet.

### Coverage
**90% minimum**, enforced in CI. Per-package exemptions MUST be justified in PR
description and recorded in `.ktn-linter.yaml` `exclude:` section.

### Review
1. Human review (PR).
2. **CodeRabbit** (automatic on PR, `.coderabbit.yaml`).
3. **Qodo Merge** (complementary IA review, `.pr_agent.toml`).
4. Local `/review` skill before pushing (tiers 1-3).

## 7.0 Git Workflow

- **Never commit to `main` directly.**
- **Branch naming**: `feat/*`, `fix/*`, `refactor/*`, `docs/*`, `chore/*`, `perf/*`.
- **Conventional commits** required (`feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert|merge|wip`).
- **No AI references in commits** (`Co-Authored-By: Claude/AI`, robot emoji, etc.).
- **PR required** (even solo). CI must be green.
- **Squash-only merge** into `main`.
- Use `/git --commit` and `/git --merge` skills.

## 8.0 Versioning — Semver Strict

- `v0.x.y` during development (current phase).
- `v1.0.0` when the public API is stable.
- Git tags signed: `git tag -s vX.Y.Z -m "Release X.Y.Z"`.
- `MODULE.bazel` `version` field matches latest tag.

## 9.0 Performance Discipline

- **Benchmarks mandatory** for any new hot-path code in `kernel/` (add `*_bench_test.go`).
- **Benchmark comparison** runs automatically on PR (`benchmark-compare` job).
- **No regression > 5%** without explicit justification in PR.
- `scripts/bench_manager.py` manages baseline database.

## 10.0 Testing Discipline

- **Every public function** must have tests.
- Table-driven tests preferred (`[]struct{ name, input, want }`).
- **Race detector** runs in CI.
- Test files colocated: `foo.go` + `foo_test.go` in same directory.
- External tests (`package foo_test`) OK for black-box testing.

## 11.0 Documentation

- Every package has a `README.md` with usage + performance notes.
- Every exported symbol has one-line godoc (Go style).
- No multi-paragraph godoc unless strictly necessary.

## 12.0 MCP-First & GrepAI-First

See `~/.claude/CLAUDE.md` for global rules:
- `mcp__grepai__*` for semantic search **before** Grep.
- `mcp__github__*` for GitHub API **before** `gh`.
- Fallback to CLI only when MCP unavailable.

## 13.0 Claude Skills

All skills in `~/.claude/commands/` are available. Key ones for this project:
- `/plan` — design new packages, refactors.
- `/do` — iterative execution (picks up approved `/plan`).
- `/review` — 3-tier review (agents + Qodo + CodeRabbit).
- `/lint` — ktn-linter (148 rules, 8 phases).
- `/git` — branch + conventional commit + PR.
- `/feature` — requirements traceability matrix.

## 14.0 Safeguards

- Never delete `.claude/`, `.devcontainer/`, `scripts/` without explicit user confirmation.
- When renaming packages: update BUILD.bazel + run gazelle + verify all imports.
- When moving content: move, never delete logic.

## 15.0 Context Hierarchy

```
/workspace/CLAUDE.md              ← THIS FILE (project)
├── .devcontainer/CLAUDE.md       ← container setup
│   └── images/.claude/CLAUDE.md  ← Claude container layer
├── pkg/kernel/CLAUDE.md          ← (optional) kernel-layer rules
├── pkg/core/CLAUDE.md            ← (optional) core-layer rules
└── pkg/component/CLAUDE.md       ← (optional) component-layer rules
```

## 16.0 Roadmap

Progressive upgrade tracked in task list. Current wave:
- [x] Delete `pkg/lib/` (obsolete with Go 1.18+ generics)
- [x] Delete `pkg/core/storage/` (stub)
- [x] Rename kernel packages: `kbuffer`→`pool`, `kcache`→`cache`, `kerror`→`errs`, `fs`→`files`
- [ ] Bump Go `1.24.6` → `1.26.1`
- [ ] Refactor constructors to `package.Thing(...)` style with functional options
- [ ] CI refonte: ktn-linter + Codecov + 90% gate
- [ ] First components: `logger` + `metrics` (health)
- [ ] Next: `server` component (HTTP first)
