//go:build !unsafe_no_check
// +build !unsafe_no_check

package kbuffer

// checkSafety performs goroutine safety checks in development builds
// This is only compiled when unsafe_no_check tag is NOT present
func (g *goroutineChecker) checkSafety() {
	// Allow tests to skip safety checks
	if !testingSkipSafetyCheck {
		g.checkGoroutineSafety()
	}
}
