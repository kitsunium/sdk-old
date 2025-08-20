// Package parser provides configuration parsing for YAML files.
package parser

import (
	"io"
	"os"
	"path"

	"github.com/kitsunium/sdk/pkg/core/config/normalize"
	"gopkg.in/yaml.v3"
)

// YAML is a parser for YAML configuration files.
// It supports both .yaml and .yml file extensions and flattens nested
// structures into a dot-separated key-value map.
//
// The parser handles all YAML data types including:
//   - Scalars (strings, numbers, booleans)
//   - Maps and nested maps
//   - Arrays and sequences
//   - Anchors and aliases
//   - Multi-line strings
type YAML struct {
	Path    string
	options baseParser
}

// NewYAML creates a new YAML parser instance.
//
// Parameters:
//   - path: Path to the YAML file to parse
//   - opts: Optional parser configuration options
//
// Example:
//
//	parser := NewYAML("config.yaml")
//	config, err := parser.Load()
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewYAML(path string, opts ...ParserOption) *YAML {
	y := &YAML{
		Path: path,
		options: baseParser{
			bufferSize: 8192,
			usePool:    false,
		},
	}

	for _, opt := range opts {
		opt(&y.options)
	}

	return y
}

// Type returns the parser type identifier "yaml".
func (y *YAML) Type() string {
	return "yaml"
}

// Load reads and parses a YAML file from disk.
// Validates that the file has a .yaml or .yml extension.
//
// Returns a flattened map where nested keys are joined with dots:
//   - {"database": {"host": "localhost"}} becomes {"database.host": "localhost"}
//   - Arrays are indexed: {"servers": ["a", "b"]} becomes {"servers.0": "a", "servers.1": "b"}
//
// Returns an error if:
//   - The file extension is not .yaml or .yml
//   - The file cannot be read
//   - The YAML is malformed
func (y *YAML) Load() (map[string]string, error) {
	ext := path.Ext(y.Path)
	if ext != ".yaml" && ext != ".yml" {
		return nil, ErrInvalidExtension.Newf("expected .yaml or .yml, got %s", ext)
	}

	file, err := os.Open(y.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound.Wrap(err).WithTag("path", y.Path)
		}
		return nil, ErrReadFailed.Wrap(err).WithTag("path", y.Path)
	}
	defer file.Close()

	return y.LoadReader(file)
}

// LoadReader parses YAML from an io.Reader.
// This method reads all data into memory before parsing.
func (y *YAML) LoadReader(r io.Reader) (map[string]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, ErrReadFailed.Wrap(err).WithTag("parser", "yaml")
	}

	return y.LoadBytes(data)
}

// LoadBytes parses YAML from a byte slice.
// Uses the yaml.v3 library for parsing and normalize.Map for flattening.
func (y *YAML) LoadBytes(data []byte) (map[string]string, error) {
	var config map[string]any

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, ErrYAMLParse.Wrap(err).WithDetail("size", len(data))
	}

	return normalize.Map(config), nil
}
