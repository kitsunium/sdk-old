package buffer

import (
	"unsafe"
)

// Buffer represents a fixed-size byte buffer optimized for performance.
type Buffer struct {
	b   []byte // Pre-allocated buffer
	pos int    // Current write position
	c   int    // Fixed capacity (moved last for better alignment)
}

// NewBuffer creates a new Buffer with a fixed size.
//
// Parameters:
// - size: int - The size of the buffer to allocate.
//
// Returns:
// - *Buffer: The newly created Buffer.
func NewBuffer(size int) *Buffer {
	return &Buffer{
		b:   make([]byte, size),
		c:   size,
		pos: 0,
	}
}

// Len returns the current length of the buffer.
//
// Returns:
// - int: The current length of the buffer.
//go:inline
func (b *Buffer) Len() int {
	return b.pos
}

// Cap returns the fixed capacity of the buffer.
//
// Returns:
// - int: The fixed capacity of the buffer.
//go:inline
func (b *Buffer) Cap() int {
	return b.c
}

// Write writes bytes to the buffer.
//
// Parameters:
// - p: []byte - The bytes to write to the buffer.
//
// Returns:
// - int: The number of bytes written.
// - error: An error if the buffer overflows.
func (b *Buffer) Write(p []byte) (int, error) {
	pLen := len(p)
	remaining := b.c - b.pos
	if pLen > remaining {
		return 0, ErrBufferOverflow
	}
	// Use Go 1.24 optimized copy
	n := copy(b.b[b.pos:b.pos+pLen], p)
	b.pos += n
	return n, nil
}

// WriteString writes a string to the buffer.
//
// Parameters:
// - s: string - The string to write to the buffer.
//
// Returns:
// - int: The number of bytes written.
// - error: An error if the buffer overflows.
func (b *Buffer) WriteString(s string) (int, error) {
	sLen := len(s)
	remaining := b.c - b.pos
	if sLen > remaining {
		return 0, ErrBufferOverflow
	}
	// Direct string copy (Go 1.24 optimized)
	n := copy(b.b[b.pos:], s)
	b.pos += n
	return n, nil
}

// ReWrite clears the buffer and writes new data to it.
//
// Parameters:
// - p: []byte - The bytes to write to the buffer.
//
// Returns:
// - int: The number of bytes written.
// - error: An error if the buffer overflows.
func (b *Buffer) ReWrite(p []byte) (int, error) {
	b.Free()
	return b.Write(p)
}

// ReWriteString clears the buffer and writes a new string to it.
//
// Parameters:
// - s: string - The string to write to the buffer.
//
// Returns:
// - int: The number of bytes written.
// - error: An error if the buffer overflows.
func (b *Buffer) ReWriteString(s string) (int, error) {
	b.Free()
	return b.WriteString(s)
}

// Bytes returns the current contents of the buffer up to the write position.
//
// Returns:
// - []byte: The current contents of the buffer.
//go:inline
func (b *Buffer) Bytes() []byte {
	return b.b[:b.pos]
}

// String returns the current contents of the buffer as a string.
//
// Returns:
// - string: The current contents of the buffer as a string.
func (b *Buffer) String() string {
	if b.pos == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b.b[:b.pos]), b.pos)
}

// Free resets the buffer for reuse.
func (b *Buffer) Free() {
	b.pos = 0
}

// Clear zeroes the buffer content and resets position.
func (b *Buffer) Clear() {
	clear(b.b) // Go 1.21+ builtin
	b.pos = 0
}

// Available returns the number of bytes available for writing.
//go:inline
func (b *Buffer) Available() int {
	return b.c - b.pos
}

// WriteByte writes a single byte to the buffer.
//go:inline
func (b *Buffer) WriteByte(c byte) error {
	if b.pos >= b.c {
		return ErrBufferOverflow
	}
	b.b[b.pos] = c
	b.pos++
	return nil
}

// WriteAt writes bytes at a specific offset without changing position.
func (b *Buffer) WriteAt(p []byte, offset int) (int, error) {
	pLen := len(p)
	if offset < 0 || offset > b.c-pLen {
		return 0, ErrBufferOverflow
	}
	return copy(b.b[offset:offset+pLen], p), nil
}

// Reset resets the buffer with a new backing slice.
func (b *Buffer) Reset(buf []byte) {
	b.b = buf
	b.c = len(buf)
	b.pos = 0
}

// AppendBytes appends bytes efficiently using Go 1.24 optimizations.
//go:inline
func (b *Buffer) AppendBytes(data ...byte) error {
	dataLen := len(data)
	if b.pos+dataLen > b.c {
		return ErrBufferOverflow
	}
	// Direct append pattern (optimized in Go 1.24)
	for i := range dataLen { // Go 1.22+ range over int
		b.b[b.pos+i] = data[i]
	}
	b.pos += dataLen
	return nil
}

// TryWrite attempts to write without error return for hot paths.
//go:inline
func (b *Buffer) TryWrite(p []byte) bool {
	pLen := len(p)
	if b.pos+pLen > b.c {
		return false
	}
	copy(b.b[b.pos:], p)
	b.pos += pLen
	return true
}

// RemainingSlice returns the unused portion of the buffer.
//go:inline
func (b *Buffer) RemainingSlice() []byte {
	return b.b[b.pos:b.c]
}

// Extend extends the current position without writing.
//go:inline
func (b *Buffer) Extend(n int) error {
	newPos := b.pos + n
	if newPos > b.c {
		return ErrBufferOverflow
	}
	b.pos = newPos
	return nil
}

// Truncate reduces the buffer to n bytes.
//go:inline
func (b *Buffer) Truncate(n int) {
	b.pos = min(n, b.pos) // Go 1.21+ builtin min
}

// Grow ensures the buffer has at least n bytes available.
//go:inline
func (b *Buffer) Grow(n int) error {
	if b.c-b.pos < n {
		return ErrBufferOverflow
	}
	return nil
}

// ErrBufferOverflow is returned when a write exceeds buffer capacity.
var ErrBufferOverflow = &bufferError{"buffer overflow"}

type bufferError struct {
	s string
}

func (e *bufferError) Error() string {
	return e.s
}
