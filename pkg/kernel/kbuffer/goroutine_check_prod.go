//go:build unsafe_no_check
// +build unsafe_no_check

// Package kbuffer provides ultra-optimized, lock-free byte buffers for kernel operations.
// This file contains the production build version of goroutine checking (disabled).
// When the unsafe_no_check build tag is used, all safety checks become no-ops
// for maximum performance in production environments.
package kbuffer

// goroutineChecker is a no-op in production builds.
// This empty struct provides the same interface as the development version
// but with zero memory overhead and all methods optimized away.
type goroutineChecker struct{}

// checkSafety is a no-op in production builds (unsafe_no_check).
// This completely eliminates any runtime overhead for maximum performance.
// The compiler will inline and optimize away this empty function completely.
//
//go:nosplit
func (g *goroutineChecker) checkSafety() {
	// No-op: safety checks are disabled in production
	// The compiler will inline this empty function
}

// testingSkipSafetyCheck exists for compilation compatibility but is unused in production.
// This variable maintains API compatibility with the development build but has no effect.
var testingSkipSafetyCheck bool //nolint:unused
