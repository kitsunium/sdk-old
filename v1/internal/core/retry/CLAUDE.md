<!-- updated: 2026-04-16T00:00:00Z -->
# v1/internal/core/retry

Exponential backoff + full jitter. Used by every network adapter.

## Planned API

```go
// Policy describes when to retry and how long to wait.
type Policy struct {
    InitialDelay time.Duration
    MaxDelay     time.Duration
    MaxAttempts  int
    Jitter       JitterMode // Full | Equal | None
    Classifier   func(error) bool // return true to retry
}

// Do runs fn under the policy. Returns the last error or nil.
func Do(ctx context.Context, p Policy, fn func(ctx context.Context) error) error
```

## Rules

1. Jitter defaults to Full (0..delay random) — see AWS Architecture
   Blog on "Exponential Backoff And Jitter" for the rationale.
2. `ctx` is honoured between attempts — no sleep loops that ignore
   cancellation.
3. `Classifier` MUST NOT panic; panics are re-thrown to the caller.
4. No allocation on the happy path (single-attempt success).
