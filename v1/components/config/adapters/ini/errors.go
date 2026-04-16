// Package ini — error catalog. The format-specific parse failure is
// registered here so errors.Is comparisons work against a stable
// catalog entry regardless of the wrapped cause.
package ini

import "github.com/kitsunium/sdk/v1/internal/kernel/errs"

// ErrINIParse is returned when ini content cannot be decoded.
var ErrINIParse = errs.Define(errs.Config{
	Package: "ini",
	Code:    1050,
	Message: "failed to parse INI",
})
