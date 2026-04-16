// Package common_test contains black-box tests for the lifecycle port
// interfaces. These tests mostly prove compile-time conformance —
// runtime behaviour is adapter-specific.
package common_test

import (
	"context"
	"testing"

	"github.com/kitsunium/sdk/ports/common"
)

// Compile-time assertions — if a lifecycle interface drifts, the build
// fails here before any runtime test runs.
var (
	_ common.Opener  = (*memoryLifecycle)(nil)
	_ common.Closer  = (*memoryLifecycle)(nil)
	_ common.Flusher = (*memoryLifecycle)(nil)
)

// TestLifecycleComposite proves a single struct can satisfy Opener,
// Closer, and Flusher simultaneously without method collisions.
//
// Params:
//   - t: the testing harness
func TestLifecycleComposite(t *testing.T) {
	lc := &memoryLifecycle{}
	ctx := context.Background()
	steps := []func(context.Context) error{lc.Open, lc.Flush, lc.Close}
	for _, step := range steps {
		//: each lifecycle call must succeed on the in-test stub
		if err := step(ctx); err != nil {
			t.Fatalf("lifecycle step: %v", err)
		}
	}
	if !lc.opened || !lc.flushed || !lc.closed {
		t.Fatalf("state: %+v", lc)
	}
}

// MemoryLifecycle is a minimal in-test implementation of the lifecycle
// triad used to validate the interfaces compile and compose.
type memoryLifecycle struct {
	// opened records that Open was called.
	opened bool
	// flushed records that Flush was called.
	flushed bool
	// closed records that Close was called.
	closed bool
}

// Open marks the stub opened.
//
// Params:
//   - _: unused context
//
// Returns:
//   - err: always nil
func (m *memoryLifecycle) Open(_ context.Context) (err error) {
	m.opened = true
	return nil
}

// Close marks the stub closed.
//
// Params:
//   - _: unused context
//
// Returns:
//   - err: always nil
func (m *memoryLifecycle) Close(_ context.Context) (err error) {
	m.closed = true
	return nil
}

// Flush marks the stub flushed.
//
// Params:
//   - _: unused context
//
// Returns:
//   - err: always nil
func (m *memoryLifecycle) Flush(_ context.Context) (err error) {
	m.flushed = true
	return nil
}
