// Package parser defines interfaces and options for configuration parsers.
package parser

import (
	"io"
)

// Parser is the base interface for all configuration parsers.
// All parsers must implement Type() for identification and Load() for parsing.
type Parser interface {
	// Type returns a unique identifier for the parser type (e.g., "json", "yaml").
	Type() string
	
	// Load parses configuration and returns a flattened key-value map.
	// Keys are normalized to lowercase with dots as separators.
	Load() (map[string]string, error)
}

// FileParser extends Parser with the ability to parse from io.Reader.
// This interface is implemented by file-based parsers (JSON, YAML, TOML, XML, INI).
type FileParser interface {
	Parser
	
	// LoadReader parses configuration from an io.Reader source.
	LoadReader(r io.Reader) (map[string]string, error)
}

// ParserOption is a functional option for configuring parser behavior.
type ParserOption func(*baseParser)

// baseParser contains common configuration options for parsers.
// These options can be used to tune performance characteristics.
type baseParser struct {
	bufferSize int  // Buffer size for reading operations
	usePool    bool // Whether to use buffer pooling (deprecated, kept for compatibility)
}

// WithBufferSize sets the buffer size for reading operations.
// Larger buffers can improve performance for large files.
//
// Default: 8192 bytes
func WithBufferSize(size int) ParserOption {
	return func(p *baseParser) {
		p.bufferSize = size
	}
}

// WithPool enables or disables buffer pooling.
// Note: Buffer pooling has been found to decrease performance in most cases
// and is disabled by default. This option is kept for backward compatibility.
//
// Default: false
func WithPool(enabled bool) ParserOption {
	return func(p *baseParser) {
		p.usePool = enabled
	}
}