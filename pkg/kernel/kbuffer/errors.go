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
type bufferError string

// Error returns the error message.
//
//go:nosplit
func (e bufferError) Error() string {
	return string(e)
}

// Is implements error comparison for errors.Is.
func (e bufferError) Is(target error) bool {
	t, ok := target.(bufferError)
	return ok && e == t
}
