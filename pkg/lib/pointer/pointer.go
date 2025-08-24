// Package pointer provides utility functions for creating pointers to values.
//
// This package simplifies the creation of pointers to literal values, which is
// particularly useful when working with APIs that require pointer parameters
// or when dealing with optional fields in structs.
//
// The package provides both a generic Convert function (recommended) and
// type-specific functions for backward compatibility.
//
// Example usage:
//
//	// Using the generic function (recommended)
//	strPtr := pointer.Convert("hello")
//	intPtr := pointer.Convert(42)
//
//	// Using type-specific functions
//	boolPtr := pointer.Bool(true)
//	floatPtr := pointer.Float64(3.14)
//
//	// Useful for struct fields with pointer types
//	type Config struct {
//	    Host     *string
//	    Port     *int
//	    Enabled  *bool
//	}
//
//	config := Config{
//	    Host:    pointer.String("localhost"),
//	    Port:    pointer.Int(8080),
//	    Enabled: pointer.Bool(true),
//	}
package pointer

// Convert returns a generic pointer to the given value.
// This single function replaces all type-specific functions for better performance.
//
//go:inline
func Convert[T any](v T) *T {
	return &v
}

// String returns a pointer to the given string value.
// Deprecated: Use Convert[string](v) instead.
//
//go:inline
func String(v string) *string {
	return &v
}

// Int returns a pointer to the given int value.
// Deprecated: Use Convert[int](v) instead for better performance with generics.
func Int(v int) *int {
	return &v
}

// Int8 returns a pointer to the given int8 value.
// Deprecated: Use Convert[int8](v) instead.
func Int8(v int8) *int8 {
	return &v
}

// Int16 returns a pointer to the given int16 value.
// Deprecated: Use Convert[int16](v) instead.
func Int16(v int16) *int16 {
	return &v
}

// Int32 returns a pointer to the given int32 value.
// Deprecated: Use Convert[int32](v) instead.
func Int32(v int32) *int32 {
	return &v
}

// Int64 returns a pointer to the given int64 value.
// Deprecated: Use Convert[int64](v) instead.
func Int64(v int64) *int64 {
	return &v
}

// Uint returns a pointer to the given uint value.
// Deprecated: Use Convert[uint](v) instead.
func Uint(v uint) *uint {
	return &v
}

// Uint8 returns a pointer to the given uint8 value.
// Deprecated: Use Convert[uint8](v) instead.
func Uint8(v uint8) *uint8 {
	return &v
}

// Uint16 returns a pointer to the given uint16 value.
// Deprecated: Use Convert[uint16](v) instead.
func Uint16(v uint16) *uint16 {
	return &v
}

// Uint32 returns a pointer to the given uint32 value.
// Deprecated: Use Convert[uint32](v) instead.
func Uint32(v uint32) *uint32 {
	return &v
}

// Uint64 returns a pointer to the given uint64 value.
// Deprecated: Use Convert[uint64](v) instead.
func Uint64(v uint64) *uint64 {
	return &v
}

// Float32 returns a pointer to the given float32 value.
// Deprecated: Use Convert[float32](v) instead.
func Float32(v float32) *float32 {
	return &v
}

// Float64 returns a pointer to the given float64 value.
// Deprecated: Use Convert[float64](v) instead.
func Float64(v float64) *float64 {
	return &v
}

// Bool returns a pointer to the given bool value.
// Deprecated: Use Convert[bool](v) instead.
func Bool(v bool) *bool {
	return &v
}

// Byte returns a pointer to the given byte value.
// Deprecated: Use Convert[byte](v) instead.
func Byte(v byte) *byte {
	return &v
}

// Rune returns a pointer to the given rune value.
// Deprecated: Use Convert[rune](v) instead.
func Rune(v rune) *rune {
	return &v
}

// Complex64 returns a pointer to the given complex64 value.
// Deprecated: Use Convert[complex64](v) instead.
func Complex64(v complex64) *complex64 {
	return &v
}

// Complex128 returns a pointer to the given complex128 value.
// Deprecated: Use Convert[complex128](v) instead.
func Complex128(v complex128) *complex128 {
	return &v
}
