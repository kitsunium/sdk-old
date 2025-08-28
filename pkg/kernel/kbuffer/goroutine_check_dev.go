//go:build !unsafe_no_check
// +build !unsafe_no_check

package kbuffer

// checkSafety performs goroutine safety checks in development builds
// This is only compiled when unsafe_no_check tag is NOT present
//
//go:inline
func (g *goroutineChecker) checkSafety() {
	if debugMode {
		g.checkGoroutineSafety()
	}
}