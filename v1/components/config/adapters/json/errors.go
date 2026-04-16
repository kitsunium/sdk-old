// Package json — error catalog. The format-specific parse failure is
// registered here so errors.Is comparisons work against a stable
// catalog entry regardless of the wrapped cause.
package json

import "github.com/kitsunium/sdk/v1/internal/kernel/errs"

// ErrJSONParse is returned when json content cannot be decoded.
var ErrJSONParse = errs.Define(errs.Config{
	Package: "json",
	Code:    1010,
	Message: "failed to parse JSON",
})
