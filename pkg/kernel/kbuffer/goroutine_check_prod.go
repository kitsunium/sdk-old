//go:build unsafe_no_check
// +build unsafe_no_check

package kbuffer

// debugMode is disabled in production builds for zero overhead
func init() {
	debugMode = false
}
