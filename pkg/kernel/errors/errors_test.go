package errors_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	kerrors "github.com/kitsunium/sdk/pkg/kernel/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewError(t *testing.T) {
	t.Run("CreateNewError", func(t *testing.T) {
		err := kerrors.New(http.StatusNotFound, 1, "resource not found")
		require.NotNil(t, err)
		assert.Equal(t, http.StatusNotFound, err.HTTPCode())
		assert.Equal(t, 1, err.ExitCode())
		assert.Equal(t, "resource not found", err.Message())
		assert.Equal(t, "resource not found", err.Error())
	})

	t.Run("CreateErrorWithTags", func(t *testing.T) {
		err := kerrors.New(http.StatusInternalServerError, 1, "database error", "db", "critical")
		require.NotNil(t, err)
		assert.True(t, err.HasTag("db"))
		assert.True(t, err.HasTag("critical"))
		assert.False(t, err.HasTag("network"))
	})

	t.Run("NewfFormattedError", func(t *testing.T) {
		err := kerrors.Newf(http.StatusBadRequest, 1, "invalid %s: %d", "id", 123)
		require.NotNil(t, err)
		assert.Equal(t, "invalid id: 123", err.Message())
	})

	t.Run("BackwardCompatibility", func(t *testing.T) {
		err := kerrors.New(404, 1, "not found")
		assert.Equal(t, uint16(404), err.Code())
	})
}

func TestErrorCause(t *testing.T) {
	t.Run("WithCause", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := kerrors.New(http.StatusInternalServerError, 1, "wrapper error")
		errWithCause := err.WithCause(originalErr)
		
		assert.Equal(t, "wrapper error: original error", errWithCause.Error())
		assert.Equal(t, originalErr, errWithCause.Cause())
		assert.Equal(t, originalErr, errWithCause.Unwrap())
	})

	t.Run("WrapError", func(t *testing.T) {
		originalErr := errors.New("disk full")
		wrapped := kerrors.Wrap(originalErr, http.StatusInternalServerError, 1, "storage error")
		
		require.NotNil(t, wrapped)
		assert.Equal(t, "storage error: disk full", wrapped.Error())
		assert.Equal(t, originalErr, wrapped.Cause())
	})

	t.Run("WrapNilError", func(t *testing.T) {
		wrapped := kerrors.Wrap(nil, http.StatusInternalServerError, 1, "storage error")
		assert.Nil(t, wrapped)
	})

	t.Run("WrapSDKError", func(t *testing.T) {
		sdkErr := kerrors.New(http.StatusNotFound, 1, "not found")
		wrapped := kerrors.Wrap(sdkErr, http.StatusInternalServerError, 1, "wrapper")
		
		require.NotNil(t, wrapped)
		assert.Equal(t, http.StatusNotFound, wrapped.HTTPCode()) // Preserves original code
	})

	t.Run("WrapfFormattedError", func(t *testing.T) {
		originalErr := errors.New("connection refused")
		wrapped := kerrors.Wrapf(originalErr, http.StatusServiceUnavailable, 1, "failed to connect to %s", "database")
		
		require.NotNil(t, wrapped)
		assert.Equal(t, "failed to connect to database: connection refused", wrapped.Error())
	})
}

func TestErrorComparison(t *testing.T) {
	t.Run("IsComparison", func(t *testing.T) {
		err1 := kerrors.New(http.StatusNotFound, 1, "not found")
		err2 := kerrors.New(http.StatusNotFound, 1, "not found")
		err3 := kerrors.New(http.StatusNotFound, 2, "not found") // Different exit code
		
		assert.True(t, err1.Is(err2))
		assert.False(t, err1.Is(err3))
		
		// Test with errors.Is
		assert.True(t, kerrors.Is(err1, err2))
		assert.False(t, kerrors.Is(err1, err3))
	})

	t.Run("AsConversion", func(t *testing.T) {
		err := kerrors.New(http.StatusNotFound, 1, "not found")
		
		var target *kerrors.Error
		assert.True(t, kerrors.As(err, &target))
		assert.Equal(t, err, target)
	})
}

func TestTags(t *testing.T) {
	t.Run("AddRemoveTags", func(t *testing.T) {
		err := kerrors.New(http.StatusInternalServerError, 1, "error")
		
		// Add tags
		err.AddTag("database", "critical")
		assert.True(t, err.HasTag("database"))
		assert.True(t, err.HasTag("critical"))
		
		// Remove tag
		err.RemoveTag("critical")
		assert.True(t, err.HasTag("database"))
		assert.False(t, err.HasTag("critical"))
		
		// Get all tags
		tags := err.Tags()
		assert.Contains(t, tags, "database")
		assert.NotContains(t, tags, "critical")
	})

	t.Run("MethodChaining", func(t *testing.T) {
		err := kerrors.New(http.StatusInternalServerError, 1, "error").
			AddTag("db").
			AddTag("critical").
			RemoveTag("critical")
		
		assert.True(t, err.HasTag("db"))
		assert.False(t, err.HasTag("critical"))
	})
}

func TestDetails(t *testing.T) {
	t.Run("WithDetail", func(t *testing.T) {
		err := kerrors.New(http.StatusBadRequest, 1, "validation error").
			WithDetail("field", "email").
			WithDetail("value", "invalid@")
		
		val, ok := err.GetDetail("field")
		assert.True(t, ok)
		assert.Equal(t, "email", val)
		
		val, ok = err.GetDetail("value")
		assert.True(t, ok)
		assert.Equal(t, "invalid@", val)
		
		_, ok = err.GetDetail("missing")
		assert.False(t, ok)
	})

	t.Run("WithDetails", func(t *testing.T) {
		details := map[string]interface{}{
			"user_id": 123,
			"action":  "delete",
			"reason":  "unauthorized",
		}
		
		err := kerrors.New(http.StatusForbidden, 1, "permission denied").
			WithDetails(details)
		
		allDetails := err.Details()
		assert.Equal(t, 123, allDetails["user_id"])
		assert.Equal(t, "delete", allDetails["action"])
		assert.Equal(t, "unauthorized", allDetails["reason"])
	})

	t.Run("DetailsImmutability", func(t *testing.T) {
		err := kerrors.New(http.StatusBadRequest, 1, "error").
			WithDetail("key", "value")
		
		// Get details and try to modify
		details := err.Details()
		details["key"] = "modified"
		details["new"] = "added"
		
		// Original should be unchanged
		val, _ := err.GetDetail("key")
		assert.Equal(t, "value", val)
		_, ok := err.GetDetail("new")
		assert.False(t, ok)
	})
}

func TestHTTPHelpers(t *testing.T) {
	t.Run("IsClientError", func(t *testing.T) {
		clientErr := kerrors.New(http.StatusBadRequest, 1, "bad request")
		serverErr := kerrors.New(http.StatusInternalServerError, 1, "server error")
		successErr := kerrors.New(http.StatusOK, 0, "ok")
		
		assert.True(t, clientErr.IsClientError())
		assert.False(t, serverErr.IsClientError())
		assert.False(t, successErr.IsClientError())
	})

	t.Run("IsServerError", func(t *testing.T) {
		clientErr := kerrors.New(http.StatusBadRequest, 1, "bad request")
		serverErr := kerrors.New(http.StatusInternalServerError, 1, "server error")
		successErr := kerrors.New(http.StatusOK, 0, "ok")
		
		assert.False(t, clientErr.IsServerError())
		assert.True(t, serverErr.IsServerError())
		assert.False(t, successErr.IsServerError())
	})

	t.Run("HTTPStatusText", func(t *testing.T) {
		assert.Equal(t, "Not Found", kerrors.HTTPStatusText(http.StatusNotFound))
		assert.Equal(t, "Internal Server Error", kerrors.HTTPStatusText(http.StatusInternalServerError))
	})
}

func TestRegistry(t *testing.T) {
	t.Run("ListErrors", func(t *testing.T) {
		// Clear registry first
		kerrors.ClearRegistry()
		
		// Create errors
		kerrors.New(http.StatusBadRequest, 1, "bad request")
		kerrors.New(http.StatusNotFound, 1, "not found")
		kerrors.New(http.StatusInternalServerError, 1, "internal error")
		
		errs := kerrors.ListErrors()
		assert.Len(t, errs, 3)
		assert.Contains(t, errs, "bad request")
		assert.Contains(t, errs, "not found")
		assert.Contains(t, errs, "internal error")
	})

	t.Run("GetError", func(t *testing.T) {
		kerrors.ClearRegistry()
		
		original := kerrors.New(http.StatusNotFound, 1, "resource not found")
		
		retrieved, ok := kerrors.GetError("resource not found")
		assert.True(t, ok)
		assert.Equal(t, original, retrieved)
		
		_, ok = kerrors.GetError("non-existent")
		assert.False(t, ok)
	})

	t.Run("RegistryIsolation", func(t *testing.T) {
		kerrors.ClearRegistry()
		
		err := kerrors.New(http.StatusBadRequest, 1, "test error")
		errCopy := err.WithDetail("key", "value")
		
		// Original in registry should not have the detail
		registered, _ := kerrors.GetError("test error")
		_, hasDetail := registered.GetDetail("key")
		assert.False(t, hasDetail)
		
		// Copy should have the detail
		val, ok := errCopy.GetDetail("key")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})
}

func TestStandardErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *kerrors.Error
		httpCode int
		exitCode int
		message  string
	}{
		{"NotFound", kerrors.ErrNotFound, http.StatusNotFound, 1, "not found"},
		{"BadRequest", kerrors.ErrBadRequest, http.StatusBadRequest, 1, "bad request"},
		{"Unauthorized", kerrors.ErrUnauthorized, http.StatusUnauthorized, 1, "unauthorized"},
		{"Forbidden", kerrors.ErrForbidden, http.StatusForbidden, 1, "forbidden"},
		{"Internal", kerrors.ErrInternal, http.StatusInternalServerError, 1, "internal server error"},
		{"Conflict", kerrors.ErrConflict, http.StatusConflict, 1, "conflict"},
		{"Unprocessable", kerrors.ErrUnprocessable, http.StatusUnprocessableEntity, 1, "unprocessable entity"},
		{"TooManyRequests", kerrors.ErrTooManyRequests, http.StatusTooManyRequests, 1, "too many requests"},
		{"ServiceUnavailable", kerrors.ErrServiceUnavailable, http.StatusServiceUnavailable, 1, "service unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.httpCode, tt.err.HTTPCode())
			assert.Equal(t, tt.exitCode, tt.err.ExitCode())
			assert.Equal(t, tt.message, tt.err.Message())
		})
	}
}

func TestErrorCloning(t *testing.T) {
	t.Run("CloneIndependence", func(t *testing.T) {
		original := kerrors.New(http.StatusBadRequest, 1, "original")
		original.AddTag("original-tag")
		original = original.WithDetail("key", "original-value")
		
		// Create clone through WithCause and add modifications
		clone := original.WithCause(errors.New("cause")).
			WithTag("clone-tag").
			WithDetail("key", "clone-value")
		
		// Check original is unchanged
		assert.True(t, original.HasTag("original-tag"))
		assert.False(t, original.HasTag("clone-tag"))
		
		val, _ := original.GetDetail("key")
		assert.Equal(t, "original-value", val)
		
		// Check clone has both tags and new detail
		assert.True(t, clone.HasTag("original-tag"))
		assert.True(t, clone.HasTag("clone-tag"))
		
		val, _ = clone.GetDetail("key")
		assert.Equal(t, "clone-value", val)
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("ConcurrentTagOperations", func(t *testing.T) {
		err := kerrors.New(http.StatusInternalServerError, 1, "concurrent error")
		
		done := make(chan bool)
		
		// Concurrent writes
		go func() {
			for i := 0; i < 100; i++ {
				err.AddTag(fmt.Sprintf("tag%d", i))
			}
			done <- true
		}()
		
		// Concurrent reads
		go func() {
			for i := 0; i < 100; i++ {
				_ = err.HasTag(fmt.Sprintf("tag%d", i))
				_ = err.Tags()
			}
			done <- true
		}()
		
		// Concurrent removes
		go func() {
			for i := 0; i < 100; i++ {
				err.RemoveTag(fmt.Sprintf("tag%d", i))
			}
			done <- true
		}()
		
		// Wait for all goroutines
		for i := 0; i < 3; i++ {
			<-done
		}
	})

	t.Run("ConcurrentDetailOperations", func(t *testing.T) {
		err := kerrors.New(http.StatusInternalServerError, 1, "concurrent error")
		
		done := make(chan bool)
		
		// Concurrent detail additions
		go func() {
			for i := 0; i < 100; i++ {
				err.WithDetail(fmt.Sprintf("key%d", i), i)
			}
			done <- true
		}()
		
		// Concurrent detail reads
		go func() {
			for i := 0; i < 100; i++ {
				_ = err.Details()
				_, _ = err.GetDetail(fmt.Sprintf("key%d", i))
			}
			done <- true
		}()
		
		// Wait for all goroutines
		for i := 0; i < 2; i++ {
			<-done
		}
	})
}

func TestUnwrapCompatibility(t *testing.T) {
	t.Run("UnwrapChain", func(t *testing.T) {
		baseErr := errors.New("base error")
		wrapped1 := fmt.Errorf("wrapped1: %w", baseErr)
		wrapped2 := kerrors.Wrap(wrapped1, http.StatusInternalServerError, 1, "wrapped2")
		
		// Test unwrapping
		assert.Equal(t, wrapped1, kerrors.Unwrap(wrapped2))
		assert.True(t, errors.Is(wrapped2, baseErr))
	})
}

func ExampleNew() {
	err := kerrors.New(http.StatusNotFound, 1, "user not found").
		AddTag("database").
		WithDetail("user_id", 123)
	
	fmt.Printf("HTTP Code: %d\n", err.HTTPCode())
	fmt.Printf("Exit Code: %d\n", err.ExitCode())
	fmt.Printf("Message: %s\n", err.Message())
	fmt.Printf("Has 'database' tag: %t\n", err.HasTag("database"))
	
	// Output:
	// HTTP Code: 404
	// Exit Code: 1
	// Message: user not found
	// Has 'database' tag: true
}

func ExampleWrap() {
	originalErr := errors.New("connection refused")
	wrappedErr := kerrors.Wrap(originalErr, http.StatusServiceUnavailable, 1, "database connection failed")
	
	fmt.Printf("Error: %s\n", wrappedErr.Error())
	fmt.Printf("Is server error: %t\n", wrappedErr.IsServerError())
	
	// Output:
	// Error: database connection failed: connection refused
	// Is server error: true
}