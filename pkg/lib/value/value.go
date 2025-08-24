// Package value provides utility functions for safely dereferencing pointers.
//
// This package complements the pointer package by providing safe ways to extract
// values from pointers, returning appropriate zero values for nil pointers.
// It's particularly useful when working with APIs that return pointer types
// or when dealing with optional struct fields.
//
// The package provides both generic functions (recommended) and type-specific
// functions for backward compatibility.
//
// Example usage:
//
//	// Using the generic function (recommended)
//	var strPtr *string
//	str := value.Convert(strPtr) // Returns "" (zero value)
//
//	intPtr := pointer.Convert(42)
//	val := value.Convert(intPtr) // Returns 42
//
//	// Using ConvertOr for custom defaults
//	var portPtr *int
//	port := value.ConvertOr(portPtr, 8080) // Returns 8080
//
//	// Using type-specific functions (deprecated)
//	boolVal := value.Bool(boolPtr) // Returns false if nil
package value

// Convert returns the value of a pointer or the zero value if nil.
// This generic function replaces all type-specific functions for better performance.
//
//go:inline
func Convert[T any](ptr *T) T {
	if ptr == nil {
		var zero T
		return zero
	}
	return *ptr
}

// ConvertOr returns the value of a pointer or a default value if nil.
// Useful when you need a specific default value instead of zero value.
//
//go:inline
func ConvertOr[T any](ptr *T, defaultValue T) T {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// String dereferences the given *string pointer.
// If the pointer is nil, it returns an empty string.
// Deprecated: Use Convert[string](ptr) instead.
//
//go:inline
func String(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// Int dereferences the given *int pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[int](ptr) instead.
func Int(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Int8 dereferences the given *int8 pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[int8](ptr) instead.
func Int8(ptr *int8) int8 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Int16 dereferences the given *int16 pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[int16](ptr) instead.
func Int16(ptr *int16) int16 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Int32 dereferences the given *int32 pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[int32](ptr) instead.
func Int32(ptr *int32) int32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Int64 dereferences the given *int64 pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[int64](ptr) instead.
func Int64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Uint dereferences the given *uint pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[uint](ptr) instead.
func Uint(ptr *uint) uint {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Uint8 dereferences the given *uint8 pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[uint8](ptr) instead.
func Uint8(ptr *uint8) uint8 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Uint16 dereferences the given *uint16 pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[uint16](ptr) instead.
func Uint16(ptr *uint16) uint16 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Uint32 dereferences the given *uint32 pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[uint32](ptr) instead.
func Uint32(ptr *uint32) uint32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Uint64 dereferences the given *uint64 pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[uint64](ptr) instead.
func Uint64(ptr *uint64) uint64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Float32 dereferences the given *float32 pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[float32](ptr) instead.
func Float32(ptr *float32) float32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Float64 dereferences the given *float64 pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[float64](ptr) instead.
func Float64(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Bool dereferences the given *bool pointer.
// If the pointer is nil, it returns false.
// Deprecated: Use Convert[bool](ptr) instead.
func Bool(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

// Byte dereferences the given *byte pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[byte](ptr) instead.
func Byte(ptr *byte) byte {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Rune dereferences the given *rune pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[rune](ptr) instead.
func Rune(ptr *rune) rune {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Complex64 dereferences the given *complex64 pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[complex64](ptr) instead.
func Complex64(ptr *complex64) complex64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Complex128 dereferences the given *complex128 pointer.
// If the pointer is nil, it returns 0.
// Deprecated: Use Convert[complex128](ptr) instead.
func Complex128(ptr *complex128) complex128 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
