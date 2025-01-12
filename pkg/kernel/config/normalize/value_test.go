package normalize_test

import (
	"testing"

	"github.com/kistunium/sdk/pkg/kernel/config/normalize"
	"github.com/stretchr/testify/assert"
)

func TestValue(t *testing.T) {
	t.Run("NoQuotes", func(t *testing.T) {
		input := "hello"
		expected := "hello"
		result := normalize.Value(input)
		assert.Equal(t, expected, result, "Value without quotes should remain unchanged")
	})

	t.Run("SingleQuotes", func(t *testing.T) {
		input := "'hello'"
		expected := "hello"
		result := normalize.Value(input)
		assert.Equal(t, expected, result, "Value with single quotes should have them removed")
	})

	t.Run("DoubleQuotes", func(t *testing.T) {
		input := "\"hello\""
		expected := "hello"
		result := normalize.Value(input)
		assert.Equal(t, expected, result, "Value with double quotes should have them removed")
	})

	t.Run("MixedQuotes", func(t *testing.T) {
		input := "\"hello'"
		expected := "\"hello'"
		result := normalize.Value(input)
		assert.Equal(t, expected, result, "Mixed quotes should remain unchanged")
	})

	t.Run("SpacesWithQuotes", func(t *testing.T) {
		input := "  'hello'  "
		expected := "hello"
		result := normalize.Value(input)
		assert.Equal(t, expected, result, "Surrounding spaces and quotes should be removed")
	})

	t.Run("OnlySpaces", func(t *testing.T) {
		input := "   "
		expected := ""
		result := normalize.Value(input)
		assert.Equal(t, expected, result, "Value with only spaces should return an empty string")
	})

	t.Run("EmptyString", func(t *testing.T) {
		input := ""
		expected := ""
		result := normalize.Value(input)
		assert.Equal(t, expected, result, "Empty string should remain unchanged")
	})

	t.Run("UnmatchedSingleQuote", func(t *testing.T) {
		input := "'hello"
		expected := "'hello"
		result := normalize.Value(input)
		assert.Equal(t, expected, result, "Unmatched single quote should remain unchanged")
	})

	t.Run("UnmatchedDoubleQuote", func(t *testing.T) {
		input := "\"hello"
		expected := "\"hello"
		result := normalize.Value(input)
		assert.Equal(t, expected, result, "Unmatched double quote should remain unchanged")
	})
}
