package json

import (
	"github.com/kitsunium/sdk/v1/components/config/parser"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSON_Type(t *testing.T) {
	j := NewJSON("test.json")
	if j.Type() != "json" {
		t.Errorf("Type() = %q, want %q", j.Type(), "json")
	}
}

func TestJSON_NewJSON(t *testing.T) {
	// Test without options
	j1 := NewJSON("test.json")
	if j1.Path != "test.json" {
		t.Errorf("Path = %q, want %q", j1.Path, "test.json")
	}
	if j1.options.BufferSize != 8192 {
		t.Errorf("bufferSize = %d, want %d", j1.options.BufferSize, 8192)
	}
	if j1.options.UsePool != false {
		t.Errorf("usePool = %v, want %v", j1.options.UsePool, false)
	}

	// Test with options
	j2 := NewJSON("test.json", parser.WithBufferSize(4096), parser.WithPool(true))
	if j2.options.BufferSize != 4096 {
		t.Errorf("bufferSize = %d, want %d", j2.options.BufferSize, 4096)
	}
	if j2.options.UsePool != true {
		t.Errorf("usePool = %v, want %v", j2.options.UsePool, true)
	}
}

func TestJSON_Load_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "test.json")

	content := `{
		"database": {
			"host": "localhost",
			"port": 5432,
			"name": "testdb"
		},
		"server": {
			"host": "0.0.0.0",
			"port": 8080
		}
	}`

	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	j := NewJSON(jsonPath)
	result, err := j.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	expected := map[string]string{
		"database.host": "localhost",
		"database.port": "5432",
		"database.name": "testdb",
		"server.host":   "0.0.0.0",
		"server.port":   "8080",
	}

	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q, want %q", key, value, expectedValue)
		}
	}
}

func TestJSON_Load_InvalidExtension(t *testing.T) {
	j := NewJSON("test.txt")
	_, err := j.Load()
	if err == nil {
		t.Error("Load() should error on invalid extension")
	}
	if !errors.Is(err, parser.ErrInvalidExtension) {
		t.Errorf("Expected parser.ErrInvalidExtension, got: %v", err)
	}
}

func TestJSON_Load_NonExistentFile(t *testing.T) {
	j := NewJSON("/non/existent/file.json")
	_, err := j.Load()
	if err == nil {
		t.Error("Load() should error on non-existent file")
	}
	if !errors.Is(err, parser.ErrFileNotFound) {
		t.Errorf("Expected parser.ErrFileNotFound, got: %v", err)
	}
}

func TestJSON_LoadReader(t *testing.T) {
	content := `{
		"key1": "value1",
		"key2": "value2",
		"nested": {
			"key3": "value3"
		}
	}`

	j := NewJSON("")
	reader := strings.NewReader(content)
	result, err := j.LoadReader(reader)
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}

	expected := map[string]string{
		"key1":        "value1",
		"key2":        "value2",
		"nested.key3": "value3",
	}

	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q, want %q", key, value, expectedValue)
		}
	}
}

func TestJSON_LoadReader_Error(t *testing.T) {
	j := NewJSON("")
	reader := &jsonErrorReader{err: io.ErrUnexpectedEOF}

	_, err := j.LoadReader(reader)
	if err == nil {
		t.Error("LoadReader() should return error from reader")
	}
	if !errors.Is(err, parser.ErrReadFailed) {
		t.Errorf("Expected parser.ErrReadFailed, got: %v", err)
	}
}

type jsonErrorReader struct {
	err error
}

func (r *jsonErrorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func TestJSON_LoadBytes_AllTypes(t *testing.T) {
	content := []byte(`{
		"string": "value",
		"int": 42,
		"float": 3.14,
		"bool_true": true,
		"bool_false": false,
		"null": null,
		"array": ["item1", "item2", 3, true, null],
		"nested": {
			"deep": {
				"value": "nested_value"
			}
		},
		"array_of_objects": [
			{"id": 1, "name": "first"},
			{"id": 2, "name": "second"}
		]
	}`)

	// Also test with a custom decoder that might produce int64 or json.Number
	// This simulates edge cases in JSON parsing

	j := NewJSON("")
	result, err := j.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	expected := map[string]string{
		"string":                  "value",
		"int":                     "42",
		"float":                   "3.14",
		"bool.true":               "true",
		"bool.false":              "false",
		"null":                    "",
		"array.0":                 "item1",
		"array.1":                 "item2",
		"array.2":                 "3",
		"array.3":                 "true",
		"array.4":                 "",
		"nested.deep.value":       "nested_value",
		"array.of.objects.0.id":   "1",
		"array.of.objects.0.name": "first",
		"array.of.objects.1.id":   "2",
		"array.of.objects.1.name": "second",
	}

	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func TestJSON_LoadBytes_InvalidJSON(t *testing.T) {
	testCases := []struct {
		name    string
		content []byte
	}{
		{"empty", []byte("")},
		{"invalid syntax", []byte("{invalid json}")},
		{"incomplete", []byte(`{"key": `)},
		{"trailing comma", []byte(`{"key": "value",}`)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			j := NewJSON("")
			_, err := j.LoadBytes(tc.content)
			if err == nil {
				t.Error("LoadBytes() should error on invalid JSON")
			}
			if !errors.Is(err, ErrJSONParse) {
				t.Errorf("Expected error about parsing JSON, got: %v", err)
			}
		})
	}
}

func TestJSON_LoadBytes_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		content  []byte
		expected map[string]string
	}{
		{
			name:     "empty object",
			content:  []byte(`{}`),
			expected: map[string]string{},
		},
		{
			name:     "empty array",
			content:  []byte(`{"array": []}`),
			expected: map[string]string{},
		},
		{
			name:     "nested empty objects",
			content:  []byte(`{"a": {}, "b": {"c": {}}}`),
			expected: map[string]string{},
		},
		{
			name:    "unicode strings",
			content: []byte(`{"emoji": "😀", "chinese": "你好", "arabic": "مرحبا"}`),
			expected: map[string]string{
				"emoji":   "😀",
				"chinese": "你好",
				"arabic":  "مرحبا",
			},
		},
		{
			name:    "special characters",
			content: []byte(`{"quotes": "\"quoted\"", "newline": "line1\nline2", "tab": "tab\there"}`),
			expected: map[string]string{
				"quotes":  "quoted", // normalize.Value strips quotes
				"newline": "line1\nline2",
				"tab":     "tab\there",
			},
		},
		{
			name:    "large integers",
			content: []byte(`{"big": 9223372036854775807, "negative": -9223372036854775808}`),
			expected: map[string]string{
				"big":      "9223372036854775807",
				"negative": "-9223372036854775808",
			},
		},
		{
			name:    "scientific notation",
			content: []byte(`{"sci1": 1.23e10, "sci2": 1.23e-10}`),
			expected: map[string]string{
				"sci1": "1.23e10",  // Preserve original scientific notation
				"sci2": "1.23e-10", // Preserve original scientific notation
			},
		},
		{
			name:    "mixed array types",
			content: []byte(`{"mixed": [1, "two", true, null, {"nested": "value"}]}`),
			expected: map[string]string{
				"mixed.0":        "1",
				"mixed.1":        "two",
				"mixed.2":        "true",
				"mixed.3":        "",
				"mixed.4.nested": "value",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			j := NewJSON("")
			result, err := j.LoadBytes(tc.content)
			if err != nil {
				t.Fatalf("LoadBytes() error = %v", err)
			}

			if len(result) != len(tc.expected) {
				t.Errorf("Result has %d items, want %d", len(result), len(tc.expected))
			}

			for key, expectedValue := range tc.expected {
				if value, ok := result[key]; !ok || value != expectedValue {
					t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
				}
			}
		})
	}
}

func TestJSON_LoadBytes_SmallSize(t *testing.T) {
	// Test with very small JSON to trigger the minimum size allocation
	content := []byte(`{"a": 1}`)

	j := NewJSON("")
	result, err := j.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	if result["a"] != "1" {
		t.Errorf("a = %q, want %q", result["a"], "1")
	}
}

func TestJSON_NormalizeAnyValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "test", "test"},
		{"float64_int", float64(42), "42"},
		{"float64_decimal", 3.14, "3.14"},
		{"bool_true", true, "true"},
		{"bool_false", false, "false"},
		{"nil", nil, ""},
		{"int64", int64(123), "123"},
		{"json.Number", json.Number("456"), "456"},
		{"other_type", struct{}{}, "{}"},
		{"array", []int{1, 2, 3}, "[1 2 3]"},
		{"map", map[string]int{"a": 1}, "map[a:1]"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeAnyValue(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeAnyValue(%v) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestJSON_LoadBytes_DefaultCase(t *testing.T) {
	// Test to ensure the default case works (though it shouldn't be hit in normal operation)
	// This test ensures our defensive programming is tested
	j := NewJSON("")

	// Test with complex nested arrays that might contain unexpected types
	content := []byte(`{
		"arrays": [
			[1, 2, 3],
			["a", "b", "c"],
			[true, false, null],
			[{"nested": "object"}, {"another": "one"}]
		]
	}`)

	result, err := j.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	// Verify we handle nested arrays correctly
	// Note: nested arrays become strings when encountered in normalizeAnyValue
	if result["arrays.0"] != "[1 2 3]" {
		t.Errorf("arrays.0 = %q, want %q", result["arrays.0"], "[1 2 3]")
	}
	if result["arrays.1"] != "[a b c]" {
		t.Errorf("arrays.1 = %q, want %q", result["arrays.1"], "[a b c]")
	}
	if result["arrays.2"] != "[true false <nil>]" {
		t.Errorf("arrays.2 = %q, want %q", result["arrays.2"], "[true false <nil>]")
	}
}

func TestJSON_ComplexNesting(t *testing.T) {
	// Create deeply nested JSON
	type DeepStruct struct {
		Level1 struct {
			Level2 struct {
				Level3 struct {
					Level4 struct {
						Value string `json:"value"`
					} `json:"level4"`
				} `json:"level3"`
			} `json:"level2"`
		} `json:"level1"`
	}

	data := DeepStruct{}
	data.Level1.Level2.Level3.Level4.Value = "deep"

	jsonBytes, _ := json.Marshal(data)

	j := NewJSON("")
	result, err := j.LoadBytes(jsonBytes)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	expectedKey := "level1.level2.level3.level4.value"
	if result[expectedKey] != "deep" {
		t.Errorf("%s = %q, want %q", expectedKey, result[expectedKey], "deep")
	}
}

func TestJSON_LargeDocument(t *testing.T) {
	// Test with a large document to ensure stack allocation works
	data := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		section := make(map[string]interface{})
		for j := 0; j < 10; j++ {
			section[fmt.Sprintf("key_%d", j)] = fmt.Sprintf("value_%d_%d", i, j)
		}
		data[fmt.Sprintf("section_%d", i)] = section
	}

	jsonBytes, _ := json.Marshal(data)

	j := NewJSON("")
	result, err := j.LoadBytes(jsonBytes)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	// Check a few samples
	if result["section.0.key.0"] != "value_0_0" {
		t.Errorf("section.0.key.0 = %q, want %q", result["section.0.key.0"], "value_0_0")
	}
	if result["section.99.key.9"] != "value_99_9" {
		t.Errorf("section.99.key.9 = %q, want %q", result["section.99.key.9"], "value_99_9")
	}
}

func TestJSON_LoadBytes_ExhaustiveTypes(t *testing.T) {
	// This test ensures EVERY possible JSON type is tested
	content := []byte(`{
		"string": "hello",
		"number_int": 42,
		"number_float": 3.14159,
		"number_negative": -123,
		"number_zero": 0,
		"number_scientific": 1.23e10,
		"bool_true": true,
		"bool_false": false,
		"null_value": null,
		"empty_string": "",
		"unicode_string": "Hello 世界 🌍 مرحبا",
		"escaped_string": "Line1\nLine2\tTab\"Quote\"",
		
		"object_empty": {},
		"object_simple": {
			"key": "value"
		},
		"object_nested": {
			"level1": {
				"level2": {
					"level3": {
						"deep": "value"
					}
				}
			}
		},
		
		"array_empty": [],
		"array_strings": ["a", "b", "c"],
		"array_numbers": [1, 2.5, -3, 0, 1e5],
		"array_bools": [true, false, true],
		"array_nulls": [null, null],
		"array_mixed": [
			"string",
			123,
			true,
			null,
			{"nested": "object"},
			["nested", "array"]
		],
		
		"array_of_arrays": [
			[1, 2, 3],
			["a", "b", "c"],
			[true, false],
			[null],
			[[1, 2], [3, 4]]
		],
		
		"array_of_objects": [
			{"id": 1, "name": "first"},
			{"id": 2, "name": "second", "nested": {"key": "value"}},
			{}
		],
		
		"complex_nesting": {
			"array_in_object": [1, 2, {"key": [true, false, null]}],
			"object_in_array": [
				{
					"deep": {
						"array": [
							{"id": 1},
							{"id": 2, "data": [1, 2, 3]}
						]
					}
				}
			]
		},
		
		"edge_cases": {
			"very_long_number": 9223372036854775807,
			"very_small_number": 0.000000000001,
			"empty_array_in_object": {"items": []},
			"null_in_object": {"value": null},
			"unicode_key_名前": "value",
			"special-chars!@#$": "special"
		}
	}`)

	j := NewJSON("")
	result, err := j.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	// Test a comprehensive set of values
	tests := map[string]string{
		// Basic types
		"string":            "hello",
		"number.int":        "42",
		"number.float":      "3.14159",
		"number.negative":   "-123",
		"number.zero":       "0",
		"number.scientific": "1.23e10",
		"bool.true":         "true",
		"bool.false":        "false",
		"null.value":        "",
		"empty.string":      "",
		"unicode.string":    "Hello 世界 🌍 مرحبا",

		// Nested objects
		"object.simple.key":                       "value",
		"object.nested.level1.level2.level3.deep": "value",

		// Arrays
		"array.strings.0":      "a",
		"array.strings.1":      "b",
		"array.numbers.0":      "1",
		"array.numbers.1":      "2.5",
		"array.numbers.2":      "-3",
		"array.bools.0":        "true",
		"array.bools.1":        "false",
		"array.nulls.0":        "",
		"array.mixed.0":        "string",
		"array.mixed.1":        "123",
		"array.mixed.2":        "true",
		"array.mixed.3":        "",
		"array.mixed.4.nested": "object",

		// Arrays of arrays (become strings)
		"array.of.arrays.0": "[1 2 3]",
		"array.of.arrays.1": "[a b c]",
		"array.of.arrays.2": "[true false]",
		"array.of.arrays.3": "[<nil>]",
		"array.of.arrays.4": "[[1 2] [3 4]]",

		// Arrays of objects
		"array.of.objects.0.id":         "1",
		"array.of.objects.0.name":       "first",
		"array.of.objects.1.id":         "2",
		"array.of.objects.1.nested.key": "value",

		// Complex nesting
		"complex.nesting.array.in.object.0":       "1",
		"complex.nesting.array.in.object.1":       "2",
		"complex.nesting.array.in.object.2.key.0": "true",
		"complex.nesting.array.in.object.2.key.1": "false",
		"complex.nesting.array.in.object.2.key.2": "",

		// Edge cases
		"edge.cases.very.long.number":     "9223372036854775807",
		"edge.cases.very.small.number":    "0.000000000001",
		"edge.cases.null.in.object.value": "",
		"edge.cases.unicode.key.名前":       "value",
		"edge.cases.special.chars!@#$":    "special",
	}

	for key, expected := range tests {
		if actual, exists := result[key]; !exists {
			t.Errorf("Key %q not found in result", key)
		} else if actual != expected {
			t.Errorf("Key %q = %q, want %q", key, actual, expected)
		}
	}

	// Verify we handle all types
	if len(result) < 50 {
		t.Errorf("Expected at least 50 keys, got %d", len(result))
	}
}

func TestJSON_RealWorldExample(t *testing.T) {
	content := []byte(`{
		"name": "myapp",
		"version": "1.0.0",
		"dependencies": {
			"express": "^4.17.1",
			"mongodb": "^3.6.0"
		},
		"scripts": {
			"start": "node index.js",
			"test": "jest",
			"build": "webpack"
		},
		"author": {
			"name": "John Doe",
			"email": "john@example.com"
		},
		"keywords": ["web", "api", "rest"],
		"config": {
			"port": 3000,
			"database": {
				"host": "localhost",
				"port": 27017,
				"name": "mydb"
			}
		}
	}`)

	j := NewJSON("")
	result, err := j.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	checks := map[string]string{
		"name":                 "myapp",
		"version":              "1.0.0",
		"dependencies.express": "^4.17.1",
		"scripts.start":        "node index.js",
		"author.email":         "john@example.com",
		"keywords.0":           "web",
		"keywords.2":           "rest",
		"config.port":          "3000",
		"config.database.name": "mydb",
	}

	for key, expectedValue := range checks {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func BenchmarkJSON_LoadBytes_Small(b *testing.B) {
	content := []byte(`{
		"key1": "value1",
		"key2": 42,
		"nested": {
			"key3": true
		}
	}`)

	j := NewJSON("")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = j.LoadBytes(content)
	}
}

func BenchmarkJSON_LoadBytes_Medium(b *testing.B) {
	data := make(map[string]interface{})
	for i := 0; i < 20; i++ {
		section := make(map[string]interface{})
		for j := 0; j < 10; j++ {
			section[fmt.Sprintf("key_%d", j)] = fmt.Sprintf("value_%d_%d", i, j)
		}
		data[fmt.Sprintf("section_%d", i)] = section
	}
	content, _ := json.Marshal(data)

	j := NewJSON("")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = j.LoadBytes(content)
	}
}

func BenchmarkJSON_LoadBytes_Large(b *testing.B) {
	data := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		section := make(map[string]interface{})
		for j := 0; j < 20; j++ {
			section[fmt.Sprintf("key_%d", j)] = fmt.Sprintf("value_%d_%d", i, j)
		}
		data[fmt.Sprintf("section_%d", i)] = section
	}
	content, _ := json.Marshal(data)

	j := NewJSON("")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = j.LoadBytes(content)
	}
}

func BenchmarkJSON_LoadReader(b *testing.B) {
	content := `{
		"database": {
			"host": "localhost",
			"port": 5432
		},
		"server": {
			"host": "0.0.0.0",
			"port": 8080
		}
	}`

	j := NewJSON("")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, _ = j.LoadReader(reader)
	}
}
