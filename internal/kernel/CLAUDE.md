<!-- updated: 2026-04-16T00:00:00Z -->
# internal/kernel

Stdlib-only primitives. **Private to the SDK** (under `internal/` — Go enforces external-import rejection at the compiler level).

## Dependency contract

- Imports allowed: Go stdlib only.
- Imports forbidden: `components/`, `adapters/`, `ports/`, `internal/core/`, any third-party module.
- Enforced by `internal/core/contract/arch_test.go` on every CI run — a `go/ast` walk rejects violations.

## Packages

| Package | Purpose |
|---|---|
| [`pool`](./pool) | Reusable byte buffers + global sync.Pool |
| [`cache`](./cache) | Generic in-memory caches (LRU, Sharded, Atomic) |
| [`errs`](./errs) | Typed error catalog with `Define`/`Instance` |
| [`files`](./files) | File/Directory/Archive/Stats abstractions |

## Conventions

1. Package = domain, function = specialization. New code uses the bare form (`pool.Buffer(n, ...)`); legacy `New*` constructors are deprecated — do not add new ones.
2. Benchmarks mandatory for hot-path code: `*_bench_test.go` colocated.
3. Arch-specific code uses `//go:build amd64 || arm64`.
4. No `fmt.Errorf` in new code — `errs.Define` + `Instance(...)`.

## Validation

```bash
bazel test //internal/kernel/...
go test -race -bench=. -benchmem ./internal/kernel/...
```
