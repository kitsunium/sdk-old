package parser

import (
	"errors"
	"os"
	"testing"
)

func TestARGS_Type(t *testing.T) {
	args := NewARGS(false)
	if args.Type() != "args" {
		t.Errorf("Type() = %q, want %q", args.Type(), "args")
	}
}

func TestARGS_Load_SkipFirst(t *testing.T) {
	// Save and restore os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"program", "--key=value", "--flag"}
	
	args := NewARGS(true)
	result, err := args.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	
	// Should skip "program" and parse the rest
	if result["key"] != "value" {
		t.Errorf("key = %q, want %q", result["key"], "value")
	}
	if result["flag"] != "true" {
		t.Errorf("flag = %q, want %q", result["flag"], "true")
	}
}

func TestARGS_Load_NoSkip(t *testing.T) {
	// Save and restore os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"--first=value", "--second", "value2"}
	
	args := NewARGS(false)
	result, err := args.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	
	if result["first"] != "value" {
		t.Errorf("first = %q, want %q", result["first"], "value")
	}
	if result["second"] != "value2" {
		t.Errorf("second = %q, want %q", result["second"], "value2")
	}
}

func TestARGS_Load_EmptyArgs(t *testing.T) {
	// Save and restore os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	// Test with program name only
	os.Args = []string{"program"}
	
	args := NewARGS(true)
	result, err := args.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
	
	// Test with empty Args
	os.Args = []string{}
	args2 := NewARGS(false)
	result2, err := args2.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	
	if len(result2) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result2))
	}
}

func TestARGS_ParseArgs_Basic(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected map[string]string
	}{
		{
			name: "equals format",
			args: []string{"--key=value", "--another=test"},
			expected: map[string]string{
				"key":     "value",
				"another": "test",
			},
		},
		{
			name: "space format",
			args: []string{"--key", "value", "--another", "test"},
			expected: map[string]string{
				"key":     "value",
				"another": "test",
			},
		},
		{
			name: "boolean flags",
			args: []string{"--verbose", "--debug", "--quiet"},
			expected: map[string]string{
				"verbose": "true",
				"debug":   "true",
				"quiet":   "true",
			},
		},
		{
			name: "single dash",
			args: []string{"-v", "-d", "-q"},
			expected: map[string]string{
				"v": "true",
				"d": "true",
				"q": "true",
			},
		},
		{
			name: "mixed formats",
			args: []string{"--key=value", "-v", "--flag", "arg", "-x"},
			expected: map[string]string{
				"key":  "value",
				"v":    "true",
				"flag": "arg",
				"x":    "true",
			},
		},
	}
	
	parser := NewARGS(false)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parser.ParseArgs(tc.args)
			if err != nil {
				t.Fatalf("ParseArgs() error = %v", err)
			}
			
			if len(result) != len(tc.expected) {
				t.Errorf("Result has %d items, want %d", len(result), len(tc.expected))
			}
			
			for key, expectedValue := range tc.expected {
				if value, ok := result[key]; !ok || value != expectedValue {
					t.Errorf("key %q = %q, want %q", key, value, expectedValue)
				}
			}
		})
	}
}

func TestARGS_ParseArgs_Normalization(t *testing.T) {
	args := []string{
		"--DATABASE_URL=postgres://localhost",
		"--Redis_Host", "127.0.0.1",
		"--UPPER_CASE_KEY=value",
		"--Mixed_Case", "  trimmed  ",
		"--quoted=\"quoted value\"",
	}
	
	parser := NewARGS(false)
	result, err := parser.ParseArgs(args)
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	
	// Check key normalization
	if result["database.url"] != "postgres://localhost" {
		t.Errorf("database.url = %q, want %q", result["database.url"], "postgres://localhost")
	}
	if result["redis.host"] != "127.0.0.1" {
		t.Errorf("redis.host = %q, want %q", result["redis.host"], "127.0.0.1")
	}
	if result["upper.case.key"] != "value" {
		t.Errorf("upper.case.key = %q, want %q", result["upper.case.key"], "value")
	}
	
	// Check value normalization
	if result["mixed.case"] != "trimmed" {
		t.Errorf("mixed.case = %q, want %q", result["mixed.case"], "trimmed")
	}
	if result["quoted"] != "quoted value" {
		t.Errorf("quoted = %q, want %q", result["quoted"], "quoted value")
	}
}

func TestARGS_ParseArgs_EdgeCases(t *testing.T) {
	parser := NewARGS(false)
	
	// Empty args
	result, err := parser.ParseArgs([]string{})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty args")
	}
	
	// Args with empty strings
	result, err = parser.ParseArgs([]string{"", "--key=value", ""})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("key = %q, want %q", result["key"], "value")
	}
	
	// Non-flag args (should be skipped)
	result, err = parser.ParseArgs([]string{"not-a-flag", "another", "--real-flag=yes"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if len(result) != 1 || result["real-flag"] != "yes" {
		t.Errorf("Should only parse real flags")
	}
	
	// Empty flag (just dashes)
	result, err = parser.ParseArgs([]string{"--", "---", "--real=value"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if result["real"] != "value" {
		t.Errorf("real = %q, want %q", result["real"], "value")
	}
	
	// Flag with empty value
	result, err = parser.ParseArgs([]string{"--key=", "--another="})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if result["key"] != "" {
		t.Errorf("key = %q, want empty string", result["key"])
	}
	if result["another"] != "" {
		t.Errorf("another = %q, want empty string", result["another"])
	}
}

func TestARGS_ParseArgs_ComplexScenarios(t *testing.T) {
	parser := NewARGS(false)
	
	// Complex real-world scenario
	args := []string{
		"--config-file=/etc/app/config.yaml",
		"--log-level", "debug",
		"-v",
		"--enable-feature-a",
		"--disable-feature-b",
		"--port=8080",
		"--host", "0.0.0.0",
		"--ssl", "non-flag-value", // This will be taken as value for --ssl
		"--database-url=postgres://user:pass@localhost:5432/db?sslmode=disable",
	}
	
	result, err := parser.ParseArgs(args)
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	
	expected := map[string]string{
		"config-file":       "/etc/app/config.yaml",
		"log-level":         "debug",
		"v":                 "true",
		"enable-feature-a":  "true",
		"disable-feature-b": "true",
		"port":              "8080",
		"host":              "0.0.0.0",
		"ssl":               "non-flag-value",
		"database-url":      "postgres://user:pass@localhost:5432/db?sslmode=disable",
	}
	
	for key, expectedValue := range expected {
		if value, ok := result[key]; !ok || value != expectedValue {
			t.Errorf("key %q = %q, want %q", key, value, expectedValue)
		}
	}
}

func TestARGS_ParseArgsStrict(t *testing.T) {
	parser := NewARGS(false)
	
	// Valid strict args
	validArgs := []string{"--key=value", "-f", "file.txt", "--verbose"}
	result, err := parser.ParseArgsStrict(validArgs)
	if err != nil {
		t.Fatalf("ParseArgsStrict() with valid args error = %v", err)
	}
	
	if result["key"] != "value" {
		t.Errorf("key = %q, want %q", result["key"], "value")
	}
	if result["f"] != "file.txt" {
		t.Errorf("f = %q, want %q", result["f"], "file.txt")
	}
	if result["verbose"] != "true" {
		t.Errorf("verbose = %q, want %q", result["verbose"], "true")
	}
	
	// Invalid strict args (non-flag argument)
	invalidArgs := []string{"not-a-flag", "--valid-flag"}
	_, err = parser.ParseArgsStrict(invalidArgs)
	if err == nil {
		t.Error("ParseArgsStrict() should error on non-flag arguments")
	}
	if !errors.Is(err, ErrARGSInvalid) {
		t.Errorf("ParseArgsStrict() expected ErrARGSInvalid, got: %v", err)
	}
}

func TestARGS_ParseArgsStrict_EdgeCases(t *testing.T) {
	parser := NewARGS(false)
	
	// Empty args
	result, err := parser.ParseArgsStrict([]string{})
	if err != nil {
		t.Fatalf("ParseArgsStrict() error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty args")
	}
	
	// Args with empty strings (should be skipped)
	result, err = parser.ParseArgsStrict([]string{"", "--key=value", ""})
	if err != nil {
		t.Fatalf("ParseArgsStrict() error = %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("key = %q, want %q", result["key"], "value")
	}
	
	// Just dashes (should handle gracefully)
	result, err = parser.ParseArgsStrict([]string{"--", "--key=value"})
	if err != nil {
		t.Fatalf("ParseArgsStrict() error = %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("key = %q, want %q", result["key"], "value")
	}
	
	// Flag at end without value
	result, err = parser.ParseArgsStrict([]string{"--flag1", "--flag2"})
	if err != nil {
		t.Fatalf("ParseArgsStrict() error = %v", err)
	}
	if result["flag1"] != "true" {
		t.Errorf("flag1 = %q, want %q", result["flag1"], "true")
	}
	if result["flag2"] != "true" {
		t.Errorf("flag2 = %q, want %q", result["flag2"], "true")
	}
}

func BenchmarkARGS_ParseArgs(b *testing.B) {
	args := []string{
		"--database-url=postgres://localhost:5432/test",
		"--redis-host=localhost",
		"--redis-port=6379",
		"--app-name=test-app",
		"--log-level", "debug",
		"--max-connections", "100",
		"-v",
		"--enable-feature",
	}
	
	parser := NewARGS(false)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseArgs(args)
	}
}

func BenchmarkARGS_ParseArgsStrict(b *testing.B) {
	args := []string{
		"--database-url=postgres://localhost:5432/test",
		"--redis-host=localhost",
		"--redis-port=6379",
		"--app-name=test-app",
		"--log-level", "debug",
		"--max-connections", "100",
		"-v",
		"--enable-feature",
	}
	
	parser := NewARGS(false)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseArgsStrict(args)
	}
}