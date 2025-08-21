package kbuffer

import (
	"bytes"
	"errors"
	"testing"
)

func TestNewBuffer(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		want     int
	}{
		{"positive capacity", 1024, 1024},
		{"zero capacity", 0, defaultBufferSize},
		{"negative capacity", -1, defaultBufferSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuffer(tt.capacity)
			if b.Cap() != tt.want {
				t.Errorf("NewBuffer(%d) capacity = %d, want %d", tt.capacity, b.Cap(), tt.want)
			}
			if b.Len() != 0 {
				t.Errorf("NewBuffer(%d) length = %d, want 0", tt.capacity, b.Len())
			}
		})
	}
}

func TestBuffer_Write(t *testing.T) {
	b := NewBuffer(10)

	// Test successful write
	data := []byte("hello")
	n, err := b.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() = %d, want %d", n, len(data))
	}
	if !bytes.Equal(b.Bytes(), data) {
		t.Errorf("Bytes() = %v, want %v", b.Bytes(), data)
	}

	// Test write with remaining space
	more := []byte(" test")
	n, err = b.Write(more)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(more) {
		t.Errorf("Write() = %d, want %d", n, len(more))
	}

	// Test overflow
	overflow := []byte("overflow")
	_, err = b.Write(overflow)
	if !errors.Is(err, ErrBufferOverflow) {
		t.Errorf("Write() error = %v, want ErrBufferOverflow", err)
	}

	// Test empty write
	n, err = b.Write([]byte{})
	if err != nil {
		t.Errorf("Write(empty) error = %v", err)
	}
	if n != 0 {
		t.Errorf("Write(empty) = %d, want 0", n)
	}
}

func TestBuffer_WriteString(t *testing.T) {
	b := NewBuffer(10)

	// Test successful write
	s := "hello"
	n, err := b.WriteString(s)
	if err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if n != len(s) {
		t.Errorf("WriteString() = %d, want %d", n, len(s))
	}
	if b.String() != s {
		t.Errorf("String() = %v, want %v", b.String(), s)
	}

	// Test overflow
	_, err = b.WriteString("overflow")
	if !errors.Is(err, ErrBufferOverflow) {
		t.Errorf("WriteString() error = %v, want ErrBufferOverflow", err)
	}

	// Test empty string
	b.Reset()
	n, err = b.WriteString("")
	if err != nil {
		t.Errorf("WriteString(empty) error = %v", err)
	}
	if n != 0 {
		t.Errorf("WriteString(empty) = %d, want 0", n)
	}
}

func TestBuffer_WriteByte(t *testing.T) {
	b := NewBuffer(2)

	// Test successful writes
	if err := b.WriteByte('a'); err != nil {
		t.Fatalf("WriteByte() error = %v", err)
	}
	if err := b.WriteByte('b'); err != nil {
		t.Fatalf("WriteByte() error = %v", err)
	}

	// Test overflow
	err := b.WriteByte('c')
	if !errors.Is(err, ErrBufferOverflow) {
		t.Errorf("WriteByte() error = %v, want ErrBufferOverflow", err)
	}

	if b.String() != "ab" {
		t.Errorf("String() = %v, want 'ab'", b.String())
	}
}

func TestBuffer_WriteAt(t *testing.T) {
	b := NewBuffer(10)
	b.WriteString("hello")

	// Test valid write
	n, err := b.WriteAt([]byte("XX"), 1)
	if err != nil {
		t.Fatalf("WriteAt() error = %v", err)
	}
	if n != 2 {
		t.Errorf("WriteAt() = %d, want 2", n)
	}
	if string(b.data[:5]) != "hXXlo" {
		t.Errorf("data = %v, want 'hXXlo'", string(b.data[:5]))
	}

	// Test negative offset
	_, err = b.WriteAt([]byte("test"), -1)
	if !errors.Is(err, ErrInvalidOffset) {
		t.Errorf("WriteAt(negative) error = %v, want ErrInvalidOffset", err)
	}

	// Test offset beyond capacity
	_, err = b.WriteAt([]byte("test"), 100)
	if !errors.Is(err, ErrInvalidOffset) {
		t.Errorf("WriteAt(beyond) error = %v, want ErrInvalidOffset", err)
	}

	// Test partial write at boundary
	n, err = b.WriteAt([]byte("12345"), 8)
	if err != nil {
		t.Fatalf("WriteAt(boundary) error = %v", err)
	}
	if n != 2 {
		t.Errorf("WriteAt(boundary) = %d, want 2", n)
	}
}

func TestBuffer_TryWrite(t *testing.T) {
	b := NewBuffer(5)

	// Test successful write
	if !b.TryWrite([]byte("hello")) {
		t.Error("TryWrite() = false, want true")
	}

	// Test failed write
	if b.TryWrite([]byte("more")) {
		t.Error("TryWrite(overflow) = true, want false")
	}
}

func TestBuffer_String(t *testing.T) {
	b := NewBuffer(10)

	// Test empty buffer
	if s := b.String(); s != "" {
		t.Errorf("String() = %q, want empty", s)
	}

	// Test with content
	b.WriteString("test")
	if s := b.String(); s != "test" {
		t.Errorf("String() = %q, want 'test'", s)
	}
}

func TestBuffer_Bytes(t *testing.T) {
	b := NewBuffer(10)

	// Test empty buffer
	if len(b.Bytes()) != 0 {
		t.Errorf("Bytes() length = %d, want 0", len(b.Bytes()))
	}

	// Test with content
	data := []byte("test")
	b.Write(data)
	if !bytes.Equal(b.Bytes(), data) {
		t.Errorf("Bytes() = %v, want %v", b.Bytes(), data)
	}
}

func TestBuffer_LenCapAvailable(t *testing.T) {
	b := NewBuffer(10)

	if b.Len() != 0 {
		t.Errorf("Len() = %d, want 0", b.Len())
	}
	if b.Cap() != 10 {
		t.Errorf("Cap() = %d, want 10", b.Cap())
	}
	if b.Available() != 10 {
		t.Errorf("Available() = %d, want 10", b.Available())
	}

	b.Write([]byte("hello"))

	if b.Len() != 5 {
		t.Errorf("Len() = %d, want 5", b.Len())
	}
	if b.Cap() != 10 {
		t.Errorf("Cap() = %d, want 10", b.Cap())
	}
	if b.Available() != 5 {
		t.Errorf("Available() = %d, want 5", b.Available())
	}
}

func TestBuffer_Reset(t *testing.T) {
	b := NewBuffer(10)
	b.Write([]byte("hello"))

	b.Reset()

	if b.Len() != 0 {
		t.Errorf("Len() after Reset = %d, want 0", b.Len())
	}
	if b.Available() != 10 {
		t.Errorf("Available() after Reset = %d, want 10", b.Available())
	}
}

func TestBuffer_Clear(t *testing.T) {
	b := NewBuffer(10)
	b.Write([]byte("hello"))

	b.Clear()

	if b.Len() != 0 {
		t.Errorf("Len() after Clear = %d, want 0", b.Len())
	}

	// Verify data was zeroed
	for i := 0; i < 5; i++ {
		if b.data[i] != 0 {
			t.Errorf("data[%d] = %d, want 0", i, b.data[i])
		}
	}
}

func TestBuffer_Truncate(t *testing.T) {
	b := NewBuffer(10)
	b.Write([]byte("hello"))

	// Truncate to smaller
	b.Truncate(3)
	if b.Len() != 3 {
		t.Errorf("Len() after Truncate(3) = %d, want 3", b.Len())
	}

	// Truncate to larger (no effect)
	b.Truncate(10)
	if b.Len() != 3 {
		t.Errorf("Len() after Truncate(10) = %d, want 3", b.Len())
	}

	// Truncate to negative (becomes 0)
	b.Truncate(-1)
	if b.Len() != 0 {
		t.Errorf("Len() after Truncate(-1) = %d, want 0", b.Len())
	}
}

func TestBuffer_Grow(t *testing.T) {
	b := NewBuffer(10)
	b.Write([]byte("hello"))

	// Test successful grow
	if err := b.Grow(5); err != nil {
		t.Errorf("Grow(5) error = %v", err)
	}

	// Test grow beyond capacity
	err := b.Grow(6)
	if !errors.Is(err, ErrBufferOverflow) {
		t.Errorf("Grow(6) error = %v, want ErrBufferOverflow", err)
	}
}

func TestBuffer_Extend(t *testing.T) {
	b := NewBuffer(10)
	b.Write([]byte("hello"))

	// Test valid extend
	if err := b.Extend(3); err != nil {
		t.Fatalf("Extend(3) error = %v", err)
	}
	if b.Len() != 8 {
		t.Errorf("Len() after Extend(3) = %d, want 8", b.Len())
	}

	// Test extend beyond capacity
	err := b.Extend(5)
	if !errors.Is(err, ErrInvalidOffset) {
		t.Errorf("Extend(5) error = %v, want ErrInvalidOffset", err)
	}

	// Test negative extend
	err = b.Extend(-1)
	if !errors.Is(err, ErrInvalidOffset) {
		t.Errorf("Extend(-1) error = %v, want ErrInvalidOffset", err)
	}
}

func TestBuffer_RemainingSlice(t *testing.T) {
	b := NewBuffer(10)
	b.Write([]byte("hello"))

	remaining := b.RemainingSlice()
	if len(remaining) != 5 {
		t.Errorf("RemainingSlice() length = %d, want 5", len(remaining))
	}
	if cap(remaining) != 5 {
		t.Errorf("RemainingSlice() capacity = %d, want 5", cap(remaining))
	}
}

func TestBuffer_AppendBytes(t *testing.T) {
	b := NewBuffer(10)

	// Test successful append
	if err := b.AppendBytes('h', 'e', 'l', 'l', 'o'); err != nil {
		t.Fatalf("AppendBytes() error = %v", err)
	}
	if b.String() != "hello" {
		t.Errorf("String() = %q, want 'hello'", b.String())
	}

	// Test overflow
	err := b.AppendBytes('1', '2', '3', '4', '5', '6')
	if !errors.Is(err, ErrBufferOverflow) {
		t.Errorf("AppendBytes(overflow) error = %v, want ErrBufferOverflow", err)
	}
}

func TestBuffer_Clone(t *testing.T) {
	b := NewBuffer(20)
	b.Write([]byte("hello"))

	clone := b.Clone()

	// Verify clone has same content
	if !bytes.Equal(clone.Bytes(), b.Bytes()) {
		t.Errorf("Clone().Bytes() = %v, want %v", clone.Bytes(), b.Bytes())
	}
	if clone.Cap() != b.Cap() {
		t.Errorf("Clone().Cap() = %d, want %d", clone.Cap(), b.Cap())
	}
	if clone.Len() != b.Len() {
		t.Errorf("Clone().Len() = %d, want %d", clone.Len(), b.Len())
	}

	// Verify independent memory by modifying clone
	n, err := clone.Write([]byte(" world"))
	if err != nil {
		t.Fatalf("Clone.Write() error = %v", err)
	}
	if n != 6 {
		t.Errorf("Clone.Write() = %d, want 6", n)
	}

	// Clone should have new content
	if string(clone.Bytes()) != "hello world" {
		t.Errorf("Clone modified content = %q, want 'hello world'", string(clone.Bytes()))
	}

	// Original should be unchanged
	if string(b.Bytes()) != "hello" {
		t.Errorf("Original modified after clone write = %q, want 'hello'", string(b.Bytes()))
	}

	// Verify they don't share memory
	if bytes.Equal(clone.Bytes(), b.Bytes()) {
		t.Error("Clone() shares memory with original")
	}
}

func TestBufferError(t *testing.T) {
	err := bufferError("test error")

	if err.Error() != "test error" {
		t.Errorf("Error() = %q, want 'test error'", err.Error())
	}

	// Test Is method
	if !err.Is(bufferError("test error")) {
		t.Error("Is() = false for same error")
	}
	if err.Is(bufferError("different")) {
		t.Error("Is() = true for different error")
	}
	if err.Is(errors.New("test error")) {
		t.Error("Is() = true for different error type")
	}
}
