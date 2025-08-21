package normalize_test

import (
	"testing"

	"github.com/kitsunium/sdk/pkg/kernel/config/normalize"
	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	t.Run("EmptyMap", func(t *testing.T) {
		input := map[string]any{}
		expected := map[string]string{}
		result := normalize.Map(input)
		assert.Equal(t, expected, result, "Flattening an empty map should return an empty map")
	})

	t.Run("FlatMap", func(t *testing.T) {
		input := map[string]any{
			"key1": "value1",
			"key2": 42,
		}
		expected := map[string]string{
			"key1": "value1",
			"key2": "42",
		}
		result := normalize.Map(input)
		assert.Equal(t, expected, result, "Flattening a flat map should not change its structure")
	})

	t.Run("NestedMap", func(t *testing.T) {
		input := map[string]any{
			"key1": map[string]any{
				"nested1": "value1",
				"nested2": 42,
			},
		}
		expected := map[string]string{
			"key1.nested1": "value1",
			"key1.nested2": "42",
		}
		result := normalize.Map(input)
		assert.Equal(t, expected, result, "Flattening a nested map should produce dot-separated keys")
	})

	t.Run("MapWithSlices", func(t *testing.T) {
		input := map[string]any{
			"key1": []any{"value1", 42},
		}
		expected := map[string]string{
			"key1.0": "value1",
			"key1.1": "42",
		}
		result := normalize.Map(input)
		assert.Equal(t, expected, result, "Flattening a map with slices should index each slice element")
	})

	t.Run("ComplexMap", func(t *testing.T) {
		input := map[string]any{
			"key1": map[string]any{
				"nested1": []any{
					"value1",
					map[string]any{
						"deep": 42,
					},
				},
			},
		}
		expected := map[string]string{
			"key1.nested1.0":      "value1",
			"key1.nested1.1.deep": "42",
		}
		result := normalize.Map(input)
		assert.Equal(t, expected, result, "Flattening a deeply nested map should handle complex cases correctly")
	})
}
