<!-- updated: 2026-04-16T00:00:00Z -->
# v1/internal/core

Stateful composition on top of `internal/kernel`. **Generic, reusable
across components.**

## Contract

| | Rule |
|---|---|
| **Imports allowed** | `internal/kernel/*`, Go stdlib, vetted 3rd-party |
| **Imports forbidden** | `components/`, `adapters/`, `ports/` |
| **State** | Allowed and encouraged (pool of buffers, LRU entries, token buckets, timers, worker pools) |
| **Reusability** | A core primitive must be usable by at least two components (or be plausibly so) — otherwise it belongs inside the component |
| **Allocation** | Zero on steady state when feasible. Bench-tracked. |
| **Enforcement** | `contract/arch_external_test.go` (tags=archcheck) |

## Packages

| Package | Status | Purpose |
|---|---|---|
| [`pool`](./pool) | MIGRATED | Byte buffers (Safe / Unsafe / Sharded) + global sync.Pool |
| [`cache`](./cache) | MIGRATED | LRU, Sharded, Atomic caches |
| [`normalize`](./normalize) | MIGRATED | Key/value canonicalization, zero-alloc string↔bytes |
| [`contract`](./contract) | MIGRATED | Layer boundary enforcement (archcheck tag) |
| [`ring`](./ring) | SKELETON | Lock-free MPSC/SPMC ring buffer |
| [`ratelimit`](./ratelimit) | SKELETON | Token bucket, leaky bucket |
| [`lifecycle`](./lifecycle) | SKELETON | Opener/Closer/Flusher orchestration + TTL |
| [`fanout`](./fanout) | SKELETON | Ring + worker pool for N-way fanout |
| [`backpressure`](./backpressure) | SKELETON | Flow control (credit, window) |
| [`retry`](./retry) | SKELETON | Exponential backoff + jitter |

## Motivation per primitive

- **`ring`** — the backbone of `fanout`. Each sink owns a ring; producers
  never block.
- **`ratelimit`** — first consumer is the logger (switch between
  sync-per-record and batched when rate crosses N/s). Reused by any
  client doing rate-aware I/O.
- **`lifecycle`** — every component needs Opener/Closer/Flusher
  orchestration. Lives in core so logger, config, and future
  components share it.
- **`fanout`** — used by the logger to push one record to N sinks.
  Reusable by metrics, tracer, any future multi-destination emitter.
- **`backpressure`** — credit/window mechanics for cloud adapters
  (CloudWatch Logs accepts batch N events; we must never over-commit).
- **`retry`** — standard exponential backoff. Used by every network
  adapter.

## Conventions

1. State is explicit — every core type exposes its mutable state via
   typed methods; no hidden globals.
2. Constructors use the sentence form: `ratelimit.Tokens(rate, opts...)`.
3. All state-holding types implement `Close(ctx) error` when they
   allocate goroutines or timers.
4. No component-specific logic — "this is useful for the logger" is
   a warning sign. Extract to `components/<x>` if truly specific.

## Validation

```bash
cd v1 && go test -race -bench=. -benchmem ./internal/core/...
cd v1 && go test -tags=archcheck ./internal/core/contract/...
bazel test //v1/internal/core/...
```
