# Package kbuffer

## Overview

The `kbuffer` package provides high-performance, zero-allocation byte buffers
optimized for kernel-level operations and system programming. It implements
fixed-capacity buffers with carefully designed memory layouts for optimal CPU
cache performance, making it ideal for performance-critical paths where memory
allocations must be avoided.

## Features

- **Zero-Allocation Operations**: String/byte conversions without memory
  allocation
- **Cache-Line Optimization**: Struct layout aligned for optimal CPU cache
  performance
- **Compiler Optimizations**: Aggressive inlining with `//go:inline` and
  `//go:nosplit` directives
- **Memory Pooling**: Reusable buffer pools to minimize GC pressure
- **Thread-Safe Pool**: Safe concurrent access to buffer pools
- **Comprehensive Safety**: Strict bounds checking despite unsafe operations

## Installation

```go
import "github.com/kitsunium/sdk/pkg/kernel/kbuffer"
```

## Core Components

### Buffer

The main `Buffer` type provides a fixed-capacity byte buffer with
zero-allocation operations:

```go
// Create a new buffer with 1KB capacity
buf := kbuffer.NewBuffer(1024)

// Write data without allocations
buf.WriteString("Hello ")
buf.WriteString("World")

// Get result as string (zero-allocation)
result := buf.String() // "Hello World"

// Get result as bytes
data := buf.Bytes() // []byte("Hello World")

// Reset for reuse
buf.Reset()
```

### Pool

Buffer pools for efficient buffer reuse:

```go
// Get a buffer from the default pool
buf := kbuffer.Get()
defer kbuffer.Put(buf) // Return to pool when done

// Use the buffer
buf.WriteString("temporary data")
processData(buf.Bytes())
```

Custom pools with specific sizes:

```go
// Create a pool for 4KB buffers
pool := kbuffer.NewPool(4096)

// Get and return buffers
buf := pool.Get()
defer pool.Put(buf)
```

## API Reference

### Buffer Methods

#### Writing Operations

```go
// Write bytes to buffer
n, err := buf.Write([]byte("data"))

// Write string without allocation
n, err := buf.WriteString("text")

// Write single byte
err := buf.WriteByte('A')

// Write at specific offset
n, err := buf.WriteAt([]byte("data"), offset)

// Try write without error (for hot paths)
if buf.TryWrite(data) {
    // Success
}

// Append multiple bytes
err := buf.AppendBytes('H', 'e', 'l', 'l', 'o')
```

#### Reading Operations

```go
// Get written bytes
data := buf.Bytes()

// Get as string (zero-allocation)
str := buf.String()

// Get buffer length
length := buf.Len()

// Get remaining capacity
available := buf.Available()

// Get unused portion
remaining := buf.RemainingSlice()
```

#### Buffer Management

```go
// Reset position (keep memory)
buf.Reset()

// Clear content (zero memory)
buf.Clear()

// Truncate to n bytes
buf.Truncate(100)

// Ensure space available
err := buf.Grow(256)

// Extend position without writing
err := buf.Extend(10)

// Clone buffer with new memory
clone := buf.Clone()
```

### Pool Functions

```go
// Default pool operations
buf := kbuffer.Get()        // Get from default pool
kbuffer.Put(buf)            // Return to default pool

// Custom pool
pool := kbuffer.NewPool(8192)
buf := pool.Get()
pool.Put(buf)

// Get pool statistics
stats := pool.Stats()
fmt.Printf("Allocated: %d, In use: %d\n",
    stats.TotalAllocated, stats.InUse)
```

## Performance Characteristics

### Zero-Allocation Operations

The package uses unsafe operations to avoid allocations:

```go
// Traditional approach (allocates)
data := []byte(str)

// kbuffer approach (zero allocation)
buf.WriteString(str)
```

### Cache-Line Optimization

The Buffer struct is designed with CPU cache lines in mind:

```go
type Buffer struct {
    // Hot path fields in first cache line (64 bytes)
    data []byte   // 24 bytes (slice header)
    pos  int32    // 4 bytes
    cap  int32    // 4 bytes
    _    [32]byte // Padding to 64 bytes
}
```

### Compiler Directives

Strategic use of compiler directives for optimization:

- `//go:inline` - Forces inlining for small functions
- `//go:nosplit` - Prevents stack growth checks
- `//go:noescape` - Prevents heap escape (internal use)

## Use Cases

### High-Frequency Logging

```go
var bufPool = kbuffer.NewPool(512)

func LogMessage(level, msg string) {
    buf := bufPool.Get()
    defer bufPool.Put(buf)

    buf.WriteString(time.Now().Format(time.RFC3339))
    buf.WriteByte(' ')
    buf.WriteString(level)
    buf.WriteString(": ")
    buf.WriteString(msg)

    // Write to output (zero-allocation path)
    output.Write(buf.Bytes())
}
```

### Protocol Encoding

```go
func EncodePacket(cmd byte, payload []byte) []byte {
    buf := kbuffer.NewBuffer(len(payload) + 5)

    buf.WriteByte(0xAA)              // Start byte
    buf.WriteByte(cmd)                // Command
    buf.WriteByte(byte(len(payload))) // Length
    buf.Write(payload)                // Data
    buf.WriteByte(0x55)              // End byte

    return buf.Bytes()
}
```

### String Building

```go
func BuildQuery(table string, conditions []string) string {
    buf := kbuffer.Get()
    defer kbuffer.Put(buf)

    buf.WriteString("SELECT * FROM ")
    buf.WriteString(table)

    if len(conditions) > 0 {
        buf.WriteString(" WHERE ")
        for i, cond := range conditions {
            if i > 0 {
                buf.WriteString(" AND ")
            }
            buf.WriteString(cond)
        }
    }

    return buf.String() // Zero-allocation conversion
}
```

## Benchmarks

Performance comparison with standard library:

```
BenchmarkKBuffer_Write-8         500000000    3.2 ns/op     0 B/op    0 allocs/op
BenchmarkStdBuffer_Write-8       100000000   12.4 ns/op     0 B/op    0 allocs/op

BenchmarkKBuffer_String-8       1000000000    0.3 ns/op     0 B/op    0 allocs/op
BenchmarkStdBuffer_String-8      300000000    4.8 ns/op    48 B/op    1 allocs/op

BenchmarkPool_GetPut-8           200000000    8.7 ns/op     0 B/op    0 allocs/op
BenchmarkNewBuffer-8              50000000   31.2 ns/op  1024 B/op    1 allocs/op
```

## Safety Considerations

### Unsafe Operations

The package uses `unsafe` for performance but maintains safety through:

1. **Bounds Checking**: All operations verify capacity before access
2. **Immutable Capacity**: Buffers have fixed size, preventing overflows
3. **Clear Documentation**: Unsafe operations are clearly marked
4. **Comprehensive Testing**: Race detector and fuzzing coverage

### Memory Sharing

Be aware of memory sharing when using `Bytes()` and `String()`:

```go
buf := kbuffer.NewBuffer(100)
buf.WriteString("data")

// This slice shares memory with buffer
slice := buf.Bytes()

// Modifying buffer affects slice
buf.Reset()
buf.WriteString("new")
// slice content may change!

// Safe approach: clone if needed
safeCopy := make([]byte, len(slice))
copy(safeCopy, slice)
```

## Error Handling

The package defines specific errors for different failure modes:

```go
var (
    ErrBufferOverflow = errors.New("buffer overflow")
    ErrInvalidOffset  = errors.New("invalid offset")
    ErrPoolClosed     = errors.New("pool is closed")
)
```

Example error handling:

```go
if _, err := buf.Write(data); err != nil {
    if errors.Is(err, kbuffer.ErrBufferOverflow) {
        // Handle overflow
        buf = kbuffer.NewBuffer(buf.Cap() * 2)
    }
}
```

## Best Practices

1. **Use Pools for Temporary Buffers**: Always use pools for short-lived buffers
2. **Defer Put Operations**: Ensure buffers return to pool with defer
3. **Size Appropriately**: Choose buffer sizes based on expected data
4. **Clear Sensitive Data**: Use `Clear()` for security-sensitive content
5. **Avoid Escape**: Keep buffers local to prevent heap allocation

## Thread Safety

- Individual buffers are NOT thread-safe
- Buffer pools ARE thread-safe
- Use separate buffers per goroutine or synchronize access

## Dependencies

This package has no external dependencies beyond the Go standard library.

## License

Part of the Kitsunium SDK. See the main repository for license information.
