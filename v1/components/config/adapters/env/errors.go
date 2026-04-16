// Package env — error catalog. The format-specific parse failure is
// registered here so errors.Is comparisons work against a stable
// catalog entry regardless of the wrapped cause.
package env

import "github.com/kitsunium/sdk/v1/internal/kernel/errs"

// ErrENVParse is returned when env content cannot be decoded.
var ErrENVParse = errs.Define(errs.Config{
	Package: "env",
	Code:    1060,
	Message: "failed to parse environment variable",
})
