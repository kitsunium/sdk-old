// Package yaml — error catalog. The format-specific parse failure is
// registered here so errors.Is comparisons work against a stable
// catalog entry regardless of the wrapped cause.
package yaml

import "github.com/kitsunium/sdk/v1/internal/kernel/errs"

// ErrYAMLParse is returned when yaml content cannot be decoded.
var ErrYAMLParse = errs.Define(errs.Config{
	Package: "yaml",
	Code:    1020,
	Message: "failed to parse YAML",
})
