package kerror

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestDefine(t *testing.T) {
	// Clear registry for clean test
	ClearRegistry()
	
	tests := []struct {
		name      string
		config    KConfig
		wantPanic bool
	}{
		{
			name: "basic definition",
			config: KConfig{
				Code:    404,
				Message: "not found",
			},
		},
		{
			name: "with package",
			config: KConfig{
				Package: "testpkg",
				Code:    500,
				Message: "server error",
			},
		},
		{
			name: "auto message from HTTP status",
			config: KConfig{
				Code: 401,
			},
		},
		{
			name: "custom error code",
			config: KConfig{
				Code:    999,
				Message: "custom error",
			},
		},
		{
			name: "duplicate code in same package",
			config: KConfig{
				Package: "dup",
				Code:    100,
			},
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Define() should panic for duplicate code")
					}
				}()
			}
			
			err := Define(tt.config)
			
			if err.Code() != tt.config.Code {
				t.Errorf("Code() = %d, want %d", err.Code(), tt.config.Code)
			}
			
			if tt.config.Package != "" && err.Package() != tt.config.Package {
				t.Errorf("Package() = %s, want %s", err.Package(), tt.config.Package)
			}
			
			if tt.config.Message != "" && err.Message() != tt.config.Message {
				t.Errorf("Message() = %s, want %s", err.Message(), tt.config.Message)
			}
			
			if err.ID() == 0 {
				t.Error("ID() should not be 0")
			}
		})
	}
}

func TestDefineDuplicate(t *testing.T) {
	ClearRegistry()
	
	// Define first error
	_ = Define(KConfig{
		Package: "testdup",
		Code:    200,
		Message: "first",
	})
	
	// Try to define duplicate - should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Should panic on duplicate error code")
		} else {
			msg := fmt.Sprint(r)
			if !strings.Contains(msg, "duplicate error code") {
				t.Errorf("Panic message should mention duplicate, got: %s", msg)
			}
		}
	}()
	
	_ = Define(KConfig{
		Package: "testdup",
		Code:    200,
		Message: "second",
	})
}

func TestGetCallerPackage(t *testing.T) {
	// Test package detection
	pkg := getCallerPackage()
	if pkg == "" {
		t.Error("getCallerPackage() returned empty string")
	}
	
	// Test caching
	pkg2 := getCallerPackage()
	if pkg != pkg2 {
		t.Error("getCallerPackage() should return consistent results")
	}
}

func TestGetCallerPackageEdgeCases(t *testing.T) {
	// Test with different call depths
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pkg := getCallerPackage()
			if pkg == "" {
				t.Error("getCallerPackage() failed in goroutine")
			}
		}()
	}
	wg.Wait()
}

func TestKErrorMethods(t *testing.T) {
	ClearRegistry()
	
	err := Define(KConfig{
		Package: "methods",
		Code:    418,
		Message: "I'm a teapot",
	})
	
	// Test all getter methods
	if err.ID() == 0 {
		t.Error("ID() should not be 0")
	}
	
	if err.Package() != "methods" {
		t.Errorf("Package() = %s, want methods", err.Package())
	}
	
	if err.Code() != 418 {
		t.Errorf("Code() = %d, want 418", err.Code())
	}
	
	if err.Message() != "I'm a teapot" {
		t.Errorf("Message() = %s, want I'm a teapot", err.Message())
	}
	
	if err.Error() != "I'm a teapot" {
		t.Errorf("Error() = %s, want I'm a teapot", err.Error())
	}
}

func TestKErrorIs(t *testing.T) {
	ClearRegistry()
	
	err1 := Define(KConfig{Code: 404})
	err2 := Define(KConfig{Code: 500})
	
	tests := []struct {
		name   string
		err    KError
		target error
		want   bool
	}{
		{
			name:   "same KError",
			err:    err1,
			target: err1,
			want:   true,
		},
		{
			name:   "different KError",
			err:    err1,
			target: err2,
			want:   false,
		},
		{
			name:   "KError pointer same",
			err:    err1,
			target: &err1,
			want:   true,
		},
		{
			name:   "KError pointer different",
			err:    err1,
			target: &err2,
			want:   false,
		},
		{
			name:   "nil pointer",
			err:    err1,
			target: (*KError)(nil),
			want:   false,
		},
		{
			name:   "different type",
			err:    err1,
			target: errors.New("other"),
			want:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Is(tt.target); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKErrorMarshalJSON(t *testing.T) {
	ClearRegistry()
	
	err := Define(KConfig{
		Package: "json",
		Code:    400,
		Message: "bad request",
	})
	
	data, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		t.Fatalf("MarshalJSON() error = %v", jsonErr)
	}
	
	var result map[string]interface{}
	if jsonErr := json.Unmarshal(data, &result); jsonErr != nil {
		t.Fatalf("Failed to unmarshal: %v", jsonErr)
	}
	
	if result["package"] != "json" {
		t.Errorf("JSON package = %v, want json", result["package"])
	}
	
	if result["code"] != float64(400) {
		t.Errorf("JSON code = %v, want 400", result["code"])
	}
	
	if result["message"] != "bad request" {
		t.Errorf("JSON message = %v, want bad request", result["message"])
	}
	
	if _, ok := result["id"]; !ok {
		t.Error("JSON missing id field")
	}
}

func TestKErrorUnmarshalJSON(t *testing.T) {
	jsonStr := `{"id":123,"package":"test","code":500,"message":"error"}`
	
	var err KError
	if jsonErr := json.Unmarshal([]byte(jsonStr), &err); jsonErr != nil {
		t.Fatalf("UnmarshalJSON() error = %v", jsonErr)
	}
	
	if err.ID() != 123 {
		t.Errorf("ID() = %d, want 123", err.ID())
	}
	
	if err.Package() != "test" {
		t.Errorf("Package() = %s, want test", err.Package())
	}
	
	if err.Code() != 500 {
		t.Errorf("Code() = %d, want 500", err.Code())
	}
	
	if err.Message() != "error" {
		t.Errorf("Message() = %s, want error", err.Message())
	}
}

func TestKErrorUnmarshalJSONInvalid(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantErr: true,
		},
		{
			name:    "missing fields",
			json:    `{}`,
			wantErr: false,
		},
		{
			name:    "wrong types",
			json:    `{"id":"string","code":"notanumber"}`,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err KError
			jsonErr := json.Unmarshal([]byte(tt.json), &err)
			if (jsonErr != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", jsonErr, tt.wantErr)
			}
		})
	}
}

func TestKErrorString(t *testing.T) {
	ClearRegistry()
	
	err := Define(KConfig{
		Package: "str",
		Code:    403,
		Message: "forbidden",
	})
	
	str := err.String()
	if !strings.Contains(str, "str") {
		t.Errorf("String() should contain package, got: %s", str)
	}
	if !strings.Contains(str, "403") {
		t.Errorf("String() should contain code, got: %s", str)
	}
	if !strings.Contains(str, "forbidden") {
		t.Errorf("String() should contain message, got: %s", str)
	}
	if !strings.Contains(str, fmt.Sprintf("%d", err.ID())) {
		t.Errorf("String() should contain ID, got: %s", str)
	}
}

func TestHTTPStatusTextCaching(t *testing.T) {
	ClearRegistry()
	
	// Clear cache
	httpStatusTextCache = sync.Map{}
	
	// Test standard HTTP status
	err1 := Define(KConfig{Code: 200})
	if err1.Message() != http.StatusText(200) {
		t.Errorf("Message should be HTTP status text, got: %s", err1.Message())
	}
	
	// Second call should use cache
	err2 := Define(KConfig{Code: 201})
	if err2.Message() != http.StatusText(201) {
		t.Errorf("Message should be HTTP status text, got: %s", err2.Message())
	}
	
	// Test non-standard code
	err3 := Define(KConfig{Code: 999})
	if !strings.Contains(err3.Message(), "999") {
		t.Errorf("Message for unknown code should contain code number, got: %s", err3.Message())
	}
}

func TestDefineWithMetrics(t *testing.T) {
	ClearRegistry()
	
	// Enable metrics
	Configure(GlobalConfig{
		EnableMetrics:  true,
		DefaultPackage: "metrics",
	})
	
	// Set up metrics collector
	metrics := NewSimpleMetrics()
	SetMetricsCollector(metrics)
	
	// Define error
	_ = Define(KConfig{
		Code:    500,
		Message: "server error",
	})
	
	// Check metrics were recorded
	snapshot := metrics.GetMetrics()
	if snapshot == nil {
		t.Fatal("Metrics should not be nil")
	}
	
	// Disable metrics
	Configure(GlobalConfig{
		EnableMetrics: false,
	})
}

func TestErrorCounterIncrement(t *testing.T) {
	ClearRegistry()
	
	// Get initial counter value
	initial := atomic.LoadUint32(&errorCounter)
	
	// Define multiple errors
	count := 5
	for i := 0; i < count; i++ {
		_ = Define(KConfig{Code: 100 + i})
	}
	
	// Check counter increased
	final := atomic.LoadUint32(&errorCounter)
	if final != initial+uint32(count) {
		t.Errorf("Counter should increase by %d, got %d", count, final-initial)
	}
}

func TestDefineConcurrency(t *testing.T) {
	ClearRegistry()
	
	var wg sync.WaitGroup
	errors := make([]KError, 100)
	
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errors[idx] = Define(KConfig{
				Code:    100 + idx,
				Message: fmt.Sprintf("error %d", idx),
			})
		}(i)
	}
	
	wg.Wait()
	
	// Verify all errors have unique IDs
	seen := make(map[uint32]bool)
	for _, err := range errors {
		if seen[err.ID()] {
			t.Errorf("Duplicate ID found: %d", err.ID())
		}
		seen[err.ID()] = true
	}
}

func TestCallerPackageCaching(t *testing.T) {
	// Clear cache
	callerPackageCache = sync.Map{}
	
	// First call
	pkg1 := getCallerPackage()
	
	// Second call should use cache
	pkg2 := getCallerPackage()
	
	if pkg1 != pkg2 {
		t.Error("Package name should be cached")
	}
	
	// Verify cache has entry
	pc, _, _, _ := runtime.Caller(1)
	if cached, ok := callerPackageCache.Load(pc); !ok {
		t.Error("Cache should contain entry")
	} else if cached.(string) == "" {
		t.Error("Cached value should not be empty")
	}
}

func TestDefineAutoPackageDetection(t *testing.T) {
	ClearRegistry()
	
	// Test without explicit package
	err := Define(KConfig{Code: 300})
	
	if err.Package() == "" {
		t.Error("Package should be auto-detected")
	}
	
	if err.Package() == GetConfig().DefaultPackage {
		t.Error("Should not fall back to default package in normal conditions")
	}
}

func TestDefineWithEmptyMessage(t *testing.T) {
	ClearRegistry()
	
	tests := []struct {
		name         string
		code         int
		wantNonEmpty bool
	}{
		{
			name:         "standard HTTP code",
			code:         404,
			wantNonEmpty: true,
		},
		{
			name:         "non-standard code",
			code:         999,
			wantNonEmpty: true,
		},
		{
			name:         "1xx code",
			code:         100,
			wantNonEmpty: true,
		},
		{
			name:         "2xx code",
			code:         204,
			wantNonEmpty: true,
		},
		{
			name:         "3xx code",
			code:         301,
			wantNonEmpty: true,
		},
		{
			name:         "4xx code",
			code:         429,
			wantNonEmpty: true,
		},
		{
			name:         "5xx code",
			code:         503,
			wantNonEmpty: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Define(KConfig{Code: tt.code})
			
			if tt.wantNonEmpty && err.Message() == "" {
				t.Errorf("Message should not be empty for code %d", tt.code)
			}
		})
	}
}