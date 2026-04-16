<!-- updated: 2026-04-16T00:00:00Z -->
# v1/internal/core/lifecycle

Opener / Closer / Flusher orchestration. Every component that
manages a set of lifecycle-bearing resources uses this package.

## Planned API

```go
// Group holds an ordered set of lifecycle-bearing objects. Open is
// called in insertion order; Flush and Close in reverse order.
type Group struct{ /* ... */ }

func NewGroup() *Group
func (g *Group) Register(id string, x any) // x may implement any
                                            // subset of Opener/
                                            // Closer/Flusher
func (g *Group) Open(ctx context.Context) error
func (g *Group) Flush(ctx context.Context) error
func (g *Group) Close(ctx context.Context) error

// TTL schedules a function to run after d. Used by caches and
// ratelimiters that need periodic maintenance.
type TTL struct{ /* ... */ }
func NewTTL(d time.Duration, fn func()) *TTL
func (t *TTL) Stop()
func (t *TTL) Reset(d time.Duration)
```

## Rules

1. Open/Close are idempotent — re-calling MUST NOT panic.
2. Close implies Flush iff the caller didn't call Flush first.
3. Errors from multiple members aggregate via `errors.Join`; the
   Group never short-circuits on the first failure when closing.
4. `TTL` uses `clock.Ticker` internally — no direct `time.Timer`.
