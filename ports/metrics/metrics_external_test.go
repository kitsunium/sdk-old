// Package metrics_test contains black-box tests for the metrics port
// contract.
package metrics_test

import (
	"context"
	"testing"

	"github.com/kitsunium/sdk/ports/metrics"
)

// Compile-time assertions — if Exporter or the re-exported lifecycle
// aliases drift, the build fails here.
var (
	_ metrics.Exporter = (*memoryExporter)(nil)
	_ metrics.Opener   = (*memoryExporter)(nil)
	_ metrics.Closer   = (*memoryExporter)(nil)
	_ metrics.Flusher  = (*memoryExporter)(nil)
)

// TestExporterBatch verifies the Exporter port accepts a batch.
//
// Params:
//   - t: the testing harness
func TestExporterBatch(t *testing.T) {
	e := &memoryExporter{}
	err := e.Export(context.Background(), []metrics.SampleEvent{
		{Name: "m", Value: 1, Kind: metrics.SampleCounter},
		{Name: "m", Value: 2, Kind: metrics.SampleGauge},
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(e.batches) != 1 || len(e.batches[0]) != 2 {
		t.Fatalf("unexpected batches: %#v", e.batches)
	}
}

// MemoryExporter is a minimal in-test implementation of Exporter
// covering the full lifecycle.
type memoryExporter struct {
	// batches captures every payload passed to Export.
	batches [][]metrics.SampleEvent
	// opened records that Open was called.
	opened bool
	// flushed records that Flush was called.
	flushed bool
	// closed records that Close was called.
	closed bool
}

// Name identifies the in-test exporter.
//
// Returns:
//   - name: always "memory-metrics"
func (m *memoryExporter) Name() (name string) { return "memory-metrics" }

// Export records the batch for later assertion.
//
// Params:
//   - _: unused context
//   - s: the batch to record
//
// Returns:
//   - err: always nil
func (m *memoryExporter) Export(_ context.Context, s []metrics.SampleEvent) (err error) {
	m.batches = append(m.batches, s)
	return nil
}

// Open marks the exporter opened.
//
// Params:
//   - _: unused context
//
// Returns:
//   - err: always nil
func (m *memoryExporter) Open(_ context.Context) (err error) {
	m.opened = true
	return nil
}

// Close marks the exporter closed.
//
// Params:
//   - _: unused context
//
// Returns:
//   - err: always nil
func (m *memoryExporter) Close(_ context.Context) (err error) {
	m.closed = true
	return nil
}

// Flush marks the exporter flushed.
//
// Params:
//   - _: unused context
//
// Returns:
//   - err: always nil
func (m *memoryExporter) Flush(_ context.Context) (err error) {
	m.flushed = true
	return nil
}
