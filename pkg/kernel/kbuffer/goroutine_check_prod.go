//go:build unsafe_no_check
// +build unsafe_no_check

package kbuffer

// checkSafety is a no-op in production builds (unsafe_no_check)
// This completely eliminates any runtime overhead
//
//go:nosplit
func (g *goroutineChecker) checkSafety() {
	// No-op: safety checks are disabled in production
	// The compiler will inline this empty function
}
