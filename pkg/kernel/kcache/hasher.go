// Package kcache provides cache implementations with configurable thread safety.
// This file contains hash function implementations including FNV-1a, xxHash, and CityHash algorithms.
package kcache

import (
	"reflect"
	"unsafe"
)

// Ensure fnvHasher implements Hasher interface
var _ Hasher = (*fnvHasher)(nil)

// fnvHasher implements FNV-1a hashing for cache keys.
// Optimized for speed with type-specific paths.
type fnvHasher struct {
	// No state needed for FNV
}

// newFNVHasher creates a new FNV-1a hasher.
//
//go:inline
func newFNVHasher() Hasher {
	return &fnvHasher{}
}

// Hash computes FNV-1a hash for any key type.
// Uses type switches for optimized paths.
func (h *fnvHasher) Hash(key interface{}) uint64 {
	// Type switch for common types
	switch k := key.(type) {
	case string:
		return hashString(k)
	case []byte:
		return hashBytes(k)
	case int:
		return hashInt(int64(k))
	case int64:
		return hashInt(k)
	case int32:
		return hashInt(int64(k))
	case int16:
		return hashInt(int64(k))
	case int8:
		return hashInt(int64(k))
	case uint:
		return hashUint(uint64(k))
	case uint64:
		return hashUint(k)
	case uint32:
		return hashUint(uint64(k))
	case uint16:
		return hashUint(uint64(k))
	case uint8:
		return hashUint(uint64(k))
	case float64:
		return hashFloat64(k)
	case float32:
		return hashFloat64(float64(k))
	case bool:
		if k {
			return FNVOffsetBasis ^ 1
		}
		return FNVOffsetBasis
	case nil:
		return FNVOffsetBasis
	default:
		// Fallback to reflection-based hashing
		return hashInterface(key)
	}
}

// Equal checks if two keys are equal.
// Optimized for common types.
func (h *fnvHasher) Equal(a, b interface{}) bool {
	// Handle nil cases
	if a == nil || b == nil {
		return a == b
	}

	// Type assertion for common types
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case []byte:
		bv, ok := b.([]byte)
		return ok && bytesEqual(av, bv)
	case int:
		bv, ok := b.(int)
		return ok && av == bv
	case int32:
		bv, ok := b.(int32)
		return ok && av == bv
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case uint:
		bv, ok := b.(uint)
		return ok && av == bv
	case uint32:
		bv, ok := b.(uint32)
		return ok && av == bv
	case uint64:
		bv, ok := b.(uint64)
		return ok && av == bv
	case float32:
		bv, ok := b.(float32)
		if !ok {
			return false
		}
		// Handle NaN case: NaN should equal NaN for cache purposes
		if av != av && bv != bv {
			return true
		}
		return av == bv
	case float64:
		bv, ok := b.(float64)
		if !ok {
			return false
		}
		// Handle NaN case: NaN should equal NaN for cache purposes
		if av != av && bv != bv {
			return true
		}
		return av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	default:
		// Check if both are functions - compare by pointer
		va := reflect.ValueOf(a)
		vb := reflect.ValueOf(b)
		if va.Kind() == reflect.Func && vb.Kind() == reflect.Func {
			// Functions are equal if they have the same pointer
			if va.IsNil() && vb.IsNil() {
				return true
			}
			if va.IsNil() || vb.IsNil() {
				return false
			}
			return va.Pointer() == vb.Pointer()
		}

		// Check if both are channels - compare by pointer
		if va.Kind() == reflect.Chan && vb.Kind() == reflect.Chan {
			// Channels are equal if they have the same pointer
			if va.IsNil() && vb.IsNil() {
				return true
			}
			if va.IsNil() || vb.IsNil() {
				return false
			}
			return va.Pointer() == vb.Pointer()
		}

		// For other types, use reflect.DeepEqual
		// This handles structs, arrays, slices, maps, etc.
		return reflect.DeepEqual(a, b)
	}
}

// hashString computes FNV-1a hash for strings.
// Optimized with unsafe string access.
//
//go:inline
//go:nosplit
func hashString(s string) uint64 {
	// Access string bytes without allocation
	p := unsafe.Pointer((*reflect.StringHeader)(unsafe.Pointer(&s)).Data)
	n := len(s)

	hash := FNVOffsetBasis
	for i := 0; i < n; i++ {
		hash ^= uint64(*(*byte)(unsafe.Add(p, i)))
		hash *= FNVPrime
	}
	return hash
}

// hashBytes computes FNV-1a hash for byte slices.
// Direct memory access for speed.
//
//go:inline
//go:nosplit
func hashBytes(b []byte) uint64 {
	hash := FNVOffsetBasis
	for i := 0; i < len(b); i++ {
		hash ^= uint64(b[i])
		hash *= FNVPrime
	}
	return hash
}

// hashInt computes FNV-1a hash for integers.
// Mixes bits for better distribution.
//
//go:inline
//go:nosplit
func hashInt(i int64) uint64 {
	hash := FNVOffsetBasis

	// Mix bytes of integer
	hash ^= uint64(i & 0xFF)
	hash *= FNVPrime
	hash ^= uint64((i >> 8) & 0xFF)
	hash *= FNVPrime
	hash ^= uint64((i >> 16) & 0xFF)
	hash *= FNVPrime
	hash ^= uint64((i >> 24) & 0xFF)
	hash *= FNVPrime
	hash ^= uint64((i >> 32) & 0xFF)
	hash *= FNVPrime
	hash ^= uint64((i >> 40) & 0xFF)
	hash *= FNVPrime
	hash ^= uint64((i >> 48) & 0xFF)
	hash *= FNVPrime
	hash ^= uint64((i >> 56) & 0xFF)
	hash *= FNVPrime

	return hash
}

// hashUint computes FNV-1a hash for unsigned integers.
// Similar to hashInt but for unsigned types.
//
//go:inline
//go:nosplit
func hashUint(u uint64) uint64 {
	return hashInt(int64(u))
}

// hashFloat64 computes FNV-1a hash for floats.
// Treats float bits as integer for hashing.
//
//go:inline
//go:nosplit
func hashFloat64(f float64) uint64 {
	// Convert float bits to uint64
	bits := *(*uint64)(unsafe.Pointer(&f))

	// Handle special cases
	if f == 0 {
		return FNVOffsetBasis // Both +0 and -0 hash the same
	}
	if f != f { // NaN
		return FNVOffsetBasis ^ 0x7FF8000000000001
	}

	return hashUint(bits)
}

// hashInterface computes hash for arbitrary interface types.
// Uses reflection as fallback for unknown types.
func hashInterface(key interface{}) uint64 {
	// Get type and value via reflection
	v := reflect.ValueOf(key)

	// Handle different kinds
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return FNVOffsetBasis
		}
		// Hash pointer address
		return hashUint(uint64(v.Pointer()))

	case reflect.Slice:
		if v.IsNil() {
			return FNVOffsetBasis
		}
		// Hash slice elements
		hash := FNVOffsetBasis
		for i := 0; i < v.Len(); i++ {
			elemHash := hashInterface(v.Index(i).Interface())
			hash ^= elemHash
			hash *= FNVPrime
		}
		return hash

	case reflect.Array:
		// Hash array elements
		hash := FNVOffsetBasis
		for i := 0; i < v.Len(); i++ {
			elemHash := hashInterface(v.Index(i).Interface())
			hash ^= elemHash
			hash *= FNVPrime
		}
		return hash

	case reflect.Struct:
		// Hash struct fields
		hash := FNVOffsetBasis
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).CanInterface() {
				fieldHash := hashInterface(v.Field(i).Interface())
				hash ^= fieldHash
				hash *= FNVPrime
			}
		}
		return hash

	case reflect.Map:
		if v.IsNil() {
			return FNVOffsetBasis
		}
		// Hash map keys and values
		hash := FNVOffsetBasis
		for _, k := range v.MapKeys() {
			keyHash := hashInterface(k.Interface())
			valueHash := hashInterface(v.MapIndex(k).Interface())
			hash ^= keyHash
			hash *= FNVPrime
			hash ^= valueHash
			hash *= FNVPrime
		}
		return hash

	case reflect.Func:
		// For functions, use pointer address as hash
		if v.IsNil() {
			return FNVOffsetBasis
		}
		return hashUint(uint64(v.Pointer()))

	case reflect.Chan:
		// For channels, use pointer address as hash
		if v.IsNil() {
			return FNVOffsetBasis
		}
		return hashUint(uint64(v.Pointer()))

	default:
		// For other types, hash the type name and string representation
		hash := hashString(v.Type().String())
		if v.CanInterface() {
			hash ^= hashString(v.String())
			hash *= FNVPrime
		}
		return hash
	}
}

// bytesEqual compares byte slices for equality.
// Optimized for common sizes.
//
//go:inline
//go:nosplit
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	// Fast path for small slices
	n := len(a)
	if n <= 16 {
		for i := 0; i < n; i++ {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	// Use word-wise comparison for larger slices
	return bytesEqualLarge(a, b)
}

// bytesEqualLarge compares large byte slices using word-wise comparison.
// Processes 8 bytes at a time for better performance.
func bytesEqualLarge(a, b []byte) bool {
	n := len(a)

	// Compare 8 bytes at a time
	for i := 0; i < n-7; i += 8 {
		if *(*uint64)(unsafe.Pointer(&a[i])) != *(*uint64)(unsafe.Pointer(&b[i])) {
			return false
		}
	}

	// Compare remaining bytes
	for i := n &^ 7; i < n; i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// Alternative hash functions for specialized use cases

// xxHash implements xxHash algorithm for better performance.
// Suitable for large keys.
type xxHasher struct{}

// Hash computes xxHash for the key.
func (h *xxHasher) Hash(key interface{}) uint64 {
	// Simplified xxHash implementation
	// Full implementation would be more complex
	switch k := key.(type) {
	case string:
		return xxHashString(k)
	case []byte:
		return xxHashBytes(k)
	default:
		// Fallback to FNV for other types
		return (&fnvHasher{}).Hash(key)
	}
}

// Equal checks equality (same as FNV hasher).
func (h *xxHasher) Equal(a, b interface{}) bool {
	return (&fnvHasher{}).Equal(a, b)
}

// xxHashString computes xxHash for strings.
func xxHashString(s string) uint64 {
	// Simplified xxHash - real implementation would be more complex
	const (
		prime1 = 11400714785074694791
		prime2 = 14029467366897019727
		prime3 = 1609587929392839161
		prime4 = 9650029242287828579
		prime5 = 2870177450012600261
	)

	h := prime5 + uint64(len(s))

	// Process string bytes
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i]) * prime5
		h = (h << 11) | (h >> 53)
		h *= prime1
	}

	// Final mix
	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	h *= prime3
	h ^= h >> 32

	return h
}

// xxHashBytes computes xxHash for byte slices.
func xxHashBytes(b []byte) uint64 {
	// Convert to string and hash
	// This is safe because we don't modify the bytes
	return xxHashString(*(*string)(unsafe.Pointer(&b)))
}

// CityHash implements Google's CityHash for strings.
// Good balance of speed and distribution.
type cityHasher struct{}

// Hash computes CityHash for the key.
func (h *cityHasher) Hash(key interface{}) uint64 {
	switch k := key.(type) {
	case string:
		return cityHashString(k)
	case []byte:
		return cityHashBytes(k)
	default:
		return (&fnvHasher{}).Hash(key)
	}
}

// Equal checks equality.
func (h *cityHasher) Equal(a, b interface{}) bool {
	return (&fnvHasher{}).Equal(a, b)
}

// cityHashString computes CityHash for strings.
func cityHashString(s string) uint64 {
	// Simplified CityHash - real implementation would be more complex
	const (
		k0 = 0xc3a5c85c97cb3127
		k1 = 0xb492b66fbe98f273
		k2 = 0x9ae16a3b2f90404f
	)

	n := uint64(len(s))
	h := n * k0

	// Mix string bytes
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i]) * k1
		h = (h >> 47) ^ h
		h *= k2
	}

	return h
}

// cityHashBytes computes CityHash for byte slices.
func cityHashBytes(b []byte) uint64 {
	return cityHashString(*(*string)(unsafe.Pointer(&b)))
}

// Identity hasher for pre-hashed keys

// identityHasher uses the key itself as the hash.
// Only works with uint64 keys.
type identityHasher struct{}

// Hash returns the key as the hash (for uint64 keys only).
func (h *identityHasher) Hash(key interface{}) uint64 {
	if k, ok := key.(uint64); ok {
		return k
	}
	// Fallback for non-uint64 keys
	return (&fnvHasher{}).Hash(key)
}

// Equal checks equality.
func (h *identityHasher) Equal(a, b interface{}) bool {
	return a == b
}
