// Package xml — error catalog. The format-specific parse failure is
// registered here so errors.Is comparisons work against a stable
// catalog entry regardless of the wrapped cause.
package xml

import "github.com/kitsunium/sdk/v1/internal/kernel/errs"

// ErrXMLParse is returned when xml content cannot be decoded.
var ErrXMLParse = errs.Define(errs.Config{
	Package: "xml",
	Code:    1040,
	Message: "failed to parse XML",
})
