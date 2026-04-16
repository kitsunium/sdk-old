// Package parser provides configuration parsing for TOML files.
package parser

import (
	"io"
	"os"
	"path"

	"github.com/kitsunium/sdk/internal/core/config/normalize"
	"github.com/pelletier/go-toml/v2"
)

// TOML is a parser for TOML (Tom's Obvious, Minimal Language) configuration files.
// It flattens nested structures into a dot-separated key-value map.
//
// The parser handles all TOML data types including:
//   - Basic strings and literal strings
//   - Integers, floats, booleans
//   - Dates and times
//   - Arrays and inline tables
//   - Nested tables and table arrays
type TOML struct {
	Path    string
	options baseParser
}

// NewTOML creates a new TOML parser instance.
//
// Parameters:
//   - path: Path to the TOML file to parse
//   - opts: Optional parser configuration options
//
// Example:
//
//	parser := NewTOML("config.toml")
//	config, err := parser.Load()
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewTOML(path string, opts ...ParserOption) *TOML {
	t := &TOML{
		Path: path,
		options: baseParser{
			bufferSize: 8192,
			usePool:    false,
		},
	}

	for _, opt := range opts {
		opt(&t.options)
	}

	return t
}

// Type returns the parser type identifier "toml".
func (t *TOML) Type() string {
	return "toml"
}

// Load reads and parses a TOML file from disk.
// Validates that the file has a .toml extension.
//
// Returns a flattened map where nested keys are joined with dots:
//   - [database] host = "localhost" becomes {"database.host": "localhost"}
//   - Arrays are indexed: servers = ["a", "b"] becomes {"servers.0": "a", "servers.1": "b"}
//
// Returns an error if:
//   - The file extension is not .toml
//   - The file cannot be read
//   - The TOML is malformed
func (t *TOML) Load() (map[string]string, error) {
	if ext := path.Ext(t.Path); ext != ".toml" {
		return nil, ErrInvalidExtension.Newf("expected .toml, got %s", ext)
	}

	file, err := os.Open(t.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound.Wrap(err).WithTag("path", t.Path)
		}
		return nil, ErrReadFailed.Wrap(err).WithTag("path", t.Path)
	}
	defer file.Close()

	return t.LoadReader(file)
}

// LoadReader parses TOML from an io.Reader.
// This method reads all data into memory before parsing.
func (t *TOML) LoadReader(r io.Reader) (map[string]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, ErrReadFailed.Wrap(err).WithTag("parser", "toml")
	}

	return t.LoadBytes(data)
}

// LoadBytes parses TOML from a byte slice.
// Uses the pelletier/go-toml/v2 library for parsing and normalize.Map for flattening.
func (t *TOML) LoadBytes(data []byte) (map[string]string, error) {
	var config map[string]any

	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, ErrTOMLParse.Wrap(err).WithDetail("size", len(data))
	}

	return normalize.Map(config), nil
}
