// Package normalize provides utilities for normalizing configuration keys and values.
//
// This package offers high-performance normalization functions that transform
// configuration keys to a consistent format (lowercase with dots) and clean
// configuration values by trimming whitespace and quotes. It also provides
// map flattening capabilities to convert nested structures into flat key-value pairs.
//
// Key features:
//   - Zero-allocation key normalization using lookup tables
//   - Efficient value trimming with unsafe operations
//   - Nested map flattening with dot notation
//   - Array/slice support in map flattening
//
// Example usage:
//
//	normalized := normalize.Key("DATABASE_URL")  // "database.url"
//	value := normalize.Value(" 'localhost' ")    // "localhost"
//
//	config := map[string]any{
//	    "database": map[string]any{
//	        "host": "localhost",
//	        "port": 5432,
//	    },
//	}
//	flat := normalize.Map(config)
//	// flat["database.host"] = "localhost"
//	// flat["database.port"] = "5432"
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
// replacing underscores with dots. This function uses pre-computed lookup
// tables for optimal performance and avoids allocations when possible.
//
// The normalization process:
//   - Converts uppercase letters to lowercase
//   - Replaces underscores with dots
//   - Returns the original string if no transformation is needed
//
// Example:
//
//	Key("DATABASE_URL") // returns "database.url"
//	Key("Redis_Host")   // returns "redis.host"
//	Key("api.key")      // returns "api.key" (unchanged)
func Key(key string) string {
	if len(key) == startIndex {
		return key
	}

	if !needsKeyTransform(key) {
		return key
	}

	return transformKey(key)
}

// needsKeyTransform checks if a key needs transformation
func needsKeyTransform(key string) bool {
	for i := startIndex; i < len(key); i++ {
		c := key[i]
		if c == '_' || isUpperCase(c) {
			return true
		}
	}
	return false
}

// isUpperCase checks if a byte is an uppercase ASCII letter
func isUpperCase(c byte) bool {
	return c >= asciiUpperCaseStart && c <= asciiUpperCaseEnd
}

// transformKey applies lowercase and underscore-to-dot transformations
func transformKey(key string) string {
	result := make([]byte, len(key))
	for i := startIndex; i < len(key); i++ {
		result[i] = transformByte(key[i])
	}
	return unsafe.String(unsafe.SliceData(result), len(result))
}

// transformByte applies transformations to a single byte
func transformByte(c byte) byte {
	c = toLowerTable[c]
	c = underscoreToDot[c]
	return c
}

// Value normalizes a configuration value by trimming whitespace and
// removing surrounding quotes. Uses unsafe operations for optimal performance
// while maintaining safety through careful bounds checking.
//
// The normalization process:
//   - Trims leading and trailing whitespace (space, tab, newline, carriage return)
//   - Removes matching surrounding quotes (single or double)
//   - Returns empty string for whitespace-only values
//
// Example:
//
//	Value("  'localhost'  ") // returns "localhost"
//	Value(`"quoted"`)       // returns "quoted"
//	Value("  \n\t  ")        // returns ""
//	Value("no quotes")       // returns "no quotes"
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

// trimWhitespace removes leading and trailing whitespace characters.
// Returns the start and end indices of the non-whitespace content.
// Uses unsafe pointer arithmetic for optimal performance.
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

// trimQuotes removes matching surrounding quotes (single or double) if present.
// Only removes quotes if they match at both ends of the string.
// Returns adjusted start and end indices.
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

// isWhitespace checks if a byte is a whitespace character.
// Considers space, tab, newline, and carriage return as whitespace.
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// Map flattens a nested map[string]any into a map[string]string with
// dot-notation keys. This function recursively processes nested maps and
// arrays, converting all values to strings and normalizing keys.
//
// Features:
//   - Handles nested maps with unlimited depth
//   - Processes arrays/slices with indexed notation (e.g., "items.0", "items.1")
//   - Normalizes all keys using the Key function
//   - Normalizes string values using the Value function
//   - Pre-allocates output map with estimated capacity for efficiency
//
// Example:
//
//	input := map[string]any{
//	    "db": map[string]any{"host": "localhost", "port": 5432},
//	    "servers": []any{"server1", "server2"},
//	}
//	result := Map(input)
//	// result["db.host"] = "localhost"
//	// result["db.port"] = "5432"
//	// result["servers.0"] = "server1"
//	// result["servers.1"] = "server2"
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

// flattenContext holds context for map flattening to reduce parameter count
type flattenContext struct {
	output     map[string]string
	prefix     []string
	keyBuilder *strings.Builder
}

func flattenRecursive(output map[string]string, input map[string]any,
	prefix []string, keyBuilder *strings.Builder) {
	ctx := &flattenContext{
		output:     output,
		prefix:     prefix,
		keyBuilder: keyBuilder,
	}
	for key, value := range input {
		keyBuilder.Reset()
		buildKey(keyBuilder, prefix, key)
		processValue(ctx, value, key)
	}
}

// processValue handles different value types during flattening.
// Dispatches to appropriate handlers based on value type:
// maps are recursively flattened, arrays are indexed, and other values are stored.
func processValue(ctx *flattenContext, value any, key string) {
	switch v := value.(type) {
	case map[string]any:
		processMap(ctx, v, key)
	case []any:
		processSlice(ctx, v, key)
	default:
		storeValue(ctx.output, ctx.keyBuilder.String(), value)
	}
}

// processMap handles nested map values during flattening.
// Creates a new prefix by appending the current key and recursively processes the map.
func processMap(ctx *flattenContext, m map[string]any, key string) {
	newPrefix := append(ctx.prefix, key)
	flattenRecursive(ctx.output, m, newPrefix, ctx.keyBuilder)
}

// buildKey constructs a dot-separated key from prefix and key.
// Efficiently builds the full path using a strings.Builder.
func buildKey(keyBuilder *strings.Builder, prefix []string, key string) {
	writePrefix(keyBuilder, prefix)
	_, _ = keyBuilder.WriteString(key)
}

// writePrefix writes the prefix parts to the builder with dot separators.
// Handles empty prefixes gracefully and adds appropriate separators.
func writePrefix(keyBuilder *strings.Builder, prefix []string) {
	if len(prefix) > startIndex {
		for i, p := range prefix {
			if i > startIndex {
				_ = keyBuilder.WriteByte('.')
			}
			_, _ = keyBuilder.WriteString(p)
		}
		_ = keyBuilder.WriteByte('.')
	}
}

// processSlice handles array/slice values during map flattening.
// Array elements are indexed with dot notation (e.g., "key.0", "key.1").
// Nested maps within arrays are recursively flattened.
func processSlice(ctx *flattenContext, slice []any, key string) {
	for i, item := range slice {
		ctx.keyBuilder.Reset()
		buildArrayKey(ctx.keyBuilder, ctx.prefix, key, i)

		// Process the array item
		switch v := item.(type) {
		case map[string]any:
			itemKey := fmt.Sprintf("%s.%d", key, i)
			newPrefix := append(ctx.prefix, itemKey)
			flattenRecursive(ctx.output, v, newPrefix, ctx.keyBuilder)
		default:
			storeValue(ctx.output, ctx.keyBuilder.String(), item)
		}
	}
}

// buildArrayKey constructs a dot-separated key with array index.
// Creates keys in the format "prefix.key.index" for array elements.
func buildArrayKey(keyBuilder *strings.Builder, prefix []string,
	key string, index int) {
	writePrefix(keyBuilder, prefix)
	_, _ = keyBuilder.WriteString(key)
	_ = keyBuilder.WriteByte('.')
	_, _ = fmt.Fprintf(keyBuilder, "%d", index)
}

// storeValue stores a value in the output map with proper formatting.
// Normalizes the key and converts the value to a string representation.
// Handles nil values by storing empty strings.
func storeValue(output map[string]string, key string, value any) {
	normalizedKey := Key(key)
	switch v := value.(type) {
	case string:
		output[normalizedKey] = Value(v)
	case nil:
		output[normalizedKey] = emptyString
	default:
		output[normalizedKey] = Value(fmt.Sprintf("%v", value))
	}
}

// StringToBytesSafe converts string to []byte with allocation (safe copy).
// This function creates a new byte slice and copies the string data,
// ensuring the resulting slice is independent of the original string.
// Use this when you need to modify the bytes or keep them beyond the string's lifetime.
func StringToBytesSafe(s string) []byte {
	return []byte(s)
}

// BytesToStringSafe converts []byte to string with allocation (safe copy).
// This function creates a new string by copying the byte data,
// ensuring the string is independent of the original byte slice.
// Use this when the byte slice might be modified after conversion.
func BytesToStringSafe(b []byte) string {
	return string(b)
}
