// Package toml — error catalog. The format-specific parse failure is
// registered here so errors.Is comparisons work against a stable
// catalog entry regardless of the wrapped cause.
package toml

import "github.com/kitsunium/sdk/v1/internal/kernel/errs"

// ErrTOMLParse is returned when toml content cannot be decoded.
var ErrTOMLParse = errs.Define(errs.Config{
	Package: "toml",
	Code:    1030,
	Message: "failed to parse TOML",
})
