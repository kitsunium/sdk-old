// Package parser provides error definitions for configuration parsers.
// All errors are defined using the kerror package for enhanced tracking and observability.
package parser

import "github.com/kitsunium/sdk/pkg/kernel/kerror"

// Error definitions for the parser package.
// These errors provide structured error handling with unique codes for each error type.
var (
	// ErrInvalidExtension is returned when a file has an unsupported extension.
	// Each parser validates specific extensions (e.g., .json, .yaml, .toml).
	ErrInvalidExtension = kerror.Define(kerror.KConfig{
		Code:    1001,
		Message: "invalid file extension",
	})
	// ErrFileNotFound is returned when a configuration file does not exist.
	// This error wraps the underlying OS error for additional context.
	ErrFileNotFound = kerror.Define(kerror.KConfig{
		Code:    1002,
		Message: "file not found",
	})
	
	// ErrReadFailed is returned when a file or reader cannot be read.
	// This typically occurs due to I/O errors or permission issues.
	ErrReadFailed = kerror.Define(kerror.KConfig{
		Code:    1003,
		Message: "failed to read file",
	})

	// ErrJSONParse is returned when JSON content is malformed or invalid.
	// The error wraps the underlying json.Unmarshal error for details.
	ErrJSONParse = kerror.Define(kerror.KConfig{
		Code:    1010,
		Message: "failed to parse JSON",
	})

	// ErrYAMLParse is returned when YAML content is malformed or invalid.
	// Supports both .yaml and .yml file extensions.
	ErrYAMLParse = kerror.Define(kerror.KConfig{
		Code:    1020,
		Message: "failed to parse YAML",
	})

	// ErrTOMLParse is returned when TOML content is malformed or invalid.
	// Uses the pelletier/go-toml/v2 library for parsing.
	ErrTOMLParse = kerror.Define(kerror.KConfig{
		Code:    1030,
		Message: "failed to parse TOML",
	})

	// ErrXMLParse is returned when XML content is malformed or invalid.
	// Handles attributes, nested elements, and CDATA sections.
	ErrXMLParse = kerror.Define(kerror.KConfig{
		Code:    1040,
		Message: "failed to parse XML",
	})

	// ErrINIParse is returned when INI content is malformed or invalid.
	// Supports .ini, .cfg, and .conf file extensions.
	ErrINIParse = kerror.Define(kerror.KConfig{
		Code:    1050,
		Message: "failed to parse INI",
	})

	// ErrENVParse is returned when environment variables cannot be parsed.
	// Supports optional prefix filtering for variable selection.
	ErrENVParse = kerror.Define(kerror.KConfig{
		Code:    1060,
		Message: "failed to parse environment variable",
	})

	// ErrARGSParse is returned when command-line arguments cannot be parsed.
	// Supports both --key=value and --key value formats.
	ErrARGSParse = kerror.Define(kerror.KConfig{
		Code:    1070,
		Message: "failed to parse arguments",
	})
	
	// ErrARGSInvalid is returned when arguments don't follow expected format.
	// In strict mode, all arguments must start with - or --.
	ErrARGSInvalid = kerror.Define(kerror.KConfig{
		Code:    1071,
		Message: "invalid argument format",
	})
)