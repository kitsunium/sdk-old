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
	
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' || i == len(data)-1 {
			lineEnd := i
			if i == len(data)-1 && data[i] != '\n' {
				lineEnd = i + 1
			}
			
			// Process line
			if lineEnd > lineStart {
				line := data[lineStart:lineEnd]
				
				// Trim line
				for len(line) > 0 && (line[0] == ' ' || line[0] == '\t' || line[0] == '\r') {
					line = line[1:]
				}
				for len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t' || line[len(line)-1] == '\r') {
					line = line[:len(line)-1]
				}
				
				if len(line) > 0 && line[0] != '#' && line[0] != ';' {
					if line[0] == '[' && line[len(line)-1] == ']' {
						// Section
						currentSection = normalize.Key(iniBytesToString(line[1 : len(line)-1]))
					} else {
						// Key-value
						sepIdx := -1
						for j := 0; j < len(line); j++ {
							if line[j] == '=' || line[j] == ':' {
								sepIdx = j
								break
							}
						}
						
						if sepIdx != -1 {
							key := line[:sepIdx]
							value := line[sepIdx+1:]
							
							// Trim key
							for len(key) > 0 && (key[len(key)-1] == ' ' || key[len(key)-1] == '\t') {
								key = key[:len(key)-1]
							}
							
							// Trim value
							for len(value) > 0 && (value[0] == ' ' || value[0] == '\t') {
								value = value[1:]
							}
							for len(value) > 0 && (value[len(value)-1] == ' ' || value[len(value)-1] == '\t') {
								value = value[:len(value)-1]
							}
							
							// Handle quotes
							if len(value) >= 2 {
								first, last := value[0], value[len(value)-1]
								if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
									value = value[1 : len(value)-1]
								}
							}
							
							keyStr := iniBytesToString(key)
							if currentSection != "" {
								keyStr = currentSection + "." + keyStr
							}
							
							config[normalize.Key(keyStr)] = normalize.Value(iniBytesToString(value))
						}
					}
				}
			}
			
			lineStart = i + 1
		}
	}
	
	return config, nil
}
