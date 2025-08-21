package normalize

import (
	"testing"
)

func TestKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"UPPER", "upper"},
		{"under_score", "under.score"},
		{"MixedCase", "mixedcase"},
		{"MIXED_CASE_KEY", "mixed.case.key"},
		{"", ""},
		{"a_b_c_d", "a.b.c.d"},
	}

	for _, tt := range tests {
		result := Key(tt.input)
		if result != tt.expected {
			t.Errorf("Key(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"  trimmed  ", "trimmed"},
		{`"quoted"`, "quoted"},
		{`'single'`, "single"},
		{"  \"spaced\"  ", "spaced"},
		{"\n\ttabbed\n\t", "tabbed"},
		{"", ""},
		{"\"", "\""},
		{"   ", ""},                        // All whitespace
		{"\r\n  Windows  \r\n", "Windows"}, // Windows line endings
		{`''`, ""},                         // Empty quotes
		{`""`, ""},                         // Empty double quotes
		{"'mismatched\"", "'mismatched\""}, // Mismatched quotes
	}

	for _, tt := range tests {
		result := Value(tt.input)
		if result != tt.expected {
			t.Errorf("Value(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMap(t *testing.T) {
	input := map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
		},
		"servers": []any{
			map[string]any{"name": "web1"},
			map[string]any{"name": "web2"},
			"plain_string",
			nil,
			123,
		},
		"enabled":      true,
		"empty":        nil,
		"string_value": "  needs trimming  ",
		"nested_array": []any{
			map[string]any{
				"deep": map[string]any{
					"value": "nested",
				},
			},
		},
	}

	result := Map(input)

	expectedKeys := map[string]bool{
		"database.host":             true,
		"database.port":             true,
		"servers.0.name":            true,
		"servers.1.name":            true,
		"servers.2":                 true,
		"servers.3":                 true,
		"servers.4":                 true,
		"enabled":                   true,
		"empty":                     true,
		"string.value":              true,
		"nested.array.0.deep.value": true,
	}

	for key := range expectedKeys {
		if _, exists := result[key]; !exists {
			t.Errorf("Expected key %q not found in result", key)
		}
	}

	if result["database.host"] != "localhost" {
		t.Errorf("database.host = %q, want %q", result["database.host"], "localhost")
	}

	if result["database.port"] != "5432" {
		t.Errorf("database.port = %q, want %q", result["database.port"], "5432")
	}

	if result["servers.2"] != "plain_string" {
		t.Errorf("servers.2 = %q, want %q", result["servers.2"], "plain_string")
	}

	if result["servers.3"] != "" {
		t.Errorf("servers.3 = %q, want empty string", result["servers.3"])
	}

	if result["string.value"] != "needs trimming" {
		t.Errorf("string.value = %q, want %q", result["string.value"], "needs trimming")
	}
}

func TestStringToBytes(t *testing.T) {
	str := "test string"
	bytes := StringToBytes(str)

	if len(bytes) != len(str) {
		t.Errorf("StringToBytes length = %d, want %d", len(bytes), len(str))
	}

	for i := 0; i < len(str); i++ {
		if bytes[i] != str[i] {
			t.Errorf("StringToBytes[%d] = %v, want %v", i, bytes[i], str[i])
		}
	}
}

func TestBytesToString(t *testing.T) {
	bytes := []byte("test bytes")
	str := BytesToString(bytes)

	if str != "test bytes" {
		t.Errorf("BytesToString = %q, want %q", str, "test bytes")
	}
}

func TestProcessSliceWithPrefix(t *testing.T) {
	// Test case to cover the prefix path building in processSlice
	input := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"items": []any{
					"value1",
					"value2",
				},
			},
		},
	}

	result := Map(input)

	// Check that the array items have properly concatenated prefixes
	if result["level1.level2.items.0"] != "value1" {
		t.Errorf("level1.level2.items.0 = %q, want %q", result["level1.level2.items.0"], "value1")
	}
	if result["level1.level2.items.1"] != "value2" {
		t.Errorf("level1.level2.items.1 = %q, want %q", result["level1.level2.items.1"], "value2")
	}
}

func BenchmarkKey(b *testing.B) {
	keys := []string{
		"simple_key",
		"UPPER_CASE_KEY",
		"Mixed_Case_Key",
		"very_long_key_with_many_underscores_and_words",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, key := range keys {
			_ = Key(key)
		}
	}
}

func BenchmarkValue(b *testing.B) {
	values := []string{
		"simple",
		"  trimmed  ",
		`"quoted"`,
		`'single quoted'`,
		"  \"quoted with spaces\"  ",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, val := range values {
			_ = Value(val)
		}
	}
}

func BenchmarkMap(b *testing.B) {
	input := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": map[string]any{
					"value": "deep",
				},
			},
		},
		"array": []any{1, 2, 3, 4, 5},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Map(input)
	}
}
