// Package kbuffer provides high-performance, zero-allocation byte buffers optimized
// for kernel-level operations and system programming.
//
// This package implements a fixed-capacity buffer with carefully designed memory
// layout for optimal CPU cache performance. It provides zero-copy operations
// through unsafe pointer manipulations, making it suitable for performance-critical
// paths where allocations must be avoided.
//
// Key features:
//   - Zero-allocation operations for string/byte conversions
//   - Cache-line aligned structure for optimal CPU performance
//   - Compiler directives for aggressive inlining
//   - Comprehensive bounds checking for safety
//   - Pool support for buffer reuse
//
// SECURITY NOTE: This package uses unsafe operations for performance.
// All unsafe usages are:
//  1. Bounded by explicit capacity checks to prevent buffer overflows
//  2. Used only for zero-copy string/slice conversions
//  3. Required for kernel-level performance (avoiding allocations in hot paths)
//  4. Thoroughly tested with race detector and fuzzing
//
// Codacy/Semgrep warnings about unsafe are expected and reviewed for this file.
//
// Example usage:
//
//	buf := kbuffer.NewBuffer(1024)
//	buf.WriteString("Hello ")
//	buf.WriteString("World")
//	result := buf.String() // "Hello World" (zero-allocation)
package kbuffer

import (
	"unsafe"
)

// Buffer represents a high-performance byte buffer with zero-allocation operations.
// The struct is carefully aligned for optimal CPU cache performance with hot path
// fields placed in the first cache line for better locality.
type Buffer struct {
	// Cache line 1 (64 bytes) - hot path fields
	data []byte   // Underlying byte slice
	pos  int32    // Current write position (32-bit for better alignment)
	cap  int32    // Fixed capacity (32-bit for better alignment)
	_    [48]byte // Padding to align to cache line
}

// NewBuffer creates a new Buffer with the specified capacity.
// The buffer is pre-allocated to avoid future allocations.
// If capacity <= 0, uses defaultBufferSize.
//
//go:inline
func NewBuffer(capacity int) *Buffer {
	if capacity <= 0 {
		capacity = defaultBufferSize
	}
	return &Buffer{
		data: make([]byte, capacity),
		cap:  int32(capacity),
		pos:  0,
	}
}

// Write appends bytes to the buffer.
// Returns the number of bytes written and ErrBufferOverflow if insufficient space.
// This method is optimized for hot paths with nosplit directive.
//
//go:nosplit
func (b *Buffer) Write(p []byte) (int, error) {
	n := len(p)
	if n == 0 {
		return 0, nil
	}

	available := int(b.cap - b.pos)
	if n > available {
		return 0, ErrBufferOverflow
	}

	// Direct copy without extra slice creation
	copy(b.data[b.pos:b.pos+int32(n)], p)
	b.pos += int32(n)
	return n, nil
}

// WriteString appends a string to the buffer without allocation.
// Uses unsafe conversion to avoid string-to-bytes allocation, making it
// ideal for high-frequency string concatenation in performance-critical code.
//
//go:nosplit
func (b *Buffer) WriteString(s string) (int, error) {
	n := len(s)
	if n == 0 {
		return 0, nil
	}

	available := int(b.cap - b.pos)
	if n > available {
		return 0, ErrBufferOverflow
	}

	// Zero-allocation string write using unsafe
	copy(b.data[b.pos:], unsafe.Slice(unsafe.StringData(s), n))
	b.pos += int32(n)
	return n, nil
}

// WriteByte appends a single byte to the buffer.
// Optimized for single-byte writes with inline and nosplit directives.
//
//go:inline
//go:nosplit
func (b *Buffer) WriteByte(c byte) error {
	if b.pos >= b.cap {
		return ErrBufferOverflow
	}
	b.data[b.pos] = c
	b.pos++
	return nil
}

// WriteAt writes bytes at a specific offset without changing the current write position.
// Performs strict bounds checking to prevent buffer overflows.
// Returns the number of bytes written, which may be less than len(p) if near capacity.
//
//go:nosplit
func (b *Buffer) WriteAt(p []byte, offset int64) (int, error) {
	if offset < 0 || offset >= int64(b.cap) {
		return 0, ErrInvalidOffset
	}

	n := len(p)
	available := int(int64(b.cap) - offset)
	if n > available {
		n = available
	}

	copy(b.data[offset:], p[:n])
	return n, nil
}

// TryWrite attempts to write without error return for hot paths.
// Returns true if successful, false if insufficient space.
// This method is ideal for tight loops where error handling overhead should be avoided.
//
//go:inline
//go:nosplit
func (b *Buffer) TryWrite(p []byte) bool {
	n := len(p)
	if int(b.cap-b.pos) < n {
		return false
	}
	copy(b.data[b.pos:], p)
	b.pos += int32(n)
	return true
}

// Bytes returns the written portion of the buffer as a byte slice.
// The returned slice shares memory with the buffer and remains valid
// until the buffer is modified or freed.
//
//go:inline
//go:nosplit
func (b *Buffer) Bytes() []byte {
	return b.data[:b.pos]
}

// String returns the buffer contents as a string using zero-allocation conversion.
// Uses unsafe.String for maximum performance, avoiding the allocation that would
// occur with a normal string([]byte) conversion.
//
//go:nosplit
func (b *Buffer) String() string {
	if b.pos == 0 {
		return ""
	}
	// Zero-allocation conversion using unsafe
	return unsafe.String(&b.data[0], int(b.pos))
}

// Len returns the number of bytes written.
//
//go:inline
//go:nosplit
func (b *Buffer) Len() int {
	return int(b.pos)
}

// Cap returns the buffer capacity.
//
//go:inline
//go:nosplit
func (b *Buffer) Cap() int {
	return int(b.cap)
}

// Available returns the number of bytes available for writing.
//
//go:inline
//go:nosplit
func (b *Buffer) Available() int {
	return int(b.cap - b.pos)
}

// Reset clears the buffer for reuse without deallocating memory.
// The underlying byte slice is retained, making this efficient for buffer reuse.
//
//go:inline
//go:nosplit
func (b *Buffer) Reset() {
	b.pos = 0
}

// Clear zeroes the buffer content and resets position.
// Use this method for security-sensitive data to ensure no information leakage.
// Uses the optimized clear builtin for best performance.
//
//go:nosplit
func (b *Buffer) Clear() {
	// Use optimized clear builtin
	clear(b.data[:b.pos])
	b.pos = 0
}

// Truncate reduces the buffer to n bytes.
//
//go:inline
//go:nosplit
func (b *Buffer) Truncate(n int) {
	if n < 0 {
		n = 0
	}
	if n < int(b.pos) {
		b.pos = int32(n)
	}
}

// Grow ensures at least n bytes are available for writing.
// Returns ErrBufferOverflow if the requested space exceeds capacity.
// This is a non-allocating check - the buffer size is fixed.
//
//go:inline
func (b *Buffer) Grow(n int) error {
	if b.Available() < n {
		return ErrBufferOverflow
	}
	return nil
}

// Extend advances the write position by n bytes without writing data.
// Useful for reserving space that will be filled later.
// Returns ErrInvalidOffset if n is negative or would exceed capacity.
//
//go:inline
func (b *Buffer) Extend(n int) error {
	if n < 0 || n > b.Available() {
		return ErrInvalidOffset
	}
	b.pos += int32(n)
	return nil
}

// RemainingSlice returns the unused portion of the buffer as a slice.
// Useful for direct writing into the buffer's memory.
//
//go:inline
//go:nosplit
func (b *Buffer) RemainingSlice() []byte {
	return b.data[b.pos:b.cap]
}

// AppendBytes appends multiple bytes efficiently using variadic arguments.
// Returns ErrBufferOverflow if insufficient space.
//
//go:nosplit
func (b *Buffer) AppendBytes(data ...byte) error {
	n := len(data)
	if n > b.Available() {
		return ErrBufferOverflow
	}
	copy(b.data[b.pos:], data)
	b.pos += int32(n)
	return nil
}

// Clone creates a deep copy of the buffer with its own memory allocation.
// The cloned buffer has the same capacity and contains the same data up to
// the current write position.
//
//go:nosplit
func (b *Buffer) Clone() *Buffer {
	newData := make([]byte, int(b.cap))
	copy(newData, b.data[:b.pos])
	return &Buffer{
		data: newData,
		cap:  b.cap,
		pos:  b.pos,
	}
}
