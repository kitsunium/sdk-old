// Package parser provides configuration parsing for command-line arguments.
package parser

import (
	"os"
	"strconv"
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
	flagCount := a.countFlags(args)
	config := make(map[string]string, flagCount)

	for i := 0; i < len(args); i++ {
		key, value, skip := a.processSingleArg(args, i)
		if key != "" {
			config[normalize.Key(key)] = normalize.Value(value)
			i += skip
		}
	}

	return config, nil
}

func (a *ARGS) countFlags(args []string) int {
	count := 0
	for _, arg := range args {
		if len(arg) > 0 && arg[0] == '-' {
			count++
		}
	}
	return count
}

func (a *ARGS) processSingleArg(args []string, index int) (string, string, int) {
	arg := args[index]
	if len(arg) == 0 || arg[0] != '-' {
		return "", "", 0
	}

	// Fast path for stripping dashes
	if len(arg) > 1 && arg[1] == '-' {
		arg = arg[2:]
	} else {
		arg = arg[1:]
	}

	// Check for key=value format (most common case)
	if eqIdx := strings.IndexByte(arg, '='); eqIdx != -1 {
		return arg[:eqIdx], arg[eqIdx+1:], 0
	}

	// Check if next arg is a value
	if index+1 < len(args) && a.isValue(args[index+1]) {
		return arg, args[index+1], 1
	}

	// Boolean flag
	return arg, "true", 0
}

func (a *ARGS) isValue(arg string) bool {
	return len(arg) > 0 && arg[0] != '-'
}

// isNegativeNumber checks if a string represents a negative number.
func isNegativeNumber(s string) bool {
	if len(s) < 2 || s[0] != '-' {
		return false
	}
	// Try to parse as float (covers both int and float)
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
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

		key, value, consumed := a.parseArgument(arg, args, i)
		config[normalize.Key(key)] = normalize.Value(value)
		i += consumed
	}

	return config, nil
}

// parseArgument parses a single argument and returns the key, value, and whether next arg was consumed.
func (a *ARGS) parseArgument(arg string, args []string, index int) (key, value string, consumed int) {
	// Strip leading dashes
	key = a.stripDashes(arg)

	// Check for key=value format
	if eqIdx := strings.IndexByte(key, '='); eqIdx != -1 {
		return key[:eqIdx], key[eqIdx+1:], 0
	}

	// Handle standalone flag or flag with value
	return a.handleFlagValue(key, args, index)
}

// stripDashes removes leading dashes from the argument.
func (a *ARGS) stripDashes(arg string) string {
	if len(arg) > 1 && arg[1] == '-' {
		return arg[2:]
	}
	return arg[1:]
}

// handleFlagValue determines the value for a flag without '='.
func (a *ARGS) handleFlagValue(key string, args []string, index int) (string, string, int) {
	// No next argument available
	if index+1 >= len(args) {
		return key, "true", 0
	}

	nextArg := args[index+1]

	// Next arg is a negative number (treat as value)
	if isNegativeNumber(nextArg) {
		return key, nextArg, 1
	}

	// Next arg is another flag
	if strings.HasPrefix(nextArg, "-") {
		return key, "true", 0
	}

	// Next arg is a value
	return key, nextArg, 1
}
