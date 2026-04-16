// Package lifecycle provides orchestration helpers for objects that
// implement common.Opener / common.Closer / common.Flusher.
//
// Components register ordered sets of lifecycle-bearing things
// (e.g. sinks, sources) and let this package handle the
// open-in-order, flush-in-reverse, close-in-reverse dance plus
// context-bound deadlines.
//
// Implementation pending — see CLAUDE.md for the spec.
package lifecycle
