package value

// String dereferences the given *string pointer.
// If the pointer is nil, it returns an empty string.
func String(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// Int dereferences the given *int pointer.
// If the pointer is nil, it returns 0.
func Int(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Int8 dereferences the given *int8 pointer.
// If the pointer is nil, it returns 0.
func Int8(ptr *int8) int8 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Int16 dereferences the given *int16 pointer.
// If the pointer is nil, it returns 0.
func Int16(ptr *int16) int16 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Int32 dereferences the given *int32 pointer.
// If the pointer is nil, it returns 0.
func Int32(ptr *int32) int32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Int64 dereferences the given *int64 pointer.
// If the pointer is nil, it returns 0.
func Int64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Uint dereferences the given *uint pointer.
// If the pointer is nil, it returns 0.
func Uint(ptr *uint) uint {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Uint8 dereferences the given *uint8 pointer.
// If the pointer is nil, it returns 0.
func Uint8(ptr *uint8) uint8 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Uint16 dereferences the given *uint16 pointer.
// If the pointer is nil, it returns 0.
func Uint16(ptr *uint16) uint16 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Uint32 dereferences the given *uint32 pointer.
// If the pointer is nil, it returns 0.
func Uint32(ptr *uint32) uint32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Uint64 dereferences the given *uint64 pointer.
// If the pointer is nil, it returns 0.
func Uint64(ptr *uint64) uint64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Float32 dereferences the given *float32 pointer.
// If the pointer is nil, it returns 0.
func Float32(ptr *float32) float32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Float64 dereferences the given *float64 pointer.
// If the pointer is nil, it returns 0.
func Float64(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Bool dereferences the given *bool pointer.
// If the pointer is nil, it returns false.
func Bool(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

// Byte dereferences the given *byte pointer.
// If the pointer is nil, it returns 0.
func Byte(ptr *byte) byte {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Rune dereferences the given *rune pointer.
// If the pointer is nil, it returns 0.
func Rune(ptr *rune) rune {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Complex64 dereferences the given *complex64 pointer.
// If the pointer is nil, it returns 0.
func Complex64(ptr *complex64) complex64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Complex128 dereferences the given *complex128 pointer.
// If the pointer is nil, it returns 0.
func Complex128(ptr *complex128) complex128 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
