<!-- updated: 2026-04-16T00:00:00Z -->
# pkg/kernel

Low-level, stdlib-only wrappers. **No external imports, ever.** These packages live on the hot path of every higher layer — prioritize zero-allocation, lock-free, or cache-friendly designs over ergonomics.

## Dependency contract

- Imports allowed: Go stdlib only.
- Imports forbidden: `pkg/core/*`, `pkg/component/*`, any third-party module.
- CI will reject any PR that adds a non-stdlib import inside `pkg/kernel/`.

## Packages

| Package | Purpose | Key types / entry points |
|---|---|---|
| [`pool`](./pool) | Reusable byte buffers, sync.Pool-backed global | `Buffer`, `Sharded`, `NewSafeBuffer`, `NewUnsafeBuffer`, `NewSafeShardedBuffer`, `NewUnsafeShardedBuffer`, `Get/Put`, `GetBuffer/PutBuffer` |
| [`cache`](./cache) | In-memory caches (generic `K comparable, V any`) | `Cache[K,V]`, `LRU`, `AtomicCache`, `ShardedLRU`, `FastLRU`, `Stats` |
| [`errs`](./errs) | Typed errors with codes, registry, context propagation, metrics | `KError`, `KConfig`, `Instance`, `Result[T]`, `Define`, `GetError`, `SetMetricsCollector` |
| [`files`](./files) | File, directory, archive, stats, system abstractions | `File`, `Directory`, `Archive`, `Stats`, `System`, `Option`, `Path`, `NewFile`, `NewDirectory`, `NewArchive`, `NewStats` |

## Conventions

1. **Naming**: package = domain, function = specialization (see `pkg/CLAUDE.md`).
2. **Legacy `New*` prefix** is still present; do not add new `New*` constructors — prefer the bare form.
3. **Benchmarks mandatory** for any new hot-path code: colocate `*_bench_test.go`.
4. **Race tests**: if the package exposes concurrent types, add a `*_race_test.go` protected by `//go:build !race` **only** when the test intentionally races to validate unsafe behavior (see `pool/unsafe_buffer_race_test.go`).
5. **Build tags**: arch-specific code uses `//go:build amd64 || arm64` (see `pool/unsafe_buffer.go`). Production/debug splits use a custom tag like `unsafe_no_check` (see `pool/goroutine_check_prod.go`).
6. **No `fmt.Errorf`** for error construction in new code — go through `errs.Define` + `KError.Instance(...)`.

## Attention points detected in current code

- `pool` has an `Unsafe*` family that relies on single-owner goroutine semantics; it ships with a compile-time-disablable goroutine checker (`goroutine_check_prod.go` behind `unsafe_no_check`). Don't expose `Unsafe*` buffers across goroutines without the `Sharded` variant.
- `cache.LRU` is mutex-protected; for high-contention paths use `ShardedLRU` or `AtomicCache`.
- `errs` is the single source of error identity — register once in an `init()`, reference by ID or by `(package, code)` pair elsewhere.
- `files.Option` uses a `Path` type with helpers; don't pass raw `string` paths to new APIs.

## Validation

```bash
bazel test //pkg/kernel/...                                    # run everything with race detector
bazel test //pkg/kernel/pool:pool_test --test_arg=-bench=.     # benchmarks (Bazel)
go test -race -bench=. -benchmem ./pkg/kernel/...              # dev alternative
```
