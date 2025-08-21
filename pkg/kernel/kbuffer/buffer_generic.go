//go:build !amd64

package kbuffer

// writeOptimized falls back to standard implementation on non-AMD64.
//
//go:nosplit
func (b *Buffer) writeOptimized(p []byte) (int, error) {
	return b.Write(p)
}

// writeStringOptimized falls back to standard implementation on non-AMD64.
//
//go:nosplit
func (b *Buffer) writeStringOptimized(s string) (int, error) {
	return b.WriteString(s)
}
