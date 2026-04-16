// Package fanout dispatches one input event to N consumers, each
// protected by its own bounded ring buffer. It is the primitive the
// logger uses to push a single Record to several sinks without a
// slow sink blocking the fast ones.
//
// Each consumer owns a ring (from internal/core/ring) and a worker
// goroutine that drains it. The producer side is non-blocking: a
// full ring returns a drop signal to the emitter.
//
// Implementation pending — see CLAUDE.md for the spec.
package fanout
