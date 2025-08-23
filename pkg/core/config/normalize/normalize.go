// Package normalize provides configuration key and value normalization.
package normalize

import (
	"fmt"
	"strings"
	"unsafe"
)

// Constants for improved readability and maintainability
const (
	tableSize           = 256
	initialBuilderSize  = 64
	minQuoteLength      = 2
	emptyString         = ""
	asciiUpperCaseStart = 'A'
	asciiUpperCaseEnd   = 'Z'
	asciiCaseDiff       = 32
	startIndex          = 0
)

var (
	toLowerTable    [tableSize]byte
	underscoreToDot [tableSize]byte
)

func init() {
	for i := startIndex; i < tableSize; i++ {
		toLowerTable[i] = byte(i)
		underscoreToDot[i] = byte(i)
	}
	for i := asciiUpperCaseStart; i <= asciiUpperCaseEnd; i++ {
		toLowerTable[i] = byte(i + asciiCaseDiff)
	}
	underscoreToDot['_'] = '.'
}

// Key normalizes a configuration key by converting to lowercase and
// replacing underscores with dots.
//
// Example:
//
//	Key("DATABASE_URL") // returns "database.url"
//	Key("Redis_Host")   // returns "redis.host"
func Key(key string) string {
	if len(key) == startIndex {
		return key
	}

	needsTransform := false
	for i := startIndex; i < len(key); i++ {
		c := key[i]
		if c == '_' || (c >= asciiUpperCaseStart && c <= asciiUpperCaseEnd) {
			needsTransform = true
			break
		}
	}

	if !needsTransform {
		return key
	}

	result := make([]byte, len(key))
	for i := startIndex; i < len(key); i++ {
		c := key[i]
		c = toLowerTable[c]
		c = underscoreToDot[c]
		result[i] = c
	}

	return unsafe.String(unsafe.SliceData(result), len(result))
}

// Value normalizes a configuration value by trimming whitespace and
// removing surrounding quotes.
//
// Example:
//
//	Value("  'localhost'  ") // returns "localhost"
//	Value(`"quoted"`)       // returns "quoted"
func Value(value string) string {
	vlen := len(value)
	if vlen == startIndex {
		return value
	}

	start, end := trimWhitespace(value)

	if start >= end {
		return emptyString
	}

	start, end = trimQuotes(value, start, end)

	if start == startIndex && end == vlen {
		return value
	}

	return value[start:end]
}

// trimWhitespace removes leading and trailing whitespace
func trimWhitespace(value string) (start, end int) {
	data := unsafe.StringData(value)
	vlen := len(value)
	start, end = startIndex, vlen

	for start < end {
		c := *(*byte)(unsafe.Add(unsafe.Pointer(data), start))
		if !isWhitespace(c) {
			break
		}
		start++
	}

	for end > start {
		c := *(*byte)(unsafe.Add(unsafe.Pointer(data), end-1))
		if !isWhitespace(c) {
			break
		}
		end--
	}

	return start, end
}

// trimQuotes removes surrounding quotes if present
func trimQuotes(value string, start, end int) (int, int) {
	if end-start >= minQuoteLength {
		data := unsafe.StringData(value)
		first := *(*byte)(unsafe.Add(unsafe.Pointer(data), start))
		last := *(*byte)(unsafe.Add(unsafe.Pointer(data), end-1))

		if first == last && (first == '"' || first == '\'') {
			start++
			end--
		}
	}
	return start, end
}

// isWhitespace checks if a byte is a whitespace character
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// Map flattens a nested map[string]any into a map[string]string with
// dot-notation keys.
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
	keyBuilder.Grow(initialBuilderSize)
	flattenRecursive(output, input, nil, &keyBuilder)
	return output
}

func estimateCapacity(m map[string]any) int {
	var count int
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

func flattenRecursive(output map[string]string, input map[string]any,
	prefix []string, keyBuilder *strings.Builder) {
	for key, value := range input {
		keyBuilder.Reset()
		buildKey(keyBuilder, prefix, key)

		switch v := value.(type) {
		case map[string]any:
			newPrefix := append(prefix, key)
			flattenRecursive(output, v, newPrefix, keyBuilder)
		case []any:
			processSlice(output, v, prefix, key, keyBuilder)
		case string:
			output[Key(keyBuilder.String())] = Value(v)
		case nil:
			output[Key(keyBuilder.String())] = emptyString
		default:
			output[Key(keyBuilder.String())] = Value(fmt.Sprintf("%v", value))
		}
	}
}

// buildKey constructs a dot-separated key from prefix and key
func buildKey(keyBuilder *strings.Builder, prefix []string, key string) {
	if len(prefix) > startIndex {
		for i, p := range prefix {
			if i > startIndex {
				_ = keyBuilder.WriteByte('.')
			}
			_, _ = keyBuilder.WriteString(p)
		}
		_ = keyBuilder.WriteByte('.')
	}
	_, _ = keyBuilder.WriteString(key)
}

// processSlice handles array/slice values during map flattening.
// Array elements are indexed with dot notation (e.g., "key.0", "key.1").
// Nested maps within arrays are recursively flattened.
func processSlice(output map[string]string, slice []any, prefix []string,
	key string, keyBuilder *strings.Builder) {
	for i, item := range slice {
		keyBuilder.Reset()
		buildArrayKey(keyBuilder, prefix, key, i)

		// Process the array item
		switch v := item.(type) {
		case map[string]any:
			itemKey := fmt.Sprintf("%s.%d", key, i)
			newPrefix := append(prefix, itemKey)
			flattenRecursive(output, v, newPrefix, keyBuilder)
		case string:
			output[Key(keyBuilder.String())] = Value(v)
		case nil:
			output[Key(keyBuilder.String())] = emptyString
		default:
			output[Key(keyBuilder.String())] = Value(fmt.Sprintf("%v", item))
		}
	}
}

// buildArrayKey constructs a dot-separated key with array index
func buildArrayKey(keyBuilder *strings.Builder, prefix []string,
	key string, index int) {
	if len(prefix) > startIndex {
		for j, p := range prefix {
			if j > startIndex {
				_ = keyBuilder.WriteByte('.')
			}
			_, _ = keyBuilder.WriteString(p)
		}
		_ = keyBuilder.WriteByte('.')
	}

	_, _ = keyBuilder.WriteString(key)
	_ = keyBuilder.WriteByte('.')
	_, _ = fmt.Fprintf(keyBuilder, "%d", index)
}

// StringToBytesSafe converts string to []byte with allocation (safe copy).
func StringToBytesSafe(s string) []byte {
	return []byte(s)
}

// BytesToStringSafe converts []byte to string with allocation (safe copy).
func BytesToStringSafe(b []byte) string {
	return string(b)
}
