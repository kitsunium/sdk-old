package buffer_test

import (
	"testing"

	"github.com/kitsunium/sdk/pkg/kernel/buffer"
	"github.com/stretchr/testify/assert"
)

func TestBuffer(t *testing.T) {
	t.Run("Len and Cap", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		assert.Equal(t, 0, bb.Len(), "expected Len=0")
		assert.Equal(t, 128, bb.Cap(), "expected Cap=128")

		_, _ = bb.WriteString("test")
		assert.Equal(t, 4, bb.Len(), "expected Len=4")
		assert.Equal(t, 128, bb.Cap(), "expected Cap=128")
	})

	t.Run("Write and WriteString", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		data := []byte("Write Test")
		n, err := bb.Write(data)
		assert.NoError(t, err, "unexpected error in Write")
		assert.Equal(t, len(data), n, "expected bytes written to match length of data")
		assert.Equal(t, string(data), bb.String(), "buffer contents mismatch after Write")

		str := "String Test"
		n, err = bb.WriteString(str)
		assert.NoError(t, err, "unexpected error in WriteString")
		assert.Equal(t, len(str), n, "expected bytes written to match length of string")
		assert.Equal(t, string(data)+str, bb.String(), "buffer contents mismatch after WriteString")
	})

	t.Run("Write Overflow", func(t *testing.T) {
		bb := buffer.NewBuffer(16)
		_, err := bb.WriteString("This is a long string that exceeds capacity")
		assert.Error(t, err, "expected error when writing past buffer capacity")
	})

	t.Run("ReWrite", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		_, _ = bb.WriteString("Old Data")

		newData := []byte("New")
		n, err := bb.ReWrite(newData)
		assert.NoError(t, err, "unexpected error in ReWrite")
		assert.Equal(t, len(newData), n, "expected bytes written to match length of newData")
		assert.Equal(t, string(newData), bb.String(), "buffer contents mismatch after ReWrite")
		assert.Equal(t, newData, bb.Bytes(), "buffer contents mismatch after ReWrite")
	})

	t.Run("ReWriteString", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		_, _ = bb.WriteString("Old Data")

		newString := "New"
		n, err := bb.ReWriteString(newString)
		assert.NoError(t, err, "unexpected error in ReWriteString")
		assert.Equal(t, len(newString), n, "expected bytes written to match length of newString")
		assert.Equal(t, newString, bb.String(), "buffer contents mismatch after ReWriteString")
		assert.Equal(t, []byte(newString), bb.Bytes(), "buffer contents mismatch after ReWrite")
	})

	t.Run("Free", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		_, _ = bb.WriteString("Free Test")
		bb.Free()
		assert.Equal(t, 0, bb.Len(), "expected buffer length 0 after Free")
		assert.Equal(t, 128, bb.Cap(), "expected buffer capacity to remain unchanged after Free")
	})

	t.Run("Available", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		assert.Equal(t, 128, bb.Available(), "expected full capacity available")
		
		_, _ = bb.WriteString("Hello")
		assert.Equal(t, 123, bb.Available(), "expected 123 bytes available")
		
		bb.Free()
		assert.Equal(t, 128, bb.Available(), "expected full capacity after Free")
	})

	t.Run("WriteByte", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		
		err := bb.WriteByte('A')
		assert.NoError(t, err)
		assert.Equal(t, 1, bb.Len())
		assert.Equal(t, "A", bb.String())
		
		err = bb.WriteByte('B')
		assert.NoError(t, err)
		assert.Equal(t, 2, bb.Len())
		assert.Equal(t, "AB", bb.String())
	})

	t.Run("WriteAt", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		
		// Write initial data
		_, _ = bb.WriteString("Hello     World")
		
		// Write at specific offset
		n, err := bb.WriteAt([]byte("Go"), 6)
		assert.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.Equal(t, "Hello Go  World", bb.String())
	})

	t.Run("Grow", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		
		err := bb.Grow(128)
		assert.NoError(t, err)
		
		_, _ = bb.WriteString("Some data")
		err = bb.Grow(100)
		assert.NoError(t, err)
		
		err = bb.Grow(200)
		assert.Error(t, err)
		assert.Equal(t, buffer.ErrBufferOverflow, err)
	})

	t.Run("Reset", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		_, _ = bb.WriteString("Initial")
		
		newSlice := make([]byte, 256)
		bb.Reset(newSlice)
		
		assert.Equal(t, 0, bb.Len())
		assert.Equal(t, 256, bb.Cap())
		assert.Equal(t, 256, bb.Available())
	})

	t.Run("EmptyString", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		assert.Equal(t, "", bb.String())
	})

	t.Run("Clear", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		_, _ = bb.WriteString("Data to clear")
		bb.Clear()
		assert.Equal(t, 0, bb.Len())
		assert.Equal(t, "", bb.String())
	})

	t.Run("AppendBytes", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		err := bb.AppendBytes('H', 'e', 'l', 'l', 'o')
		assert.NoError(t, err)
		assert.Equal(t, "Hello", bb.String())
	})

	t.Run("TryWrite", func(t *testing.T) {
		bb := buffer.NewBuffer(10)
		ok := bb.TryWrite([]byte("Hello"))
		assert.True(t, ok)
		assert.Equal(t, "Hello", bb.String())
		
		ok = bb.TryWrite([]byte("World!"))
		assert.False(t, ok)
		assert.Equal(t, "Hello", bb.String())
	})

	t.Run("RemainingSlice", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		_, _ = bb.WriteString("Hello")
		remaining := bb.RemainingSlice()
		assert.Equal(t, 123, len(remaining))
	})

	t.Run("Extend", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		err := bb.Extend(10)
		assert.NoError(t, err)
		assert.Equal(t, 10, bb.Len())
		
		err = bb.Extend(200)
		assert.Error(t, err)
	})

	t.Run("Truncate", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		_, _ = bb.WriteString("Hello World")
		bb.Truncate(5)
		assert.Equal(t, 5, bb.Len())
		assert.Equal(t, "Hello", bb.String())
		
		bb.Truncate(100)
		assert.Equal(t, 5, bb.Len())
	})

	t.Run("WriteByte_Overflow", func(t *testing.T) {
		bb := buffer.NewBuffer(2)
		err := bb.WriteByte('A')
		assert.NoError(t, err)
		err = bb.WriteByte('B')
		assert.NoError(t, err)
		err = bb.WriteByte('C')
		assert.Error(t, err)
		assert.Equal(t, buffer.ErrBufferOverflow, err)
	})

	t.Run("WriteAt_NegativeOffset", func(t *testing.T) {
		bb := buffer.NewBuffer(128)
		n, err := bb.WriteAt([]byte("test"), -1)
		assert.Error(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, buffer.ErrBufferOverflow, err)
	})

	t.Run("WriteAt_Overflow", func(t *testing.T) {
		bb := buffer.NewBuffer(10)
		n, err := bb.WriteAt([]byte("test"), 8)
		assert.Error(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, buffer.ErrBufferOverflow, err)
	})

	t.Run("AppendBytes_Overflow", func(t *testing.T) {
		bb := buffer.NewBuffer(3)
		err := bb.AppendBytes('A', 'B', 'C')
		assert.NoError(t, err)
		err = bb.AppendBytes('D')
		assert.Error(t, err)
		assert.Equal(t, buffer.ErrBufferOverflow, err)
	})

	t.Run("ErrorMessage", func(t *testing.T) {
		assert.Equal(t, "buffer overflow", buffer.ErrBufferOverflow.Error())
	})

	t.Run("WriteString_Overflow", func(t *testing.T) {
		bb := buffer.NewBuffer(5)
		n, err := bb.WriteString("Hello World")
		assert.Error(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, buffer.ErrBufferOverflow, err)
	})

	t.Run("Extend_Overflow", func(t *testing.T) {
		bb := buffer.NewBuffer(10)
		err := bb.Extend(11)
		assert.Error(t, err)
		assert.Equal(t, buffer.ErrBufferOverflow, err)
	})
}
