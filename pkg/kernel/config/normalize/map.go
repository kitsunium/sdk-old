package normalize

import (
	"fmt"
	"strings"
)

// Map flattens a nested map[string]any into a map[string]string.
// Keys in the resulting map represent the flattened path using dot notation.
//
// Parameters:
// - input: map[string]any - The input map to flatten.
//
// Returns:
// - map[string]string: A flattened map with string keys and values.
func Map(input map[string]any) map[string]string {
	output := make(map[string]string, len(input))
	reduce(output, input, nil)
	return output
}

// reduce recursively processes the input map and populates the output map with flattened keys.
//
// Parameters:
// - output: map[string]string - The resulting flattened map.
// - input: map[string]any - The map to process.
// - prefix: []string - The current prefix representing the path to the current map.
func reduce(output map[string]string, input map[string]any, prefix []string) {
	for key, value := range input {
		fullKey := make([]string, len(prefix)+1)
		copy(fullKey, prefix)
		fullKey[len(prefix)] = key

		switch v := value.(type) {
		case map[string]any:
			// Recurse for nested maps
			reduce(output, v, fullKey)
		case []any:
			// Handle slices by indexing each element
			for i, item := range v {
				reduce(output, map[string]any{fmt.Sprintf("%s.%d", key, i): item}, prefix)
			}
		default:
			// Flatten the key and normalize the value
			flatKey := strings.Join(fullKey, ".")
			output[Key(flatKey)] = Value(fmt.Sprintf("%v", value))
		}
	}
}
