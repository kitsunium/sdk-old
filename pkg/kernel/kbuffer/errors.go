package kbuffer

// Error constants for buffer operations.
// Using sentinel errors for zero-allocation error handling.
const (
	errBufferOverflow = bufferError("buffer overflow")
	errInvalidOffset  = bufferError("invalid offset")
	errNilBuffer      = bufferError("nil buffer")
	errInvalidSize    = bufferError("invalid size")
)

// Exported error variables
var (
	// ErrBufferOverflow is returned when a write exceeds buffer capacity.
	ErrBufferOverflow = errBufferOverflow

	// ErrInvalidOffset is returned when an offset is out of bounds.
	ErrInvalidOffset = errInvalidOffset

	// ErrNilBuffer is returned when operating on a nil buffer.
	ErrNilBuffer = errNilBuffer

	// ErrInvalidSize is returned when a size parameter is invalid.
	ErrInvalidSize = errInvalidSize
)

// bufferError implements the error interface with zero allocations.
// Uses string constants to avoid heap allocations during error creation.
// This is critical for high-performance kernel code where allocations
// in error paths can cause performance degradation.
type bufferError string

// Error returns the error message string.
// This method has the //go:nosplit directive to prevent stack growth
// and ensure minimal overhead in error handling paths.
//
//go:nosplit
func (e bufferError) Error() string {
	return string(e)
}

// Is implements error comparison for errors.Is functionality.
// Allows for efficient error type checking without reflection.
// Used by the standard library's errors.Is() function.
func (e bufferError) Is(target error) bool {
	t, ok := target.(bufferError)
	return ok && e == t
}
