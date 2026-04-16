<!-- updated: 2026-04-16T00:00:00Z -->
# Kitsunium SDK — v1

This directory is the **v1 module** of the SDK. Import path:
`github.com/kitsunium/sdk/v1/...`.

It is independent from `v2/` and higher: each major version freezes its
own `go.mod`, its own dependency set, and its own ports/adapters
contract. Breaking changes only cross the vN boundary.

## 1.0 Target

- **OS**: Linux only, Debian-based preferred (we target a known kernel
  surface — io_uring, epoll, futex, splice/sendfile are fair game).
- **Go version**: 1.26.1, pinned in both `go.mod` and `MODULE.bazel`.
- **Performance discipline**: zero-allocation on steady-state hot paths,
  bench-gated for every kernel primitive, no `reflect` on hot paths.

## 2.0 Layout

```
v1/
├── go.mod                         # module github.com/kitsunium/sdk/v1
├── BUILD.bazel                    # gazelle root
├── CLAUDE.md                      # this file
├── components/                    # PUBLIC — DDD-shaped products
│   ├── logger/
│   └── config/
├── adapters/                      # PUBLIC — driver implementations
│   ├── logger/
│   │   ├── console/
│   │   ├── file/
│   │   ├── syslog/
│   │   └── cloud/
│   │       ├── aws/{cloudwatch,s3}/
│   │       └── grafana/loki/
│   ├── config/                    # (future — e.g. remote config store)
│   └── shared/                    # cross-adapter backends (AWS client, HTTP)
├── ports/                         # PUBLIC — interface contracts
│   ├── common/                    # Opener / Closer / Flusher
│   ├── logger/                    # Sink, EntryEvent, Severity, Format
│   └── config/                    # Accessor, Source, Watcher
└── internal/                      # Go-enforced private
    ├── kernel/                    # syscall-near, stateless, linux-only
    │   ├── errs/                  # typed error catalog
    │   ├── files/                 # file/dir syscall wrappers
    │   ├── atomicx/               # atomic helpers (stateless)
    │   ├── clock/                 # CLOCK_MONOTONIC, zero-alloc Now
    │   └── syscallx/              # epoll, futex, io_uring, mmap
    └── core/                      # stateful, generic, component-reusable
        ├── pool/                  # byte buffers + sharded/sync.Pool
        ├── cache/                 # LRU, Sharded, Atomic
        ├── normalize/             # key/value canonicalization
        ├── ring/                  # lock-free ring buffer
        ├── ratelimit/             # token/leaky bucket (used by logger→batch)
        ├── lifecycle/             # Opener/Closer/Flusher orchestration + TTL
        ├── fanout/                # ring + worker pool for multi-sink
        ├── backpressure/          # flow control
        ├── retry/                 # exponential backoff
        └── contract/              # layer arch test (tags=archcheck)
```

## 3.0 Layer contract (MECHANICALLY ENFORCED)

Enforced by `internal/core/contract/arch_external_test.go` — a `go/ast`
walker that rejects violating imports. Runs in CI via
`go test -tags=archcheck ./internal/core/contract/...`.

| Layer | May import | Forbidden |
|---|---|---|
| `internal/kernel/` | Go stdlib only | everything else in this module |
| `internal/core/` | `internal/kernel/*`, stdlib, vetted 3rd-party | `components/`, `adapters/`, `ports/` |
| `ports/` | stdlib only | everything else |
| `components/` | `ports/*`, `internal/*`, stdlib, 3rd-party | `adapters/` |
| `adapters/` | `ports/*`, `internal/*`, stdlib, 3rd-party | `components/` |

### Kernel vs core (crisp)

- **kernel** = wraps syscalls or provides **stateless** primitives. No
  background goroutines, no timers, no registries. Build tag
  `//go:build linux` systematic.
- **core** = anything that **stores state**, runs goroutines, schedules
  timers, or orchestrates lifecycle. Generic and reusable across
  components.

Examples of the split:
- `errs.Define(...)` → kernel (stateless catalog registration).
- `pool.Buffer(n)` → core (the pool itself stores state).
- `cache.LRU[K,V]` → core (entries + TTL).
- `files.Open(path)` → kernel (syscall wrapper).
- `ratelimit.Tokens(rate)` → core (bucket state + refill timer).

## 4.0 Naming conventions

Package = domain. Function = specialization. `import + call` reads as a
sentence: `pool.Buffer(1024)` = *"pool, give me a buffer of 1024"*.

1. No `New*` prefix on new constructors (`pool.Buffer(n, opts...)`).
2. Variants via functional options, not new struct names
   (`pool.Buffer(n, pool.Safe(), pool.Sharded(16))`).
3. No abbreviations in package names unless resolving a collision.
   `errs` exists because `error` is a Go keyword and `errors` is stdlib.
4. Error definitions go through `errs.Define` — no `fmt.Errorf` in new
   code, no raw `errors.New` at call sites.
5. Tests are suffixed: `_internal_test.go` (white-box),
   `_external_test.go` (black-box), `_bench_test.go`,
   `_integration_test.go`.

## 5.0 Build system

Bazel is the source of truth. Go native tooling is for day-to-day dev.

```bash
# canonical
bazel build //...
bazel test //...

# dev alternative (inside v1/)
cd v1 && go build ./... && go test ./...

# arch enforcement
cd v1 && go test -tags=archcheck ./internal/core/contract/...

# regenerate BUILD.bazel
bazel run //:gazelle
```

## 6.0 Roadmap (this version)

| Phase | Status | Scope |
|---|---|---|
| v1.0 skeleton | DONE | Directories, go.mod, migrated kernel/core/components/ports/, arch test |
| config component | SPEC | `components/config/` — see `components/config/CLAUDE.md` |
| logger component | SPEC | `components/logger/` — see `components/logger/CLAUDE.md` |
| core primitives | SKELETON | `ratelimit`, `lifecycle`, `fanout`, `ring`, `retry`, `backpressure` |
| kernel primitives | SKELETON | `atomicx`, `clock`, `syscallx` |
| adapters/logger/console | TODO | First functional adapter — reference implementation |
| adapters/logger/file | TODO | Local filesystem sink |
| adapters/logger/syslog | TODO | Unix syslog sink |
| adapters/logger/cloud/aws/cloudwatch | TODO | AWS CloudWatch Logs sink |
| adapters/logger/cloud/aws/s3 | TODO | S3 sink (batched) |
| adapters/logger/cloud/grafana/loki | TODO | Loki sink |

v2/ will break the wire contract (ports interfaces) only when we have a
concrete migration motivation documented in an ADR.

## 7.0 Performance targets

- Kernel hot paths: **zero allocation** on steady state, benchmarked.
- Core fanout: bounded queue, no goroutine leak, backpressure-aware.
- Logger single record (no sinks): ≤ 50 ns, ≤ 0 allocs on `pool`-backed
  buffer reuse.
- Logger record → 1 console sink: ≤ 300 ns, ≤ 1 alloc (the encoded
  payload).

Regressions > 5% require explicit PR justification.
