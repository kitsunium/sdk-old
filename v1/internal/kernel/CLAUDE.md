<!-- updated: 2026-04-16T00:00:00Z -->
# v1/internal/kernel

Layer 0 equivalent of the SDK. **Syscall-near, stateless, Linux-only**.

## Contract

| | Rule |
|---|---|
| **Imports allowed** | Go stdlib only |
| **Imports forbidden** | everything else in this module (core, components, adapters, ports) |
| **State** | None. No registries, no timers, no goroutines. Pure primitives or syscall wrappers. |
| **Platform** | Linux (`//go:build linux` systematic in new code). Debian-based runtime assumed. |
| **Allocation** | Zero on steady state. Bench-gated. |
| **Enforcement** | `internal/core/contract/arch_external_test.go` (tags=archcheck) |

## Packages

| Package | Status | Purpose |
|---|---|---|
| [`errs`](./errs) | MIGRATED | Typed error catalog with `Define` / `Instance` |
| [`files`](./files) | MIGRATED | File / Directory / Host / Stats syscall wrappers |
| [`atomicx`](./atomicx) | SKELETON | Atomic helpers (stateless) |
| [`clock`](./clock) | SKELETON | `CLOCK_MONOTONIC`, zero-alloc `Now()` |
| [`syscallx`](./syscallx) | SKELETON | Direct syscalls: epoll, futex, io_uring, mmap |

## Why kernel exists

A performance SDK needs a clear bucket where **"zero-alloc, stdlib-only,
bench-required"** is enforced mechanically. Without this bucket, you
can only enforce the rule by convention — and convention drifts.

If a primitive *stores state* (even transiently), it belongs in
`internal/core`, not here. Examples:

| Symbol | Layer | Reason |
|---|---|---|
| `errs.Define(...)` | kernel | Registers once at init, but no per-op state |
| `files.Open(path)` | kernel | Thin syscall wrapper |
| `atomicx.Counter.Inc()` | kernel | Stateless helper over `sync/atomic` |
| `clock.Now()` | kernel | Wraps `time.Now` via `vDSO` path |
| `pool.Buffer(n)` | **core** | Pool stores buffers |
| `cache.LRU[K,V]` | **core** | Entries + TTL |
| `ratelimit.Tokens(rate)` | **core** | Bucket state + refill |

## Conventions

1. Package = domain, function = specialization.
   `files.Open(p)`, not `files.NewFile(p)`.
2. Benchmarks mandatory for hot-path code: `*_bench_test.go`.
3. Arch-specific code uses `//go:build amd64 || arm64`.
4. No `fmt.Errorf` in new code — `errs.Define` + `Instance(...)`.
5. No 3rd-party imports. Ever.

## Validation

```bash
cd v1 && go test -race -bench=. -benchmem ./internal/kernel/...
bazel test //v1/internal/kernel/...
```
