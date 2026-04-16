// Package ports defines the interfaces (contracts) that components consume
// and adapters implement. It is the inversion-of-control seam of the SDK:
// components and adapters both depend on ports, never on each other.
//
// Ports MUST remain stdlib-only — no internal/ dependencies, no third-party
// imports. The architecture contract
// (internal/core/contract/arch_external_test.go) rejects any violation.
package ports

import (
	"context"
	"time"
)

// Severity is a monotonic ordinal used to filter Entries per sink.
// Components map their own level vocabulary onto this type so adapters
// never need to know the concrete domain (log, metrics, tracing) that
// produced the entry.
type Severity int8

// Severity ranks in ascending urgency. OFF silences the sink entirely
// and is the conventional default for disabled backends.
const (
	SeverityDebug Severity = iota
	SeverityInfo
	SeverityWarn
	SeverityError
	SeverityOff
)

// Format identifies the serialization a sink expects. Adapters signal
// which formats they support; components pick one at configure time.
type Format uint8

// Supported formats. Adapters MAY declare new values in future minor
// releases; callers MUST treat unknown values as an error, not a panic.
const (
	FormatBinary Format = iota
	FormatText
	FormatJSON
)

// Entry is a single record written to a sink. Domain-specific payloads
// (log messages, metric samples, spans) are carried in Payload as a
// typed value understood by both the component and the adapter.
type Entry struct {
	Timestamp time.Time
	Severity  Severity
	Source    string
	Payload   any
}

// Sink is the primary port implemented by every adapter. A single
// adapter MAY satisfy additional ports (for instance, an AWS S3 client
// backing both a log Sink and a metrics Exporter); the interfaces stay
// intentionally small so that composition remains cheap.
type Sink interface {
	// Name returns the sink's stable identifier (e.g. "console",
	// "aws.s3"). Repositories use this to resolve SinkID lookups.
	Name() string

	// Write hands one Entry to the sink. Implementations MUST NOT
	// block indefinitely; fanout runs behind a bounded queue and
	// uses the context deadline as the write budget.
	Write(ctx context.Context, entry Entry) error
}

// Opener is implemented by sinks that require a one-shot initialization
// step (open a socket, handshake with a cloud API, pre-allocate a
// buffer). Repositories call Open exactly once, during FromConfig.
type Opener interface {
	Open(ctx context.Context) error
}

// Closer releases any resource held by the sink. Repositories call Close
// during their own Close(ctx) in LIFO order relative to Open.
type Closer interface {
	Close(ctx context.Context) error
}

// Flusher drains any pending entry. Repositories invoke Flush before
// Close so a fast shutdown path can guarantee no data loss.
type Flusher interface {
	Flush(ctx context.Context) error
}
