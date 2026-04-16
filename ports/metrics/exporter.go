// Package metrics holds the port contracts the metrics component
// declares and adapters implement. Types here are metrics-domain
// specific (Sample, Exporter). Cross-cutting lifecycle interfaces live
// in ports/common and compose orthogonally.
package metrics

import (
	"context"

	"github.com/kitsunium/sdk/ports/common"
)

// SampleKind distinguishes counter, gauge, and histogram observations.
// Adapters MUST handle each kind; unsupported kinds return an error
// from Export rather than panicking.
type SampleKind uint8

// Supported metric kinds.
const (
	// SampleCounter is a monotonically increasing observation.
	SampleCounter SampleKind = iota
	// SampleGauge is a value that moves up and down freely.
	SampleGauge
	// SampleHistogram is a bucketed distribution of observations.
	SampleHistogram
)

// SampleRecord is a single metric observation. Components (metrics
// today, tracing later) emit SampleRecords; adapters implementing
// Exporter pump batches of them to their backend.
type SampleRecord struct {
	// Name is the metric identifier (e.g. "http_requests_total").
	Name string
	// Tags carries label key/value pairs for dimensionality.
	Tags map[string]string
	// Value is the numeric observation.
	Value float64
	// Unit annotates the measurement unit (e.g. "ms", "bytes").
	Unit string
	// Kind discriminates counter vs gauge vs histogram semantics.
	Kind SampleKind
}

// Exporter batches SampleRecords to a backend. Separate from
// logger.Sink because metrics are batch-oriented (fanout amortizes
// overhead over many observations) while logs are entry-oriented
// (per-call latency). A single adapter MAY satisfy both interfaces.
type Exporter interface {
	// Name returns the exporter's stable identifier (e.g.
	// "prometheus", "aws.cloudwatch"). Repositories use this to
	// resolve ExporterID lookups.
	Name() (name string)

	// Export dispatches a batch of samples to the backend.
	// Implementations MUST respect ctx's deadline.
	//
	// Params:
	//   - ctx: deadline for the export operation
	//   - samples: batch of observations to pump to the backend
	//
	// Returns:
	//   - err: non-nil when the batch could not be delivered
	Export(ctx context.Context, samples []SampleRecord) (err error)
}

// Re-exported lifecycle interfaces for ergonomics — adapters typically
// import one of metrics.Opener/Closer/Flusher without a second import.
type (
	// Opener is aliased from common.Opener.
	Opener = common.Opener
	// Closer is aliased from common.Closer.
	Closer = common.Closer
	// Flusher is aliased from common.Flusher.
	Flusher = common.Flusher
)
