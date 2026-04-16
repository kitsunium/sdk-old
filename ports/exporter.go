// Package ports — this file carries the metrics Exporter contract.
// The canonical package doc lives in sink.go.
package ports

import "context"

// Sample is a single metric observation. Components (currently metrics,
// tracing later) emit Samples; adapters that implement Exporter pump
// batches of Samples to their backend (CloudWatch, Prometheus, ...).
type Sample struct {
	Name  string
	Tags  map[string]string
	Value float64
	Unit  string
	Kind  SampleKind
}

// SampleKind distinguishes counter, gauge, and histogram observations.
// Adapters MUST handle each kind; unsupported kinds return an error
// from Export rather than panicking.
type SampleKind uint8

// Supported metric kinds.
const (
	SampleCounter SampleKind = iota
	SampleGauge
	SampleHistogram
)

// Exporter batches Samples to a backend. Separate from Sink because
// metrics are batch-oriented (fanout amortizes overhead over many
// observations), while logs are entry-oriented (per-call latency).
// A single adapter MAY satisfy both interfaces.
type Exporter interface {
	Name() string
	Export(ctx context.Context, samples []Sample) error
}
