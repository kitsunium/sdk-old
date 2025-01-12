package buffer

import (
	"errors"
)

// Buffer represents a fixed-size byte buffer.
type Buffer struct {
	c   int    // Fixed capacity
	b   []byte // Pre-allocated buffer
	pos int    // Current write position
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
func (b *Buffer) Len() int {
	return b.pos
}

// Cap returns the fixed capacity of the buffer.
//
// Returns:
// - int: The fixed capacity of the buffer.
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
	if len(p)+b.pos > b.c {
		return 0, errors.New("buffer overflow")
	}
	n := copy(b.b[b.pos:], p)
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
	return b.Write([]byte(s))
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
func (b *Buffer) Bytes() []byte {
	return b.b[:b.pos]
}

// String returns the current contents of the buffer as a string.
//
// Returns:
// - string: The current contents of the buffer as a string.
func (b *Buffer) String() string {
	return string(b.b[:b.pos])
}

// Free resets the buffer for reuse.
func (b *Buffer) Free() {
	b.pos = 0
}
