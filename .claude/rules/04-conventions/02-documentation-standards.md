# Documentation Standards - Kernel Code Documentation

## Purpose

Define mandatory documentation standards for kernel packages to ensure code is self-documenting, maintainable, and understandable by all team members.

## When to Use

- When writing any new code
- When defining public APIs
- When implementing complex algorithms
- When using unsafe operations
- During code review to ensure compliance

## Package Documentation

### Package Comment Format

```go
// Package kbuffer provides high-performance, thread-safe and unsafe buffer
// implementations optimized for kernel operations.
//
// The package offers two implementations:
//   - Safe: Thread-safe buffer with mutex protection
//   - Unsafe: Lock-free buffer for single-threaded use
//
// Performance characteristics:
//   - Zero-allocation in hot paths
//   - O(1) read/write operations
//   - Cache-line aligned for optimal CPU performance
//
// Usage:
//
//	// Safe version for concurrent use
//	buf := kbuffer.NewBuffer(1024)
//	defer buf.Close()
//
//	n, err := buf.Write(data)
//	if err != nil {
//	    return err
//	}
//
// See README.md for detailed benchmarks and examples.
package kbuffer
```

### Documentation File (doc.go)

```go
// doc.go

/*
Package kbuffer implements high-performance buffer operations for kernel use.

Architecture Overview

The package is structured around two core implementations:

1. Safe Buffer (buffer.go)
   - Thread-safe with minimal lock contention
   - Suitable for concurrent producers/consumers
   - ~1000 ns/op for typical operations

2. Unsafe Buffer (buffer_unsafe.go)
   - Zero-overhead, lock-free implementation
   - Single-threaded use only
   - ~400 ns/op for typical operations (60% faster)

Memory Management

All buffers use pre-allocated memory pools to minimize GC pressure:
   - Small buffers: 1KB-4KB (pooled)
   - Medium buffers: 4KB-64KB (pooled)
   - Large buffers: >64KB (direct allocation)

Thread Safety

Safe version guarantees:
   - Concurrent reads: Safe
   - Concurrent writes: Safe with serialization
   - Read during write: Safe with consistent view

Unsafe version requirements:
   - MUST be accessed from single goroutine
   - Panics on concurrent access (debug mode)
   - No checks in production (unsafe_no_check tag)
*/
package kbuffer
```

## Type Documentation

### Struct Documentation

```go
// Buffer represents a high-performance byte buffer optimized for kernel operations.
//
// The buffer automatically grows as needed and provides zero-copy operations
// where possible. It is designed to minimize allocations and maximize CPU
// cache efficiency.
//
// Memory layout:
//   [header|64 bytes padding|data...]
//   - header: 16 bytes for metadata
//   - padding: ensures data starts on cache line
//   - data: actual buffer content
//
// Concurrency: This type is thread-safe.
type Buffer struct {
    // data holds the buffer content, cache-line aligned
    data []byte

    // read position for streaming operations
    // Invariant: 0 <= off <= len(data)
    off int

    // write position for append operations
    // Invariant: 0 <= wpos <= cap(data)
    wpos int

    // mutex protects all fields
    mu sync.RWMutex
}
```

### Interface Documentation

```go
// Reader defines the interface for reading bytes from a source.
//
// Implementations must be safe for concurrent use if documented as thread-safe.
// The Read method should return io.EOF when no more data is available.
//
// Performance contract:
//   - Read should not allocate if p has sufficient capacity
//   - Implementations should read as much as possible in one call
//
// Error handling:
//   - Return io.EOF at end of stream
//   - Return io.ErrUnexpectedEOF for truncated data
//   - Wrap underlying errors with context
type Reader interface {
    // Read reads up to len(p) bytes into p.
    //
    // Returns the number of bytes read (0 <= n <= len(p)) and any error.
    // Even if Read returns n < len(p), it may use all of p as scratch space.
    //
    // Implementations must not retain p after Read returns.
    Read(p []byte) (n int, err error)
}
```

## Method Documentation

### Public Method Documentation

```go
// Write appends the contents of p to the buffer, growing it as needed.
//
// The return value n is the length of p; err is always nil.
// If the buffer becomes too large, Write will panic with ErrTooLarge.
//
// Performance characteristics:
//   - O(1) amortized time complexity
//   - Zero allocations if buffer has capacity
//   - Automatically grows by 2x when needed
//
// Concurrency: Safe for concurrent calls.
//
// Example:
//   n, err := buf.Write([]byte("hello"))
//   // n == 5, err == nil
func (b *Buffer) Write(p []byte) (n int, err error) {
    // ... implementation
}
```

### Private Method Documentation

```go
// grow increases the buffer capacity to at least n bytes.
//
// Preconditions:
//   - b.mu must be held for writing
//   - n > cap(b.data)
//
// Postconditions:
//   - cap(b.data) >= n
//   - Existing data is preserved
//
// This method uses a growth factor of 2x up to 1MB, then 1.25x for larger sizes
// to balance memory usage and allocation frequency.
func (b *Buffer) grow(n int) {
    // ... implementation
}
```

## Unsafe Code Documentation

### Unsafe Operation Documentation

```go
// unsafeStringToBytes performs a zero-copy conversion from string to []byte.
//
// WARNING: The returned byte slice shares memory with the string.
// Modifying the byte slice will corrupt the string.
//
// Safety requirements:
//   - The returned slice must not be modified
//   - The string must not be garbage collected while slice is in use
//   - The slice must not escape the current function scope
//
// Performance: Saves one allocation and copy vs []byte(s)
//
//go:nosplit
//go:noescape
func unsafeStringToBytes(s string) []byte {
    // Unsafe pointer cast to avoid allocation
    // Safe because we never modify the result
    return *(*[]byte)(unsafe.Pointer(&s))
}
```

### Memory Layout Documentation

```go
// node represents a buffer pool node with optimized memory layout.
//
// Memory layout (64 bytes total):
//   Offset  Size  Field
//   0       8     next    *node
//   8       8     data    *byte
//   16      8     size    int
//   24      8     used    int32 (atomic)
//   28      4     padding (compiler inserted)
//   32      32    pad     [32]byte (cache line padding)
//
// Cache line alignment ensures no false sharing between nodes
// when accessed from different CPU cores.
//
//go:notinheap
type node struct {
    next *node      // Next node in free list
    data *byte      // Pointer to buffer data
    size int        // Buffer size
    used int32      // Atomic usage flag
    _    [32]byte   // Padding to 64 bytes (cache line)
}
```

## Performance Documentation

### Benchmark Documentation

```go
// BenchmarkBuffer_Write measures buffer write performance.
//
// Expected performance:
//   - Safe version: ~1000 ns/op, 0 allocs/op
//   - Unsafe version: ~400 ns/op, 0 allocs/op
//   - Ratio: Unsafe should be 60% faster
//
// The benchmark tests sequential writes of 1KB chunks to measure
// throughput under typical workload conditions.
//
// Results on reference hardware (Intel i7-9700K @ 3.6GHz):
//   BenchmarkBuffer_Write_Safe-8     1000000    1050 ns/op    0 B/op    0 allocs/op
//   BenchmarkBuffer_Write_Unsafe-8   3000000     420 ns/op    0 B/op    0 allocs/op
func BenchmarkBuffer_Write(b *testing.B) {
    // ... benchmark code
}
```

## Error Documentation

### Error Variable Documentation

```go
// ErrBufferFull is returned when attempting to write to a full buffer
// that cannot be grown due to size limits.
//
// This error typically occurs when:
//   - Buffer reaches MaxBufferSize (default: 1GB)
//   - System is out of memory
//   - Write would cause integer overflow
//
// Recovery: Create a new buffer or drain existing buffer before writing.
var ErrBufferFull = errors.New("buffer: full")

// ErrInvalidOffset is returned when seeking to an invalid position.
//
// Common causes:
//   - Negative offset
//   - Offset beyond buffer length
//   - Integer overflow in offset calculation
var ErrInvalidOffset = errors.New("buffer: invalid offset")
```

## Constant Documentation

### Constant Group Documentation

```go
// Buffer size constants define standard buffer capacities.
//
// These sizes are chosen to align with common system parameters:
//   - SmallBufferSize: Typical network packet
//   - DefaultBufferSize: OS page size
//   - LargeBufferSize: L2 cache size
//   - MaxBufferSize: Prevent excessive memory use
const (
    // SmallBufferSize is used for small temporary buffers.
    SmallBufferSize = 1024 // 1KB

    // DefaultBufferSize is the standard buffer size for most operations.
    DefaultBufferSize = 4096 // 4KB

    // LargeBufferSize is used for bulk operations.
    LargeBufferSize = 65536 // 64KB

    // MaxBufferSize is the maximum allowed buffer size.
    // Attempting to grow beyond this size will return ErrBufferFull.
    MaxBufferSize = 1 << 30 // 1GB
)
```

## Comment Formatting Rules

### Inline Comments

```go
func process(data []byte) error {
    // Validate input before processing
    if len(data) == 0 {
        return ErrEmptyInput
    }

    // Pre-allocate result buffer based on input size
    // We use 2x size to handle worst-case expansion
    result := make([]byte, 0, len(data)*2)

    for i := 0; i < len(data); i++ {
        b := data[i]

        // Skip null bytes (protocol requirement)
        if b == 0 {
            continue
        }

        // Apply transformation
        // Note: This must match the inverse in decode()
        result = append(result, transform(b))
    }

    return nil
}
```

### TODO Comments

```go
// TODO(username): Optimize this loop for better cache locality
// Consider processing in 64-byte chunks to match cache line size.

// FIXME(username): This causes excessive allocations under load
// Need to implement a pool for these temporary buffers.

// NOTE: This intentionally doesn't check for overflow
// because the input is validated to be within safe bounds.

// HACK: Working around issue #123 in upstream library
// Remove this when upstream fixes the bug.
```

## Do's

✅ **Document all exported types** and functions ✅ **Explain "why"** not just "what" ✅ **Include performance characteristics** for critical paths ✅ **Document concurrency guarantees** clearly ✅
**Specify preconditions and postconditions** ✅ **Include examples** for complex APIs ✅ **Document unsafe operations thoroughly** ✅ **Use complete sentences** starting with the name ✅ **Keep line
length under 80 characters** for readability ✅ **Update documentation** when changing code

## Don'ts

❌ **Don't state the obvious** (i++ // increment i) ❌ **Don't use unclear abbreviations** in comments ❌ **Don't leave TODO comments** without attribution ❌ **Don't document unexported types**
excessively ❌ **Don't use comments to disable code** (use version control) ❌ **Don't write novels** - be concise but complete ❌ **Don't use humor or sarcasm** in documentation ❌ **Don't reference
specific line numbers** (they change) ❌ **Don't forget to document breaking changes** ❌ **Don't use markdown in godoc** comments (limited support)

## Documentation Tools

```bash
# Generate and view godoc
godoc -http=:6060
# Visit http://localhost:6060/pkg/

# Check documentation coverage
go-doc-coverage ./...

# Lint documentation
golint ./...

# Generate markdown from godoc
godocdown ./... > API.md
```

## Related Documents

- [01-naming-conventions.md](01-naming-conventions.md) - Naming standards
- [03-code-organization.md](03-code-organization.md) - Code structure
- [../02-implementation/01-interface-patterns.md](../02-implementation/01-interface-patterns.md) - Interface documentation patterns
