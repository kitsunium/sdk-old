package normalize

import (
	"testing"
)

// TestTransformKey_Coverage tests the uncovered branch in transformKey
func TestTransformKey_Coverage(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "empty key",
			key:  "",
			want: "",
		},
		{
			name: "key shorter than startIndex",
			key:  "ab",
			want: "ab",
		},
		{
			name: "key exactly at startIndex",
			key:  "abc",
			want: "abc",
		},
		{
			name: "key with uppercase after startIndex",
			key:  "abcDEF",
			want: "abcdef",
		},
		{
			name: "key with underscore after startIndex",
			key:  "abc_def",
			want: "abc.def",
		},
		{
			name: "single character",
			key:  "x",
			want: "x",
		},
		{
			name: "two characters",
			key:  "xy",
			want: "xy",
		},
		{
			name: "uppercase with underscore",
			key:  "ABCDEF_GHI",
			want: "abcdef.ghi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transformKey(tt.key)
			if got != tt.want {
				t.Errorf("transformKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// TestTransformKey_BoundaryConditions tests edge cases of transformKey
func TestTransformKey_BoundaryConditions(t *testing.T) {
	// Test with current startIndex value (which is 0)
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "transform entire key",
			key:  "ABC_DEF",
			want: "abc.def",
		},
		{
			name: "empty key",
			key:  "",
			want: "",
		},
		{
			name: "single char uppercase",
			key:  "A",
			want: "a",
		},
		{
			name: "mixed case with underscores",
			key:  "A_B_C",
			want: "a.b.c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transformKey(tt.key)
			if got != tt.want {
				t.Errorf("transformKey(%q) = %q, want %q",
					tt.key, got, tt.want)
			}
		})
	}
}
