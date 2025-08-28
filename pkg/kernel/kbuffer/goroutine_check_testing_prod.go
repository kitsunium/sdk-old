//go:build unsafe_no_check
// +build unsafe_no_check

package kbuffer

// testingSkipSafetyCheck is not used in production builds but needs to exist for tests to compile
var testingSkipSafetyCheck bool
