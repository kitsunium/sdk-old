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
