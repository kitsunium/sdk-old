// Package normalize provides configuration key and value normalization.
package normalize

import (
	"fmt"
	"strings"
	"unsafe"
)

var (
	toLowerTable    [256]byte
	underscoreToDot [256]byte
)

func init() {
	for i := 0; i < 256; i++ {
		toLowerTable[i] = byte(i)
		underscoreToDot[i] = byte(i)
	}
	for i := 'A'; i <= 'Z'; i++ {
		toLowerTable[i] = byte(i + 32)
	}
	underscoreToDot['_'] = '.'
}

// Key normalizes a configuration key by converting to lowercase and replacing underscores with dots.
//
// Example:
//
//	Key("DATABASE_URL") // returns "database.url"
//	Key("Redis_Host")   // returns "redis.host"
func Key(key string) string {
	if len(key) == 0 {
		return key
	}

	needsTransform := false
	for i := 0; i < len(key); i++ {
		c := key[i]
		if c == '_' || (c >= 'A' && c <= 'Z') {
			needsTransform = true
			break
		}
	}

	if !needsTransform {
		return key
	}

	result := make([]byte, len(key))
	for i := 0; i < len(key); i++ {
		c := key[i]
		c = toLowerTable[c]
		c = underscoreToDot[c]
		result[i] = c
	}

	return unsafe.String(unsafe.SliceData(result), len(result))
}

// Value normalizes a configuration value by trimming whitespace and removing surrounding quotes.
//
// Example:
//
//	Value("  'localhost'  ") // returns "localhost"
//	Value(`"quoted"`)       // returns "quoted"
func Value(value string) string {
	vlen := len(value)
	if vlen == 0 {
		return value
	}

	data := unsafe.StringData(value)
	start, end := 0, vlen

	for start < end {
		c := *(*byte)(unsafe.Add(unsafe.Pointer(data), start))
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

	if start >= end {
		return ""
	}

	if end-start >= 2 {
		first := *(*byte)(unsafe.Add(unsafe.Pointer(data), start))
		last := *(*byte)(unsafe.Add(unsafe.Pointer(data), end-1))

		if first == last && (first == '"' || first == '\'') {
			start++
			end--
		}
	}

	if start == 0 && end == vlen {
		return value
	}

	return value[start:end]
}

// Map flattens a nested map[string]any into a map[string]string with dot-notation keys.
//
// Example:
//
//	input := map[string]any{
//	  "db": map[string]any{"host": "localhost", "port": 5432},
//	}
//	result := Map(input)
//	// result["db.host"] = "localhost"
//	// result["db.port"] = "5432"
func Map(input map[string]any) map[string]string {
	capacity := estimateCapacity(input)
	output := make(map[string]string, capacity)
	var keyBuilder strings.Builder
	keyBuilder.Grow(64)
	flattenRecursive(output, input, nil, &keyBuilder)
	return output
}

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

func flattenRecursive(output map[string]string, input map[string]any, prefix []string, keyBuilder *strings.Builder) {
	for key, value := range input {
		keyBuilder.Reset()

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

		switch v := value.(type) {
		case map[string]any:
			newPrefix := append(prefix, key)
			flattenRecursive(output, v, newPrefix, keyBuilder)
		case []any:
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

// StringToBytesSafe converts string to []byte with allocation (safe copy).
func StringToBytesSafe(s string) []byte {
	return []byte(s)
}

// BytesToStringSafe converts []byte to string with allocation (safe copy).
func BytesToStringSafe(b []byte) string {
	return string(b)
}
