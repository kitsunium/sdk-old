package kerror

import (
	"testing"
)

func TestResult(t *testing.T) {
	// Test success result
	result := NewResult("success", true)
	if !result.Ok {
		t.Error("Result should be Ok")
	}
	if result.Value != "success" {
		t.Error("Result value should be 'success'")
	}
	if result.Unwrap() != "success" {
		t.Error("Unwrap should return 'success'")
	}
	if result.UnwrapOr("default") != "success" {
		t.Error("UnwrapOr should return 'success'")
	}

	// Test fail result
	failResult := NewResult("", false)
	if failResult.Ok {
		t.Error("Result should not be Ok")
	}
	if failResult.UnwrapOr("default") != "default" {
		t.Error("UnwrapOr should return default")
	}

	// Test panic on unwrap fail
	defer func() {
		if r := recover(); r == nil {
			t.Error("Unwrap should panic on failed Result")
		}
	}()
	failResult.Unwrap()
}

func TestResultTypes(t *testing.T) {
	// Test with int
	intResult := NewResult(42, true)
	if intResult.Value != 42 {
		t.Error("Int result value should be 42")
	}

	// Test with struct
	type TestStruct struct {
		Field string
	}
	structResult := NewResult(TestStruct{Field: "test"}, true)
	if structResult.Value.Field != "test" {
		t.Error("Struct result field should be 'test'")
	}
}