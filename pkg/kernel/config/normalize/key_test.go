package normalize_test

import (
	"testing"

	"github.com/kistunium/sdk/pkg/kernel/config/normalize"
	"github.com/stretchr/testify/assert"
)

func TestKey(t *testing.T) {
	t.Run("NoUnderscores", func(t *testing.T) {
		input := "SimpleKey"
		expected := "simplekey"
		result := normalize.Key(input)
		assert.Equal(t, expected, result, "Key without underscores should be converted to lowercase")
	})

	t.Run("WithUnderscores", func(t *testing.T) {
		input := "key_with_underscores"
		expected := "key.with.underscores"
		result := normalize.Key(input)
		assert.Equal(t, expected, result, "Key with underscores should replace underscores with dots and convert to lowercase")
	})

	t.Run("MixedCaseAndUnderscores", func(t *testing.T) {
		input := "Key_With_Mixed_CASE"
		expected := "key.with.mixed.case"
		result := normalize.Key(input)
		assert.Equal(t, expected, result, "Key with mixed case and underscores should normalize correctly")
	})

	t.Run("LeadingAndTrailingUnderscores", func(t *testing.T) {
		input := "_leading_and_trailing_"
		expected := ".leading.and.trailing."
		result := normalize.Key(input)
		assert.Equal(t, expected, result, "Leading and trailing underscores should be replaced with dots")
	})

	t.Run("MultipleConsecutiveUnderscores", func(t *testing.T) {
		input := "key__with__multiple___underscores"
		expected := "key..with..multiple...underscores"
		result := normalize.Key(input)
		assert.Equal(t, expected, result, "Consecutive underscores should be replaced with consecutive dots")
	})

	t.Run("EmptyString", func(t *testing.T) {
		input := ""
		expected := ""
		result := normalize.Key(input)
		assert.Equal(t, expected, result, "Empty string should return an empty string")
	})

	t.Run("OnlyUnderscores", func(t *testing.T) {
		input := "___"
		expected := "..."
		result := normalize.Key(input)
		assert.Equal(t, expected, result, "String with only underscores should be replaced with dots")
	})
}
