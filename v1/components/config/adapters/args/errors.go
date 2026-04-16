// Package args — error catalog. Registers catalog entries so
// errors.Is comparisons remain stable across the wrapped cause.
package args

import "github.com/kitsunium/sdk/v1/internal/kernel/errs"

var (
	// ErrARGSParse is returned when args cannot be parsed.
	ErrARGSParse = errs.Define(errs.Config{
		Package: "args",
		Code:    1070,
		Message: "failed to parse arguments",
	})
	// ErrARGSInvalid is returned when an arg does not follow the
	// expected flag format in strict mode.
	ErrARGSInvalid = errs.Define(errs.Config{
		Package: "args",
		Code:    1071,
		Message: "invalid argument format",
	})
)
