// Package logger holds the port contracts the logger component declares
// and adapters implement. Types here are logger-domain specific (Entry,
// Severity, Sink); cross-cutting lifecycle interfaces live in
// ports/common and compose orthogonally.
package logger

import (
	"context"
	"time"

	"github.com/kitsunium/sdk/v1/ports/common"
)

// Severity is a monotonic ordinal used to filter Entries per sink.
// Components map their own level vocabulary onto this type so adapters
// never need to know the concrete domain that produced the entry.
type Severity int8

// Severity ranks in ascending urgency. SeverityOff silences the sink
// entirely and is the conventional default for disabled backends.
const (
	// SeverityDebug is the lowest emittable severity.
	SeverityDebug Severity = iota
	// SeverityInfo is the normal operational severity.
	SeverityInfo
	// SeverityWarn flags recoverable anomalies worth reviewing.
	SeverityWarn
	// SeverityError flags unrecoverable failures of a single operation.
	SeverityError
	// SeverityOff silences the sink (filter sentinel, never emitted).
	SeverityOff
)

// Format identifies the serialization a sink expects. Adapters signal
// which formats they support; the logger component picks one at
// configure time.
type Format uint8

// Supported formats. Adapters MAY declare new values in future minor
// releases; callers MUST treat unknown values as an error, not a panic.
const (
	// FormatBinary requests a binary encoding (adapter-specific).
	FormatBinary Format = iota
	// FormatText requests a human-readable line format.
	FormatText
	// FormatJSON requests a single-line JSON object per entry.
	FormatJSON
)

// EntryEvent is a single log record written to a Sink. Adapters
// serialize it according to their configured Format; components produce
// it from application calls.
type EntryEvent struct {
	// Timestamp is the creation instant of the record.
	Timestamp time.Time
	// Severity is the record's urgency level.
	Severity Severity
	// Source identifies the emitting subsystem (logger instance name).
	Source string
	// Payload carries the domain message and structured fields.
	// Adapters MUST NOT mutate Payload — it may be shared by fanout.
	Payload any
}

// Sink is the primary port implemented by every logger adapter. A
// single adapter MAY satisfy additional ports (for instance, an AWS S3
// client backing both a logger.Sink and a metrics.Exporter); the
// interfaces stay intentionally small so that composition remains
// cheap.
type Sink interface {
	// Name returns the sink's stable identifier (e.g. "console",
	// "aws.s3"). Repositories use this to resolve SinkID lookups.
	Name() (name string)

	// Write hands one EntryEvent to the sink. Implementations MUST NOT
	// block indefinitely; fanout runs behind a bounded queue and uses
	// ctx's deadline as the write budget.
	//
	// Params:
	//   - ctx: deadline for the write
	//   - entry: the record to serialize and dispatch
	//
	// Returns:
	//   - err: non-nil when the record could not be written
	Write(ctx context.Context, entry EntryEvent) (err error)
}

// Re-exported lifecycle interfaces for ergonomics — adapters typically
// import one of logger.Opener/Closer/Flusher without a second import.
type (
	// Opener is aliased from common.Opener.
	Opener = common.Opener
	// Closer is aliased from common.Closer.
	Closer = common.Closer
	// Flusher is aliased from common.Flusher.
	Flusher = common.Flusher
)
