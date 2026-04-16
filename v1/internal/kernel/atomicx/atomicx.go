//go:build linux

// Package atomicx provides stateless atomic helpers built on top of
// sync/atomic. It targets Linux amd64/arm64; every primitive exposed
// here is safe for concurrent use and allocation-free.
//
// The package is intentionally minimal. If you need a state-owning
// primitive (counter that also auto-resets, gauge with history), it
// belongs in internal/core.
//
// Implementation pending — see CLAUDE.md for the spec.
package atomicx
