// Package normalize provides configuration key and value normalization.
// It transforms configuration keys to a consistent format and handles
// value trimming and quote removal.
package normalize

import (
	"fmt"
	"strings"
	"unsafe"
)

// Lookup tables for fast character conversion (initialized once)
var (
	// Pre-computed lowercase conversion table
	toLowerTable [256]byte
	// Pre-computed underscore to dot conversion
	underscoreToDot [256]byte
)

func init() {
	// Initialize lookup tables for fast conversion
	for i := 0; i < 256; i++ {
		toLowerTable[i] = byte(i)
		underscoreToDot[i] = byte(i)
	}
	// Set uppercase to lowercase conversions (A-Z -> a-z)
	for i := 'A'; i <= 'Z'; i++ {
		toLowerTable[i] = byte(i + 32) // or i | 0x20
	}
	// Set underscore to dot conversion
	underscoreToDot['_'] = '.'
}

// Key normalizes a configuration key by replacing underscores with dots and
// converting to lowercase.
//
// Examples:
//   - "DATABASE_URL" -> "database.url"
//   - "Redis_Host" -> "redis.host"
//   - "already.lowercase" -> "already.lowercase"
func Key(key string) string {
	if len(key) == 0 {
		return key
	}

	// Fast path: check if transformation is needed
	needsTransform := false
	for i := 0; i < len(key); i++ {
		c := key[i]
		if c == '_' || (c >= 'A' && c <= 'Z') {
			needsTransform = true
			break
		}
	}

	// If no transformation needed, return original string
	if !needsTransform {
		return key
	}

	// Transform using lookup tables (fastest method)
	result := make([]byte, len(key))
	for i := 0; i < len(key); i++ {
		c := key[i]
		// Apply both transformations using lookup tables
		c = toLowerTable[c]
		c = underscoreToDot[c]
		result[i] = c
	}

	// Convert back to string without allocation
	return unsafe.String(unsafe.SliceData(result), len(result))
}

// Value normalizes a configuration value by trimming whitespace and removing
// surrounding quotes.
//
// Supported whitespace characters:
//   - Space (0x20)
//   - Tab (0x09)
//   - Newline (0x0A)
//   - Carriage return (0x0D)
//
// Quote handling:
//   - Removes matching double quotes (")
//   - Removes matching single quotes (')
//   - Only removes if quotes match at both ends
//
// Examples:
//   - "  trimmed  " -> "trimmed"
//   - "'quoted'" -> "quoted"
//   - "\r\n  Windows  \r\n" -> "Windows"
func Value(value string) string {
	vlen := len(value)
	if vlen == 0 {
		return value
	}

	// Use unsafe for direct memory access
	data := unsafe.StringData(value)
	start, end := 0, vlen

	// Optimized whitespace check using bit manipulation
	// space=0x20, tab=0x09, newline=0x0A, cr=0x0D
	for start < end {
		c := *(*byte)(unsafe.Add(unsafe.Pointer(data), start))
		// Fast whitespace check: most common first (space), then others
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			start++
		} else {
			break
		}
	}

	for end > start {
		c := *(*byte)(unsafe.Add(unsafe.Pointer(data), end-1))
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			end--
		} else {
			break
		}
	}

	// Early return for empty after trim
	if start >= end {
		return ""
	}

	// Check for quotes only if we have at least 2 chars
	if end-start >= 2 {
		first := *(*byte)(unsafe.Add(unsafe.Pointer(data), start))
		last := *(*byte)(unsafe.Add(unsafe.Pointer(data), end-1))

		// Optimized quote check with single comparison when possible
		if first == last && (first == '"' || first == '\'') {
			start++
			end--
		}
	}

	// Fast return (no allocation if unchanged)
	if start == 0 && end == vlen {
		return value
	}

	return value[start:end]
}

// Map flattens a nested map[string]any into a map[string]string with dot-notation
// keys representing the hierarchical structure.
//
// Supported value types:
//   - Nested maps (map[string]any) - recursively flattened
//   - Arrays ([]any) - indexed with dot notation (e.g., "key.0", "key.1")
//   - Strings - normalized via Value()
//   - nil - converted to empty string
//   - Other types - converted via fmt.Sprintf and normalized
//
// Examples:
//
//	{"db": {"host": "localhost", "port": 5432}} -> {"db.host": "localhost", "db.port": "5432"}
//	{"servers": ["a", "b"]} -> {"servers.0": "a", "servers.1": "b"}
func Map(input map[string]any) map[string]string {
	capacity := estimateCapacity(input)
	output := make(map[string]string, capacity)
	var keyBuilder strings.Builder
	keyBuilder.Grow(64)
	flattenRecursive(output, input, nil, &keyBuilder)
	return output
}

// Helper functions for Map() implementation

// estimateCapacity calculates the approximate number of flat key-value pairs
// that will result from flattening a nested map.
func estimateCapacity(m map[string]any) int {
	count := 0
	for _, v := range m {
		switch v := v.(type) {
		case map[string]any:
			count += estimateCapacity(v)
		case []any:
			count += len(v)
		default:
			count++
		}
	}
	return count
}

// flattenRecursive recursively flattens a nested map structure into dot-notation keys.
// The prefix slice tracks the current path in the hierarchy.
func flattenRecursive(output map[string]string, input map[string]any, prefix []string, keyBuilder *strings.Builder) {
	for key, value := range input {
		keyBuilder.Reset()

		// Build the full key path
		if len(prefix) > 0 {
			for i, p := range prefix {
				if i > 0 {
					keyBuilder.WriteByte('.')
				}
				keyBuilder.WriteString(p)
			}
			keyBuilder.WriteByte('.')
		}
		keyBuilder.WriteString(key)

		// Process the value based on its type
		switch v := value.(type) {
		case map[string]any:
			// Recursively process nested maps
			newPrefix := append(prefix, key)
			flattenRecursive(output, v, newPrefix, keyBuilder)
		case []any:
			// Process arrays with index notation
			processSlice(output, v, prefix, key, keyBuilder)
		case string:
			output[Key(keyBuilder.String())] = Value(v)
		case nil:
			output[Key(keyBuilder.String())] = ""
		default:
			output[Key(keyBuilder.String())] = Value(fmt.Sprintf("%v", value))
		}
	}
}

// processSlice handles array/slice values during map flattening.
// Array elements are indexed with dot notation (e.g., "key.0", "key.1").
// Nested maps within arrays are recursively flattened.
func processSlice(output map[string]string, slice []any, prefix []string, key string, keyBuilder *strings.Builder) {
	for i, item := range slice {
		keyBuilder.Reset()

		// Build key with array index
		if len(prefix) > 0 {
			for j, p := range prefix {
				if j > 0 {
					keyBuilder.WriteByte('.')
				}
				keyBuilder.WriteString(p)
			}
			keyBuilder.WriteByte('.')
		}

		keyBuilder.WriteString(key)
		keyBuilder.WriteByte('.')
		fmt.Fprintf(keyBuilder, "%d", i)

		// Process the array item
		switch v := item.(type) {
		case map[string]any:
			itemKey := fmt.Sprintf("%s.%d", key, i)
			newPrefix := append(prefix, itemKey)
			flattenRecursive(output, v, newPrefix, keyBuilder)
		case string:
			output[Key(keyBuilder.String())] = Value(v)
		case nil:
			output[Key(keyBuilder.String())] = ""
		default:
			output[Key(keyBuilder.String())] = Value(fmt.Sprintf("%v", item))
		}
	}
}

// StringToBytes converts a string to a byte slice without allocation.
// This function uses unsafe operations to directly access the underlying
// string data.
//
// WARNING: The returned byte slice shares memory with the input string.
// Modifying the byte slice may cause undefined behavior. Use this function
// only when you need read-only access to the string bytes.
func StringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// BytesToString converts a byte slice to a string without allocation.
// This function uses unsafe operations to create a string header that
// references the same underlying memory as the byte slice.
//
// WARNING: The returned string shares memory with the input byte slice.
// The byte slice must not be modified after calling this function, as it
// would violate Go's string immutability guarantee.
func BytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
