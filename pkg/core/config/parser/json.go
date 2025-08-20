// Package parser provides configuration parsing utilities for various formats.
package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/kitsunium/sdk/pkg/core/config/normalize"
)

// JSON is a JSON configuration parser that flattens nested
// structures into a flat key-value map with dot-separated keys.
// It supports all JSON types including arrays and nested objects.
//
// Example:
//
//	parser := NewJSON("config.json")
//	config, err := parser.Load()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Access nested values with dot notation
//	dbHost := config["database.host"]
//	port := config["server.port"]
type JSON struct {
	Path    string
	options baseParser
}

// NewJSON creates a new JSON parser instance.
//
// Options can be provided to customize the parser behavior:
//   - WithBufferSize(size): Set the buffer size for reading (default: 8192)
//   - WithPool(enabled): Enable/disable buffer pooling (default: false)
//
// Example:
//
//	parser := NewJSON("config.json", WithBufferSize(16384))
func NewJSON(path string, opts ...ParserOption) *JSON {
	j := &JSON{
		Path: path,
		options: baseParser{
			bufferSize: 8192,
			usePool:    false, // Pool doesn't improve performance for JSON
		},
	}

	for _, opt := range opts {
		opt(&j.options)
	}

	return j
}

// Type returns the parser type identifier "json".
func (j *JSON) Type() string {
	return "json"
}

// Load reads and parses a JSON file from disk.
// The file path must have a .json extension.
//
// Returns a flattened map where nested keys are joined with dots:
//   - {"a": {"b": "c"}} becomes {"a.b": "c"}
//   - Arrays are indexed: {"arr": [1, 2]} becomes {"arr.0": "1", "arr.1": "2"}
//
// Returns an error if:
//   - The file extension is not .json
//   - The file cannot be read
//   - The JSON is malformed
func (j *JSON) Load() (map[string]string, error) {
	if ext := path.Ext(j.Path); ext != ".json" {
		return nil, ErrInvalidExtension.Newf("expected .json, got %s", ext)
	}

	data, err := os.ReadFile(j.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound.Wrap(err).WithTag("path", j.Path)
		}
		return nil, ErrReadFailed.Wrap(err).WithTag("path", j.Path)
	}

	return j.LoadBytes(data)
}

// LoadReader parses JSON from an io.Reader.
//
// This method reads all data from the reader into memory before parsing.
// For large files, consider using Load() which reads directly from disk.
func (j *JSON) LoadReader(r io.Reader) (map[string]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, ErrReadFailed.Wrap(err).WithTag("parser", "json")
	}

	return j.LoadBytes(data)
}

// fastFloat64ToString converts float64 to string.
// Integers are formatted without decimal points.
func fastFloat64ToString(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// normalizeJSONNumber converts json.Number to string.
// Tries to preserve exact representation for integers,
// and converts scientific notation appropriately.
func normalizeJSONNumber(n json.Number) string {
	// First try as int64 to preserve exact integers
	if i, err := n.Int64(); err == nil {
		return strconv.FormatInt(i, 10)
	}

	// Then try as float64
	if f, err := n.Float64(); err == nil {
		return fastFloat64ToString(f)
	}

	// Fallback to string representation
	return string(n)
}

// LoadBytes parses JSON from a byte slice.
//
// This method:
//  1. Unmarshals JSON into a map structure
//  2. Flattens nested structures using a stack-based approach
//  3. Normalizes all keys to lowercase with dots instead of underscores
//  4. Converts all values to strings
//
// Example:
//
//	data := []byte(`{"db": {"host": "localhost"}}`)
//	config, err := parser.LoadBytes(data)
//	// config["db.host"] == "localhost"
func (j *JSON) LoadBytes(data []byte) (map[string]string, error) {
	var config map[string]any

	// Try standard unmarshal first (faster path)
	if err := json.Unmarshal(data, &config); err == nil {
		// Fast path succeeded, process the result
		result, processErr := j.processConfig(config, len(data))
		if processErr == nil {
			return result, nil
		}
		// If processing fails due to precision issues, fall back to UseNumber
	}

	// Fallback: Use decoder with UseNumber to preserve large integers
	config = nil // Reset config
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&config); err != nil {
		return nil, ErrJSONParse.Wrap(err).WithDetail("size", len(data))
	}

	return j.processConfig(config, len(data))
}

// processConfig converts the parsed JSON config to a flat string map
func (j *JSON) processConfig(config map[string]any, dataSize int) (map[string]string, error) {
	// Better size estimation: ~1 key per 30 bytes of JSON
	estimatedSize := max(dataSize/30, 16)
	result := make(map[string]string, estimatedSize)

	// Pre-allocate string builder for key construction
	var keyBuilder strings.Builder
	keyBuilder.Grow(256)

	// Use stack for iteration instead of recursion to improve performance
	type stackItem struct {
		data   map[string]any
		prefix string
	}

	// Pre-size stack based on typical nesting depth
	stack := make([]stackItem, 1, 8)
	stack[0] = stackItem{data: config, prefix: ""}

	for len(stack) > 0 {
		// Pop from stack
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		prefixLen := len(current.prefix)
		needsDot := prefixLen > 0

		for key, value := range current.data {
			// Optimized key construction
			keyBuilder.Reset()
			if needsDot {
				keyBuilder.WriteString(current.prefix)
				keyBuilder.WriteByte('.')
			}
			keyBuilder.WriteString(key)
			fullKey := keyBuilder.String()

			switch v := value.(type) {
			case string:
				result[normalize.Key(fullKey)] = normalize.Value(v)
			case map[string]any:
				stack = append(stack, stackItem{data: v, prefix: fullKey})
			case []any:
				for i, item := range v {
					// Optimized array key construction
					arrayKeyBuilder := &keyBuilder
					arrayKeyBuilder.Reset()
					arrayKeyBuilder.WriteString(fullKey)
					arrayKeyBuilder.WriteByte('.')
					arrayKeyBuilder.WriteString(strconv.Itoa(i))
					itemKey := arrayKeyBuilder.String()

					if subMap, ok := item.(map[string]any); ok {
						stack = append(stack, stackItem{data: subMap, prefix: itemKey})
					} else {
						result[normalize.Key(itemKey)] = normalizeAnyValue(item)
					}
				}
			case json.Number:
				result[normalize.Key(fullKey)] = normalizeJSONNumber(v)
			case float64:
				result[normalize.Key(fullKey)] = fastFloat64ToString(v)
			case bool:
				if v {
					result[normalize.Key(fullKey)] = "true"
				} else {
					result[normalize.Key(fullKey)] = "false"
				}
			case nil:
				result[normalize.Key(fullKey)] = ""
			default:
				// Fallback for any unexpected types
				result[normalize.Key(fullKey)] = normalize.Value(fmt.Sprint(v))
			}
		}
	}

	return result, nil
}

// normalizeAnyValue converts any JSON value type to its string representation.
// Handles all JSON types including those from Decoder.UseNumber().
func normalizeAnyValue(v any) string {
	switch val := v.(type) {
	case string:
		return normalize.Value(val)
	case json.Number:
		return normalizeJSONNumber(val)
	case float64:
		return fastFloat64ToString(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return ""
	case int64:
		return strconv.FormatInt(val, 10)
	default:
		// This handles any unexpected types
		return normalize.Value(fmt.Sprint(val))
	}
}
