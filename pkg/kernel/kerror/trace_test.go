package kerror

import (
	"context"
	"testing"
)

// Test WithContext with actual trace and span IDs
func TestWithContextWithTraceAndSpanIDs(t *testing.T) {
	ClearRegistry()
	
	// Save original functions
	origExtractTrace := extractTraceIDFunc
	origExtractSpan := extractSpanIDFunc
	
	// Set mock functions
	extractTraceIDFunc = func(ctx context.Context) string {
		return "trace-123"
	}
	extractSpanIDFunc = func(ctx context.Context) string {
		return "span-456"
	}
	
	// Restore after test
	defer func() {
		extractTraceIDFunc = origExtractTrace
		extractSpanIDFunc = origExtractSpan
	}()
	
	// Create error and instance
	err := Define(KConfig{Code: 500})
	inst := err.New()
	defer inst.Release()
	
	// Add context
	ctx := context.Background()
	inst.WithContext(ctx)
	
	// Verify tags were added
	if trace, ok := inst.Tag("trace_id"); !ok || trace != "trace-123" {
		t.Errorf("trace_id not set correctly: got %v, ok=%v", trace, ok)
	}
	
	if span, ok := inst.Tag("span_id"); !ok || span != "span-456" {
		t.Errorf("span_id not set correctly: got %v, ok=%v", span, ok)
	}
	
	// Verify context is set
	if inst.Context() != ctx {
		t.Error("Context not set")
	}
}

// Test WithContext with only trace ID
func TestWithContextOnlyTraceID(t *testing.T) {
	ClearRegistry()
	
	// Save original functions
	origExtractTrace := extractTraceIDFunc
	origExtractSpan := extractSpanIDFunc
	
	// Set mock functions
	extractTraceIDFunc = func(ctx context.Context) string {
		return "trace-only"
	}
	extractSpanIDFunc = func(ctx context.Context) string {
		return "" // No span ID
	}
	
	// Restore after test
	defer func() {
		extractTraceIDFunc = origExtractTrace
		extractSpanIDFunc = origExtractSpan
	}()
	
	// Create error and instance
	err := Define(KConfig{Code: 404})
	inst := err.New()
	defer inst.Release()
	
	// Add context
	ctx := context.Background()
	inst.WithContext(ctx)
	
	// Verify only trace_id tag was added
	if trace, ok := inst.Tag("trace_id"); !ok || trace != "trace-only" {
		t.Errorf("trace_id not set correctly: got %v, ok=%v", trace, ok)
	}
	
	if _, ok := inst.Tag("span_id"); ok {
		t.Error("span_id should not be set")
	}
}

// Test WithContext with only span ID
func TestWithContextOnlySpanID(t *testing.T) {
	ClearRegistry()
	
	// Save original functions
	origExtractTrace := extractTraceIDFunc
	origExtractSpan := extractSpanIDFunc
	
	// Set mock functions
	extractTraceIDFunc = func(ctx context.Context) string {
		return "" // No trace ID
	}
	extractSpanIDFunc = func(ctx context.Context) string {
		return "span-only"
	}
	
	// Restore after test
	defer func() {
		extractTraceIDFunc = origExtractTrace
		extractSpanIDFunc = origExtractSpan
	}()
	
	// Create error and instance
	err := Define(KConfig{Code: 403})
	inst := err.New()
	defer inst.Release()
	
	// Add context
	ctx := context.Background()
	inst.WithContext(ctx)
	
	// Verify only span_id tag was added
	if _, ok := inst.Tag("trace_id"); ok {
		t.Error("trace_id should not be set")
	}
	
	if span, ok := inst.Tag("span_id"); !ok || span != "span-only" {
		t.Errorf("span_id not set correctly: got %v, ok=%v", span, ok)
	}
}

// Test default extract functions
func TestDefaultExtractFunctions(t *testing.T) {
	ctx := context.Background()
	
	// Test default implementations return empty strings
	if id := defaultExtractTraceID(ctx); id != "" {
		t.Errorf("defaultExtractTraceID should return empty string, got %s", id)
	}
	
	if id := defaultExtractSpanID(ctx); id != "" {
		t.Errorf("defaultExtractSpanID should return empty string, got %s", id)
	}
	
	// Test with TODO context
	todoCtx := context.TODO()
	if id := defaultExtractTraceID(todoCtx); id != "" {
		t.Errorf("defaultExtractTraceID with TODO should return empty string, got %s", id)
	}
	
	if id := defaultExtractSpanID(todoCtx); id != "" {
		t.Errorf("defaultExtractSpanID with TODO should return empty string, got %s", id)
	}
}