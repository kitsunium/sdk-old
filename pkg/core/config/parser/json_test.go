package parser

import (
	"bytes"
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
	if j1.options.bufferSize != 8192 {
		t.Errorf("bufferSize = %d, want %d", j1.options.bufferSize, 8192)
	}
	if j1.options.usePool != false {
		t.Errorf("usePool = %v, want %v", j1.options.usePool, false)
	}

	// Test with options
	j2 := NewJSON("test.json", WithBufferSize(4096), WithPool(true))
	if j2.options.bufferSize != 4096 {
		t.Errorf("bufferSize = %d, want %d", j2.options.bufferSize, 4096)
	}
	if j2.options.usePool != true {
		t.Errorf("usePool = %v, want %v", j2.options.usePool, true)
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
	if !errors.Is(err, ErrInvalidExtension) {
		t.Errorf("Expected ErrInvalidExtension, got: %v", err)
	}
}

func TestJSON_Load_NonExistentFile(t *testing.T) {
	j := NewJSON("/non/existent/file.json")
	_, err := j.Load()
	if err == nil {
		t.Error("Load() should error on non-existent file")
	}
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("Expected ErrFileNotFound, got: %v", err)
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
	if !errors.Is(err, ErrReadFailed) {
		t.Errorf("Expected ErrReadFailed, got: %v", err)
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
		"edge.cases.special-chars!@#$":    "special",
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

// Tests for missing coverage in JSON parser - error paths and edge cases

func TestJSON_LoadBytes_ExceedsMaxSize(t *testing.T) {
	// Test JSON size validation - create data larger than MaxJSONSize (10MB)
	j := NewJSON("")

	// Create data that exceeds MaxJSONSize
	bigData := make([]byte, MaxJSONSize+1)
	copy(bigData, []byte(`{"key": "`))
	for i := 10; i < len(bigData)-2; i++ {
		bigData[i] = 'a'
	}
	copy(bigData[len(bigData)-2:], []byte(`"}`))

	_, err := j.LoadBytes(bigData)
	if err == nil {
		t.Error("LoadBytes() should error on data exceeding MaxJSONSize")
	}
	if !errors.Is(err, ErrJSONParse) {
		t.Errorf("Expected ErrJSONParse for size violation, got: %v", err)
	}
}

func TestJSON_LoadBytes_TrailingData(t *testing.T) {
	// Test detection of trailing data after valid JSON
	j := NewJSON("")

	// Valid JSON followed by extra data
	content := []byte(`{"key": "value"} {"extra": "data"}`)

	_, err := j.LoadBytes(content)
	if err == nil {
		t.Error("LoadBytes() should error on trailing data")
	}
	if !errors.Is(err, ErrJSONParse) {
		t.Errorf("Expected ErrJSONParse for trailing data, got: %v", err)
	}
}

func TestJSON_LoadBytes_MaxDepthExceeded(t *testing.T) {
	// Test maximum nesting depth protection
	j := NewJSON("")

	// Create deeply nested JSON that exceeds MaxJSONDepth
	var content strings.Builder
	content.WriteString(`{`)
	for i := 0; i < MaxJSONDepth+5; i++ {
		content.WriteString(fmt.Sprintf(`"level%d": {`, i))
	}
	content.WriteString(`"deep": "value"`)
	for i := 0; i < MaxJSONDepth+5; i++ {
		content.WriteString(`}`)
	}
	content.WriteString(`}`)

	_, err := j.LoadBytes([]byte(content.String()))
	if err == nil {
		t.Error("LoadBytes() should error on excessive nesting depth")
	}
	if !errors.Is(err, ErrJSONParse) {
		t.Errorf("Expected ErrJSONParse for depth violation, got: %v", err)
	}
}

func TestJSON_Load_ReadError(t *testing.T) {
	// Test general read error handling (not file not found)
	// Create a file that exists but can't be read (permission error simulation)
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "unreadable.json")

	// Write file first
	if err := os.WriteFile(jsonPath, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Make it unreadable by changing permissions
	if err := os.Chmod(jsonPath, 0000); err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	// Restore permissions after test
	defer os.Chmod(jsonPath, 0644)

	j := NewJSON(jsonPath)
	_, err := j.Load()
	if err == nil {
		t.Error("Load() should error on unreadable file")
	}
	if !errors.Is(err, ErrReadFailed) {
		t.Errorf("Expected ErrReadFailed for read error, got: %v", err)
	}
}

func TestJSON_ProcessValue_AllTypeBranches(t *testing.T) {
	// Test all type branches in processValue function
	testCases := []struct {
		name     string
		content  []byte
		expected map[string]string
	}{
		{
			name:    "int64 type",
			content: []byte(`{"int64": 9223372036854775807}`),
			expected: map[string]string{
				"int64": "9223372036854775807",
			},
		},
		{
			name:    "default type case",
			content: []byte(`{"custom": 123}`),
			expected: map[string]string{
				"custom": "123",
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

			for key, expected := range tc.expected {
				if actual, ok := result[key]; !ok || actual != expected {
					t.Errorf("key %q = %q, want %q", key, actual, expected)
				}
			}
		})
	}
}

func TestJSON_ProcessArray_NestedArrayError(t *testing.T) {
	// Test array processing with nested structures that might cause depth errors
	j := NewJSON("")

	// Create an array with deeply nested maps to test depth handling
	var content strings.Builder
	content.WriteString(`{"array": [{`)
	for i := 0; i < MaxJSONDepth-5; i++ {
		content.WriteString(fmt.Sprintf(`"level%d": {`, i))
	}
	content.WriteString(`"deep": "value"`)
	for i := 0; i < MaxJSONDepth-5; i++ {
		content.WriteString(`}`)
	}
	content.WriteString(`}]}`)

	_, err := j.LoadBytes([]byte(content.String()))
	// This should succeed as it's within limits
	if err != nil {
		t.Errorf("LoadBytes() should handle deep but valid nesting, got error: %v", err)
	}
}

func TestJSON_FastFloat64ToString_EdgeCases(t *testing.T) {
	// Test the fastFloat64ToString helper function edge cases
	testCases := []struct {
		input    float64
		expected string
	}{
		{42.0, "42"},
		{42.5, "42.5"},
		{0.0, "0"},
		{-0.0, "0"},
		{1e10, "10000000000"},
		{1e-10, "1e-10"},
		{3.14159265359, "3.14159265359"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("float64_%v", tc.input), func(t *testing.T) {
			result := fastFloat64ToString(tc.input)
			if result != tc.expected {
				t.Errorf("fastFloat64ToString(%v) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestJSON_NormalizeJSONNumber_Coverage(t *testing.T) {
	// Test normalizeJSONNumber function coverage
	number := json.Number("123.456")
	result := normalizeJSONNumber(number)
	if result != "123.456" {
		t.Errorf("normalizeJSONNumber() = %q, want %q", result, "123.456")
	}

	// Test with scientific notation
	number2 := json.Number("1.23e10")
	result2 := normalizeJSONNumber(number2)
	if result2 != "1.23e10" {
		t.Errorf("normalizeJSONNumber() = %q, want %q", result2, "1.23e10")
	}
}

func TestJSON_BuildKey_Coverage(t *testing.T) {
	// Test buildKey function coverage
	j := &JSON{}
	keyBuilder := &strings.Builder{}

	// Test with prefix and dot
	result := j.buildKey(keyBuilder, "prefix", "key", true)
	expected := "prefix.key"
	if result != expected {
		t.Errorf("buildKey() = %q, want %q", result, expected)
	}

	// Test without prefix (no dot needed)
	result2 := j.buildKey(keyBuilder, "", "key", false)
	expected2 := "key"
	if result2 != expected2 {
		t.Errorf("buildKey() = %q, want %q", result2, expected2)
	}
}

func TestJSON_ArrayProcessing_ComplexNesting(t *testing.T) {
	// Test complex array processing scenarios
	content := []byte(`{
		"complex": [
			{"nested": [{"deep": "value1"}]},
			{"nested": [{"deep": "value2"}]}
		]
	}`)

	j := NewJSON("")
	result, err := j.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	// Verify complex nested array structure
	expected := map[string]string{
		"complex.0.nested.0.deep": "value1",
		"complex.1.nested.0.deep": "value2",
	}

	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
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

// Concurrent Tests

func TestJSON_LoadReader_Concurrent(t *testing.T) {
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

	const numGoroutines = 100
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			j := NewJSON("")
			reader := strings.NewReader(content)
			result, err := j.LoadReader(reader)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			if result["database.host"] != "localhost" {
				t.Errorf("Concurrent test %d: database.host = %q, want %q", i, result["database.host"], "localhost")
			}
			if result["server.port"] != "8080" {
				t.Errorf("Concurrent test %d: server.port = %q, want %q", i, result["server.port"], "8080")
			}
		case err := <-errors:
			t.Errorf("Concurrent test error: %v", err)
		}
	}
}

func TestJSON_LoadBytes_Concurrent(t *testing.T) {
	content := []byte(`{
		"array": ["item1", "item2", "item3"],
		"object": {
			"nested": {
				"value": "test"
			}
		},
		"number": 42,
		"bool": true
	}`)

	const numGoroutines = 50
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			j := NewJSON("")
			result, err := j.LoadBytes(content)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			if result["array.0"] != "item1" {
				t.Errorf("Concurrent bytes test %d: array.0 = %q, want %q", i, result["array.0"], "item1")
			}
			if result["object.nested.value"] != "test" {
				t.Errorf("Concurrent bytes test %d: object.nested.value = %q, want %q", i, result["object.nested.value"], "test")
			}
			if result["number"] != "42" {
				t.Errorf("Concurrent bytes test %d: number = %q, want %q", i, result["number"], "42")
			}
		case err := <-errors:
			t.Errorf("Concurrent bytes test error: %v", err)
		}
	}
}

func TestJSON_Load_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()

	const numGoroutines = 20
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			jsonPath := filepath.Join(tmpDir, fmt.Sprintf("test_%d.json", id))
			content := fmt.Sprintf(`{
				"worker_id": %d,
				"host": "localhost",
				"port": %d,
				"config": {
					"timeout": 30,
					"retries": 3
				}
			}`, id, 8080+id)

			if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
				errors <- err
				return
			}

			j := NewJSON(jsonPath)
			result, err := j.Load()
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			if result["host"] != "localhost" {
				t.Errorf("Concurrent load test: host = %q, want %q", result["host"], "localhost")
			}
			if result["config.timeout"] != "30" {
				t.Errorf("Concurrent load test: config.timeout = %q, want %q", result["config.timeout"], "30")
			}
		case err := <-errors:
			t.Errorf("Concurrent load test error: %v", err)
		}
	}
}

// Panic Recovery Tests

func TestJSON_LoadReader_PanicRecovery(t *testing.T) {
	malformedContents := []string{
		"",                              // empty
		"{",                             // incomplete
		"}",                             // incomplete
		`{"key":}`,                      // missing value
		`{"key": "value",}`,             // trailing comma
		string([]byte{0, 1, 2, 3, 255}), // binary data
		strings.Repeat(`{"key": "value"},`, 10000),         // very large malformed
		`{"` + strings.Repeat("key", 1000) + `": "value"}`, // very long key
		`{"key": "` + strings.Repeat("value", 1000) + `"}`, // very long value
		`{"key": "value\x00with\x00nulls"}`,                // null bytes
		`{"测试": "值"}`,                                      // unicode
		`{"\u0000": "null char key"}`,                      // null character in key
		`{"key": "\u{invalid}"}`,                           // invalid unicode
	}

	for i, content := range malformedContents {
		t.Run(fmt.Sprintf("malformed_input_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("LoadReader panicked with input %d: %v", i, r)
				}
			}()

			j := NewJSON("")
			reader := strings.NewReader(content)
			_, _ = j.LoadReader(reader)
		})
	}
}

func TestJSON_LoadBytes_PanicRecovery(t *testing.T) {
	panicInputs := [][]byte{
		nil,                    // nil slice
		{},                     // empty slice
		{0},                    // single null byte
		make([]byte, 10000000), // very large empty content
		bytes.Repeat([]byte(`{"key": "value"},`), 100000), // extremely large malformed
		[]byte(`{`),                       // just opening brace
		[]byte(`}`),                       // just closing brace
		[]byte(strings.Repeat(`{`, 1000)), // many opening braces
		[]byte(strings.Repeat(`}`, 1000)), // many closing braces
		[]byte(`{"key": ` + strings.Repeat(`[`, 1000) + `}`), // unbalanced brackets
	}

	for i, content := range panicInputs {
		t.Run(fmt.Sprintf("panic_input_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("LoadBytes panicked with input %d: %v", i, r)
				}
			}()

			j := NewJSON("")
			_, _ = j.LoadBytes(content)
		})
	}
}

func TestJSON_Load_PanicRecovery(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name    string
		setup   func() string
		cleanup func(string)
	}{
		{
			name: "empty_file",
			setup: func() string {
				path := filepath.Join(tmpDir, "empty.json")
				os.WriteFile(path, []byte{}, 0644)
				return path
			},
			cleanup: func(path string) { os.Remove(path) },
		},
		{
			name: "binary_file",
			setup: func() string {
				path := filepath.Join(tmpDir, "binary.json")
				os.WriteFile(path, make([]byte, 1000), 0644)
				return path
			},
			cleanup: func(path string) { os.Remove(path) },
		},
		{
			name: "very_large_file",
			setup: func() string {
				path := filepath.Join(tmpDir, "large.json")
				largeObject := make(map[string]interface{})
				for i := 0; i < 10000; i++ {
					largeObject[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
				}
				content, _ := json.Marshal(largeObject)
				os.WriteFile(path, content, 0644)
				return path
			},
			cleanup: func(path string) { os.Remove(path) },
		},
		{
			name: "unicode_file",
			setup: func() string {
				path := filepath.Join(tmpDir, "unicode.json")
				content := `{"测试": "值", "🌍": "world", "العربية": "arabic"}`
				os.WriteFile(path, []byte(content), 0644)
				return path
			},
			cleanup: func(path string) { os.Remove(path) },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setup()
			defer tc.cleanup(path)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Load panicked with %s: %v", tc.name, r)
				}
			}()

			j := NewJSON(path)
			_, _ = j.Load()
		})
	}
}

// Multi-threaded Benchmarks

func BenchmarkJSON_LoadReader_Concurrent(b *testing.B) {
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

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			j := NewJSON("")
			reader := strings.NewReader(content)
			_, _ = j.LoadReader(reader)
		}
	})
}

func BenchmarkJSON_LoadBytes_Concurrent_Small(b *testing.B) {
	content := []byte(`{
		"key1": "value1",
		"key2": 42,
		"nested": {
			"key3": true
		}
	}`)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			j := NewJSON("")
			_, _ = j.LoadBytes(content)
		}
	})
}

func BenchmarkJSON_LoadBytes_Concurrent_Large(b *testing.B) {
	data := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		section := make(map[string]interface{})
		for j := 0; j < 10; j++ {
			section[fmt.Sprintf("key_%d", j)] = fmt.Sprintf("value_%d_%d", i, j)
		}
		data[fmt.Sprintf("section_%d", i)] = section
	}
	content, _ := json.Marshal(data)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			j := NewJSON("")
			_, _ = j.LoadBytes(content)
		}
	})
}
