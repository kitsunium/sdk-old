<!-- updated: 2026-04-16T00:00:00Z -->
# pkg — Kitsunium SDK

Root of the public SDK. Three strict unidirectional layers.

## Layer map

```
pkg/component/   → logger, metrics                 (ready-to-use products)
      ↑
pkg/core/        → config/parser, config/normalize (composable bricks)
      ↑
pkg/kernel/      → pool, cache, errs, files        (stdlib wrappers, hot paths)
      ↑
                  Go stdlib
```

## Dependency rules (ENFORCED)

| Layer | May depend on | MUST NOT depend on |
|---|---|---|
| `pkg/kernel/*` | Go stdlib only | Anything else in the repo, external deps |
| `pkg/core/*` | `pkg/kernel/*`, stdlib | `pkg/component/*`, stdlib where a kernel wrapper exists |
| `pkg/component/*` | `pkg/core/*`, `pkg/kernel/*`, stdlib | Anything above itself |

Forbidden: reverse deps, layer skipping, non-stdlib deps in `kernel`.

## Current packages (tracked in git)

| Path | Package | Status |
|---|---|---|
| `pkg/kernel/pool` | `pool` | active — lock-free byte buffers (amd64/arm64), sync.Pool-backed global |
| `pkg/kernel/cache` | `cache` | active — LRU, atomic LRU, sharded LRU, FastLRU |
| `pkg/kernel/errs` | `errs` | active — error registry + instances + metrics + context propagation |
| `pkg/kernel/files` | `files` | active — File, Directory, Archive, Stats, System abstractions |
| `pkg/core/config/parser` | `parser` | active — JSON / YAML / TOML / INI / XML / ENV / ARGS parsers |
| `pkg/core/config/normalize` | `normalize` | active — key/value normalization, zero-alloc string↔bytes |
| `pkg/component/logger` | `logger` | active — `JSON(...)`, `Text(...)` with typed `Field` API |
| `pkg/component/metrics` | `metrics` | active — `Counter`, `Gauge`, `Health` primitives |

## Naming convention (target state)

> Package = domain. Function = specialization.
> Read `import + call` as a sentence: `pool.Buffer(1024)` = "pool, give me a buffer of 1024".

### Status vs target

Constructors currently use the legacy `New*` prefix (`NewSafeBuffer`, `NewLRU`, `NewJSON`, …). The roadmap (CLAUDE.md §16, "Refactor constructors") migrates them to the bare form with functional options:

| Legacy | Target |
|---|---|
| `pool.NewSafeBuffer(1024)` | `pool.Buffer(1024, pool.Safe())` |
| `pool.NewUnsafeShardedBuffer(1024, 16)` | `pool.Buffer(1024, pool.Unsafe(), pool.Sharded(16))` |
| `cache.NewLRU[K,V](1000)` | `cache.LRU[K,V](1000)` |
| `cache.NewShardedLRU(cap, n)` | `cache.LRU(cap, cache.Sharded(n))` |
| `parser.NewYAML(path)` | `parser.YAML.Load(path)` |
| `errs.Define(KConfig{...})` | keep (already domain-first) |

Until the migration lands, **do not** introduce new `New*` constructors — add the bare form and delegate from `New*` for back-compat during transition.

## Rules for changes in `pkg/`

1. `go.mod` require list must stay empty except for stdlib. External deps are rejected in `kernel/`; `core/` and `component/` need explicit PR justification.
2. After adding/removing/renaming a `.go` file, run `bazel run //:gazelle` so BUILD.bazel stays truthful.
3. New public symbol ⇒ one-line godoc + colocated `_test.go` + bench in `kernel/` hot paths.
4. Coverage gate: **90%** enforced in CI.
5. Every package keeps a top-level `README.md` (usage + perf notes).

## Validation commands

```bash
bazel build //pkg/...           # layer-wide build
bazel test //pkg/...            # layer-wide tests (race detector on)
bazel run //:gazelle            # regenerate BUILD.bazel after file moves
```
