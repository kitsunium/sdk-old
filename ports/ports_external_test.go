// Package ports_test contains black-box tests for the ports contracts.
package ports_test

import (
	"context"
	"testing"
	"time"

	"github.com/kitsunium/sdk/ports"
)

// Compile-time assertions — these fail the build, not a runtime test,
// if memorySink / memoryExporter drift from their port contract.
var (
	_ ports.Sink     = (*memorySink)(nil)
	_ ports.Opener   = (*memorySink)(nil)
	_ ports.Closer   = (*memorySink)(nil)
	_ ports.Flusher  = (*memorySink)(nil)
	_ ports.Exporter = (*memoryExporter)(nil)
)

// TestSinkWrite exercises Name + Write on a trivial in-test sink.
func TestSinkWrite(t *testing.T) {
	s := &memorySink{}
	if s.Name() != "memory" {
		t.Fatalf("Name: got %q", s.Name())
	}
	err := s.Write(context.Background(), ports.Entry{
		Timestamp: time.Unix(0, 0),
		Severity:  ports.SeverityInfo,
		Source:    "t",
		Payload:   "hi",
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if len(s.entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(s.entries))
	}
}

// TestExporterBatch verifies the Exporter port accepts a batch.
func TestExporterBatch(t *testing.T) {
	e := &memoryExporter{}
	err := e.Export(context.Background(), []ports.Sample{
		{Name: "m", Value: 1, Kind: ports.SampleCounter},
		{Name: "m", Value: 2, Kind: ports.SampleGauge},
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(e.batches) != 1 || len(e.batches[0]) != 2 {
		t.Fatalf("unexpected batches: %#v", e.batches)
	}
}

// TestLifecycleComposite proves a single struct can satisfy Opener,
// Closer, and Flusher simultaneously without method collisions.
func TestLifecycleComposite(t *testing.T) {
	s := &memorySink{}
	ctx := context.Background()
	for _, step := range []func(context.Context) error{s.Open, s.Flush, s.Close} {
		if err := step(ctx); err != nil {
			t.Fatalf("lifecycle step: %v", err)
		}
	}
	if !s.opened || !s.flushed || !s.closed {
		t.Fatalf("state: %+v", s)
	}
}

// memorySink is a minimal in-test implementation of the full sink
// lifecycle used to validate the interfaces compile and compose.
type memorySink struct {
	entries                 []ports.Entry
	opened, flushed, closed bool
}

// Name identifies the in-test sink.
func (m *memorySink) Name() string { return "memory" }

// Write appends e to the in-memory log, never blocks.
func (m *memorySink) Write(_ context.Context, e ports.Entry) error {
	m.entries = append(m.entries, e)
	return nil
}

// Open marks the sink opened.
func (m *memorySink) Open(_ context.Context) error {
	m.opened = true
	return nil
}

// Close marks the sink closed.
func (m *memorySink) Close(_ context.Context) error {
	m.closed = true
	return nil
}

// Flush marks the sink flushed.
func (m *memorySink) Flush(_ context.Context) error {
	m.flushed = true
	return nil
}

// memoryExporter is a minimal in-test implementation of Exporter.
type memoryExporter struct {
	batches [][]ports.Sample
}

// Name identifies the in-test exporter.
func (m *memoryExporter) Name() string { return "memory-metrics" }

// Export records the batch for later assertion.
func (m *memoryExporter) Export(_ context.Context, s []ports.Sample) error {
	m.batches = append(m.batches, s)
	return nil
}
