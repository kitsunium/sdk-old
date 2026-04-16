<!-- updated: 2026-04-16T00:00:00Z -->
# internal/kernel/pool

Reusable byte buffers and global `sync.Pool`. Hottest path in the SDK — changes here must be benchmarked.

## Files

| File | Role |
|---|---|
| `interface.go` | `Buffer`, `Pool`, `Sharded`, `Option`; constructors `NewSafeBuffer`, `NewUnsafeBuffer`, `NewSafeShardedBuffer`, `GetGlobalPool`; sizing & sentinel-error constants |
| `global.go` | Package-level shared pool: `Get`, `Put`, `GetBuffer`, `PutBuffer`, `SetGlobalClearOnPut`, `SetGlobalMaxSize` |
| `safe_buffer.go` | Mutex-guarded `safeBuffer` (spinlock with exponential backoff) |
| `safe_sharded.go` | Sharded variant of `safeBuffer` for high-contention writers |
| `unsafe_buffer.go` | Lock-free `unsafeBuffer` — **amd64/arm64 only** (`//go:build amd64 \|\| arm64`) |
| `pool_bench_test.go` | Benchmarks — regression > 5% blocks PR |
| `unsafe_buffer_race_test.go` | `//go:build !race` — intentionally races on unsafe buffer to verify failure mode |
| `legacy_test_helpers_test.go` | No-op shims for `testingSkipSafetyCheck` + `getCurrentGID` kept for legacy-test compatibility; no runtime effect |

## Buffer API surface

`Buffer` methods:
- Writes: `Write`, `WriteString`, `WriteByte`, `WriteAt`, `TryWrite`, `AppendBytes`
- Reads: `Bytes`, `String`, `BytesUnsafe`, `RemainingSlice`, `Clone`
- Sizing: `Len`, `Cap`, `Available`, `Grow`, `Extend`, `Truncate`
- Lifecycle: `Reset`, `Clear`

`Sharded` adds: `WriteToShard`, `ShardCount`, `Balance`.

## Buffer choice (callers must pick deliberately)

| Constructor | Thread-safe | Typical perf | Use when |
|---|---|---|---|
| `NewUnsafeBuffer(n)` | ❌ no | ~2-3 ns/op | single goroutine owns the buffer for its lifetime |
| `NewSafeBuffer(n)` | ✅ spinlock | ~15-25 ns/op | 2-10 concurrent writers |
| `NewSafeShardedBuffer(n, s)` | ✅ per-shard spinlock | ~70-85 ns/op at 100 goroutines | high-contention write paths |

Spinlock over `sync.Mutex`: shorter critical sections + exponential backoff + cache locality beat mutex by ~2-3× at low contention. Handing an `unsafeBuffer` across goroutines is undefined behaviour — there is no runtime check; pick `NewSafeBuffer` instead.

## Rules

1. **Never** hand an `unsafeBuffer` to another goroutine — it has no synchronization. For shared access use `NewSafeBuffer`; for high-contention writers use `NewSafeShardedBuffer`.
2. `SetGlobalMaxSize` caps pooled buffer capacity to avoid retaining oversized allocations; don't raise without a benchmark.
3. `SetGlobalClearOnPut(true)` zeroes memory on return — required when the buffer may carry secrets.
4. New methods on `Buffer` must be added to **both** `safeBuffer` and `unsafeBuffer` (+ sharded variant) in the same PR with matching semantics.
5. Any change under `//go:build amd64 || arm64` must compile on both (`GOARCH=amd64 go build ./...`, `GOARCH=arm64 go build ./...`).
6. Target naming migration (roadmap §16): `NewSafeBuffer(n)` → `pool.Buffer(n, pool.Safe())`, `NewSafeShardedBuffer(n, s)` → `pool.Buffer(n, pool.Safe(), pool.Sharded(s))`. Do not add new `New*` symbols.

## Deliberately absent

Removed during the 2026-04 rebuild (see `/workspace/.claude/contexts/kernel-pool-audit.md`):
- `goroutine_check.go` / `goroutine_check_prod.go` — dev-mode goroutine owner assertion used a full `runtime.Stack` capture per write. Catastrophic on hot path; collision-prone hash stood in for a real GID; the `unsafe_no_check` build tag that gated it is gone.
- `unsafe_sharded.go` + `NewUnsafeShardedBuffer` — logically incoherent (non-thread-safe sharding) AND contained a bug where `selectShard()` always returned shard 0.
- `prewarm()` in global — allocations returned to `sync.Pool` at package init get evicted at the first GC.
- Dead constants from `interface.go`: `likelyTrue/False`, `prefetchRead/Write`, `alignment16/32`, `ptrSize/wordSize`, `optimalIOSize`, `stateFlagLocked/ReadOnly`, shadowing `min/max` helpers.
- `Factory`, `Writer`, `Reader` interfaces — zero implementations, zero consumers outside tests. `Pool` is kept (return type of `GetGlobalPool`).
- Constants `likelyTrue`/`likelyFalse`/`prefetchRead`/`prefetchWrite` (Go has no compiler hints for these), `ptrSize`/`wordSize` (unused), `alignment16`/`alignment32` (no SIMD path in Go code), `optimalIOSize` (unreferenced), `spinLimit`/`backoffInitial`/`backoffMax` (not used by the inline spinLock backoff in `safe_buffer.go`), `stateFlagLocked`/`stateFlagReadOnly` (declared, never set/read), `errNilBuffer`/`errConcurrentModification` (never returned).

## Validation

```bash
bazel test //internal/kernel/pool/...
go test -race -bench=. -benchmem ./internal/kernel/pool
```
