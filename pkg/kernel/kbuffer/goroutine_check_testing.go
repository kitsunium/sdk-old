//go:build !unsafe_no_check
// +build !unsafe_no_check

package kbuffer

// testingSkipSafetyCheck is used only in tests to temporarily disable safety checks
// This is only available in development builds (not in production with unsafe_no_check)
var testingSkipSafetyCheck bool
