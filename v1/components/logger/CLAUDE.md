<!-- updated: 2026-04-16T00:00:00Z -->
# components/logger — Cahier des charges

Structured, high-performance logger with **pluggable multi-sink
architecture**. Components declare the ports they need; adapters in
`adapters/logger/*` provide the concrete I/O backends (console,
filesystem, syslog, S3, CloudWatch, Loki, …).

## 1.0 Scope & non-scope

**In scope**
- Structured logging API (`Debug/Info/Warn/Error` + typed `Field`).
- Multiple simultaneous sinks per logger instance (fanout).
- Zero-reflect serialization on hot paths; typed field constructors.
- Mockability — `ports/logger.Sink` is a small interface trivially
  faked in unit tests.
- Adaptive rate limiting per sink (`internal/core/ratelimit`) to flip
  between sync-per-record and batched delivery under load.
- Graceful lifecycle (`Opener`/`Closer`/`Flusher`) per sink.

**Out of scope**
- Log querying, indexing, or retention policy — that belongs in the
  observability backends (CloudWatch, Loki).
- Sampling heuristics beyond rate-based batching.
- File rotation — delegated to `adapters/logger/file`.

## 2.0 DDD structure

```
components/logger/
├── domain/                 # entities + value objects (no I/O)
│   ├── level.go            # Level enum (Debug..Off)
│   ├── field.go            # typed Field + constructors
│   ├── record.go           # Record (immutable message + fields)
│   └── severity_map.go     # domain Level ↔ ports.Severity
├── application/            # use cases (orchestrates domain + ports)
│   ├── emitter.go          # Emit(Record) → fanout
│   ├── fanout.go           # ring + workerpool per sink
│   ├── batcher.go          # ratelimit-driven batcher
│   └── registry.go         # Repository — sinks by ID
├── ports/                  # port contracts (re-export ports/logger)
│   └── ports.go
├── logger.go               # facade — `logger.JSON(...)`, `logger.Text(...)`
├── options.go              # functional options
└── CLAUDE.md               # this file
```

## 3.0 Public API (target v1.0.0)

```go
// Facade constructors — read as "logger, give me X".
logger.JSON(opts ...Option) Logger     // JSON-formatted, multi-sink
logger.Text(opts ...Option) Logger     // Human-readable, multi-sink

// Interface — what callers consume.
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(err error, msg string, fields ...Field)
    With(fields ...Field) Logger
    Flush(ctx context.Context) error
    Close(ctx context.Context) error
}

// Field constructors — zero-reflection.
logger.String(k, v string) Field
logger.Int(k string, v int) Field
logger.Int64(k string, v int64) Field
logger.Float(k string, v float64) Field
logger.Bool(k string, v bool) Field
logger.Err(err error) Field              // keyed "error"
logger.NamedErr(k string, err error) Field
logger.Any(k string, v any) Field        // escape hatch — discouraged

// Options — compose sinks + tune behaviour.
logger.WithLevel(Level) Option
logger.WithSink(ports_logger.Sink, ...SinkOption) Option
logger.WithTimeFormat(layout string) Option
logger.WithSource(name string) Option
logger.WithClock(func() time.Time) Option

// Per-sink options — attached via WithSink(sink, logger.SinkRate(...), ...)
logger.SinkRate(perSec int) SinkOption         // ratelimit.Tokens threshold
logger.SinkBatchAbove(perSec int) SinkOption   // switch to batch above N/s
logger.SinkBatchSize(n int) SinkOption         // batch emit size
logger.SinkBufferDepth(n int) SinkOption       // ring buffer depth
logger.SinkTimeout(d time.Duration) SinkOption // per-Write deadline
```

## 4.0 Contracts

### 4.1 Multi-sink fanout

- Each sink runs **behind its own bounded ring buffer** (backed by
  `internal/core/ring`) and its own worker goroutine pool
  (`internal/core/fanout`).
- A slow or broken sink MUST NOT block the emitter. When the ring
  fills, the logger drops records for that sink and increments its
  drop counter (exposed via `Flush` return metadata).
- `Flush(ctx)` MUST drain every sink or return
  `context.DeadlineExceeded` — never partial silent success.
- `Close(ctx)` implies `Flush(ctx)` then `Sink.Close` in reverse-order
  of registration.

### 4.2 Adaptive batching

- Default mode per sink: **synchronous one-record-per-Write** (lowest
  latency, fine for console).
- When the sink's rate crosses `SinkBatchAbove(N)`, the emitter
  switches to **batched mode**: records are buffered until
  `SinkBatchSize(n)` is reached or a `SinkTimeout` elapses, then a
  single batched Write is issued.
- The switch is driven by `internal/core/ratelimit` — the logger never
  polls or sleeps; the ratelimiter triggers mode transitions on
  token-bucket state.
- Rationale: CloudWatch/S3 pricing and HTTP overhead punish per-record
  calls. Console/syslog do not — per-record stays the default.

### 4.3 Error semantics

- `Error(err, msg, fields...)` — **err comes first** intentionally so
  the type system enforces that error logs carry one. Do not reorder.
- `With(fields...) Logger` — returns a child logger with pre-bound
  fields. MUST NOT share mutable state with the parent beyond the
  underlying writer and sink refs.
- Default level: `LevelInfo`. Default output: `os.Stderr` iff no
  explicit sink is registered. Registering any sink via `WithSink`
  **removes** the implicit stderr default.

### 4.4 Mockability

- `ports/logger.Sink` is the entire contract an adapter exposes.
  Tests register a `testSink` struct with a captured `Write` and
  assert on the resulting slice of `EntryEvent`.
- No global logger state. No package-level `init`. No singletons.

## 5.0 Dependency rules (in this subtree)

Forbidden:
- `components/logger` imports **no** adapter package.
- `components/logger` imports **no** `ports/logger` private helpers
  (only the public interface surface).

Allowed:
- `ports/common`, `ports/logger` (interfaces).
- `internal/kernel/errs`, `internal/kernel/clock`.
- `internal/core/pool` (record buffer), `internal/core/ring`,
  `internal/core/fanout`, `internal/core/ratelimit`,
  `internal/core/lifecycle`.

## 6.0 Performance targets

| Operation | Target | Measurement |
|---|---|---|
| Single `Info` call (Sink=nil) | ≤ 50 ns | `go test -bench -benchmem` |
| Single `Info` call (1 console sink, sync) | ≤ 300 ns, ≤ 1 alloc | same |
| `Info` call (1 sink, batched mode) | ≤ 80 ns, 0 allocs | same |
| `With(3 fields)` | ≤ 80 ns, 1 alloc | same |
| Fanout 1 record → 3 sinks | ≤ 500 ns, 0 allocs (ring) | same |

Benchmarks: `components/logger/*_bench_test.go`. CI compares against
baseline in `scripts/bench_manager.py` with 5% regression budget.

## 7.0 Test plan

1. **Unit**: every field constructor, every level, every option.
2. **Mock**: fake sink asserts record order, field presence, timing.
3. **Fanout**: 3 sinks, slow sink must not block fast sinks.
4. **Backpressure**: ring full → drops counted, never blocked.
5. **Mode transition**: emit 1k/s → ratelimiter flips to batched.
6. **Race**: `-race` on every test binary.
7. **Bench**: per-operation regression tracking.

## 8.0 Migration from legacy logger

The pre-v1 logger (kept in git history) exposes `JSON(opts...)` and
`Text(opts...)` with a single `io.Writer` output. The v1 API keeps
those constructors but **`WithOutput(io.Writer)` is replaced by
`WithSink(ports_logger.Sink, ...)`**. Consumers migrate by wrapping
their writer in `adapters/logger/console` (or their own `Sink`).

Old code:
```go
lg := logger.JSON(logger.WithOutput(os.Stderr), logger.WithLevel(logger.LevelDebug))
```

New code:
```go
lg := logger.JSON(
    logger.WithSink(console.Sink(os.Stderr)),
    logger.WithLevel(logger.LevelDebug),
)
```

## 9.0 Validation

```bash
cd v1 && go test -race ./components/logger/...
cd v1 && go test -bench=. -benchmem ./components/logger/...
bazel test //v1/components/logger/...
```
