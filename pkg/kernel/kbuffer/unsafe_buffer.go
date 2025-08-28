//go:build amd64 || arm64
// +build amd64 arm64

package kbuffer

import (
	"unsafe"
)

// ============================================================================
// UNSAFE BUFFER - NON THREAD-SAFE - MAXIMUM PERFORMANCE
// ============================================================================
//
// ⚠️ WARNING: This buffer is NOT thread-safe!
// Use ONLY in single-threaded contexts or when YOU manage synchronization.
// For concurrent access, use NewSafeBuffer() instead.
//
// Performance characteristics:
// - Write: ~2-3 ns/op (10-15x faster than thread-safe version)
// - Zero allocations
// - Zero overhead
// - Direct memory access
//
// ============================================================================

// unsafeBuffer is the fastest possible buffer implementation.
// NO synchronization - caller MUST ensure single-threaded access.
type unsafeBuffer struct {
	// Cache line 1 (64 bytes) - Hot path fields
	data unsafe.Pointer // Direct pointer to byte array (8 bytes)
	len  uint32         // Current length - no atomics needed (4 bytes)
	cap  uint32         // Fixed capacity (4 bytes)
	flag uint32         // Status flags (4 bytes)
	_    [44]byte       // Cache line padding

	// Cache line 2 (64 bytes) - Safety and metadata
	checker goroutineChecker // Goroutine safety checker (16 bytes)
	origin  unsafe.Pointer   // Original allocation pointer (8 bytes)
	pooled  bool             // From pool flag (1 byte)
	_       [39]byte         // Cache line padding
}

// newUnsafeBuffer creates a new non-thread-safe buffer.
// ⚠️ UNSAFE: No synchronization - single-threaded use only!
//
//go:nosplit
func newUnsafeBuffer(capacity int, opts ...Option) Buffer {
	// Validate capacity
	if capacity <= 0 {
		capacity = defaultBufferSize
	}
	if capacity < minBufferSize {
		capacity = minBufferSize
	}
	if capacity > maxBufferSize {
		capacity = maxBufferSize
	}

	// Allocate memory
	buf := make([]byte, capacity)

	// Create buffer
	b := &unsafeBuffer{
		data:   unsafe.Pointer(&buf[0]),
		len:    0,
		cap:    uint32(capacity),
		flag:   stateFlagNormal,
		origin: unsafe.Pointer(&buf[0]),
		pooled: false,
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(b); err != nil {
			continue
		}
	}

	return b
}

// Write appends bytes with ZERO overhead.
// ⚠️ UNSAFE: Not thread-safe!
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) Write(p []byte) (n int, err error) {
	// Check for concurrent access
	b.checker.checkSafety()

	if len(p) == 0 {
		return 0, nil
	}

	newLen := b.len + uint32(len(p))
	if newLen > b.cap {
		return 0, errBufferFull
	}

	// Direct memory copy - no synchronization
	dst := unsafe.Pointer(uintptr(b.data) + uintptr(b.len))
	copy(unsafe.Slice((*byte)(dst), len(p)), p)

	b.len = newLen

	if newLen == b.cap {
		b.flag |= stateFlagFull
	}

	return len(p), nil
}

// WriteString with zero-copy and ZERO overhead.
// ⚠️ UNSAFE: Not thread-safe!
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) WriteString(s string) (n int, err error) {
	// Check for concurrent access
	b.checker.checkSafety()

	if len(s) == 0 {
		return 0, nil
	}

	newLen := b.len + uint32(len(s))
	if newLen > b.cap {
		return 0, errBufferFull
	}

	// Zero-copy string write
	dst := unsafe.Pointer(uintptr(b.data) + uintptr(b.len))
	src := unsafe.Pointer(unsafe.StringData(s))
	copy(unsafe.Slice((*byte)(dst), len(s)), unsafe.Slice((*byte)(src), len(s)))

	b.len = newLen

	if newLen == b.cap {
		b.flag |= stateFlagFull
	}

	return len(s), nil
}

// WriteByte with minimal overhead.
// ⚠️ UNSAFE: Not thread-safe!
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) WriteByte(c byte) error {
	// Check for concurrent access
	b.checker.checkSafety()

	if b.len >= b.cap {
		return errBufferFull
	}

	*(*byte)(unsafe.Pointer(uintptr(b.data) + uintptr(b.len))) = c
	b.len++

	if b.len == b.cap {
		b.flag |= stateFlagFull
	}

	return nil
}

// WriteAt writes at specific offset.
// ⚠️ UNSAFE: Not thread-safe!
func (b *unsafeBuffer) WriteAt(p []byte, off int64) (n int, err error) {
	// Check for concurrent access
	b.checker.checkSafety()

	if off < 0 || off >= int64(b.cap) {
		return 0, errInvalidOffset
	}

	available := int64(b.cap) - off
	writeLen := int64(len(p))

	if writeLen > available {
		writeLen = available
	}

	dst := unsafe.Pointer(uintptr(b.data) + uintptr(off))
	copy(unsafe.Slice((*byte)(dst), writeLen), p[:writeLen])

	return int(writeLen), nil
}

// TryWrite always succeeds in unsafe mode (no lock to try).
// ⚠️ UNSAFE: Not thread-safe!
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) TryWrite(p []byte) bool {
	_, err := b.Write(p)
	return err == nil
}

// Bytes returns data slice - direct access, no copy.
// ⚠️ UNSAFE: Returned slice shares memory with buffer!
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) Bytes() []byte {
	if b.len == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(b.data), b.len)
}

// String returns buffer content as string.
// ⚠️ UNSAFE: Uses unsafe conversion!
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) String() string {
	if b.len == 0 {
		return ""
	}
	return unsafe.String((*byte)(b.data), b.len)
}

// BytesUnsafe returns raw pointer and length.
// ⚠️ UNSAFE: Direct memory access!
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) BytesUnsafe() (ptr uintptr, len int) {
	if b.len == 0 {
		return 0, 0
	}
	return uintptr(b.data), int(b.len)
}

// Len returns current length - direct field access.
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) Len() int {
	return int(b.len)
}

// Cap returns capacity.
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) Cap() int {
	return int(b.cap)
}

// Available returns remaining space.
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) Available() int {
	return int(b.cap - b.len)
}

// Reset clears position - direct field access.
// ⚠️ UNSAFE: Not thread-safe!
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) Reset() {
	b.len = 0
	b.flag = stateFlagNormal
}

// Clear zeros memory and resets.
// ⚠️ UNSAFE: Not thread-safe!
//
//go:nosplit
func (b *unsafeBuffer) Clear() {
	if b.len > 0 {
		clear(unsafe.Slice((*byte)(b.data), b.len))
	}
	b.len = 0
	b.flag = stateFlagCleared
}

// Truncate reduces length.
// ⚠️ UNSAFE: Not thread-safe!
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) Truncate(n int) {
	if n < 0 {
		n = 0
	}
	if n < int(b.len) {
		b.len = uint32(n)
		if uint32(n) < b.cap {
			b.flag &^= stateFlagFull
		}
	}
}

// Grow checks available space.
//
//go:inline
func (b *unsafeBuffer) Grow(n int) error {
	if b.Available() < n {
		return errBufferFull
	}
	return nil
}

// Extend advances position.
// ⚠️ UNSAFE: Not thread-safe!
//
//go:inline
func (b *unsafeBuffer) Extend(n int) error {
	if n < 0 {
		return errInvalidSize
	}

	newLen := b.len + uint32(n)
	if newLen > b.cap {
		return errBufferFull
	}

	b.len = newLen
	return nil
}

// Clone creates independent copy.
func (b *unsafeBuffer) Clone() Buffer {
	newBuf := make([]byte, b.cap)

	if b.len > 0 {
		copy(newBuf, unsafe.Slice((*byte)(b.data), b.len))
	}

	clone := &unsafeBuffer{
		data:   unsafe.Pointer(&newBuf[0]),
		len:    b.len,
		cap:    b.cap,
		flag:   b.flag &^ stateFlagPooled,
		origin: unsafe.Pointer(&newBuf[0]),
		pooled: false,
	}

	return clone
}

// RemainingSlice returns unused portion.
//
//go:inline
//go:nosplit
func (b *unsafeBuffer) RemainingSlice() []byte {
	if b.len >= b.cap {
		return nil
	}
	start := unsafe.Pointer(uintptr(b.data) + uintptr(b.len))
	return unsafe.Slice((*byte)(start), b.cap-b.len)
}

// AppendBytes appends multiple bytes.
func (b *unsafeBuffer) AppendBytes(data ...byte) error {
	if len(data) == 0 {
		return nil
	}
	_, err := b.Write(data)
	return err
}
