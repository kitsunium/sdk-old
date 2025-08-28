//go:build unsafe_no_check
// +build unsafe_no_check

package kbuffer

// init sets debugMode to false for production builds
func init() {
	debugMode = false
}

// checkSafety is a no-op in production builds (unsafe_no_check)
// This completely eliminates any runtime overhead
//
//go:nosplit
//go:inline
func (g *goroutineChecker) checkSafety() {
	// No-op: safety checks are disabled in production
	// The compiler will inline this empty function
}
