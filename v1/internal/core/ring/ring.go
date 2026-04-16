// Package ring provides a lock-free ring buffer used as the backbone
// of fan-out dispatching inside the SDK.
//
// The default configuration is multi-producer / single-consumer
// (MPSC), which matches the logger's emitter → worker topology. A
// single-producer / multi-consumer variant is planned for metrics
// fanning out to per-scrape HTTP handlers.
//
// Implementation pending — see CLAUDE.md for the spec.
package ring
