package kbuffer

// Buffer size constants for optimal memory allocation and performance.
//
// These constants are carefully chosen based on:
//   - CPU cache line sizes (typically 64 bytes)
//   - Memory page sizes (typically 4KB)
//   - Common I/O operation sizes
//   - Balance between memory usage and copy overhead
const (
	// defaultBufferSize is the default buffer size when none is specified.
	// 4KB aligns with typical memory page size for optimal memory management.
	defaultBufferSize = 4096 // 4KB
)
