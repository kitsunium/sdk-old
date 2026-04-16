// Package common holds port interfaces shared across every component
// (logger, metrics, future tracer). The lifecycle triad Opener / Closer /
// Flusher composes orthogonally with the component-specific ports such
// as logger.Sink or metrics.Exporter — a single adapter implements
// whichever subset it needs.
package common

import "context"

// Opener is implemented by adapters that require a one-shot
// initialization step (open a socket, handshake with a cloud API,
// pre-allocate a buffer). Repositories call Open exactly once, during
// FromConfig. Failure to open MUST return an error; panics inside Open
// are a contract violation.
//
// Params:
//   - ctx: deadline for the Open operation; implementations MUST honour ctx.Err
//
// Returns:
//   - err: non-nil if the adapter cannot be brought into an operational state
type Opener interface {
	Open(ctx context.Context) (err error)
}

// Closer releases any resource held by an adapter. Repositories call
// Close during their own shutdown path in LIFO order relative to Open.
// Close is idempotent — multiple invocations MUST NOT panic.
//
// Params:
//   - ctx: deadline for the Close operation
//
// Returns:
//   - err: non-nil only when cleanup fails in an actionable way
type Closer interface {
	Close(ctx context.Context) (err error)
}

// Flusher drains any buffered state before a graceful shutdown.
// Repositories invoke Flush before Close so a fast shutdown path can
// guarantee no data loss. Implementations MUST respect ctx and return
// context.DeadlineExceeded when the drain cannot finish in time.
//
// Params:
//   - ctx: deadline for the flush
//
// Returns:
//   - err: non-nil when pending data could not be drained
type Flusher interface {
	Flush(ctx context.Context) (err error)
}
