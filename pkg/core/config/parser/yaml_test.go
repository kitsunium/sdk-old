package parser

import (
	"errors"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestYAML_Type(t *testing.T) {
	yaml := NewYAML("test.yaml")
	if yaml.Type() != "yaml" {
		t.Errorf("Type() = %q, want %q", yaml.Type(), "yaml")
	}
}

func TestYAML_NewYAML(t *testing.T) {
	// Test without options
	y1 := NewYAML("test.yaml")
	if y1.Path != "test.yaml" {
		t.Errorf("Path = %q, want %q", y1.Path, "test.yaml")
	}
	if y1.options.bufferSize != 8192 {
		t.Errorf("bufferSize = %d, want %d", y1.options.bufferSize, 8192)
	}
	if y1.options.usePool != false {
		t.Errorf("usePool = %v, want %v", y1.options.usePool, false)
	}
	
	// Test with options
	y2 := NewYAML("test.yml", WithBufferSize(8192), WithPool(false))
	if y2.options.bufferSize != 8192 {
		t.Errorf("bufferSize = %d, want %d", y2.options.bufferSize, 8192)
	}
	if y2.options.usePool != false {
		t.Errorf("usePool = %v, want %v", y2.options.usePool, false)
	}
}

func TestYAML_Load_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "test.yaml")
	
	content := `# YAML Configuration
database:
  host: localhost
  port: 5432
  name: testdb
  enabled: true

server:
  host: 0.0.0.0
  port: 8080
  timeout: 30.5

paths:
  data: /var/data
  logs: /var/logs
`
	
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	yaml := NewYAML(yamlPath)
	result, err := yaml.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	
	expected := map[string]string{
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
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func TestYAML_Load_YMLExtension(t *testing.T) {
	tmpDir := t.TempDir()
	ymlPath := filepath.Join(tmpDir, "test.yml")
	
	content := `key: value`
	
	if err := os.WriteFile(ymlPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	yaml := NewYAML(ymlPath)
	result, err := yaml.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	
	if result["key"] != "value" {
		t.Errorf("key = %q, want %q", result["key"], "value")
	}
}

func TestYAML_Load_InvalidExtension(t *testing.T) {
	yaml := NewYAML("test.txt")
	_, err := yaml.Load()
	if err == nil {
		t.Error("Load() should error on invalid extension")
	}
	if !errors.Is(err, ErrInvalidExtension) {
		t.Errorf("Expected error about invalid extension, got: %v", err)
	}
}

func TestYAML_Load_NonExistentFile(t *testing.T) {
	yaml := NewYAML("/non/existent/file.yaml")
	_, err := yaml.Load()
	if err == nil {
		t.Error("Load() should error on non-existent file")
	}
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("Expected error about opening file, got: %v", err)
	}
}

func TestYAML_LoadReader_WithPool(t *testing.T) {
	content := `
section1:
  key1: value1
  key2: 42
  key3: true

section2:
  nested:
    inner: value
  array:
    - item1
    - item2
    - item3
`
	
	yaml := NewYAML("", WithPool(true))
	reader := strings.NewReader(content)
	result, err := yaml.LoadReader(reader)
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}
	
	expected := map[string]string{
		"section1.key1":        "value1",
		"section1.key2":        "42",
		"section1.key3":        "true",
		"section2.nested.inner": "value",
		"section2.array.0":     "item1",
		"section2.array.1":     "item2",
		"section2.array.2":     "item3",
	}
	
	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func TestYAML_LoadReader_WithoutPool(t *testing.T) {
	content := `
section:
  key: value
  number: 123
`
	
	yaml := NewYAML("", WithPool(false))
	reader := strings.NewReader(content)
	result, err := yaml.LoadReader(reader)
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}
	
	expected := map[string]string{
		"section.key":    "value",
		"section.number": "123",
	}
	
	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func TestYAML_LoadReader_ErrorWithPool(t *testing.T) {
	yaml := NewYAML("", WithPool(true))
	reader := &yamlErrorReader{err: io.ErrUnexpectedEOF}
	
	_, err := yaml.LoadReader(reader)
	if err == nil {
		t.Error("LoadReader() should return error from reader")
	}
	if !errors.Is(err, ErrReadFailed) {
		t.Errorf("Expected error about reading YAML, got: %v", err)
	}
}

func TestYAML_LoadReader_ErrorWithoutPool(t *testing.T) {
	yaml := NewYAML("", WithPool(false))
	reader := &yamlErrorReader{err: io.ErrUnexpectedEOF}
	
	_, err := yaml.LoadReader(reader)
	if err == nil {
		t.Error("LoadReader() should return error from reader")
	}
	if !errors.Is(err, ErrReadFailed) {
		t.Errorf("Expected error about reading YAML, got: %v", err)
	}
}

func TestYAML_LoadReader_InvalidYAML(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		pool    bool
	}{
		{"invalid syntax with pool", "key: [unclosed", true},
		{"invalid syntax without pool", "key: [unclosed", false},
		{"invalid anchor with pool", "<<: *undefined", true},
		{"invalid anchor without pool", "<<: *undefined", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			yaml := NewYAML("", WithPool(tc.pool))
			reader := strings.NewReader(tc.content)
			
			_, err := yaml.LoadReader(reader)
			if err == nil {
				t.Error("LoadReader() should error on invalid YAML")
			}
			if !errors.Is(err, ErrYAMLParse) {
				t.Errorf("Expected error about parsing YAML, got: %v", err)
			}
		})
	}
}

func TestYAML_LoadBytes(t *testing.T) {
	content := []byte(`
# Complete YAML test
title: Test

owner:
  name: Tom Preston-Werner
  dob: 1979-05-27T07:32:00-08:00

database:
  enabled: true
  ports:
    - 8000
    - 8001
    - 8002
  data:
    - [delta, phi]
    - [3.14]
  temp_targets:
    cpu: 79.5
    case: 72.0

servers:
  alpha:
    ip: 10.0.0.1
    role: frontend
  beta:
    ip: 10.0.0.2
    role: backend

products:
  - name: Hammer
    sku: 738594937
  - name: Nail
    sku: 284758393
    color: gray
`)
	
	yaml := NewYAML("")
	result, err := yaml.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}
	
	// Check some key values
	expected := map[string]string{
		"title":                   "Test",
		"owner.name":             "Tom Preston-Werner",
		"database.enabled":       "true",
		"database.ports.0":       "8000",
		"database.ports.1":       "8001",
		"database.ports.2":       "8002",
		"database.temp.targets.cpu": "79.5",
		"database.temp.targets.case": "72",
		"servers.alpha.ip":       "10.0.0.1",
		"servers.alpha.role":     "frontend",
		"servers.beta.ip":        "10.0.0.2",
		"servers.beta.role":      "backend",
		"products.0.name":        "Hammer",
		"products.0.sku":         "738594937",
		"products.1.name":        "Nail",
		"products.1.sku":         "284758393",
		"products.1.color":       "gray",
	}
	
	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q (exists=%v), want %q", key, value, ok, expectedValue)
		}
	}
}

func TestYAML_LoadBytes_InvalidYAML(t *testing.T) {
	testCases := []struct {
		name    string
		content []byte
	}{
		{"invalid syntax", []byte("key: [unclosed")},
		{"invalid anchor", []byte("<<: *undefined")},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			yaml := NewYAML("")
			_, err := yaml.LoadBytes(tc.content)
			if err == nil {
				t.Error("LoadBytes() should error on invalid YAML")
			}
			if !errors.Is(err, ErrYAMLParse) {
				t.Errorf("Expected error about parsing YAML, got: %v", err)
			}
		})
	}
}

func TestYAML_AllTypes(t *testing.T) {
	// Test ALL possible YAML types
	content := []byte(`
# Strings
str1: "I'm a string."
str2: 'You can "quote" me.'
str3: |
  Multi-line
  string
str4: >
  Folded
  string
str5: Plain string

# Numbers
int1: 99
int2: 42
int3: 0
int4: -17
hex: 0xDEADBEEF
oct: 0755
float1: 1.0
float2: 3.1415
float3: -0.01
scientific: 6.626e-34

# Booleans
bool1: true
bool2: false
bool3: yes
bool4: no
bool5: on
bool6: off

# Null
null1: ~
null2: null
null3:

# Dates
date1: 2002-12-14
datetime: 2001-12-15T02:59:43.1Z

# Arrays
integers: [1, 2, 3]
colors:
  - red
  - yellow
  - green
nested_arrays:
  - [1, 2]
  - [3, 4, 5]
mixed_array: [1, "two", 3.0, true]
empty_array: []

# Objects/Maps
inline: {first: Tom, last: Preston-Werner}
mapping:
  key: value
  another: value2

# Anchors and Aliases
defaults: &defaults
  adapter: postgres
  host: localhost

development:
  database: dev_db
  <<: *defaults

production:
  database: prod_db
  <<: *defaults
  host: prod.example.com

# Complex nested
deeply:
  nested:
    structure:
      with:
        many:
          levels: value

# Special types
binary: !!binary |
  R0lGODlhAQABAIAAAAAAAP///w==
set: !!set
  ? item1
  ? item2
  ? item3
omap: !!omap
  - key1: value1
  - key2: value2

# Unicode
unicode: 你好世界 🌍 مرحبا
special: "@#$%^&*()"
`)
	
	yaml := NewYAML("")
	result, err := yaml.LoadBytes(content)
	if err != nil {
		t.Fatalf("LoadBytes() error = %v", err)
	}
	
	// Verify various types
	checks := map[string]string{
		"str1": "I'm a string.",
		"str2": "You can \"quote\" me.",
		"str5": "Plain string",
		"int1": "99",
		"int2": "42",
		"int3": "0",
		"int4": "-17",
		"float1": "1",
		"float2": "3.1415",
		"float3": "-0.01",
		"bool1": "true",
		"bool2": "false",
		"bool3": "yes",
		"bool4": "no",
		"bool5": "on",
		"bool6": "off",
		"null1": "",
		"null2": "",
		"integers.0": "1",
		"colors.1": "yellow",
		"inline.first": "Tom",
		"mapping.key": "value",
		"development.database": "dev_db",
		"development.adapter": "postgres",
		"development.host": "localhost",
		"production.database": "prod_db",
		"production.adapter": "postgres",
		"production.host": "prod.example.com",
		"deeply.nested.structure.with.many.levels": "value",
		"unicode": "你好世界 🌍 مرحبا",
		"special": "@#$%^&*()",
	}
	
	for key, expectedValue := range checks {
		if value, ok := result[key]; !ok {
			t.Errorf("Key %q not found in result", key)
		} else if value != expectedValue {
			t.Errorf("Key %q = %q, want %q", key, value, expectedValue)
		}
	}
}

// Helper type for testing reader errors
type yamlErrorReader struct {
	err error
}

func (r *yamlErrorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func BenchmarkYAML_LoadBytes_Small(b *testing.B) {
	content := []byte(`
section:
  key1: value1
  key2: 42
  key3: true
`)
	
	yaml := NewYAML("")
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, _ = yaml.LoadBytes(content)
	}
}

func BenchmarkYAML_LoadBytes_Large(b *testing.B) {
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&buf, "section%d:\n", i)
		for j := 0; j < 20; j++ {
			fmt.Fprintf(&buf, "  key%d: value_%d_%d\n", j, i, j)
		}
	}
	content := buf.Bytes()
	
	yaml := NewYAML("")
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, _ = yaml.LoadBytes(content)
	}
}

func BenchmarkYAML_LoadReader_WithPool(b *testing.B) {
	content := `
database:
  host: localhost
  port: 5432
server:
  host: 0.0.0.0
  port: 8080
`
	
	yaml := NewYAML("", WithPool(true))
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, _ = yaml.LoadReader(reader)
	}
}

func BenchmarkYAML_LoadReader_WithoutPool(b *testing.B) {
	content := `
database:
  host: localhost
  port: 5432
server:
  host: 0.0.0.0
  port: 8080
`
	
	yaml := NewYAML("", WithPool(false))
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, _ = yaml.LoadReader(reader)
	}
}