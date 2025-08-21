package kbuffer

import (
	"unsafe"
)

// Buffer represents a high-performance byte buffer with zero-allocation operations.
// The struct is carefully aligned for optimal CPU cache performance.
type Buffer struct {
	// Cache line 1 (64 bytes) - hot path fields
	data []byte   // Underlying byte slice
	pos  int32    // Current write position (32-bit for better alignment)
	cap  int32    // Fixed capacity (32-bit for better alignment)
	_    [48]byte // Padding to align to cache line
}

// NewBuffer creates a new Buffer with the specified capacity.
// The buffer is pre-allocated to avoid future allocations.
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
// Returns the number of bytes written and any error.
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

	// Use unsafe for zero-copy operation
	dst := unsafe.Slice(&b.data[b.pos], available)
	copy(dst, p)
	b.pos += int32(n)
	return n, nil
}

// WriteString appends a string to the buffer without allocation.
// Uses unsafe conversion to avoid string-to-bytes allocation.
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
	src := unsafe.Slice(unsafe.StringData(s), n)
	dst := unsafe.Slice(&b.data[b.pos], available)
	copy(dst, src)
	b.pos += int32(n)
	return n, nil
}

// WriteByte appends a single byte to the buffer.
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

// WriteAt writes bytes at a specific offset without changing position.
// This method performs bounds checking for security.
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

// Bytes returns the written portion of the buffer.
// The returned slice shares memory with the buffer.
//
//go:inline
//go:nosplit
func (b *Buffer) Bytes() []byte {
	return b.data[:b.pos]
}

// String returns the buffer contents as a string using zero-allocation conversion.
// Uses unsafe.String for maximum performance.
//
//go:nosplit
func (b *Buffer) String() string {
	if b.pos == 0 {
		return ""
	}
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
//
//go:inline
//go:nosplit
func (b *Buffer) Reset() {
	b.pos = 0
}

// Clear zeroes the buffer content and resets position.
// Use for security-sensitive data.
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
//
//go:inline
func (b *Buffer) Grow(n int) error {
	if b.Available() < n {
		return ErrBufferOverflow
	}
	return nil
}

// Extend advances the write position by n bytes without writing.
// Returns error if n would exceed capacity.
//
//go:inline
func (b *Buffer) Extend(n int) error {
	if n < 0 || n > b.Available() {
		return ErrInvalidOffset
	}
	b.pos += int32(n)
	return nil
}

// RemainingSlice returns the unused portion of the buffer.
//
//go:inline
//go:nosplit
func (b *Buffer) RemainingSlice() []byte {
	return b.data[b.pos:b.cap]
}

// AppendBytes appends multiple bytes efficiently.
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

// Clone creates a copy of the buffer with its own memory.
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
