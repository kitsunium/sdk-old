# AGENTS.md

Instructions for AI coding agents (Claude Code, CodeRabbit, Qodo Merge,
GitHub Copilot) working on this repository.

## Project

**Kitsunium SDK** — Go library providing optimized wrappers around the standard
library, organized in strict unidirectional layers, so business projects can
import ready-to-use components.

Module: `github.com/kitsunium/sdk`
Language: Go 1.26.1
Build: Bazel (source of truth) + Go native (dev alt)
Linter: `ktn-linter` (only, no alternatives)

## Golden Rules (non-negotiable)

### 1. Respect the layer hierarchy

```
pkg/component/  → may import core/, kernel/, stdlib
pkg/core/       → may import kernel/ only (no stdlib directly)
pkg/kernel/     → stdlib only (no internal imports)
```

- **Never** add an import that goes against the arrows.
- **Never** add an external third-party dep to `pkg/kernel/*`.
- **Never** skip a layer (component → stdlib directly).

### 2. Follow the naming convention

Package = **domain**. Function = **specialization**.

Correct:
```go
pool.Buffer(1024)
cache.LRU(1000)
errs.New("E001", "NotFound", "resource not found")
files.Stats("/path/to/file")
logger.JSON(logger.WithLevel(logger.Info))
metrics.Counter("requests_total")
```

Incorrect:
```go
pool.NewBuffer(1024)           // no New prefix
buffer.New(1024)               // package = domain (pool), not function
kerror.New(...)                // old k* prefix retired
fs.Stats(...)                  // fs was renamed files
```

### 3. Constructor arguments

- Small-arity positional: `cache.LRU(capacity)`.
- Variants/modifiers: functional options `...Option`.
- No optional booleans in positional args — always opts.

Example:
```go
buf := pool.Buffer(1024, pool.Unsafe(), pool.Sharded(16))
```

### 4. Errors

Use `pkg/kernel/errs` for all error creation in core/ and component/:
```go
var ErrNotFound = errs.New(404, "NOT_FOUND", "resource not found")
return err.WithDetail("id", userID)
```

### 5. Testing

- Table-driven tests preferred.
- Use stdlib `testing` only (no testify, no gocheck).
- Benchmarks mandatory for hot-path code in `kernel/`.
- Tests colocated: `foo.go` + `foo_test.go`.

### 6. BUILD.bazel

After every file creation/deletion/rename in `pkg/`:
```bash
bazel run //:gazelle
```

If gazelle unavailable, manually update:
- `name` (usually matches directory name)
- `importpath`
- `srcs` (add/remove files)
- `deps` (add/remove imports)
- For tests: `embed = [":package_name"]`

## What NOT to add

- No logging framework dependency (use `pkg/component/logger`).
- No HTTP framework dependency (future `pkg/component/server`).
- No ORM (out of scope).
- No testify / mock frameworks (stdlib only).
- No Codacy config (removed).
- No golangci-lint, no prettier, no go vet (replaced by ktn-linter).

## Commit rules

- Conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`, `perf:`, `test:`, `build:`.
- **No AI attribution** (`Co-Authored-By: Claude`, "Generated with AI", etc.) —
  enforced by `.githooks/pre-push`.
- Signed commits preferred (`git commit -S`).
- No commits to `main` directly — always via PR + squash merge.

## PR rules

- Title = conventional commit line.
- Description must state WHY, not just WHAT.
- Must pass CI (ktn-linter + test + build + coverage ≥ 90%).
- Must be reviewed by CodeRabbit + Qodo (automatic).
- Benchmark regressions > 5% require justification.

## When in doubt

1. Re-read `/workspace/CLAUDE.md`.
2. Use `/plan` skill to design before implementing.
3. Use `/review` skill before pushing.
4. Ask — don't invent rules.
