// Package parser provides configuration parsing for command-line arguments.
package parser

import (
	"os"
	"strings"

	"github.com/kitsunium/sdk/pkg/core/config/normalize"
)

// ARGS is a parser for command-line arguments.
// It converts command-line flags into a key-value configuration map.
//
// Supported formats:
//   - --key=value or -key=value
//   - --key value or -key value
//   - --flag (boolean, sets to "true")
//   - Double dash (--) and single dash (-) prefixes
type ARGS struct {
	SkipFirst bool // Skip first argument (usually program name)
}

// NewARGS creates a new command-line argument parser.
//
// Parameters:
//   - skipFirst: Skip the first argument (typically the program name)
//
// Example:
//
//	// Parse args skipping program name
//	parser := NewARGS(true)
//	config, _ := parser.Load()
//
//	// --database-url=localhost becomes {"database.url": "localhost"}
//	// --verbose becomes {"verbose": "true"}
func NewARGS(skipFirst bool) *ARGS {
	return &ARGS{
		SkipFirst: skipFirst,
	}
}

// Type returns the parser type identifier "args".
func (a *ARGS) Type() string {
	return "args"
}

// Load parses command-line arguments from os.Args.
// Automatically handles the skipFirst setting.
//
// Returns a map where:
//   - Keys are normalized (lowercase, dash/underscore to dot)
//   - Boolean flags are set to "true"
//   - Values are preserved as-is
func (a *ARGS) Load() (map[string]string, error) {
	args := os.Args
	start := 0

	if a.SkipFirst && len(args) > 0 {
		start = 1
	}

	if start >= len(args) {
		return make(map[string]string), nil
	}

	return a.ParseArgs(args[start:])
}

// ParseArgs parses a specific list of arguments.
// This is the core parsing method used by Load().
//
// Handles:
//   - Key=value format
//   - Key value format (space-separated)
//   - Boolean flags (no value means "true")
//   - Single and double dash prefixes
func (a *ARGS) ParseArgs(args []string) (map[string]string, error) {
	// Pre-count flags for optimal map allocation
	flagCount := 0
	for _, arg := range args {
		if len(arg) > 0 && arg[0] == '-' {
			flagCount++
		}
	}

	// Allocate map with exact size to prevent rehashing
	config := make(map[string]string, flagCount)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if len(arg) == 0 {
			continue
		}

		if arg[0] == '-' {
			if len(arg) > 1 && arg[1] == '-' {
				arg = arg[2:]
			} else {
				arg = arg[1:]
			}
		} else {
			continue // Skip non-flag arguments
		}

		eqIdx := strings.IndexByte(arg, '=')
		if eqIdx != -1 {
			key := arg[:eqIdx]
			value := arg[eqIdx+1:]
			config[normalize.Key(key)] = normalize.Value(value)
		} else if i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
			config[normalize.Key(arg)] = normalize.Value(args[i+1])
			i++
		} else {
			config[normalize.Key(arg)] = "true"
		}
	}

	return config, nil
}

// ParseArgsStrict parses arguments with strict validation.
// All arguments must start with - or -- prefix.
//
// Returns ErrARGSInvalid if any argument doesn't follow the expected format.
// This is useful for CLI tools that require all arguments to be flags.
func (a *ARGS) ParseArgsStrict(args []string) (map[string]string, error) {
	config := make(map[string]string, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if len(arg) == 0 {
			continue
		}

		if !strings.HasPrefix(arg, "-") {
			return nil, ErrARGSInvalid.Newf("expected flag starting with -, got: %s", arg)
		}

		if arg[0] == '-' {
			if len(arg) > 1 && arg[1] == '-' {
				arg = arg[2:]
			} else {
				arg = arg[1:]
			}
		}

		eqIdx := strings.IndexByte(arg, '=')
		if eqIdx != -1 {
			key := arg[:eqIdx]
			value := arg[eqIdx+1:]
			config[normalize.Key(key)] = normalize.Value(value)
		} else if i+1 < len(args) {
			if strings.HasPrefix(args[i+1], "-") {
				config[normalize.Key(arg)] = "true"
			} else {
				config[normalize.Key(arg)] = normalize.Value(args[i+1])
				i++
			}
		} else {
			config[normalize.Key(arg)] = "true"
		}
	}

	return config, nil
}
