// Package parser defines the abstract configuration parser contract.
//
// Concrete implementations live in subpackages under
// components/config/adapters/ — one per source format (json, yaml,
// toml, ini, xml, env, args). Each adapter satisfies Parser and, for
// file-based formats, FileParser.
//
// All adapters produce a map[string]string with keys normalized via
// internal/core/normalize: lowercase, dot-separated, `_` and `-`
// collapsed to `.`.
package parser

import (
	"io"

	"github.com/kitsunium/sdk/v1/internal/kernel/errs"
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

// Options holds the shared configuration every adapter can tune via
// With* helpers. Adapters embed or reference an Options value and
// apply functional options against it at construction time.
type Options struct {
	// BufferSize is the I/O buffer size for reading sources. Larger
	// buffers reduce syscall count for large files.
	BufferSize int
	// UsePool routes reads through internal/core/pool. Disabled by
	// default — pooling has shown neutral-to-negative impact under
	// benchmark for file-sized payloads.
	UsePool bool
}

// Option is a functional option for adapter construction.
type Option func(*Options)

// WithBufferSize sets the buffer size for reading operations.
// Default: 8192 bytes. Recommended: 8192 for small files, 65536 for
// large files.
func WithBufferSize(size int) Option {
	return func(o *Options) {
		o.BufferSize = size
	}
}

// WithPool enables or disables buffer pooling for memory reuse.
// Default: false. Kept for adapters that document a perf win under
// their own benchmarks.
func WithPool(enabled bool) Option {
	return func(o *Options) {
		o.UsePool = enabled
	}
}

// Shared error catalog — errors that any adapter may raise. Adapters
// MAY re-wrap these with their own format-specific codes, but should
// prefer reusing these when the failure is format-agnostic (extension
// mismatch, missing file, I/O failure).
var (
	// ErrInvalidExtension is returned when a file has an unsupported extension.
	ErrInvalidExtension = errs.Define(errs.Config{
		Package: "parser",
		Code:    1001,
		Message: "invalid file extension",
	})
	// ErrFileNotFound is returned when a configuration file does not exist.
	ErrFileNotFound = errs.Define(errs.Config{
		Package: "parser",
		Code:    1002,
		Message: "file not found",
	})
	// ErrReadFailed is returned when a file or reader cannot be read.
	ErrReadFailed = errs.Define(errs.Config{
		Package: "parser",
		Code:    1003,
		Message: "failed to read file",
	})
)
