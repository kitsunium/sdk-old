// Package logger_test contains black-box tests for the logger port
// contract. Tests validate compile-time conformance and the minimal
// Write round-trip.
package logger_test

import (
	"context"
	"testing"
	"time"

	"github.com/kitsunium/sdk/ports/logger"
)

// Compile-time assertions — if Sink or the re-exported lifecycle
// aliases drift, the build fails here.
var (
	_ logger.Sink    = (*memorySink)(nil)
	_ logger.Opener  = (*memorySink)(nil)
	_ logger.Closer  = (*memorySink)(nil)
	_ logger.Flusher = (*memorySink)(nil)
)

// TestSinkWrite exercises Name + Write on a trivial in-test sink.
//
// Params:
//   - t: the testing harness
func TestSinkWrite(t *testing.T) {
	s := &memorySink{}
	if s.Name() != "memory" {
		t.Fatalf("Name: got %q", s.Name())
	}
	err := s.Write(context.Background(), logger.EntryEvent{
		Timestamp: time.Unix(0, 0),
		Severity:  logger.SeverityInfo,
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

// MemorySink is a minimal in-test implementation of the full sink
// lifecycle used to validate the interfaces compile and compose.
type memorySink struct {
	// entries captures every EntryEvent passed to Write.
	entries []logger.EntryEvent
	// opened records that Open was called.
	opened bool
	// flushed records that Flush was called.
	flushed bool
	// closed records that Close was called.
	closed bool
}

// Name identifies the in-test sink.
//
// Returns:
//   - name: always "memory"
func (m *memorySink) Name() (name string) { return "memory" }

// Write appends e to the in-memory log, never blocks.
//
// Params:
//   - _: unused context
//   - e: the entry to record
//
// Returns:
//   - err: always nil
func (m *memorySink) Write(_ context.Context, e logger.EntryEvent) (err error) {
	m.entries = append(m.entries, e)
	return nil
}

// Open marks the sink opened.
//
// Params:
//   - _: unused context
//
// Returns:
//   - err: always nil
func (m *memorySink) Open(_ context.Context) (err error) {
	m.opened = true
	return nil
}

// Close marks the sink closed.
//
// Params:
//   - _: unused context
//
// Returns:
//   - err: always nil
func (m *memorySink) Close(_ context.Context) (err error) {
	m.closed = true
	return nil
}

// Flush marks the sink flushed.
//
// Params:
//   - _: unused context
//
// Returns:
//   - err: always nil
func (m *memorySink) Flush(_ context.Context) (err error) {
	m.flushed = true
	return nil
}
