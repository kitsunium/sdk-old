package kbuffer

import (
	"testing"
)

func TestBuffer(t *testing.T) {
	t.Run("Len and Cap", func(t *testing.T) {
		bb := NewBuffer(128)
		Equal(t, 0, bb.Len(), "expected Len=0")
		Equal(t, 128, bb.Cap(), "expected Cap=128")

		_, _ = bb.WriteString("test")
		Equal(t, 4, bb.Len(), "expected Len=4")
		Equal(t, 128, bb.Cap(), "expected Cap=128")
	})

	t.Run("Write and WriteString", func(t *testing.T) {
		bb := NewBuffer(128)
		data := []byte("Write Test")
		n, err := bb.Write(data)
		NoError(t, err, "unexpected error in Write")
		Equal(t, len(data), n, "expected bytes written to match length of data")
		Equal(t, string(data), bb.String(), "buffer contents mismatch after Write")

		str := "String Test"
		n, err = bb.WriteString(str)
		NoError(t, err, "unexpected error in WriteString")
		Equal(t, len(str), n, "expected bytes written to match length of string")
		Equal(t, string(data)+str, bb.String(), "buffer contents mismatch after WriteString")
	})

	t.Run("Write Overflow", func(t *testing.T) {
		bb := NewBuffer(16)
		_, err := bb.WriteString("This is a long string that exceeds capacity")
		Error(t, err, "expected error when writing past buffer capacity")
	})

	t.Run("ReWrite", func(t *testing.T) {
		bb := NewBuffer(128)
		_, _ = bb.WriteString("Old Data")

		newData := []byte("New")
		n, err := bb.ReWrite(newData)
		NoError(t, err, "unexpected error in ReWrite")
		Equal(t, len(newData), n, "expected bytes written to match length of newData")
		Equal(t, string(newData), bb.String(), "buffer contents mismatch after ReWrite")
		Equal(t, newData, bb.Bytes(), "buffer contents mismatch after ReWrite")
	})

	t.Run("ReWriteString", func(t *testing.T) {
		bb := NewBuffer(128)
		_, _ = bb.WriteString("Old Data")

		newString := "New"
		n, err := bb.ReWriteString(newString)
		NoError(t, err, "unexpected error in ReWriteString")
		Equal(t, len(newString), n, "expected bytes written to match length of newString")
		Equal(t, newString, bb.String(), "buffer contents mismatch after ReWriteString")
		Equal(t, []byte(newString), bb.Bytes(), "buffer contents mismatch after ReWrite")
	})

	t.Run("Free", func(t *testing.T) {
		bb := NewBuffer(128)
		_, _ = bb.WriteString("Free Test")
		bb.Free()
		Equal(t, 0, bb.Len(), "expected buffer length 0 after Free")
		Equal(t, 128, bb.Cap(), "expected buffer capacity to remain unchanged after Free")
	})

	t.Run("Available", func(t *testing.T) {
		bb := NewBuffer(128)
		Equal(t, 128, bb.Available(), "expected full capacity available")

		_, _ = bb.WriteString("Hello")
		Equal(t, 123, bb.Available(), "expected 123 bytes available")

		bb.Free()
		Equal(t, 128, bb.Available(), "expected full capacity after Free")
	})

	t.Run("WriteByte", func(t *testing.T) {
		bb := NewBuffer(128)

		err := bb.WriteByte('A')
		NoError(t, err)
		Equal(t, 1, bb.Len())
		Equal(t, "A", bb.String())

		err = bb.WriteByte('B')
		NoError(t, err)
		Equal(t, 2, bb.Len())
		Equal(t, "AB", bb.String())
	})

	t.Run("WriteAt", func(t *testing.T) {
		bb := NewBuffer(128)

		// Write initial data
		_, _ = bb.WriteString("Hello     World")

		// Write at specific offset
		n, err := bb.WriteAt([]byte("Go"), 6)
		NoError(t, err)
		Equal(t, 2, n)
		Equal(t, "Hello Go  World", bb.String())
	})

	t.Run("Grow", func(t *testing.T) {
		bb := NewBuffer(128)

		err := bb.Grow(128)
		NoError(t, err)

		_, _ = bb.WriteString("Some data")
		err = bb.Grow(100)
		NoError(t, err)

		err = bb.Grow(200)
		Error(t, err)
		Equal(t, ErrBufferOverflow, err)
	})

	t.Run("Reset", func(t *testing.T) {
		bb := NewBuffer(128)
		_, _ = bb.WriteString("Initial")

		newSlice := make([]byte, 256)
		bb.Reset(newSlice)

		Equal(t, 0, bb.Len())
		Equal(t, 256, bb.Cap())
		Equal(t, 256, bb.Available())
	})

	t.Run("EmptyString", func(t *testing.T) {
		bb := NewBuffer(128)
		Equal(t, "", bb.String())
	})

	t.Run("Clear", func(t *testing.T) {
		bb := NewBuffer(128)
		_, _ = bb.WriteString("Data to clear")
		bb.Clear()
		Equal(t, 0, bb.Len())
		Equal(t, "", bb.String())
	})

	t.Run("AppendBytes", func(t *testing.T) {
		bb := NewBuffer(128)
		err := bb.AppendBytes('H', 'e', 'l', 'l', 'o')
		NoError(t, err)
		Equal(t, "Hello", bb.String())
	})

	t.Run("TryWrite", func(t *testing.T) {
		bb := NewBuffer(10)
		ok := bb.TryWrite([]byte("Hello"))
		True(t, ok)
		Equal(t, "Hello", bb.String())

		ok = bb.TryWrite([]byte("World!"))
		False(t, ok)
		Equal(t, "Hello", bb.String())
	})

	t.Run("RemainingSlice", func(t *testing.T) {
		bb := NewBuffer(128)
		_, _ = bb.WriteString("Hello")
		remaining := bb.RemainingSlice()
		Equal(t, 123, len(remaining))
	})

	t.Run("Extend", func(t *testing.T) {
		bb := NewBuffer(128)
		err := bb.Extend(10)
		NoError(t, err)
		Equal(t, 10, bb.Len())

		err = bb.Extend(200)
		Error(t, err)
	})

	t.Run("Truncate", func(t *testing.T) {
		bb := NewBuffer(128)
		_, _ = bb.WriteString("Hello World")
		bb.Truncate(5)
		Equal(t, 5, bb.Len())
		Equal(t, "Hello", bb.String())

		bb.Truncate(100)
		Equal(t, 5, bb.Len())
	})

	t.Run("WriteByte_Overflow", func(t *testing.T) {
		bb := NewBuffer(2)
		err := bb.WriteByte('A')
		NoError(t, err)
		err = bb.WriteByte('B')
		NoError(t, err)
		err = bb.WriteByte('C')
		Error(t, err)
		Equal(t, ErrBufferOverflow, err)
	})

	t.Run("WriteAt_NegativeOffset", func(t *testing.T) {
		bb := NewBuffer(128)
		n, err := bb.WriteAt([]byte("test"), -1)
		Error(t, err)
		Equal(t, 0, n)
		Equal(t, ErrBufferOverflow, err)
	})

	t.Run("WriteAt_Overflow", func(t *testing.T) {
		bb := NewBuffer(10)
		n, err := bb.WriteAt([]byte("test"), 8)
		Error(t, err)
		Equal(t, 0, n)
		Equal(t, ErrBufferOverflow, err)
	})

	t.Run("AppendBytes_Overflow", func(t *testing.T) {
		bb := NewBuffer(3)
		err := bb.AppendBytes('A', 'B', 'C')
		NoError(t, err)
		err = bb.AppendBytes('D')
		Error(t, err)
		Equal(t, ErrBufferOverflow, err)
	})

	t.Run("ErrorMessage", func(t *testing.T) {
		Equal(t, "buffer overflow", ErrBufferOverflow.Error())
	})

	t.Run("WriteString_Overflow", func(t *testing.T) {
		bb := NewBuffer(5)
		n, err := bb.WriteString("Hello World")
		Error(t, err)
		Equal(t, 0, n)
		Equal(t, ErrBufferOverflow, err)
	})

	t.Run("Extend_Overflow", func(t *testing.T) {
		bb := NewBuffer(10)
		err := bb.Extend(11)
		Error(t, err)
		Equal(t, ErrBufferOverflow, err)
	})
}
