package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTOML_Type(t *testing.T) {
	toml := NewTOML("test.toml")
	if toml.Type() != "toml" {
		t.Errorf("Type() = %q, want %q", toml.Type(), "toml")
	}
}

func TestTOML_NewTOML(t *testing.T) {
	// Test without options
	t1 := NewTOML("test.toml")
	if t1.Path != "test.toml" {
		t.Errorf("Path = %q, want %q", t1.Path, "test.toml")
	}
	if t1.options.bufferSize != 8192 {
		t.Errorf("bufferSize = %d, want %d", t1.options.bufferSize, 8192)
	}
	if t1.options.usePool != false {
		t.Errorf("usePool = %v, want %v", t1.options.usePool, false)
	}

	// Test with options
	t2 := NewTOML("test.toml", WithBufferSize(8192), WithPool(false))
	if t2.options.bufferSize != 8192 {
		t.Errorf("bufferSize = %d, want %d", t2.options.bufferSize, 8192)
	}
	if t2.options.usePool != false {
		t.Errorf("usePool = %v, want %v", t2.options.usePool, false)
	}
}

func TestTOML_Load_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	tomlPath := filepath.Join(tmpDir, "test.toml")

	content := `
# This is a comment
title = "TOML Example"

[database]
host = "localhost"
port = 5432
name = "testdb"
enabled = true

[server]
host = "0.0.0.0"
port = 8080
timeout = 30.5

[paths]
data = "/var/data"
logs = "/var/logs"
`

	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	toml := NewTOML(tomlPath)
	result, err := toml.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	expected := map[string]string{
		"title":            "TOML Example",
		"database.host":    "localhost",
		"database.port":    "5432",
		"database.name":    "testdb",
		"database.enabled": "true",
		"server.host":      "0.0.0.0",
		"server.port":      "8080",
		"server.timeout":   "30.5",
		"paths.data":       "/var/data",
		"paths.logs":       "/var/logs",
	}

	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q, want %q", key, value, expectedValue)
		}
	}
}

func TestTOML_Load_InvalidExtension(t *testing.T) {
	toml := NewTOML("test.txt")
	_, err := toml.Load()
	if err == nil {
		t.Error("Load() should error on invalid extension")
	}
	if !errors.Is(err, ErrInvalidExtension) {
		t.Errorf("Expected error about invalid extension, got: %v", err)
	}
}

func TestTOML_Load_NonExistentFile(t *testing.T) {
	toml := NewTOML("/non/existent/file.toml")
	_, err := toml.Load()
	if err == nil {
		t.Error("Load() should error on non-existent file")
	}
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("Expected error about opening file, got: %v", err)
	}
}

func TestTOML_LoadReader_WithPool(t *testing.T) {
	content := `
[section1]
key1 = "value1"
key2 = 42
key3 = true

[section2]
nested = { inner = "value" }
array = ["item1", "item2", "item3"]
`

	toml := NewTOML("", WithPool(true))
	reader := strings.NewReader(content)
	result, err := toml.LoadReader(reader)
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}

	expected := map[string]string{
		"section1.key1":         "value1",
		"section1.key2":         "42",
		"section1.key3":         "true",
		"section2.nested.inner": "value",
		"section2.array.0":      "item1",
		"section2.array.1":      "item2",
		"section2.array.2":      "item3",
	}

	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q, want %q", key, value, expectedValue)
		}
	}
}

func TestTOML_LoadReader_WithoutPool(t *testing.T) {
	content := `
[section]
key = "value"
number = 123
`

	toml := NewTOML("", WithPool(false))
	reader := strings.NewReader(content)
	result, err := toml.LoadReader(reader)
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}

	expected := map[string]string{
		"section.key":    "value",
		"section.number": "123",
	}

	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q, want %q", key, value, expectedValue)
		}
	}
}

func TestTOML_LoadReader_ErrorWithPool(t *testing.T) {
	toml := NewTOML("", WithPool(true))
	reader := &tomlErrorReader{err: io.ErrUnexpectedEOF}

	_, err := toml.LoadReader(reader)
	if err == nil {
		t.Error("LoadReader() should return error from reader")
	}
	if !errors.Is(err, ErrReadFailed) {
		t.Errorf("Expected error about reading TOML, got: %v", err)
	}
}

func TestTOML_LoadReader_ErrorWithoutPool(t *testing.T) {
	toml := NewTOML("", WithPool(false))
	reader := &tomlErrorReader{err: io.ErrUnexpectedEOF}

	_, err := toml.LoadReader(reader)
	if err == nil {
		t.Error("LoadReader() should return error from reader")
	}
	if !errors.Is(err, ErrReadFailed) {
		t.Errorf("Expected error about reading TOML, got: %v", err)
	}
}

func TestTOML_LoadReader_InvalidTOML(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		pool    bool
	}{
		{"invalid syntax with pool", "[section\nkey = value", true},
		{"invalid syntax without pool", "[section\nkey = value", false},
		{"invalid value with pool", "[section]\nkey = ", true},
		{"invalid value without pool", "[section]\nkey = ", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			toml := NewTOML("", WithPool(tc.pool))
			reader := strings.NewReader(tc.content)

			_, err := toml.LoadReader(reader)
			if err == nil {
				t.Error("LoadReader() should error on invalid TOML")
			}
			if !errors.Is(err, ErrTOMLParse) {
				t.Errorf("Expected error about parsing TOML, got: %v", err)
			}
		})
	}
}

func TestTOML_LoadBytes(t *testing.T) {
	content := []byte(`
# Complete TOML test
title = "Test"

[owner]
name = "Tom Preston-Werner"
dob = 1979-05-27T07:32:00-08:00

[database]
enabled = true
ports = [ 8000, 8001, 8002 ]
data = [ ["delta", "phi"], [3.14] ]
temp_targets = { cpu = 79.5, case = 72.0 }

[servers]

[servers.alpha]
ip = "10.0.0.1"
role = "frontend"

[servers.beta]
ip = "10.0.0.2"
role = "backend"

[[products]]
name = "Hammer"
sku = 738594937

[[products]]
name = "Nail"
sku = 284758393
color = "gray"
`)

	toml := NewTOML("")
	result, err := toml.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	// Check some key values
	expected := map[string]string{
		"title":            "Test",
		"owner.name":       "Tom Preston-Werner",
		"database.enabled": "true",
		"database.ports.0": "8000",
		"database.ports.1": "8001",
		"database.ports.2": "8002",
		// Arrays of arrays become nested structure
		"database.temp.targets.cpu":  "79.5",
		"database.temp.targets.case": "72",
		"servers.alpha.ip":           "10.0.0.1",
		"servers.alpha.role":         "frontend",
		"servers.beta.ip":            "10.0.0.2",
		"servers.beta.role":          "backend",
		"products.0.name":            "Hammer",
		"products.0.sku":             "738594937",
		"products.1.name":            "Nail",
		"products.1.sku":             "284758393",
		"products.1.color":           "gray",
	}

	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func TestTOML_LoadBytes_InvalidTOML(t *testing.T) {
	testCases := []struct {
		name    string
		content []byte
	}{
		{"invalid syntax", []byte("[section\nkey = value")},
		{"invalid value", []byte("[section]\nkey = ")},
		{"invalid type", []byte("key = 2020-13-01")}, // Invalid date
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			toml := NewTOML("")
			_, err := toml.LoadBytes(tc.content)
			if err == nil {
				t.Error("LoadBytes() should error on invalid TOML")
			}
			if !errors.Is(err, ErrTOMLParse) {
				t.Errorf("Expected error about parsing TOML, got: %v", err)
			}
		})
	}
}

func TestTOML_AllTypes(t *testing.T) {
	// Test ALL possible TOML types
	content := []byte(`
# Strings
str1 = "I'm a string."
str2 = 'You can "quote" me.'
str3 = """
Multi-line
string"""

# Integers
int1 = +99
int2 = 42
int3 = 0
int4 = -17
int5 = 1_000
int6 = 5_349_221
hex = 0xDEADBEEF
oct = 0o755
bin = 0b11010110

# Floats
flt1 = +1.0
flt2 = 3.1415
flt3 = -0.01
flt4 = 5e+22
flt5 = 1e06
flt6 = -2E-2
flt7 = 6.626e-34
inf1 = inf
inf2 = +inf
inf3 = -inf
nan1 = nan
nan2 = +nan
nan3 = -nan

# Booleans
bool1 = true
bool2 = false

# Dates and times
odt1 = 1979-05-27T07:32:00Z
odt2 = 1979-05-27T00:32:00-07:00
ldt1 = 1979-05-27T07:32:00
ld1 = 1979-05-27
lt1 = 07:32:00

# Arrays
integers = [ 1, 2, 3 ]
colors = [ "red", "yellow", "green" ]
nested_arrays = [ [ 1, 2 ], [3, 4, 5] ]
mixed_array = [ 1, "two", 3.0, true ]
empty_array = []

# Tables
[table]
key = "value"
inline = { first = "Tom", last = "Preston-Werner" }

# Array of tables
[[fruits]]
name = "apple"
color = "red"

[[fruits]]
name = "banana"
color = "yellow"

# Nested tables
[a.b.c]
deep = "value"

[x]
  [x.y]
    [x.y.z]
      w = "deep nesting"
`)

	toml := NewTOML("")
	result, err := toml.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	// Verify various types
	checks := map[string]string{
		"str1":       "I'm a string.",
		"str2":       "You can \"quote\" me.",
		"int1":       "99",
		"int2":       "42",
		"int3":       "0",
		"int4":       "-17",
		"hex":        "3735928559", // 0xDEADBEEF in decimal
		"oct":        "493",        // 0o755 in decimal
		"bin":        "214",        // 0b11010110 in decimal
		"flt1":       "1",
		"flt2":       "3.1415",
		"flt3":       "-0.01",
		"bool1":      "true",
		"bool2":      "false",
		"integers.0": "1",
		"colors.1":   "yellow",
		// Check first nested array element properly
		"table.key":          "value",
		"table.inline.first": "Tom",
		"fruits.0.name":      "apple",
		"fruits.1.color":     "yellow",
		"a.b.c.deep":         "value",
		"x.y.z.w":            "deep nesting",
	}

	for key, expectedValue := range checks {
		if value, ok := result[key]; !ok {
			t.Errorf("Key %q not found in result", key)
		} else if value != expectedValue {
			t.Errorf("Key %q = %q, want %q", key, value, expectedValue)
		}
	}
}

// Use tomlErrorReader to avoid conflict with ini_test.go
type tomlErrorReader struct {
	err error
}

func (r *tomlErrorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func BenchmarkTOML_LoadBytes_Small(b *testing.B) {
	content := []byte(`
[section]
key1 = "value1"
key2 = 42
key3 = true
`)

	toml := NewTOML("")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = toml.LoadBytes(content)
	}
}

func BenchmarkTOML_LoadBytes_Large(b *testing.B) {
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&buf, "[section%d]\n", i)
		for j := 0; j < 20; j++ {
			fmt.Fprintf(&buf, "key%d = \"value_%d_%d\"\n", j, i, j)
		}
	}
	content := buf.Bytes()

	toml := NewTOML("")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = toml.LoadBytes(content)
	}
}

func BenchmarkTOML_LoadReader_WithPool(b *testing.B) {
	content := `
[database]
host = "localhost"
port = 5432
[server]
host = "0.0.0.0"
port = 8080
`

	toml := NewTOML("", WithPool(true))
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, _ = toml.LoadReader(reader)
	}
}

// Tests for missing coverage in TOML parser - error paths and edge cases

func TestTOML_Load_ReadError(t *testing.T) {
	// Test general read error handling (not file not found)
	tmpDir := t.TempDir()
	tomlPath := filepath.Join(tmpDir, "unreadable.toml")

	// Write file first
	if err := os.WriteFile(tomlPath, []byte(`key = "value"`), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Make it unreadable by changing permissions
	if err := os.Chmod(tomlPath, 0000); err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	// Restore permissions after test
	defer os.Chmod(tomlPath, 0644)

	toml := NewTOML(tomlPath)
	_, err := toml.Load()
	if err == nil {
		t.Error("Load() should error on unreadable file")
	}
	if !errors.Is(err, ErrReadFailed) {
		t.Errorf("Expected ErrReadFailed for read error, got: %v", err)
	}
}

func TestTOML_LoadReader_ReaderError(t *testing.T) {
	// Test LoadReader with reader that fails
	toml := NewTOML("")
	reader := &tomlErrorReader{err: io.ErrUnexpectedEOF}

	_, err := toml.LoadReader(reader)
	if err == nil {
		t.Error("LoadReader() should return error from reader")
	}
	if !errors.Is(err, ErrReadFailed) {
		t.Errorf("Expected ErrReadFailed for reader error, got: %v", err)
	}
}

func TestTOML_LoadBytes_MalformedTOML(t *testing.T) {
	// Test various malformed TOML scenarios
	testCases := []struct {
		name    string
		content []byte
	}{
		{
			name:    "unclosed string",
			content: []byte(`key = "unclosed string`),
		},
		{
			name:    "invalid table header",
			content: []byte(`[invalid table name with spaces`),
		},
		{
			name:    "duplicate keys",
			content: []byte(`key = "value1"\nkey = "value2"`),
		},
		{
			name:    "invalid datetime",
			content: []byte(`date = 2020-13-01T25:00:00Z`), // Invalid month and hour
		},
		{
			name:    "invalid nested table",
			content: []byte(`[a]\n[a.b]\nb = 1\n[a]\nc = 2`), // Redefining table
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			toml := NewTOML("")
			_, err := toml.LoadBytes(tc.content)
			if err == nil {
				t.Error("LoadBytes() should error on malformed TOML")
			}
			if !errors.Is(err, ErrTOMLParse) {
				t.Errorf("Expected ErrTOMLParse for malformed TOML, got: %v", err)
			}
		})
	}
}

func TestTOML_LoadBytes_Empty(t *testing.T) {
	// Test with empty TOML content
	toml := NewTOML("")
	result, err := toml.LoadBytes([]byte{})
	if err != nil {
		t.Fatalf("LoadBytes() should handle empty content, got error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result for empty TOML, got %d items", len(result))
	}
}

func TestTOML_LoadBytes_OnlyWhitespace(t *testing.T) {
	// Test with only whitespace and comments
	content := []byte(`
		# This is just a comment
		
		# Another comment
	`)

	toml := NewTOML("")
	result, err := toml.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() should handle whitespace and comments, got error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result for whitespace-only TOML, got %d items", len(result))
	}
}

func TestTOML_LoadBytes_ComplexEdgeCases(t *testing.T) {
	// Test complex TOML edge cases
	content := []byte(`
# Global key
global = "value"

# Table with special characters
["special-table.with.dots"]
key = "table value"

# Array of tables
[[fruits]]
name = "apple"

[[fruits]]  
name = "orange"

# Nested table
[a.b.c]
nested = "deep"

# Inline table
inline = { x = 1, y = 2 }

# Various data types
string = "text"
integer = 123
float = 45.67
boolean = true
datetime = 1979-05-27T07:32:00Z

# Arrays
simple_array = [1, 2, 3]
string_array = ["a", "b", "c"]
mixed_nested = [[1, 2], [3, 4]]
`)

	toml := NewTOML("")
	result, err := toml.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}

	// Verify some key values
	expectedChecks := map[string]string{
		"global":                      "value",
		"special-table.with.dots.key": "table value",
		"fruits.0.name":               "apple",
		"fruits.1.name":               "orange",
		"a.b.c.nested":                "deep",
		"a.b.c.inline.x":              "1",
		"a.b.c.inline.y":              "2",
		"a.b.c.string":                "text",
		"a.b.c.integer":               "123",
		"a.b.c.float":                 "45.67",
		"a.b.c.boolean":               "true",
		"a.b.c.simple.array.0":        "1",
		"a.b.c.string.array.0":        "a",
		"a.b.c.mixed.nested.0":        "[1 2]",
		"a.b.c.mixed.nested.1":        "[3 4]",
	}

	for key, expected := range expectedChecks {
		if actual, exists := result[key]; !exists {
			t.Errorf("Key %q not found in result", key)
		} else if actual != expected {
			t.Errorf("Key %q = %q, want %q", key, actual, expected)
		}
	}
}

func TestTOML_LoadBytes_InvalidSyntaxVariations(t *testing.T) {
	// Test various invalid TOML syntax patterns
	invalidTOMLs := []struct {
		name    string
		content []byte
		reason  string
	}{
		{
			name:    "bare key with space",
			content: []byte(`bare key = "value"`),
			reason:  "bare keys cannot contain spaces",
		},
		{
			name:    "table name not closed",
			content: []byte(`[table\nkey = "value"`),
			reason:  "table name bracket not closed",
		},
		{
			name:    "string not closed",
			content: []byte(`key = "unclosed`),
			reason:  "string literal not properly closed",
		},
		{
			name:    "invalid escape sequence",
			content: []byte(`key = "invalid\x escape"`),
			reason:  "invalid escape sequence in string",
		},
		{
			name:    "array not closed",
			content: []byte(`key = [1, 2, 3`),
			reason:  "array not properly closed",
		},
	}

	for _, tc := range invalidTOMLs {
		t.Run(tc.name, func(t *testing.T) {
			toml := NewTOML("")
			_, err := toml.LoadBytes(tc.content)
			if err == nil {
				t.Errorf("LoadBytes() should error for %s", tc.reason)
			}
			if !errors.Is(err, ErrTOMLParse) {
				t.Errorf("Expected ErrTOMLParse for %s, got: %v", tc.reason, err)
			}
		})
	}
}

func BenchmarkTOML_LoadReader_WithoutPool(b *testing.B) {
	content := `
[database]
host = "localhost"
port = 5432
[server]
host = "0.0.0.0"
port = 8080
`

	toml := NewTOML("", WithPool(false))
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, _ = toml.LoadReader(reader)
	}
}

// Concurrent Tests

func TestTOML_LoadReader_Concurrent(t *testing.T) {
	content := `
[database]
host = "localhost"
port = 5432
name = "testdb"

[server]
host = "0.0.0.0"
port = 8080
debug = true
`

	const numGoroutines = 100
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			toml := NewTOML("")
			reader := strings.NewReader(content)
			result, err := toml.LoadReader(reader)
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
			if result["server.debug"] != "true" {
				t.Errorf("Concurrent test %d: server.debug = %q, want %q", i, result["server.debug"], "true")
			}
		case err := <-errors:
			t.Errorf("Concurrent test error: %v", err)
		}
	}
}

func TestTOML_LoadBytes_Concurrent(t *testing.T) {
	content := []byte(`
[section1]
key1 = "value1"
key2 = 42

[section2]
key3 = true
array = ["item1", "item2"]
`)

	const numGoroutines = 50
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			toml := NewTOML("")
			result, err := toml.LoadBytes(content)
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
			if result["section1.key1"] != "value1" {
				t.Errorf("Concurrent bytes test %d: section1.key1 = %q, want %q", i, result["section1.key1"], "value1")
			}
			if result["section2.array.0"] != "item1" {
				t.Errorf("Concurrent bytes test %d: section2.array.0 = %q, want %q", i, result["section2.array.0"], "item1")
			}
		case err := <-errors:
			t.Errorf("Concurrent bytes test error: %v", err)
		}
	}
}

// Panic Recovery Tests

func TestTOML_LoadReader_PanicRecovery(t *testing.T) {
	malformedContents := []string{
		"",                              // empty
		"[",                             // incomplete section
		"key = ",                        // incomplete value
		"key = [unclosed",               // unclosed array
		string([]byte{0, 1, 2, 3, 255}), // binary data
		strings.Repeat("[section]\nkey = \"value\"\n", 10000),        // very large
		"[" + strings.Repeat("section", 1000) + "]\nkey = \"value\"", // very long section
		"[section]\n" + strings.Repeat("key", 1000) + " = \"value\"", // very long key
		"[section]\nkey = \"" + strings.Repeat("value", 1000) + "\"", // very long value
		"[section]\nkey = \"value\x00with\x00nulls\"",                // null bytes
		"[测试]\n键 = \"值\"",                                            // unicode
	}

	for i, content := range malformedContents {
		t.Run(fmt.Sprintf("malformed_input_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("LoadReader panicked with input %d: %v", i, r)
				}
			}()

			toml := NewTOML("")
			reader := strings.NewReader(content)
			_, _ = toml.LoadReader(reader)
		})
	}
}

func TestTOML_LoadBytes_PanicRecovery(t *testing.T) {
	panicInputs := [][]byte{
		nil,                   // nil slice
		{},                    // empty slice
		{0},                   // single null byte
		make([]byte, 1000000), // very large empty content
		bytes.Repeat([]byte("[section]\nkey = \"value\"\n"), 50000), // extremely large
		[]byte("[incomplete"),             // incomplete section
		[]byte("key = [unclosed"),         // unclosed array
		[]byte(strings.Repeat("[", 1000)), // many opening brackets
	}

	for i, content := range panicInputs {
		t.Run(fmt.Sprintf("panic_input_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("LoadBytes panicked with input %d: %v", i, r)
				}
			}()

			toml := NewTOML("")
			_, _ = toml.LoadBytes(content)
		})
	}
}

// Multi-threaded Benchmarks

func BenchmarkTOML_LoadReader_Concurrent(b *testing.B) {
	content := `
[database]
host = "localhost" 
port = 5432
[server]
host = "0.0.0.0"
port = 8080
`

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			toml := NewTOML("")
			reader := strings.NewReader(content)
			_, _ = toml.LoadReader(reader)
		}
	})
}

func BenchmarkTOML_LoadBytes_Concurrent(b *testing.B) {
	content := []byte(`
[section]
key1 = "value1"
key2 = 42
key3 = true
`)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			toml := NewTOML("")
			_, _ = toml.LoadBytes(content)
		}
	})
}
