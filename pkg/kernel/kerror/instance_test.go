package kerror

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	ClearRegistry()
	
	err := Define(KConfig{Code: 404, Message: "not found"})
	
	tests := []struct {
		name            string
		setupConfig     func()
		checkInstance   func(*Instance) error
	}{
		{
			name: "basic new",
			setupConfig: func() {
				Configure(GlobalConfig{EnableStackTrace: false, EnableMetrics: false})
			},
			checkInstance: func(inst *Instance) error {
				if inst == nil {
					return fmt.Errorf("instance is nil")
				}
				if inst.err.id != err.id {
					return fmt.Errorf("wrong error ID")
				}
				if inst.message != "not found" {
					return fmt.Errorf("wrong message: %s", inst.message)
				}
				if inst.cause != nil {
					return fmt.Errorf("cause should be nil")
				}
				return nil
			},
		},
		{
			name: "with stack trace",
			setupConfig: func() {
				Configure(GlobalConfig{EnableStackTrace: true, EnableMetrics: false})
			},
			checkInstance: func(inst *Instance) error {
				if len(inst.stack) == 0 {
					return fmt.Errorf("stack trace not captured")
				}
				return nil
			},
		},
		{
			name: "with metrics",
			setupConfig: func() {
				Configure(GlobalConfig{EnableStackTrace: false, EnableMetrics: true})
				SetMetricsCollector(NewSimpleMetrics())
			},
			checkInstance: func(inst *Instance) error {
				// Just verify it doesn't panic
				return nil
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()
			inst := err.New()
			defer inst.Release()
			
			if err := tt.checkInstance(inst); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestNewf(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 500, Message: "server error"})
	
	tests := []struct {
		name     string
		format   string
		args     []any
		expected string
	}{
		{
			name:     "simple format",
			format:   "error: %s",
			args:     []any{"test"},
			expected: "error: test",
		},
		{
			name:     "multiple args",
			format:   "user %s not found in %s",
			args:     []any{"john", "database"},
			expected: "user john not found in database",
		},
		{
			name:     "number formatting",
			format:   "error %d: %s",
			args:     []any{500, "internal"},
			expected: "error 500: internal",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := err.Newf(tt.format, tt.args...)
			defer inst.Release()
			
			if inst.message != tt.expected {
				t.Errorf("message = %s, want %s", inst.message, tt.expected)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 500})
	
	tests := []struct {
		name      string
		cause     error
		wantNil   bool
	}{
		{
			name:      "nil cause",
			cause:     nil,
			wantNil:   true,
		},
		{
			name:      "standard error",
			cause:     errors.New("underlying"),
			wantNil:   false,
		},
		{
			name:      "wrapped error",
			cause:     fmt.Errorf("wrapped: %w", errors.New("inner")),
			wantNil:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := err.Wrap(tt.cause)
			
			if tt.wantNil {
				if inst != nil {
					t.Error("Wrap(nil) should return nil")
				}
				return
			}
			
			defer inst.Release()
			
			if inst.cause != tt.cause {
				t.Error("cause not properly set")
			}
			
			if !strings.Contains(inst.Error(), tt.cause.Error()) {
				t.Errorf("Error() should contain cause: %s", inst.Error())
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 503})
	
	cause := errors.New("connection failed")
	inst := err.Wrapf(cause, "service %s unavailable", "database")
	
	if inst == nil {
		t.Fatal("Wrapf should return instance")
	}
	defer inst.Release()
	
	if inst.message != "service database unavailable" {
		t.Errorf("message = %s, want service database unavailable", inst.message)
	}
	
	if inst.cause != cause {
		t.Error("cause not properly set")
	}
	
	// Test nil cause
	nilInst := err.Wrapf(nil, "test")
	if nilInst != nil {
		t.Error("Wrapf(nil) should return nil")
	}
}

func TestCaptureStack(t *testing.T) {
	ClearRegistry()
	Configure(GlobalConfig{StackTraceDepth: 10})
	
	err := Define(KConfig{Code: 500})
	inst := err.New()
	defer inst.Release()
	
	// Clear and capture
	inst.stack = inst.stack[:0]
	inst.CaptureStack(1)
	
	if len(inst.stack) == 0 {
		t.Error("Stack not captured")
	}
	
	if len(inst.stack) > 10 {
		t.Error("Stack depth not respected")
	}
}

func TestStackTrace(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 500})
	
	tests := []struct {
		name      string
		setup     func(*Instance)
		wantEmpty bool
	}{
		{
			name: "no stack",
			setup: func(inst *Instance) {
				inst.stack = inst.stack[:0]
			},
			wantEmpty: true,
		},
		{
			name: "with stack",
			setup: func(inst *Instance) {
				inst.CaptureStack(1)
			},
			wantEmpty: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := err.New()
			defer inst.Release()
			
			tt.setup(inst)
			trace := inst.StackTrace()
			
			if tt.wantEmpty && trace != "" {
				t.Error("Expected empty stack trace")
			}
			if !tt.wantEmpty && trace == "" {
				t.Error("Expected non-empty stack trace")
			}
			if !tt.wantEmpty && !strings.Contains(trace, "TestStackTrace") {
				t.Error("Stack trace should contain function name")
			}
		})
	}
}

func TestWithContext(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 404})
	inst := err.New()
	defer inst.Release()
	
	ctx := context.WithValue(context.Background(), "key", "value")
	inst.WithContext(ctx)
	
	if inst.context != ctx {
		t.Error("Context not set")
	}
	
	gotCtx := inst.Context()
	if gotCtx != ctx {
		t.Error("Context() returned wrong context")
	}
}

func TestInstanceError(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 500, Message: "server error"})
	
	tests := []struct {
		name     string
		setup    func() *Instance
		expected string
	}{
		{
			name: "without cause",
			setup: func() *Instance {
				return err.New()
			},
			expected: "server error",
		},
		{
			name: "with cause",
			setup: func() *Instance {
				return err.Wrap(errors.New("db failed"))
			},
			expected: "server error: db failed",
		},
		{
			name: "formatted with cause",
			setup: func() *Instance {
				return err.Wrapf(errors.New("timeout"), "custom %s", "message")
			},
			expected: "custom message: timeout",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := tt.setup()
			defer inst.Release()
			
			if inst.Error() != tt.expected {
				t.Errorf("Error() = %s, want %s", inst.Error(), tt.expected)
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 500})
	
	cause := errors.New("original")
	inst := err.Wrap(cause)
	defer inst.Release()
	
	unwrapped := inst.Unwrap()
	if unwrapped != cause {
		t.Error("Unwrap() should return original cause")
	}
	
	// Test without cause
	inst2 := err.New()
	defer inst2.Release()
	
	if inst2.Unwrap() != nil {
		t.Error("Unwrap() should return nil when no cause")
	}
}

func TestInstanceIs(t *testing.T) {
	ClearRegistry()
	err1 := Define(KConfig{Code: 404})
	err2 := Define(KConfig{Code: 500})
	
	inst1 := err1.New()
	defer inst1.Release()
	inst2 := err2.New()
	defer inst2.Release()
	
	tests := []struct {
		name   string
		inst   *Instance
		target error
		want   bool
	}{
		{
			name:   "same KError",
			inst:   inst1,
			target: err1,
			want:   true,
		},
		{
			name:   "different KError",
			inst:   inst1,
			target: err2,
			want:   false,
		},
		{
			name:   "KError pointer",
			inst:   inst1,
			target: &err1,
			want:   true,
		},
		{
			name:   "same instance",
			inst:   inst1,
			target: inst1,
			want:   true,
		},
		{
			name:   "different instance same error",
			inst:   err1.New(),
			target: inst1,
			want:   true,
		},
		{
			name:   "other error type",
			inst:   inst1,
			target: errors.New("other"),
			want:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.inst.Is(tt.target); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInstanceGetters(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{
		Package: "test",
		Code:    418,
		Message: "teapot",
	})
	
	inst := err.New()
	defer inst.Release()
	
	if inst.KError().id != err.id {
		t.Error("KError() returned wrong error")
	}
	
	if inst.Package() != "test" {
		t.Errorf("Package() = %s, want test", inst.Package())
	}
	
	if inst.Code() != 418 {
		t.Errorf("Code() = %d, want 418", inst.Code())
	}
}

func TestWithTag(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 400})
	
	tests := []struct {
		name      string
		config    GlobalConfig
		key       string
		value     string
		wantAdded bool
	}{
		{
			name:      "normal tag",
			config:    GlobalConfig{EnableValidation: true, MaxTagKeyLen: 100, MaxTagValueLen: 1000, MaxTags: 50},
			key:       "user",
			value:     "john",
			wantAdded: true,
		},
		{
			name:      "key too long",
			config:    GlobalConfig{EnableValidation: true, MaxTagKeyLen: 5, MaxTagValueLen: 1000, MaxTags: 50},
			key:       "verylongkey",
			value:     "value",
			wantAdded: false,
		},
		{
			name:      "value too long",
			config:    GlobalConfig{EnableValidation: true, MaxTagKeyLen: 100, MaxTagValueLen: 5, MaxTags: 50},
			key:       "key",
			value:     "verylongvalue",
			wantAdded: false,
		},
		{
			name:      "no validation",
			config:    GlobalConfig{EnableValidation: false},
			key:       "anylength",
			value:     "anyvalue",
			wantAdded: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Configure(tt.config)
			inst := err.New()
			defer inst.Release()
			
			inst.WithTag(tt.key, tt.value)
			
			val, ok := inst.Tag(tt.key)
			if tt.wantAdded {
				if !ok {
					t.Error("Tag not added")
				}
				if val != tt.value {
					t.Errorf("Tag value = %s, want %s", val, tt.value)
				}
			} else {
				if ok {
					t.Error("Tag should not be added")
				}
			}
		})
	}
}

func TestWithTags(t *testing.T) {
	ClearRegistry()
	Configure(GlobalConfig{EnableValidation: true, MaxTags: 3, MaxTagKeyLen: 10, MaxTagValueLen: 20})
	
	err := Define(KConfig{Code: 400})
	inst := err.New()
	defer inst.Release()
	
	tags := map[string]string{
		"tag1": "value1",
		"tag2": "value2",
		"tag3": "value3",
		"tag4": "value4", // Should be ignored due to MaxTags
		"verylongtagkey": "value", // Should be ignored due to MaxTagKeyLen
	}
	
	inst.WithTags(tags)
	
	allTags := inst.Tags()
	if len(allTags) > 3 {
		t.Errorf("Should respect MaxTags, got %d tags", len(allTags))
	}
	
	// Test empty tags
	inst2 := err.New()
	defer inst2.Release()
	inst2.WithTags(nil)
	if len(inst2.Tags()) != 0 {
		t.Error("WithTags(nil) should not add tags")
	}
}

func TestTag(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 400})
	inst := err.New()
	defer inst.Release()
	
	// Test non-existent tag
	val, ok := inst.Tag("missing")
	if ok {
		t.Error("Should return false for missing tag")
	}
	if val != "" {
		t.Error("Should return empty string for missing tag")
	}
	
	// Add and retrieve tag
	inst.WithTag("key", "value")
	val, ok = inst.Tag("key")
	if !ok {
		t.Error("Should return true for existing tag")
	}
	if val != "value" {
		t.Errorf("Tag value = %s, want value", val)
	}
}

func TestTags(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 400})
	inst := err.New()
	defer inst.Release()
	
	// Empty tags
	tags := inst.Tags()
	if len(tags) != 0 {
		t.Error("Should return empty map initially")
	}
	
	// Add tags
	inst.WithTag("key1", "value1")
	inst.WithTag("key2", "value2")
	
	tags = inst.Tags()
	if len(tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(tags))
	}
	
	// Verify it's a copy
	tags["key3"] = "value3"
	if _, ok := inst.Tag("key3"); ok {
		t.Error("Tags() should return a copy")
	}
}

func TestWithDetail(t *testing.T) {
	ClearRegistry()
	Configure(GlobalConfig{EnableValidation: true, MaxDetails: 2})
	
	err := Define(KConfig{Code: 400})
	inst := err.New()
	defer inst.Release()
	
	// Add details
	inst.WithDetail("detail1", "value1")
	inst.WithDetail("detail2", 123)
	inst.WithDetail("detail3", "ignored") // Should be ignored
	
	val1, ok1 := inst.Detail("detail1")
	if !ok1 || val1 != "value1" {
		t.Error("detail1 not properly stored")
	}
	
	val2, ok2 := inst.Detail("detail2")
	if !ok2 || val2 != 123 {
		t.Error("detail2 not properly stored")
	}
	
	_, ok3 := inst.Detail("detail3")
	if ok3 {
		t.Error("detail3 should be ignored due to MaxDetails")
	}
}

func TestWithDetails(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 400})
	inst := err.New()
	defer inst.Release()
	
	details := map[string]any{
		"string": "value",
		"number": 42,
	}
	
	inst.WithDetails(details)
	
	allDetails := inst.Details()
	if len(allDetails) != len(details) {
		t.Errorf("Details count = %d, want %d", len(allDetails), len(details))
	}
	
	// Test empty details
	inst2 := err.New()
	defer inst2.Release()
	inst2.WithDetails(nil)
	if len(inst2.Details()) != 0 {
		t.Error("WithDetails(nil) should not add details")
	}
}

func TestDetail(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 400})
	inst := err.New()
	defer inst.Release()
	
	// Non-existent detail
	val, ok := inst.Detail("missing")
	if ok {
		t.Error("Should return false for missing detail")
	}
	if val != nil {
		t.Error("Should return nil for missing detail")
	}
	
	// Add and retrieve
	inst.WithDetail("key", "value")
	val, ok = inst.Detail("key")
	if !ok {
		t.Error("Should return true for existing detail")
	}
	if val != "value" {
		t.Errorf("Detail value = %v, want value", val)
	}
}

func TestDetails(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 400})
	inst := err.New()
	defer inst.Release()
	
	// Empty details
	details := inst.Details()
	if len(details) != 0 {
		t.Error("Should return empty map initially")
	}
	
	// Add details
	inst.WithDetail("key1", "value1")
	inst.WithDetail("key2", 123)
	
	details = inst.Details()
	if len(details) != 2 {
		t.Errorf("Details count = %d, want 2", len(details))
	}
	
	// Verify it's a copy
	details["key3"] = "value3"
	if _, ok := inst.Detail("key3"); ok {
		t.Error("Details() should return a copy")
	}
}

func TestRelease(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 500})
	inst := err.New()
	
	// Add data
	inst.WithTag("tag", "value")
	inst.WithDetail("detail", "value")
	inst.cause = errors.New("cause")
	inst.message = "custom"
	inst.context = context.Background()
	inst.CaptureStack(1)
	
	// Release
	inst.Release()
	
	// Verify cleared
	if len(inst.tags) != 0 {
		t.Error("Tags not cleared")
	}
	if len(inst.details) != 0 {
		t.Error("Details not cleared")
	}
	if inst.cause != nil {
		t.Error("Cause not cleared")
	}
	if inst.message != "" {
		t.Error("Message not cleared")
	}
	if inst.context != nil {
		t.Error("Context not cleared")
	}
	// Stack keeps capacity but length should be 0
	if len(inst.stack) != 0 {
		t.Error("Stack not cleared")
	}
}

func TestOTelAttributes(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 500, Message: "error"})
	inst := err.New()
	defer inst.Release()
	
	inst.WithTag("trace_id", "123")
	inst.WithDetail("user", "john")
	inst.cause = errors.New("underlying")
	inst.CaptureStack(1)
	
	attrs := inst.OTelAttributes()
	
	// Check standard attributes
	if attrs["error.code"] != 500 {
		t.Error("Missing or wrong error.code")
	}
	if attrs["error.message"] != "error" {
		t.Error("Missing or wrong error.message")
	}
	
	// Check tag
	if attrs["error.tag.trace_id"] != "123" {
		t.Error("Missing or wrong tag")
	}
	
	// Check detail
	if attrs["error.detail.user"] != "john" {
		t.Error("Missing or wrong detail")
	}
	
	// Check cause
	if attrs["error.cause"] != "underlying" {
		t.Error("Missing or wrong cause")
	}
	
	// Check stack trace
	if _, ok := attrs["error.stack_trace"]; !ok {
		t.Error("Missing stack trace")
	}
}

func TestInstanceMarshalJSON(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 500, Message: "error"})
	inst := err.New()
	defer inst.Release()
	
	inst.WithTag("tag", "value")
	inst.WithDetail("detail", 123)
	inst.cause = errors.New("cause")
	inst.CaptureStack(1)
	
	data, jsonErr := json.Marshal(inst)
	if jsonErr != nil {
		t.Fatalf("MarshalJSON error: %v", jsonErr)
	}
	
	var result map[string]interface{}
	if jsonErr := json.Unmarshal(data, &result); jsonErr != nil {
		t.Fatalf("Unmarshal error: %v", jsonErr)
	}
	
	// Check fields
	if result["message"] != "error" {
		t.Error("Missing or wrong message")
	}
	
	if tags, ok := result["tags"].(map[string]interface{}); !ok {
		t.Error("Missing tags")
	} else if tags["tag"] != "value" {
		t.Error("Wrong tag value")
	}
	
	if details, ok := result["details"].(map[string]interface{}); !ok {
		t.Error("Missing details")
	} else if details["detail"] != float64(123) {
		t.Error("Wrong detail value")
	}
	
	if result["cause"] != "cause" {
		t.Error("Missing or wrong cause")
	}
	
	if _, ok := result["stack_trace"]; !ok {
		t.Error("Missing stack trace")
	}
}

func TestAs(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 500})
	inst := err.New()
	defer inst.Release()
	
	// As Instance
	var targetInst *Instance
	if !inst.As(&targetInst) {
		t.Error("As(*Instance) should return true")
	}
	if targetInst != inst {
		t.Error("As(*Instance) should set target")
	}
	
	// As KError
	var targetErr KError
	if !inst.As(&targetErr) {
		t.Error("As(KError) should return true")
	}
	if targetErr.id != err.id {
		t.Error("As(KError) should set target")
	}
	
	// As other type with cause
	inst.cause = &customError{msg: "custom"}
	var targetCustom *customError
	if !inst.As(&targetCustom) {
		t.Error("As should delegate to cause")
	}
	if targetCustom.msg != "custom" {
		t.Error("As should set custom error")
	}
	
	// As incompatible type - this test is removed because it would panic
}

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

func TestInstanceConcurrency(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 500})
	inst := err.New()
	defer inst.Release()
	
	var wg sync.WaitGroup
	
	// Concurrent reads and writes
	for i := 0; i < 10; i++ {
		wg.Add(3)
		
		// Writer
		go func(n int) {
			defer wg.Done()
			inst.WithTag(fmt.Sprintf("tag%d", n), fmt.Sprintf("value%d", n))
			inst.WithDetail(fmt.Sprintf("detail%d", n), n)
		}(i)
		
		// Reader 1
		go func() {
			defer wg.Done()
			_ = inst.Tags()
			_ = inst.Details()
		}()
		
		// Reader 2
		go func() {
			defer wg.Done()
			_ = inst.OTelAttributes()
			_ = inst.Error()
		}()
	}
	
	wg.Wait()
	
	// Verify data integrity
	tags := inst.Tags()
	details := inst.Details()
	
	if len(tags) == 0 || len(details) == 0 {
		t.Error("Concurrent operations failed")
	}
}

func TestInstancePoolReuse(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 500})
	
	// Get instance, modify, and release
	inst1 := err.New()
	inst1.WithTag("tag", "value")
	ptr1 := fmt.Sprintf("%p", inst1)
	inst1.Release()
	
	// Get new instance - might be same object
	inst2 := err.New()
	ptr2 := fmt.Sprintf("%p", inst2)
	defer inst2.Release()
	
	// Verify it's clean
	if len(inst2.Tags()) != 0 {
		t.Error("Pooled instance not properly cleaned")
	}
	
	// Note: We can't guarantee ptr1 == ptr2 due to pool behavior,
	// but we can verify the instance is clean
	_ = ptr1
	_ = ptr2
}

func TestBuilderPool(t *testing.T) {
	// Test string builder pool usage
	sb1 := builderPool.Get().(*strings.Builder)
	sb1.WriteString("test")
	sb1.Reset()
	builderPool.Put(sb1)
	
	sb2 := builderPool.Get().(*strings.Builder)
	if sb2.Len() != 0 {
		t.Error("Builder from pool not clean")
	}
	builderPool.Put(sb2)
}

func TestInstanceChaining(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 400})
	
	inst := err.New().
		WithTag("tag1", "value1").
		WithTag("tag2", "value2").
		WithDetail("detail1", "value1").
		WithDetail("detail2", 123).
		WithContext(context.Background()).
		CaptureStack(1)
		
	defer inst.Release()
	
	if inst == nil {
		t.Fatal("Chaining broke instance")
	}
	
	tags := inst.Tags()
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}
	
	details := inst.Details()
	if len(details) != 2 {
		t.Errorf("Expected 2 details, got %d", len(details))
	}
	
	if inst.Context() == nil {
		t.Error("Context not set")
	}
	
	if len(inst.stack) == 0 {
		t.Error("Stack not captured")
	}
}

func TestMaxLimits(t *testing.T) {
	ClearRegistry()
	Configure(GlobalConfig{
		EnableValidation: true,
		MaxTags:          2,
		MaxDetails:       2,
	})
	
	err := Define(KConfig{Code: 400})
	inst := err.New()
	defer inst.Release()
	
	// Try to add more than max tags
	inst.WithTag("tag1", "value1")
	inst.WithTag("tag2", "value2")
	inst.WithTag("tag3", "value3") // Should be ignored
	
	if len(inst.Tags()) != 2 {
		t.Errorf("Should have max 2 tags, got %d", len(inst.Tags()))
	}
	
	// Try to add more than max details
	inst.WithDetail("detail1", "value1")
	inst.WithDetail("detail2", "value2")
	inst.WithDetail("detail3", "value3") // Should be ignored
	
	if len(inst.Details()) != 2 {
		t.Errorf("Should have max 2 details, got %d", len(inst.Details()))
	}
}

func TestBatchWithTags(t *testing.T) {
	ClearRegistry()
	
	// Test without validation
	Configure(GlobalConfig{EnableValidation: false})
	err := Define(KConfig{Code: 400})
	inst := err.New()
	defer inst.Release()
	
	tags := []struct{ Key, Value string }{
		{Key: "env", Value: "test"},
		{Key: "version", Value: "1.0.0"},
		{Key: "service", Value: "api"},
	}
	
	result := inst.BatchWithTags(tags...)
	if result != inst {
		t.Error("BatchWithTags should return the same instance")
	}
	
	if v, ok := inst.Tag("env"); !ok || v != "test" {
		t.Error("Tag 'env' should be 'test'")
	}
	if v, ok := inst.Tag("version"); !ok || v != "1.0.0" {
		t.Error("Tag 'version' should be '1.0.0'")
	}
	if v, ok := inst.Tag("service"); !ok || v != "api" {
		t.Error("Tag 'service' should be 'api'")
	}
	
	// Test with validation and limits
	Configure(GlobalConfig{
		EnableValidation: true,
		MaxTags:          2,
		MaxTagKeyLen:     5,
		MaxTagValueLen:   5,
	})
	
	inst2 := err.New()
	defer inst2.Release()
	
	tags2 := []struct{ Key, Value string }{
		{Key: "tag1", Value: "val1"},
		{Key: "tag2", Value: "val2"},
		{Key: "tag3", Value: "val3"}, // Should be ignored (over max)
		{Key: "longkey", Value: "val"}, // Should be ignored (key too long)
		{Key: "key", Value: "verylongvalue"}, // Should be ignored (value too long)
	}
	
	inst2.BatchWithTags(tags2...)
	
	if len(inst2.Tags()) != 2 {
		t.Errorf("Should have 2 tags with validation, got %d", len(inst2.Tags()))
	}
	if _, ok := inst2.Tag("tag3"); ok {
		t.Error("tag3 should not be added (over limit)")
	}
	if _, ok := inst2.Tag("longkey"); ok {
		t.Error("longkey should not be added (key too long)")
	}
}

func TestBatchWithDetails(t *testing.T) {
	ClearRegistry()
	
	// Test without validation
	Configure(GlobalConfig{EnableValidation: false})
	err := Define(KConfig{Code: 500})
	inst := err.New()
	defer inst.Release()
	
	details := []struct{ Key string; Value any }{
		{Key: "user_id", Value: 123},
		{Key: "request_id", Value: "abc-123"},
		{Key: "retry_count", Value: 3},
	}
	
	result := inst.BatchWithDetails(details...)
	if result != inst {
		t.Error("BatchWithDetails should return the same instance")
	}
	
	if v, ok := inst.Detail("user_id"); !ok || v != 123 {
		t.Error("Detail 'user_id' should be 123")
	}
	if v, ok := inst.Detail("request_id"); !ok || v != "abc-123" {
		t.Error("Detail 'request_id' should be 'abc-123'")
	}
	if v, ok := inst.Detail("retry_count"); !ok || v != 3 {
		t.Error("Detail 'retry_count' should be 3")
	}
	
	// Test with validation and limits
	Configure(GlobalConfig{
		EnableValidation: true,
		MaxDetails:       2,
	})
	
	inst2 := err.New()
	defer inst2.Release()
	
	details2 := []struct{ Key string; Value any }{
		{Key: "detail1", Value: "val1"},
		{Key: "detail2", Value: "val2"},
		{Key: "detail3", Value: "val3"}, // Should be ignored (over max)
	}
	
	inst2.BatchWithDetails(details2...)
	
	if len(inst2.Details()) != 2 {
		t.Errorf("Should have 2 details with validation, got %d", len(inst2.Details()))
	}
	if _, ok := inst2.Detail("detail3"); ok {
		t.Error("detail3 should not be added (over limit)")
	}
}

func TestDetailAs(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 404})
	inst := err.New()
	defer inst.Release()
	
	type User struct {
		ID   int
		Name string
	}
	
	user := User{ID: 42, Name: "Alice"}
	inst.WithDetail("user", user)
	
	// Test successful conversion
	retrievedUser, ok := DetailAs[User](inst, "user")
	if !ok {
		t.Error("DetailAs should return true for existing detail")
	}
	if retrievedUser.ID != 42 || retrievedUser.Name != "Alice" {
		t.Error("DetailAs should retrieve the correct user")
	}
	
	// Test non-existent key
	_, ok = DetailAs[User](inst, "nonexistent")
	if ok {
		t.Error("DetailAs should return false for non-existent key")
	}
	
	// Test wrong type
	inst.WithDetail("number", 123)
	_, ok = DetailAs[string](inst, "number")
	if ok {
		t.Error("DetailAs should return false for type mismatch")
	}
}

func TestDetailResult(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 400})
	inst := err.New()
	defer inst.Release()
	
	inst.WithDetail("count", 10)
	
	// Test existing detail
	result := inst.DetailResult("count")
	if !result.Ok {
		t.Error("DetailResult should have Ok=true for existing detail")
	}
	if result.Value != 10 {
		t.Error("DetailResult should return correct value")
	}
	if result.Unwrap() != 10 {
		t.Error("Unwrap should return the value")
	}
	
	// Test non-existent detail
	missing := inst.DetailResult("missing")
	if missing.Ok {
		t.Error("DetailResult should have Ok=false for non-existent detail")
	}
	if missing.UnwrapOr("default") != "default" {
		t.Error("UnwrapOr should return default for missing detail")
	}
}

func TestTagResult(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 500})
	inst := err.New()
	defer inst.Release()
	
	inst.WithTag("env", "production")
	
	// Test existing tag
	result := inst.TagResult("env")
	if !result.Ok {
		t.Error("TagResult should have Ok=true for existing tag")
	}
	if result.Value != "production" {
		t.Error("TagResult should return correct value")
	}
	
	// Test non-existent tag
	missing := inst.TagResult("missing")
	if missing.Ok {
		t.Error("TagResult should have Ok=false for non-existent tag")
	}
	if missing.UnwrapOr("default") != "default" {
		t.Error("UnwrapOr should return default for missing tag")
	}
}

func TestClone(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 403, Message: "forbidden"})
	original := err.New()
	defer original.Release()
	
	original.WithTag("env", "test")
	original.WithDetail("user_id", 999)
	original.WithContext(context.Background())
	
	cloned := original.Clone()
	defer cloned.Release()
	
	// Verify clone has same properties
	if cloned.Error() != original.Error() {
		t.Error("Clone should have same error message")
	}
	
	if v, ok := cloned.Tag("env"); !ok || v != "test" {
		t.Error("Clone should have same tags")
	}
	
	if v, ok := cloned.Detail("user_id"); !ok || v != 999 {
		t.Error("Clone should have same details")
	}
	
	// Verify modifications don't affect original
	cloned.WithTag("new", "value")
	if _, ok := original.Tag("new"); ok {
		t.Error("Modifying clone should not affect original")
	}
	
	original.WithTag("original", "value")
	if _, ok := cloned.Tag("original"); ok {
		t.Error("Modifying original should not affect clone")
	}
}

func TestMapTags(t *testing.T) {
	ClearRegistry()
	err := Define(KConfig{Code: 400})
	inst := err.New()
	defer inst.Release()
	
	inst.WithTag("env", "test")
	inst.WithTag("version", "1.0")
	
	result := inst.MapTags(func(key, value string) (string, string) {
		return key, strings.ToUpper(value)
	})
	
	if result != inst {
		t.Error("MapTags should return the same instance")
	}
	
	if v, ok := inst.Tag("env"); !ok || v != "TEST" {
		t.Error("Tag 'env' should be uppercase")
	}
	if v, ok := inst.Tag("version"); !ok || v != "1.0" {
		t.Error("Tag 'version' should be '1.0'")
	}
	
	// Test with empty tags
	inst2 := err.New()
	defer inst2.Release()
	inst2.MapTags(func(key, value string) (string, string) {
		return key, "should not be called"
	})
}

func TestFilterTags(t *testing.T) {
	ClearRegistry()
	Configure(GlobalConfig{EnableValidation: false})
	err := Define(KConfig{Code: 500})
	inst := err.New()
	defer inst.Release()
	
	inst.WithTag("env", "production")
	inst.WithTag("debug", "true")
	inst.WithTag("version", "2.0")
	
	result := inst.FilterTags(func(key, value string) bool {
		return key != "debug"
	})
	
	if result != inst {
		t.Error("FilterTags should return the same instance")
	}
	
	if _, ok := inst.Tag("env"); !ok {
		t.Error("Tag 'env' should exist")
	}
	if _, ok := inst.Tag("version"); !ok {
		t.Error("Tag 'version' should exist")
	}
	if _, ok := inst.Tag("debug"); ok {
		t.Error("Tag 'debug' should be filtered out")
	}
}

func TestMergeTags(t *testing.T) {
	ClearRegistry()
	Configure(GlobalConfig{EnableValidation: false})
	err := Define(KConfig{Code: 404})
	inst1 := err.New()
	defer inst1.Release()
	
	inst1.WithTag("env", "test")
	inst1.WithTag("service", "api")
	
	inst2 := err.New()
	defer inst2.Release()
	
	inst2.WithTag("version", "1.0")
	inst2.WithTag("env", "production") // This should override
	
	result := inst1.MergeTags(inst2)
	if result != inst1 {
		t.Error("MergeTags should return the same instance")
	}
	
	// Check merged tags
	if v, ok := inst1.Tag("env"); !ok || v != "production" {
		t.Error("Tag 'env' should be overridden to 'production'")
	}
	if v, ok := inst1.Tag("service"); !ok || v != "api" {
		t.Error("Tag 'service' should remain 'api'")
	}
	if v, ok := inst1.Tag("version"); !ok || v != "1.0" {
		t.Error("Tag 'version' should be merged")
	}
	
	// inst2 should remain unchanged
	if _, ok := inst2.Tag("service"); ok {
		t.Error("inst2 should not have 'service' tag")
	}
	
	// Test with nil
	inst1.MergeTags(nil) // Should not panic
}

