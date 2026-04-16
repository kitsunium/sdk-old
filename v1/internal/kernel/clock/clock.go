//go:build linux

// Package clock exposes monotonic time primitives. It wraps the
// runtime vDSO-backed time.Now when available and is the only clock
// source the SDK should reach for on hot paths.
//
// The package is stateless: every call returns a fresh value and no
// per-instance setup is required.
//
// Implementation pending — see CLAUDE.md for the spec.
package clock
