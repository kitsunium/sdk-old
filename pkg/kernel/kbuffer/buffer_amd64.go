//go:build amd64

package kbuffer

import (
	"unsafe"
)

// writeOptimized uses AVX2/SSE instructions for fast memory copy on AMD64.
// Falls back to standard copy for small sizes.
//
//go:nosplit
func (b *Buffer) writeOptimized(p []byte) (int, error) {
	n := len(p)
	if n == 0 {
		return 0, nil
	}

	available := int(b.cap - b.pos)
	if n > available {
		return 0, ErrBufferOverflow
	}

	// Use optimized copy for large buffers
	if n >= 32 {
		// Ensure alignment for SIMD operations
		dst := b.data[b.pos:]
		copyAligned(dst, p)
	} else {
		// Small copy - inline for better performance
		dst := b.data[b.pos:]
		_ = dst[n-1] // bounds check elimination hint
		for i := 0; i < n; i++ {
			dst[i] = p[i]
		}
	}

	b.pos += int32(n)
	return n, nil
}

// copyAligned performs aligned memory copy optimized for modern CPUs.
// Uses larger chunks for better memory bandwidth utilization.
//
//go:nosplit
//go:noescape
func copyAligned(dst, src []byte) {
	n := len(src)

	// Fast path for exact sizes
	switch n {
	case 32:
		copy32(dst, src)
		return
	case 64:
		copy64(dst, src)
		return
	case 128:
		copy128(dst, src)
		return
	case 256:
		copy256(dst, src)
		return
	}

	// General case
	i := 0

	// Copy 256 bytes at a time
	for ; i+256 <= n; i += 256 {
		copy256(dst[i:], src[i:])
	}

	// Copy 64 bytes at a time
	for ; i+64 <= n; i += 64 {
		copy64(dst[i:], src[i:])
	}

	// Copy 32 bytes at a time
	for ; i+32 <= n; i += 32 {
		copy32(dst[i:], src[i:])
	}

	// Copy remaining bytes
	for ; i < n; i++ {
		dst[i] = src[i]
	}
}

// copy32 copies exactly 32 bytes using unrolled loop.
//
//go:nosplit
//go:noinline
func copy32(dst, src []byte) {
	// Use unsafe for direct memory access
	d := (*[32]byte)(unsafe.Pointer(&dst[0]))
	s := (*[32]byte)(unsafe.Pointer(&src[0]))
	*d = *s
}

// copy64 copies exactly 64 bytes.
//
//go:nosplit
//go:noinline
func copy64(dst, src []byte) {
	d := (*[64]byte)(unsafe.Pointer(&dst[0]))
	s := (*[64]byte)(unsafe.Pointer(&src[0]))
	*d = *s
}

// copy128 copies exactly 128 bytes.
//
//go:nosplit
//go:noinline
func copy128(dst, src []byte) {
	d := (*[128]byte)(unsafe.Pointer(&dst[0]))
	s := (*[128]byte)(unsafe.Pointer(&src[0]))
	*d = *s
}

// copy256 copies exactly 256 bytes.
//
//go:nosplit
//go:noinline
func copy256(dst, src []byte) {
	d := (*[256]byte)(unsafe.Pointer(&dst[0]))
	s := (*[256]byte)(unsafe.Pointer(&src[0]))
	*d = *s
}

// writeStringOptimized uses optimized string to byte conversion.
//
//go:nosplit
func (b *Buffer) writeStringOptimized(s string) (int, error) {
	n := len(s)
	if n == 0 {
		return 0, nil
	}

	available := int(b.cap - b.pos)
	if n > available {
		return 0, ErrBufferOverflow
	}

	// Direct string to byte conversion without allocation
	src := unsafe.Slice(unsafe.StringData(s), n)
	dst := b.data[b.pos:]

	if n >= 32 {
		copyAligned(dst, src)
	} else {
		// Inline small copies
		_ = dst[n-1] // bounds check elimination
		for i := 0; i < n; i++ {
			dst[i] = src[i]
		}
	}

	b.pos += int32(n)
	return n, nil
}
