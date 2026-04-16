<!-- updated: 2026-04-16 -->
# Kitsunium SDK — Project Instructions

## 1.0 Vision

High-performance Go SDK where a business project imports **ready-to-use
components** (logger, metrics, server, …) from a single stable surface and
stays insulated from the stdlib wrappers, caches, and pools that power them.

## 2.0 Architecture — Ports & Adapters over a Private Core

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

### Layers

| Directory | Visibility | Purpose |
|---|---|---|
| `internal/kernel/` | private (Go `internal/`) | Stdlib-only primitives (`pool`, `cache`, `errs`, `files`) |
| `internal/core/` | private | Composition on top of kernel (`config/parser`, `config/normalize`, future `fanout`, `lifecycle`, `contract`) |
| `ports/` | **public** | Interfaces (`Sink`, `Entry`, `Exporter`, lifecycle). Zero impl. |
| `components/` | **public** | Consumable products (`logger`, `metrics`, future `server`). Declare the ports they need. |
| `adapters/` | **public** | Implementations of ports (`console`, `syslog`, `aws/cloudwatch`, …). |

### Dependency rules (MECHANICALLY ENFORCED)

| Layer | May import | MUST NOT import |
|---|---|---|
| `internal/kernel/` | Go stdlib only | anything else in the repo |
| `internal/core/` | `internal/kernel/*`, stdlib, vetted 3rd-party | `components/`, `adapters/`, `ports/` |
| `ports/` | Go stdlib only | anything else in the repo |
| `components/` | `ports/`, `internal/*`, stdlib, 3rd-party | `adapters/` |
| `adapters/` | `ports/`, `internal/*`, stdlib, 3rd-party | `components/` |

Enforced by `internal/core/contract/arch_external_test.go` — a `go/ast`
walker that rejects any violating import. Runs via `go test -tags=archcheck
./internal/core/contract/...` in CI. Bazel sandbox skips it (no workspace
access) — `go test` is the authoritative enforcer.

### Naming convention (for new code)

Package = domain. Function = specialization. Read `import + call` as a
sentence: `pool.Buffer(1024)` = *"pool, give me a buffer of 1024"*.

1. No `New*` prefix on constructors in new APIs (`pool.Buffer(n, opts...)`).
2. Variants via functional options, not struct names (`pool.Buffer(n, pool.Safe(), pool.Sharded(16))`).
3. No abbreviations in package names unless resolving a collision (`errs` exists because `error` is a Go keyword AND `errors` is stdlib).
4. Plural OK when it reads better (`files`, `components`).

## 3.0 Build System

**Bazel is the source of truth.** Go native tooling is for day-to-day dev.

| Task | Canonical | Dev alternative |
|---|---|---|
| Build | `bazel build //...` | `go build ./...` |
| Test | `bazel test //...` | `go test ./...` |
| Arch enforcement | `go test -tags=archcheck ./internal/core/contract/...` | — |
| Coverage | `bazel coverage --combined_report=lcov //...` | `go test -cover ./...` |
| Regenerate BUILD | `bazel run //:gazelle` | — |

Always run `gazelle` after adding/removing Go files.

## 4.0 Go Version

Pinned to **Go 1.26.1** (`go.mod`, `MODULE.bazel`, `.github/workflows/ci.yml`). Bumps update all three atomically.

## 5.0 Quality Gates

**Linting:** `ktn-linter` only (`.ktn-linter.yaml`). No golangci-lint, no prettier.
**Coverage:** 90% minimum in CI. Exemptions justified in PR + `.ktn-linter.yaml` `exclude:`.
**Review:** human + CodeRabbit (`.coderabbit.yaml`) + Qodo Merge (`.pr_agent.toml`) + local `/review` skill before pushing.

## 6.0 Git Workflow

- Never commit to `main` directly.
- Branch naming: `feat/*`, `fix/*`, `refactor/*`, `docs/*`, `chore/*`, `perf/*`.
- Conventional commits (`feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert|merge|wip`).
- **No AI references** in commits (`Co-Authored-By: Claude/AI`, robot emoji, etc.).
- PR required, CI green, squash-only merge into `main`.
- Use `/git --commit` and `/git --merge`.

## 7.0 Versioning (Semver strict)

- `v0.x.y` during development (current phase).
- `v1.0.0` when the public API (`ports/`, `components/`, `adapters/`) is stable.
- Signed tags: `git tag -s vX.Y.Z -m "Release X.Y.Z"`. `MODULE.bazel` `version` matches latest tag.

## 8.0 Performance Discipline

- Benchmarks mandatory for hot-path code in `internal/kernel/` (`*_bench_test.go`).
- No regression > 5% without explicit PR justification.
- Baseline database: `scripts/bench_manager.py`.

## 9.0 Testing Discipline

- Every public function tested.
- Table-driven tests preferred.
- Race detector on in CI.
- File suffixes enforced by ktn-linter: `_internal_test.go` (white-box), `_external_test.go` (black-box), `_bench_test.go` (benchmarks), `_integration_test.go`.

## 10.0 Documentation

- Every public package has a `README.md` (usage + perf notes).
- Every exported symbol has a one-line godoc.

## 11.0 MCP-First / GrepAI-First

See `~/.claude/CLAUDE.md` for global rules. `mcp__grepai__*` before `Grep`, `mcp__github__*` before `gh`.

## 12.0 Skills

`/plan`, `/do`, `/review`, `/lint`, `/git`, `/feature`, `/search`, `/docs`.

## 13.0 Safeguards

- Never delete `.claude/`, `.devcontainer/`, `scripts/` without explicit user confirmation.
- When renaming packages: update BUILD.bazel + gazelle + verify imports + re-run arch test.
- When moving content: move, never delete logic.

## 14.0 Context Hierarchy

```
/workspace/CLAUDE.md                    ← this file (project)
├── .devcontainer/CLAUDE.md             ← container setup
│   └── images/.claude/CLAUDE.md        ← Claude container layer
├── ports/                              ← (no CLAUDE.md — contracts only)
├── components/CLAUDE.md                ← component contract v1
├── adapters/CLAUDE.md                  ← (future) adapter conventions
└── internal/
    ├── kernel/CLAUDE.md                ← kernel-layer rules
    └── core/CLAUDE.md                  ← core-layer rules
```

## 15.0 Roadmap — Deep Restructure (in progress)

Target: replace the old `kernel / core / component` triple with a
ports-and-adapters model where `components/` is the single public surface
for end users, `internal/*` is Go-enforced private, and `adapters/` carries
concrete I/O (including external-dep-heavy providers like AWS).

- [x] **Phase 0-1**: `pkg/* → internal/*` + `pkg/component → components` physical move, imports rewritten, Bazel regenerated
- [x] **Phase 1b**: `internal/core/contract/arch_external_test.go` — mechanical layering enforcement
- [x] **Phase 2**: `ports/` package — `Sink`, `Entry`, `Exporter`, `Opener/Closer/Flusher`
- [ ] **Phase 3**: component contract v1 — uniform `FromConfig` / `Repository` / `Instance(name, sink, sinks...)` interfaces
- [ ] **Phase 4**: `internal/core/fanout/` — ring buffer + worker pool per sink
- [ ] **Phase 5**: `adapters/console/` + `adapters/stream/` (first adapters)
- [ ] **Phase 6**: `components/logger` rewrite against the new contract
- [ ] **Phase 7**: `adapters/syslog/` + `adapters/file/`
- [ ] **Phase 8**: `components/metrics` rewrite, same contract
- [ ] **Phase 9**: `adapters/cloud/aws/cloudwatch/` — first external-dep adapter
- [ ] **Phase 10**: public docs, migration guide
