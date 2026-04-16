package errs

import (
	"strings"
	"testing"
)

func TestCallerPackage(t *testing.T) {
	tests := []struct {
		name    string
		skip    int
		wantNot string
	}{
		{name: "valid frame", skip: 0, wantNot: "unknown"},
		{name: "too deep skip", skip: 9999, wantNot: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := callerPackage(tt.skip)
			if tt.name == "too deep skip" {
				if got != "unknown" {
					t.Errorf("deep skip: expected 'unknown', got %q", got)
				}
				return
			}
			if got == "" {
				t.Error("callerPackage should not return empty string")
			}
			if !strings.Contains(got, "errs") {
				t.Logf("callerPackage(0) = %q", got)
			}
		})
	}
}

func TestClearRegistry(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "register then clear", cfg: Config{Package: "cleartest", Code: 1, Message: "m"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(clearRegistry)
			Define(tt.cfg)
			clearRegistry()
			// After clearing, we should be able to register the same code again without panic.
			Define(tt.cfg)
		})
	}
}
