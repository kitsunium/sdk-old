// Package parser provides a flexible configuration parsing framework with support
// for multiple formats including JSON, YAML, TOML, XML, INI, environment variables,
// and command-line arguments.
//
// The package defines common interfaces for all parsers and provides implementations
// that normalize configuration data into flat key-value maps with dot-notation keys.
// This design allows for easy configuration merging from multiple sources and
// consistent access patterns regardless of the original format.
//
// Example usage:
//
//	// Parse JSON configuration
//	jsonParser := parser.NewJSON("config.json")
//	config, err := jsonParser.Load()
//
//	// Parse environment variables
//	envParser := parser.NewEnv(parser.WithPrefix("APP_"))
//	envConfig, err := envParser.Load()
//
//	// Parse command-line arguments
//	argsParser := parser.NewArgs(os.Args[1:])
//	argsConfig, err := argsParser.Load()
package parser

import (
	"io"
)

// Parser is the base interface for all configuration parsers.
// Implementations must provide a Type identifier and a Load method
// that returns normalized configuration as a flat key-value map.
type Parser interface {
	// Type returns a unique identifier for the parser type (e.g., "json", "yaml").
	Type() string

	// Load parses configuration and returns a flattened key-value map.
	// Keys are normalized to lowercase with dots as separators.
	Load() (map[string]string, error)
}

// FileParser extends Parser with the ability to parse from io.Reader.
// This interface is implemented by file-based parsers (JSON, YAML, TOML, XML, INI)
// and allows for parsing from various sources like files, network streams, or buffers.
type FileParser interface {
	Parser

	// LoadReader parses configuration from an io.Reader source.
	LoadReader(r io.Reader) (map[string]string, error)
}

// ParserOption is a functional option for configuring parser behavior.
// Options can be passed to parser constructors to customize their operation.
type ParserOption func(*baseParser)

// baseParser contains common configuration options shared by all parser implementations.
// These options control performance characteristics like buffer sizes and pooling behavior.
type baseParser struct {
	bufferSize int  // Buffer size for reading operations
	usePool    bool // Whether to use buffer pooling (deprecated, kept for compatibility)
}

// WithBufferSize sets the buffer size for reading operations.
// Larger buffers can improve performance for large files by reducing
// the number of read syscalls. Consider increasing for files > 1MB.
//
// Default: 8192 bytes
// Recommended: 8192 for small files, 65536 for large files
func WithBufferSize(size int) ParserOption {
	return func(p *baseParser) {
		p.bufferSize = size
	}
}

// WithPool enables or disables buffer pooling for memory reuse.
// Note: Buffer pooling has been found to decrease performance in most cases
// due to synchronization overhead and is disabled by default.
// This option is maintained for backward compatibility only.
//
// Default: false
// Deprecated: Buffer pooling is not recommended for general use
func WithPool(enabled bool) ParserOption {
	return func(p *baseParser) {
		p.usePool = enabled
	}
}
