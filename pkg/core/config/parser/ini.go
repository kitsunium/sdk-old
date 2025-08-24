// Package parser provides configuration parsing for INI files.
package parser

import (
	"io"
	"os"
	"path"
	"unsafe"

	"github.com/kitsunium/sdk/pkg/core/config/normalize"
)

// INI is a parser for INI configuration files.
// It supports .ini, .cfg, and .conf file extensions.
//
// The parser handles:
//   - Sections [section]
//   - Key-value pairs with = or : separators
//   - Comments with # or ; prefixes
//   - Quoted values (single or double quotes)
//   - Whitespace trimming
//   - Section-prefixed keys (section.key)
type INI struct {
	Path    string
	options baseParser
}

// NewINI creates a new INI parser instance.
//
// Parameters:
//   - path: Path to the INI file to parse
//   - opts: Optional parser configuration options
//
// Example:
//
//	parser := NewINI("config.ini")
//	config, err := parser.Load()
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewINI(path string, opts ...ParserOption) *INI {
	i := &INI{
		Path: path,
		options: baseParser{
			bufferSize: 8192,
			usePool:    false,
		},
	}

	for _, opt := range opts {
		opt(&i.options)
	}

	return i
}

// Type returns the parser type identifier "ini".
func (i *INI) Type() string {
	return "ini"
}

// Load reads and parses an INI file from disk.
// Validates that the file has a .ini, .cfg, or .conf extension.
//
// Returns a flattened map where section keys are prefixed:
//   - [database] host=localhost becomes {"database.host": "localhost"}
//   - Global keys (before any section) have no prefix
//
// Returns an error if:
//   - The file extension is not .ini, .cfg, or .conf
//   - The file cannot be read
func (i *INI) Load() (map[string]string, error) {
	if ext := path.Ext(i.Path); ext != ".ini" && ext != ".cfg" && ext != ".conf" {
		return nil, ErrInvalidExtension.Newf("expected .ini, .cfg or .conf, got %s", ext)
	}

	data, err := os.ReadFile(i.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound.Wrap(err).WithTag("path", i.Path)
		}
		return nil, ErrReadFailed.Wrap(err).WithTag("path", i.Path)
	}

	return i.LoadBytes(data)
}

// LoadReader parses INI from an io.Reader.
// This method reads all data into memory before parsing.
func (i *INI) LoadReader(r io.Reader) (map[string]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, ErrReadFailed.Wrap(err).WithTag("parser", "ini")
	}
	return i.LoadBytes(data)
}

// iniBytesToString converts bytes to string without allocation.
// Uses unsafe pointer casting. This is safe because we don't modify the underlying bytes.
func iniBytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// LoadBytes parses INI from a byte slice.
// Implements a single-pass parser with direct byte manipulation.
func (i *INI) LoadBytes(data []byte) (map[string]string, error) {
	config := make(map[string]string, 128)
	var currentSection string
	lineStart := 0

	for idx := 0; idx < len(data); idx++ {
		if data[idx] == '\n' || idx == len(data)-1 {
			lineEnd := i.findLineEnd(idx, data)
			if lineEnd > lineStart {
				i.processLine(data[lineStart:lineEnd], &currentSection, config)
			}
			lineStart = idx + 1
		}
	}
	return config, nil
}

// findLineEnd determines the end of the current line.
func (i *INI) findLineEnd(idx int, data []byte) int {
	if idx == len(data)-1 && data[idx] != '\n' {
		return idx + 1
	}
	return idx
}

// processLine processes a single INI line.
func (i *INI) processLine(line []byte, currentSection *string, config map[string]string) {
	line = i.trimBytes(line)
	if len(line) == 0 || line[0] == '#' || line[0] == ';' {
		return
	}

	if line[0] == '[' && line[len(line)-1] == ']' {
		*currentSection = normalize.Key(iniBytesToString(line[1 : len(line)-1]))
	} else {
		i.parseKeyValue(line, *currentSection, config)
	}
}

// parseKeyValue parses a key-value pair from a line.
func (i *INI) parseKeyValue(line []byte, section string, config map[string]string) {
	sepIdx := i.findSeparator(line)
	if sepIdx == -1 {
		return
	}

	key := i.trimRight(line[:sepIdx])
	value := i.processValue(line[sepIdx+1:])

	keyStr := iniBytesToString(key)
	if section != "" {
		keyStr = section + "." + keyStr
	}
	config[normalize.Key(keyStr)] = normalize.Value(iniBytesToString(value))
}

// findSeparator finds the position of '=' or ':' in a line.
func (i *INI) findSeparator(line []byte) int {
	for j := 0; j < len(line); j++ {
		if line[j] == '=' || line[j] == ':' {
			return j
		}
	}
	return -1
}

// processValue trims and unquotes a value.
func (i *INI) processValue(value []byte) []byte {
	value = i.trimBytes(value)
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

// trimBytes removes leading and trailing whitespace from bytes.
func (i *INI) trimBytes(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t' || b[0] == '\r') {
		b = b[1:]
	}
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}

// trimRight removes trailing whitespace from bytes.
func (i *INI) trimRight(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t') {
		b = b[:len(b)-1]
	}
	return b
}
