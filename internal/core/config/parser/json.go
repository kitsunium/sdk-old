package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/kitsunium/sdk/internal/core/config/normalize"
)

const (
	// MaxJSONSize defines the maximum size of JSON input (10MB)
	MaxJSONSize = 10 * 1024 * 1024
	// MaxJSONDepth defines maximum nesting depth to prevent stack overflow
	MaxJSONDepth = 100
)

// JSON implements a high-performance JSON configuration parser that flattens
// nested structures into a flat key-value map with dot-separated keys.
// It supports all JSON types including arrays, nested objects, and preserves
// numeric precision using json.Number.
//
// Features:
//   - Zero-copy parsing with bytes.Reader
//   - Stack-based iteration to avoid recursion overhead
//   - Pre-allocated data structures for performance
//   - Protection against malicious inputs (size and depth limits)
//   - Precise number handling without floating-point errors
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

// fastFloat64ToString converts float64 to string with optimal formatting.
// Integers are formatted without decimal points to maintain readability.
// Uses 'g' format for automatic precision selection on floating-point values.
func fastFloat64ToString(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// normalizeJSONNumber converts json.Number to string while preserving precision.
// Returns the original string representation to avoid floating-point errors
// with large integers or precise decimal values.
func normalizeJSONNumber(n json.Number) string {
	// Return the original string representation to preserve precision
	// This avoids issues with large integers or precise decimals
	return n.String()
}

// LoadBytes parses JSON from a byte slice.
//
// This method:
//  1. Validates JSON size to prevent DoS attacks
//  2. Unmarshals JSON into a map structure
//  3. Flattens nested structures using a stack-based approach
//  4. Normalizes all keys to lowercase with dots instead of underscores
//  5. Converts all values to strings
//
// Example:
//
//	data := []byte(`{"db": {"host": "localhost"}}`)
//	config, err := parser.LoadBytes(data)
//	// config["db.host"] == "localhost"
func (j *JSON) LoadBytes(data []byte) (map[string]string, error) {
	// Validate JSON size to prevent DoS attacks
	if len(data) > MaxJSONSize {
		return nil, ErrJSONParse.Newf("JSON size exceeds maximum allowed: %d > %d bytes", len(data), MaxJSONSize)
	}

	if len(data) == 0 {
		return nil, ErrJSONParse.Newf("empty JSON input")
	}

	var config map[string]any

	// Use bytes.Reader for better performance and to avoid string allocation
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	if err := decoder.Decode(&config); err != nil {
		return nil, ErrJSONParse.Wrap(err).WithDetail("size", len(data))
	}

	// Check for trailing data by attempting to decode another value
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, ErrJSONParse.Newf("trailing data after JSON value")
	}

	return j.processConfig(config, len(data))
}

// processConfig converts the parsed JSON config to a flat string map.
// Uses stack-based iteration for better performance and memory efficiency
// compared to recursive approaches.
// stackItem represents an item in the processing stack.
type stackItem struct {
	data   map[string]any
	prefix string
	depth  int
}

// jsonProcessContext holds context for JSON processing to reduce parameter count
type jsonProcessContext struct {
	depth      int
	stack      *[]stackItem
	result     map[string]string
	keyBuilder *strings.Builder
}

func (j *JSON) processConfig(config map[string]any, dataSize int) (map[string]string, error) {
	estimatedSize := max(dataSize/30, 16)
	result := make(map[string]string, estimatedSize)

	var keyBuilder strings.Builder
	keyBuilder.Grow(256)

	stack := []stackItem{{data: config, prefix: "", depth: 0}}

	ctx := &jsonProcessContext{
		depth:      0,
		stack:      &stack,
		result:     result,
		keyBuilder: &keyBuilder,
	}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		ctx.depth = current.depth

		if err := j.processStackItem(current, ctx); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// processStackItem processes a single stack item.
func (j *JSON) processStackItem(current stackItem, ctx *jsonProcessContext) error {
	needsDot := len(current.prefix) > 0

	for key, value := range current.data {
		fullKey := j.buildKey(ctx.keyBuilder, current.prefix, key, needsDot)

		if err := j.processValue(fullKey, value, ctx); err != nil {
			return err
		}
	}
	return nil
}

// buildKey constructs a full key path.
func (j *JSON) buildKey(keyBuilder *strings.Builder, prefix, key string, needsDot bool) string {
	keyBuilder.Reset()
	if needsDot {
		keyBuilder.WriteString(prefix)
		keyBuilder.WriteByte('.')
	}
	keyBuilder.WriteString(key)
	return keyBuilder.String()
}

// processValue processes a single value based on its type.
func (j *JSON) processValue(fullKey string, value any, ctx *jsonProcessContext) error {
	switch v := value.(type) {
	case string:
		ctx.result[normalize.Key(fullKey)] = normalize.Value(v)
	case map[string]any:
		return j.processMap(fullKey, v, ctx)
	case []any:
		return j.processArray(fullKey, v, ctx)
	case json.Number:
		ctx.result[normalize.Key(fullKey)] = normalizeJSONNumber(v)
	case float64:
		ctx.result[normalize.Key(fullKey)] = fastFloat64ToString(v)
	case bool:
		ctx.result[normalize.Key(fullKey)] = strconv.FormatBool(v)
	case nil:
		ctx.result[normalize.Key(fullKey)] = ""
	default:
		ctx.result[normalize.Key(fullKey)] = normalize.Value(fmt.Sprint(v))
	}
	return nil
}

// processMap handles nested map processing.
func (j *JSON) processMap(fullKey string, data map[string]any, ctx *jsonProcessContext) error {
	if ctx.depth >= MaxJSONDepth {
		return ErrJSONParse.Newf("JSON nesting depth exceeds maximum: %d", MaxJSONDepth)
	}
	*ctx.stack = append(*ctx.stack, stackItem{data: data, prefix: fullKey, depth: ctx.depth + 1})
	return nil
}

// processArray handles array processing.
func (j *JSON) processArray(fullKey string, arr []any, ctx *jsonProcessContext) error {
	for i, item := range arr {
		itemKey := j.buildArrayKey(ctx.keyBuilder, fullKey, i)

		if subMap, ok := item.(map[string]any); ok {
			if err := j.processMap(itemKey, subMap, ctx); err != nil {
				return err
			}
		} else {
			ctx.result[normalize.Key(itemKey)] = normalizeAnyValue(item)
		}
	}
	return nil
}

// buildArrayKey constructs a key for an array element.
func (j *JSON) buildArrayKey(keyBuilder *strings.Builder, fullKey string, index int) string {
	keyBuilder.Reset()
	keyBuilder.WriteString(fullKey)
	keyBuilder.WriteByte('.')
	keyBuilder.WriteString(strconv.Itoa(index))
	return keyBuilder.String()
}

// normalizeAnyValue converts any JSON value type to its string representation.
// Handles all JSON types including json.Number from Decoder.UseNumber(),
// ensuring type safety and consistent string conversion.
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
