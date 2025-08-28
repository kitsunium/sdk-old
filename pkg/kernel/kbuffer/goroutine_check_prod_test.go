//go:build unsafe_no_check
// +build unsafe_no_check

package kbuffer

import (
	"testing"
)

// TestProductionBuildTag tests that debug mode is disabled in production builds.
func TestProductionBuildTag(t *testing.T) {
	// In production build (with unsafe_no_check tag), debug mode should be false
	if debugMode != false {
		t.Error("debugMode should be false in production build (unsafe_no_check tag)")
	}
}

// TestProductionPerformance verifies no overhead in production.
func TestProductionPerformance(t *testing.T) {
	var checker goroutineChecker

	// In production, checkSafety should be a no-op
	for i := 0; i < 1000000; i++ {
		checker.checkSafety()
	}

	// Writes counter should remain at 0 (no checks performed)
	if checker.writes.Load() != 0 {
		t.Errorf("writes counter = %d in production, want 0", checker.writes.Load())
	}
}

// TestProductionInit verifies initialization in production mode.
func TestProductionInit(t *testing.T) {
	// Verify that init() function was called and set debugMode to false
	if debugMode != false {
		t.Error("init() should set debugMode to false in production build")
	}
}
