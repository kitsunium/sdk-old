package kbuffer

// Buffer size constants
const (
	// defaultBufferSize is the default buffer size when none is specified.
	defaultBufferSize = 4096 // 4KB

	// Common buffer sizes for optimization
	size64B   = 64
	size256B  = 256
	size512B  = 512
	size1KB   = 1024
	size2KB   = 2048
	size4KB   = 4096
	size8KB   = 8192
	size16KB  = 16384
	size32KB  = 32768
	size64KB  = 65536
	size128KB = 131072
	size256KB = 262144
	size512KB = 524288
	size1MB   = 1048576
)

// Cache line size for CPU optimization
const cacheLineSize = 64
