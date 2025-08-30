package parser

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestENV_Type(t *testing.T) {
	env := NewENV("")
	if env.Type() != "env" {
		t.Errorf("Type() = %q, want %q", env.Type(), "env")
	}
}

func TestENV_Load_NoPrefix(t *testing.T) {
	// Set test environment variables
	testVars := map[string]string{
		"TEST_VAR_1": "value1",
		"TEST_VAR_2": "value2",
		"OTHER_VAR":  "other_value",
	}

	for k, v := range testVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	env := NewENV("")
	result, err := env.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should include all env vars
	if result["test.var.1"] != "value1" {
		t.Errorf("test.var.1 = %q, want %q", result["test.var.1"], "value1")
	}
	if result["test.var.2"] != "value2" {
		t.Errorf("test.var.2 = %q, want %q", result["test.var.2"], "value2")
	}
	if result["other.var"] != "other_value" {
		t.Errorf("other.var = %q, want %q", result["other.var"], "other_value")
	}
}

func TestENV_Load_WithPrefix(t *testing.T) {
	// Set test environment variables
	testVars := map[string]string{
		"APP_DATABASE_URL": "postgres://localhost",
		"APP_REDIS_HOST":   "127.0.0.1",
		"APP_DEBUG":        "true",
		"OTHER_VAR":        "should_not_appear",
		"ANOTHER":          "also_not",
	}

	for k, v := range testVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	env := NewENV("APP_")
	result, err := env.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should only include APP_ prefixed vars, with prefix stripped
	if result["database.url"] != "postgres://localhost" {
		t.Errorf("database.url = %q, want %q", result["database.url"], "postgres://localhost")
	}
	if result["redis.host"] != "127.0.0.1" {
		t.Errorf("redis.host = %q, want %q", result["redis.host"], "127.0.0.1")
	}
	if result["debug"] != "true" {
		t.Errorf("debug = %q, want %q", result["debug"], "true")
	}

	// Should not include non-prefixed vars
	if _, exists := result["other.var"]; exists {
		t.Error("other.var should not exist in result")
	}
	if _, exists := result["another"]; exists {
		t.Error("another should not exist in result")
	}
}

func TestENV_Load_PrefixWithUnderscore(t *testing.T) {
	// Test that underscore after prefix is stripped
	testVars := map[string]string{
		"CONFIG_SERVER_HOST": "localhost",
		"CONFIG_SERVER_PORT": "8080",
		"CONFIGVALUE":        "direct", // Without underscore
	}

	for k, v := range testVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	env := NewENV("CONFIG")
	result, err := env.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Underscore after prefix should be stripped
	if result["server.host"] != "localhost" {
		t.Errorf("server.host = %q, want %q", result["server.host"], "localhost")
	}
	if result["server.port"] != "8080" {
		t.Errorf("server.port = %q, want %q", result["server.port"], "8080")
	}
	if result["value"] != "direct" {
		t.Errorf("value = %q, want %q", result["value"], "direct")
	}
}

func TestENV_Load_InvalidEnvVar(t *testing.T) {
	// Test with invalid env var format (line 55 coverage)
	// Simulate by having an env var that starts with "="
	os.Setenv("=INVALID", "value")
	os.Setenv("VALID_VAR", "")
	defer os.Unsetenv("=INVALID")
	defer os.Unsetenv("VALID_VAR")

	env := NewENV("")
	result, err := env.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Empty value should still be included
	if val, exists := result["valid.var"]; !exists || val != "" {
		t.Errorf("valid.var = %q, exists = %v, want empty string", val, exists)
	}
	// Invalid var should be skipped (key would be empty after split)
	if _, exists := result[""]; exists {
		t.Error("Empty key should not exist in result")
	}
}

func TestENV_Load_Normalization(t *testing.T) {
	// Test key and value normalization
	testVars := map[string]string{
		"UPPER_CASE_KEY":        "  trimmed value  ",
		"Mixed_Case_Key":        "\"quoted\"",
		"already.lowercase":     "'single quotes'",
		"WITH_MANY_UNDERSCORES": "\t\twhitespace\t\t",
	}

	for k, v := range testVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	env := NewENV("")
	result, err := env.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check key normalization (uppercase to lowercase, underscore to dot)
	if result["upper.case.key"] != "trimmed value" {
		t.Errorf("upper.case.key = %q, want %q", result["upper.case.key"], "trimmed value")
	}
	if result["mixed.case.key"] != "quoted" {
		t.Errorf("mixed.case.key = %q, want %q", result["mixed.case.key"], "quoted")
	}
	if result["already.lowercase"] != "single quotes" {
		t.Errorf("already.lowercase = %q, want %q", result["already.lowercase"], "single quotes")
	}
	if result["with.many.underscores"] != "whitespace" {
		t.Errorf("with.many.underscores = %q, want %q", result["with.many.underscores"], "whitespace")
	}
}

func TestENV_LoadFiltered(t *testing.T) {
	// Set test environment variables
	testVars := map[string]string{
		"INCLUDE_THIS": "yes",
		"SKIP_THIS":    "no",
		"INCLUDE_ALSO": "yes2",
		"EXCLUDE_ME":   "nope",
		"=INVALID_KEY": "skip", // Invalid key for line 86 coverage
	}

	for k, v := range testVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	env := NewENV("")
	filter := func(envLine string) bool {
		// Now filter receives full "KEY=VALUE" line
		return strings.HasPrefix(envLine, "INCLUDE_THIS=") || strings.HasPrefix(envLine, "INCLUDE_ALSO=")
	}

	result, err := env.LoadFiltered(filter)
	if err != nil {
		t.Fatalf("LoadFiltered() error = %v", err)
	}

	// Should only include filtered vars
	if result["include.this"] != "yes" {
		t.Errorf("include.this = %q, want %q", result["include.this"], "yes")
	}
	if result["include.also"] != "yes2" {
		t.Errorf("include.also = %q, want %q", result["include.also"], "yes2")
	}

	// Should not include excluded vars
	if _, exists := result["skip.this"]; exists {
		t.Error("skip.this should not exist in result")
	}
	if _, exists := result["exclude.me"]; exists {
		t.Error("exclude.me should not exist in result")
	}
}

func TestENV_LoadFiltered_InvalidVar(t *testing.T) {
	// Test filter that rejects all
	env := NewENV("")
	filter := func(key string) bool {
		return false // Reject all
	}

	// Set some test vars
	os.Setenv("TEST_FILTER_VAR", "value")
	defer os.Unsetenv("TEST_FILTER_VAR")

	result, err := env.LoadFiltered(filter)
	if err != nil {
		t.Fatalf("LoadFiltered() error = %v", err)
	}

	// Result should be empty since filter rejects all
	if len(result) != 0 {
		t.Errorf("LoadFiltered() should return empty map when filter rejects all, got %d items", len(result))
	}
}

func TestENV_EmptyPrefix(t *testing.T) {
	// Test that empty prefix works correctly
	env1 := NewENV("")
	env2 := NewENV("NON_EXISTENT_PREFIX_")

	os.Setenv("TEST_EMPTY_PREFIX", "value")
	defer os.Unsetenv("TEST_EMPTY_PREFIX")

	result1, err := env1.Load()
	if err != nil {
		t.Fatalf("Load() with empty prefix error = %v", err)
	}

	result2, err := env2.Load()
	if err != nil {
		t.Fatalf("Load() with non-existent prefix error = %v", err)
	}

	// Empty prefix should include the var
	if result1["test.empty.prefix"] != "value" {
		t.Errorf("test.empty.prefix = %q, want %q", result1["test.empty.prefix"], "value")
	}

	// Non-existent prefix should not include it
	if _, exists := result2["test.empty.prefix"]; exists {
		t.Error("test.empty.prefix should not exist with non-matching prefix")
	}
}

func TestENV_HelperFunctions(t *testing.T) {
	e := NewENV("APP")

	t.Run("countMatchingVars", func(t *testing.T) {
		// Set up test environment
		os.Setenv("APP_KEY1", "value1")
		os.Setenv("APP_KEY2", "value2")
		os.Setenv("OTHER_KEY", "value3")
		os.Setenv("INVALID", "") // Will be skipped - no =
		defer os.Unsetenv("APP_KEY1")
		defer os.Unsetenv("APP_KEY2")
		defer os.Unsetenv("OTHER_KEY")
		defer os.Unsetenv("INVALID")

		envVars := os.Environ()
		count := e.countMatchingVars(envVars)
		// Should count APP_KEY1 and APP_KEY2
		if count < 2 {
			t.Errorf("countMatchingVars() = %d, want at least 2", count)
		}
	})

	t.Run("parseEnvVar", func(t *testing.T) {
		tests := []struct {
			name      string
			env       string
			prefix    string
			wantKey   string
			wantValue string
			wantOk    bool
		}{
			{
				name:      "valid with prefix match",
				env:       "APP_KEY=value",
				prefix:    "APP",
				wantKey:   "KEY",
				wantValue: "value",
				wantOk:    true,
			},
			{
				name:      "no equals sign",
				env:       "INVALID",
				prefix:    "",
				wantKey:   "",
				wantValue: "",
				wantOk:    false,
			},
			{
				name:      "prefix mismatch",
				env:       "OTHER_KEY=value",
				prefix:    "APP",
				wantKey:   "",
				wantValue: "",
				wantOk:    false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := NewENV(tt.prefix)
				key, value, ok := e.parseEnvVar(tt.env)
				if key != tt.wantKey || value != tt.wantValue || ok != tt.wantOk {
					t.Errorf("parseEnvVar(%q) = (%q, %q, %v), want (%q, %q, %v)",
						tt.env, key, value, ok, tt.wantKey, tt.wantValue, tt.wantOk)
				}
			})
		}
	})

	t.Run("processPrefix", func(t *testing.T) {
		// processPrefix assumes the prefix has already been removed,
		// so we test with keys that start after the prefix
		e := NewENV("APP")
		tests := []struct {
			key  string
			want string
		}{
			{"APP_KEY", "KEY"}, // Remove prefix and underscore
			{"APPKEY", "KEY"},  // Remove prefix, no underscore
			{"APP_", ""},       // Just prefix and underscore
			{"APP", ""},        // Just prefix
		}

		for _, tt := range tests {
			got := e.processPrefix(tt.key)
			if got != tt.want {
				t.Errorf("processPrefix(%q) = %q, want %q", tt.key, got, tt.want)
			}
		}
	})
}

func TestENV_LoadFiltered_EdgeCases(t *testing.T) {
	// Test LoadFiltered with various scenarios
	os.Setenv("FILTER_YES", "include")
	os.Setenv("FILTER_NO", "exclude")
	os.Setenv("INVALID", "") // No equals sign - should handle gracefully
	defer os.Unsetenv("FILTER_YES")
	defer os.Unsetenv("FILTER_NO")
	defer os.Unsetenv("INVALID")

	env := NewENV("")
	result, err := env.LoadFiltered(func(s string) bool {
		return strings.HasPrefix(s, "FILTER_YES")
	})

	if err != nil {
		t.Fatalf("LoadFiltered() error = %v", err)
	}

	if result["filter.yes"] != "include" {
		t.Errorf("filter.yes = %q, want %q", result["filter.yes"], "include")
	}

	if _, exists := result["filter.no"]; exists {
		t.Error("filter.no should not exist (filtered out)")
	}
}

func TestENV_PrefixEdgeCases(t *testing.T) {
	// Test edge cases with prefix matching
	testVars := map[string]string{
		"A":      "short",
		"AB":     "medium",
		"ABC":    "long",
		"ABCD":   "longer",
		"B_ITEM": "other",
	}

	for k, v := range testVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	// Test with "AB" prefix
	env := NewENV("AB")
	result, err := env.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should not include "A" (too short)
	if _, exists := result["a"]; exists {
		t.Error("'a' should not exist (doesn't match prefix)")
	}

	// Should include "AB" (exact match)
	if result[""] != "medium" {
		t.Errorf("empty key = %q, want %q", result[""], "medium")
	}

	// Should include "ABC" and "ABCD"
	if result["c"] != "long" {
		t.Errorf("c = %q, want %q", result["c"], "long")
	}
	if result["cd"] != "longer" {
		t.Errorf("cd = %q, want %q", result["cd"], "longer")
	}

	// Should not include "B_ITEM"
	if _, exists := result["b.item"]; exists {
		t.Error("b.item should not exist")
	}

	// Test with very long prefix that no var matches
	env2 := NewENV("VERY_LONG_PREFIX_THAT_NO_VAR_WILL_MATCH")
	result2, err := env2.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(result2) != 0 {
		t.Errorf("Expected empty result with non-matching prefix, got %d items", len(result2))
	}
}

func BenchmarkENV_Load_NoPrefix(b *testing.B) {
	// Setup environment
	for i := 0; i < 20; i++ {
		os.Setenv(fmt.Sprintf("BENCH_VAR_%d", i), fmt.Sprintf("value_%d", i))
		defer os.Unsetenv(fmt.Sprintf("BENCH_VAR_%d", i))
	}

	env := NewENV("")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = env.Load()
	}
}

// Tests for missing coverage in ENV parser - error paths and edge cases

func TestENV_CountMatchingVars_InvalidEnvVars(t *testing.T) {
	// Test countMatchingVars with invalid env var format (no equals sign)
	e := NewENV("TEST_")

	// Simulate environment variables with and without equals signs
	envVars := []string{
		"TEST_VALID_VAR=value",
		"INVALID_NO_EQUALS", // This should be skipped
		"TEST_ANOTHER=value2",
		"ALSO_INVALID_NO_EQUALS", // This should be skipped
		"OTHER_PREFIX=ignored",   // Doesn't match prefix
		"",                       // Empty string
	}

	count := e.countMatchingVars(envVars)

	// Should only count the 2 valid TEST_ prefixed vars with equals signs
	if count != 2 {
		t.Errorf("countMatchingVars() = %d, want %d", count, 2)
	}
}

func TestENV_CountMatchingVars_NoPrefix(t *testing.T) {
	// Test countMatchingVars with no prefix (should count all valid vars)
	e := NewENV("")

	envVars := []string{
		"VALID_VAR=value",
		"INVALID_NO_EQUALS",
		"ANOTHER_VALID=value2",
		"",
	}

	count := e.countMatchingVars(envVars)

	// Should count 2 valid vars (those with equals signs)
	if count != 2 {
		t.Errorf("countMatchingVars() = %d, want %d", count, 2)
	}
}

func TestENV_CountMatchingVars_EdgeCases(t *testing.T) {
	// Test various edge cases for countMatchingVars
	e := NewENV("APP_")

	testCases := []struct {
		name     string
		envVars  []string
		expected int
	}{
		{
			name:     "empty list",
			envVars:  []string{},
			expected: 0,
		},
		{
			name:     "all invalid",
			envVars:  []string{"INVALID", "ALSO_INVALID", ""},
			expected: 0,
		},
		{
			name:     "prefix at start of equals",
			envVars:  []string{"APP_=empty_value"},
			expected: 1,
		},
		{
			name:     "exact prefix match",
			envVars:  []string{"APP_KEY=value", "APPKEY=value", "APP=value"},
			expected: 1, // Only APP_KEY matches (APPKEY doesn't have underscore, APP doesn't have _)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			count := e.countMatchingVars(tc.envVars)
			if count != tc.expected {
				t.Errorf("countMatchingVars() = %d, want %d for case %s", count, tc.expected, tc.name)
			}
		})
	}
}

func TestENV_LoadFiltered_InvalidEnvFormat(t *testing.T) {
	// Test LoadFiltered with invalid env var format (missing equals)
	os.Setenv("VALID_VAR", "value")
	os.Setenv("=STARTS_WITH_EQUALS", "invalid")
	defer os.Unsetenv("VALID_VAR")
	defer os.Unsetenv("=STARTS_WITH_EQUALS")

	e := NewENV("")
	filter := func(envLine string) bool {
		// Accept all env vars
		return true
	}

	result, err := e.LoadFiltered(filter)
	if err != nil {
		t.Fatalf("LoadFiltered() error = %v", err)
	}

	// Should only include valid env vars
	if _, exists := result["valid.var"]; !exists {
		t.Error("valid.var should exist in result")
	}

	// Invalid env vars (those without proper equals) should be skipped
	if _, exists := result[""]; exists {
		t.Error("Empty key should not exist in result")
	}
}

func TestENV_LoadFiltered_FilterRejectsAll(t *testing.T) {
	// Test LoadFiltered when filter rejects everything
	os.Setenv("TEST_FILTER_VAR1", "value1")
	os.Setenv("TEST_FILTER_VAR2", "value2")
	defer os.Unsetenv("TEST_FILTER_VAR1")
	defer os.Unsetenv("TEST_FILTER_VAR2")

	e := NewENV("")
	filter := func(envLine string) bool {
		// Reject all
		return false
	}

	result, err := e.LoadFiltered(filter)
	if err != nil {
		t.Fatalf("LoadFiltered() error = %v", err)
	}

	if len(result) != 0 {
		t.Errorf("LoadFiltered() should return empty map when filter rejects all, got %d items", len(result))
	}
}

func TestENV_LoadFiltered_FilterAcceptsSpecific(t *testing.T) {
	// Test LoadFiltered with specific filter criteria
	os.Setenv("INCLUDE_ME", "include")
	os.Setenv("EXCLUDE_ME", "exclude")
	os.Setenv("INCLUDE_ALSO", "include_also")
	defer os.Unsetenv("INCLUDE_ME")
	defer os.Unsetenv("EXCLUDE_ME")
	defer os.Unsetenv("INCLUDE_ALSO")

	e := NewENV("")
	filter := func(envLine string) bool {
		// Include only vars that start with "INCLUDE_"
		return strings.HasPrefix(envLine, "INCLUDE_")
	}

	result, err := e.LoadFiltered(filter)
	if err != nil {
		t.Fatalf("LoadFiltered() error = %v", err)
	}

	// Should include the filtered vars
	if result["include.me"] != "include" {
		t.Errorf("include.me = %q, want %q", result["include.me"], "include")
	}
	if result["include.also"] != "include_also" {
		t.Errorf("include.also = %q, want %q", result["include.also"], "include_also")
	}

	// Should not include excluded vars
	if _, exists := result["exclude.me"]; exists {
		t.Error("exclude.me should not exist in result")
	}
}

func TestENV_ParseEnvVar_EdgeCases(t *testing.T) {
	// Test parseEnvVar edge cases more thoroughly
	testCases := []struct {
		name      string
		env       *ENV
		envVar    string
		wantKey   string
		wantValue string
		wantOk    bool
	}{
		{
			name:      "equals at start",
			env:       NewENV(""),
			envVar:    "=VALUE",
			wantKey:   "",
			wantValue: "",
			wantOk:    false,
		},
		{
			name:      "multiple equals signs",
			env:       NewENV(""),
			envVar:    "KEY=VALUE=MORE",
			wantKey:   "KEY",
			wantValue: "VALUE=MORE",
			wantOk:    true,
		},
		{
			name:      "empty key with prefix",
			env:       NewENV("APP_"),
			envVar:    "APP_=empty",
			wantKey:   "",
			wantValue: "empty",
			wantOk:    true,
		},
		{
			name:      "prefix longer than key",
			env:       NewENV("VERY_LONG_PREFIX"),
			envVar:    "SHORT=value",
			wantKey:   "",
			wantValue: "",
			wantOk:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key, value, ok := tc.env.parseEnvVar(tc.envVar)
			if key != tc.wantKey || value != tc.wantValue || ok != tc.wantOk {
				t.Errorf("parseEnvVar(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tc.envVar, key, value, ok, tc.wantKey, tc.wantValue, tc.wantOk)
			}
		})
	}
}

func TestENV_ProcessPrefix_EdgeCases(t *testing.T) {
	// Test processPrefix with various edge cases
	testCases := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "underscore after prefix",
			prefix:   "APP",
			key:      "APP_KEY",
			expected: "KEY",
		},
		{
			name:     "no underscore after prefix",
			prefix:   "APP",
			key:      "APPKEY",
			expected: "KEY",
		},
		{
			name:     "just prefix with underscore",
			prefix:   "APP",
			key:      "APP_",
			expected: "",
		},
		{
			name:     "just prefix without underscore",
			prefix:   "APP",
			key:      "APP",
			expected: "",
		},
		{
			name:     "multiple underscores",
			prefix:   "APP",
			key:      "APP___KEY",
			expected: "__KEY",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := NewENV(tc.prefix)
			result := e.processPrefix(tc.key)
			if result != tc.expected {
				t.Errorf("processPrefix(%q) = %q, want %q", tc.key, result, tc.expected)
			}
		})
	}
}

func BenchmarkENV_Load_WithPrefix(b *testing.B) {
	// Setup environment
	for i := 0; i < 20; i++ {
		os.Setenv(fmt.Sprintf("BENCH_VAR_%d", i), fmt.Sprintf("value_%d", i))
		defer os.Unsetenv(fmt.Sprintf("BENCH_VAR_%d", i))
	}

	env := NewENV("BENCH_")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = env.Load()
	}
}

// Concurrent Tests

func TestENV_Load_Concurrent(t *testing.T) {
	// Set up test environment variables
	testVars := map[string]string{
		"CONCURRENT_TEST_VAR_1": "value1",
		"CONCURRENT_TEST_VAR_2": "value2",
		"CONCURRENT_TEST_VAR_3": "value3",
		"OTHER_VAR":             "other_value",
	}

	for k, v := range testVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	env := NewENV("CONCURRENT_TEST_")

	// Test concurrent access to Load
	const numGoroutines = 100
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			result, err := env.Load()
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
			// Verify result consistency
			if result["var.1"] != "value1" {
				t.Errorf("Concurrent test %d: var.1 = %q, want %q", i, result["var.1"], "value1")
			}
			if result["var.2"] != "value2" {
				t.Errorf("Concurrent test %d: var.2 = %q, want %q", i, result["var.2"], "value2")
			}
			// Should not include OTHER_VAR (different prefix)
			if _, exists := result["other.var"]; exists {
				t.Errorf("Concurrent test %d: other.var should not exist", i)
			}
		case err := <-errors:
			t.Errorf("Concurrent test error: %v", err)
		}
	}
}

func TestENV_LoadFiltered_Concurrent(t *testing.T) {
	// Set up test environment variables
	testVars := map[string]string{
		"FILTER_INCLUDE_1": "include1",
		"FILTER_INCLUDE_2": "include2",
		"FILTER_EXCLUDE_1": "exclude1",
		"FILTER_EXCLUDE_2": "exclude2",
	}

	for k, v := range testVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	env := NewENV("")
	filter := func(envLine string) bool {
		return strings.HasPrefix(envLine, "FILTER_INCLUDE_")
	}

	const numGoroutines = 50
	results := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			result, err := env.LoadFiltered(filter)
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
			if result["filter.include.1"] != "include1" {
				t.Errorf("Concurrent filtered test %d: filter.include.1 = %q, want %q", i, result["filter.include.1"], "include1")
			}
			if _, exists := result["filter.exclude.1"]; exists {
				t.Errorf("Concurrent filtered test %d: filter.exclude.1 should not exist", i)
			}
		case err := <-errors:
			t.Errorf("Concurrent filtered test error: %v", err)
		}
	}
}

func TestENV_Different_Prefixes_Concurrent(t *testing.T) {
	// Set up test environment variables
	testVars := map[string]string{
		"APP_DATABASE_HOST": "localhost",
		"APP_DATABASE_PORT": "5432",
		"WEB_SERVER_HOST":   "0.0.0.0",
		"WEB_SERVER_PORT":   "8080",
		"API_TOKEN":         "secret",
	}

	for k, v := range testVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	// Test different parsers with different prefixes concurrently
	const numGoroutines = 30
	appResults := make(chan map[string]string, numGoroutines)
	webResults := make(chan map[string]string, numGoroutines)
	apiResults := make(chan map[string]string, numGoroutines)
	errors := make(chan error, numGoroutines*3)

	// APP_ prefix tests
	appEnv := NewENV("APP_")
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			result, err := appEnv.Load()
			if err != nil {
				errors <- err
				return
			}
			appResults <- result
		}(i)
	}

	// WEB_ prefix tests
	webEnv := NewENV("WEB_")
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			result, err := webEnv.Load()
			if err != nil {
				errors <- err
				return
			}
			webResults <- result
		}(i)
	}

	// API_ prefix tests
	apiEnv := NewENV("API_")
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			result, err := apiEnv.Load()
			if err != nil {
				errors <- err
				return
			}
			apiResults <- result
		}(i)
	}

	// Collect all results
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-appResults:
			if result["database.host"] != "localhost" {
				t.Errorf("APP concurrent test %d: database.host = %q, want %q", i, result["database.host"], "localhost")
			}
		case err := <-errors:
			t.Errorf("APP concurrent test error: %v", err)
		}
	}

	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-webResults:
			if result["server.host"] != "0.0.0.0" {
				t.Errorf("WEB concurrent test %d: server.host = %q, want %q", i, result["server.host"], "0.0.0.0")
			}
		case err := <-errors:
			t.Errorf("WEB concurrent test error: %v", err)
		}
	}

	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-apiResults:
			if result["token"] != "secret" {
				t.Errorf("API concurrent test %d: token = %q, want %q", i, result["token"], "secret")
			}
		case err := <-errors:
			t.Errorf("API concurrent test error: %v", err)
		}
	}
}

// Panic Recovery Tests

func TestENV_Load_PanicRecovery(t *testing.T) {
	// Test Load with various edge cases that could cause panics
	testCases := []struct {
		name    string
		prefix  string
		setup   func()
		cleanup func()
	}{
		{
			name:   "empty_prefix",
			prefix: "",
			setup: func() {
				os.Setenv("", "empty_key_value")
				os.Setenv("NORMAL_VAR", "normal_value")
			},
			cleanup: func() {
				os.Unsetenv("")
				os.Unsetenv("NORMAL_VAR")
			},
		},
		{
			name:   "very_long_prefix",
			prefix: strings.Repeat("PREFIX_", 100),
			setup: func() {
				os.Setenv(strings.Repeat("PREFIX_", 100)+"VAR", "value")
			},
			cleanup: func() {
				os.Unsetenv(strings.Repeat("PREFIX_", 100) + "VAR")
			},
		},
		{
			name:   "unicode_prefix",
			prefix: "测试_",
			setup: func() {
				os.Setenv("测试_变量", "unicode_value")
			},
			cleanup: func() {
				os.Unsetenv("测试_变量")
			},
		},
		{
			name:   "special_chars_prefix",
			prefix: "!@#$%_",
			setup: func() {
				os.Setenv("!@#$%_VAR", "special_value")
			},
			cleanup: func() {
				os.Unsetenv("!@#$%_VAR")
			},
		},
		{
			name:   "binary_data_value",
			prefix: "BIN_",
			setup: func() {
				os.Setenv("BIN_DATA", string([]byte{0, 1, 2, 3, 255}))
			},
			cleanup: func() {
				os.Unsetenv("BIN_DATA")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			defer tc.cleanup()

			env := NewENV(tc.prefix)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Load panicked with prefix %q: %v", tc.prefix, r)
				}
			}()

			_, _ = env.Load()
		})
	}
}

func TestENV_LoadFiltered_PanicRecovery(t *testing.T) {
	// Set up environment with potentially problematic variables
	testVars := map[string]string{
		"NORMAL_VAR":    "normal",
		"UNICODE_VAR":   "测试值",
		"BINARY_VAR":    string([]byte{0, 1, 2, 3, 255}),
		"VERY_LONG_VAR": strings.Repeat("value", 1000),
		"SPECIAL_CHARS": "!@#$%^&*()+={}[]|\\:;\"'<>?,./",
		"EMPTY_VAR":     "",
		"NULL_BYTES":    "value\x00with\x00nulls",
	}

	for k, v := range testVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	env := NewENV("")

	// Test various filter functions that could cause panics
	panicFilters := []struct {
		name   string
		filter func(string) bool
	}{
		{
			name: "nil_access_filter",
			filter: func(envLine string) bool {
				// Potentially problematic string operations
				return len(envLine) > 0 && envLine[0] != 0
			},
		},
		{
			name: "regex_like_filter",
			filter: func(envLine string) bool {
				// Complex string matching
				return strings.Contains(envLine, "=") &&
					len(strings.Split(envLine, "=")) == 2
			},
		},
		{
			name: "unicode_filter",
			filter: func(envLine string) bool {
				// Unicode operations
				return len([]rune(envLine)) > 5
			},
		},
		{
			name: "panic_inducing_filter",
			filter: func(envLine string) bool {
				// Potentially problematic operations
				if len(envLine) > 1000 {
					return envLine[:1000] == strings.Repeat("value", 200)
				}
				return true
			},
		},
	}

	for _, tc := range panicFilters {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("LoadFiltered panicked with %s filter: %v", tc.name, r)
				}
			}()

			_, _ = env.LoadFiltered(tc.filter)
		})
	}
}

// Multi-threaded Benchmarks

func BenchmarkENV_Load_NoPrefix_Concurrent(b *testing.B) {
	// Setup environment
	for i := 0; i < 20; i++ {
		os.Setenv(fmt.Sprintf("BENCH_CONCURRENT_VAR_%d", i), fmt.Sprintf("value_%d", i))
		defer os.Unsetenv(fmt.Sprintf("BENCH_CONCURRENT_VAR_%d", i))
	}

	env := NewENV("")
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = env.Load()
		}
	})
}

func BenchmarkENV_Load_WithPrefix_Concurrent(b *testing.B) {
	// Setup environment
	for i := 0; i < 20; i++ {
		os.Setenv(fmt.Sprintf("BENCH_PREFIX_VAR_%d", i), fmt.Sprintf("value_%d", i))
		defer os.Unsetenv(fmt.Sprintf("BENCH_PREFIX_VAR_%d", i))
	}

	env := NewENV("BENCH_PREFIX_")
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = env.Load()
		}
	})
}

func BenchmarkENV_LoadFiltered_Concurrent(b *testing.B) {
	// Setup environment
	for i := 0; i < 20; i++ {
		os.Setenv(fmt.Sprintf("BENCH_FILTER_INCLUDE_%d", i), fmt.Sprintf("include_%d", i))
		os.Setenv(fmt.Sprintf("BENCH_FILTER_EXCLUDE_%d", i), fmt.Sprintf("exclude_%d", i))
		defer os.Unsetenv(fmt.Sprintf("BENCH_FILTER_INCLUDE_%d", i))
		defer os.Unsetenv(fmt.Sprintf("BENCH_FILTER_EXCLUDE_%d", i))
	}

	env := NewENV("")
	filter := func(envLine string) bool {
		return strings.HasPrefix(envLine, "BENCH_FILTER_INCLUDE_")
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = env.LoadFiltered(filter)
		}
	})
}
