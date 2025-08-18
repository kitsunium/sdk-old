package kbuffer

import (
	"bytes"
	"reflect"
	"testing"
)

// Test helper functions to replace testify assertions

func Equal(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	
	// Special handling for byte slices
	if expectedBytes, ok1 := expected.([]byte); ok1 {
		if actualBytes, ok2 := actual.([]byte); ok2 {
			if !bytes.Equal(expectedBytes, actualBytes) {
				if len(msgAndArgs) > 0 {
					t.Errorf("%v - expected: %v, got: %v", msgAndArgs[0], expectedBytes, actualBytes)
				} else {
					t.Errorf("expected: %v, got: %v", expectedBytes, actualBytes)
				}
			}
			return
		}
	}
	
	// Use reflect.DeepEqual for complex types
	if !reflect.DeepEqual(expected, actual) {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v - expected: %v, got: %v", msgAndArgs[0], expected, actual)
		} else {
			t.Errorf("expected: %v, got: %v", expected, actual)
		}
	}
}

func NotEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if expected == actual {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v - expected not equal: %v", msgAndArgs[0], expected)
		} else {
			t.Errorf("expected not equal: %v", expected)
		}
	}
}

func True(t *testing.T, value bool, msgAndArgs ...interface{}) {
	t.Helper()
	if !value {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v - expected true, got false", msgAndArgs[0])
		} else {
			t.Error("expected true, got false")
		}
	}
}

func False(t *testing.T, value bool, msgAndArgs ...interface{}) {
	t.Helper()
	if value {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v - expected false, got true", msgAndArgs[0])
		} else {
			t.Error("expected false, got true")
		}
	}
}

func NoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v - unexpected error: %v", msgAndArgs[0], err)
		} else {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func Error(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v - expected error, got nil", msgAndArgs[0])
		} else {
			t.Error("expected error, got nil")
		}
	}
}

func Nil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	
	// Check for nil using reflection to handle nil slices correctly
	if value == nil {
		return
	}
	
	// Check if it's a nil slice/map/pointer
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Map || v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
	}
	
	if len(msgAndArgs) > 0 {
		t.Errorf("%v - expected nil, got: %v", msgAndArgs[0], value)
	} else {
		t.Errorf("expected nil, got: %v", value)
	}
}

func NotNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if value == nil {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v - expected not nil", msgAndArgs[0])
		} else {
			t.Error("expected not nil")
		}
	}
}

func Greater(t *testing.T, e1, e2 int64, msgAndArgs ...interface{}) {
	t.Helper()
	if e1 <= e2 {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v - expected %v > %v", msgAndArgs[0], e1, e2)
		} else {
			t.Errorf("expected %v > %v", e1, e2)
		}
	}
}

func GreaterOrEqual(t *testing.T, e1, e2 int, msgAndArgs ...interface{}) {
	t.Helper()
	if e1 < e2 {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v - expected %v >= %v", msgAndArgs[0], e1, e2)
		} else {
			t.Errorf("expected %v >= %v", e1, e2)
		}
	}
}

func Less(t *testing.T, e1, e2 int64, msgAndArgs ...interface{}) {
	t.Helper()
	if e1 >= e2 {
		if len(msgAndArgs) > 0 {
			t.Errorf("%v - expected %v < %v", msgAndArgs[0], e1, e2)
		} else {
			t.Errorf("expected %v < %v", e1, e2)
		}
	}
}