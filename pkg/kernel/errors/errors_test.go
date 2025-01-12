package errors_test

import (
	"testing"

	"github.com/kistunium/sdk/pkg/kernel/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	t.Run("CreateNewError", func(t *testing.T) {
		err := errors.New(404, "not found")
		assert.NotNil(t, err, "Error should not be nil")
		assert.Equal(t, uint16(404), err.Code(), "Error code mismatch")
		assert.Equal(t, "not found", err.Error(), "Error message mismatch")
	})

	t.Run("DuplicateError", func(t *testing.T) {
		err1 := errors.New(400, "bad request")
		err2 := errors.New(400, "bad request")
		assert.Equal(t, err1, err2, "Duplicate errors should reference the same object")
	})
}

func TestListErrors(t *testing.T) {
	t.Run("ListRegisteredErrors", func(t *testing.T) {
		// Add some errors
		errors.New(400, "bad request")
		errors.New(404, "not found")
		errors.New(500, "internal server error")

		errs := errors.ListErrors()
		assert.NotEmpty(t, errs, "Registered errors should not be empty")
		assert.Equal(t, 3, len(errs), "Number of registered errors mismatch")

		// Verify specific errors
		assert.Contains(t, errs, "bad request", "Error list should contain 'bad request'")
		assert.Contains(t, errs, "not found", "Error list should contain 'not found'")
		assert.Contains(t, errs, "internal server error", "Error list should contain 'internal server error'")
	})
}

func TestErrorTags(t *testing.T) {
	t.Run("CreateErrorWithTags", func(t *testing.T) {
		err := errors.New(500, "Internal Server Error", "Critical", "Database")
		assert.True(t, err.HasTag("Critical"), "Error should have 'Critical' tag")
		assert.True(t, err.HasTag("Database"), "Error should have 'Database' tag")
		assert.False(t, err.HasTag("Network"), "Error should not have 'Network' tag")
	})

	t.Run("AddAndRemoveTags", func(t *testing.T) {
		err := errors.New(500, "Internal Server Error", "Critical")
		assert.True(t, err.HasTag("Critical"), "Error should have 'Critical' tag")
		assert.False(t, err.HasTag("Network"), "Error should not have 'Network' tag")

		// Add Network tag
		err.AddTag("Network")
		assert.True(t, err.HasTag("Network"), "Error should have 'Network' tag")

		// Remove Critical tag
		err.RemoveTag("Critical")
		assert.False(t, err.HasTag("Critical"), "Error should not have 'Critical' tag")
	})

	t.Run("ListTags", func(t *testing.T) {
		err := errors.New(500, "Internal Server Error", "Critical", "Database")
		tags := err.Tags()
		assert.Contains(t, tags, "Critical", "Tags should contain 'Critical'")
		assert.Contains(t, tags, "Database", "Tags should contain 'Database'")
		assert.NotContains(t, tags, "Network", "Tags should not contain 'Network'")
	})
}
