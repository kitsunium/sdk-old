package buffer_test

import (
	"testing"

	"github.com/kistunium/sdk/pkg/kernel/buffer"
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
}

func BenchmarkBuffer(b *testing.B) {
	content := "Benchmarking Buffer WriteString"
	bytesContent := []byte(content)

	b.Run("Write", func(b *testing.B) {
		bb := buffer.NewBuffer(1024)
		for i := 0; i < b.N; i++ {
			bb.Free()
			_, _ = bb.Write(bytesContent)
		}
	})

	b.Run("WriteString", func(b *testing.B) {
		bb := buffer.NewBuffer(1024)
		for i := 0; i < b.N; i++ {
			bb.Free()
			_, _ = bb.WriteString(content)
		}
	})
}
