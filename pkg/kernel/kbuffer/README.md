# KBuffer Package

Buffer management package for the Kitsunium SDK kernel.

## Overview

The kbuffer package provides two main components:

1. **Buffer** - A fixed-size byte buffer
2. **BufferPool** - A buffer pool with automatic size management

## Features

- Thread-safe pool operations
- Automatic power-of-2 sizing for memory alignment
- Statistics tracking for monitoring pool usage
- Buffer reuse through pooling

## Buffer

A fixed-size buffer with methods for writing, reading, and manipulation.

### Basic Usage

```go
import "github.com/kitsunium/sdk/pkg/kernel/kbuffer"

// Create a new buffer with 1024 bytes capacity
buf := kbuffer.NewBuffer(1024)

// Write data
n, err := buf.Write([]byte("Hello, "))
if err != nil {
    // Handle overflow
}

// Write string
n, err = buf.WriteString("World!")

// Get content
fmt.Println(buf.String()) // "Hello, World!"
fmt.Println(buf.Bytes())  // []byte("Hello, World!")

// Check available space
available := buf.Available() // Returns remaining bytes

// Reset for reuse
buf.Free()
```

### Advanced Methods

```go
// Write single byte
err := buf.WriteByte('A')

// Try write without error
if buf.TryWrite([]byte("data")) {
    // Success
}

// Write at specific offset
n, err := buf.WriteAt([]byte("Go"), 6)

// Append bytes
err = buf.AppendBytes('H', 'e', 'l', 'l', 'o')

// Get remaining buffer slice
remaining := buf.RemainingSlice()

// Extend position without writing
err = buf.Extend(10)

// Truncate buffer
buf.Truncate(5) // Keep only first 5 bytes

// Clear buffer (zeroes content)
buf.Clear()

// Reset with new backing slice
newSlice := make([]byte, 2048)
buf.Reset(newSlice)
```

## BufferPool

A pool that manages buffers of various sizes, automatically rounding to powers
of 2 for pooling.

### Basic Usage

```go
// Use the global pool
buf := kbuffer.Get(1024)
defer kbuffer.Put(buf) // Return to pool when done

// Write to buffer
copy(buf, []byte("Hello, World!"))
```

### Pool Management

```go
// Create custom pool
pool := kbuffer.NewBufferPool()

// Configure pool
pool.SetMaxSize(1 << 20)      // Max 1MB buffers
pool.SetClearOnPut(true)      // Clear buffers for security

// Pre-warm pool
sizes := []int{256, 512, 1024, 4096, 8192}
pool.Prewarm(sizes, 10) // Pre-allocate 10 buffers of each size

// Get and return buffers
buf := pool.Get(1000)    // Will round up to 1024
defer pool.Put(buf)

// Convenience methods for common sizes
buf1k := pool.Get1K()    // 1024 bytes
buf4k := pool.Get4K()    // 4096 bytes
buf64k := pool.Get64K()  // 65536 bytes
```

### Buffer Objects with Pool

```go
// Get Buffer object from pool
buf := kbuffer.GetBuffer(1024)
defer kbuffer.PutBuffer(buf)

// Use Buffer methods
buf.WriteString("Hello, World!")
fmt.Println(buf.String())
```

### Statistics

```go
// Get pool statistics
stats := pool.GetStats()
fmt.Printf("Gets: %d\n", stats.Gets)
fmt.Printf("Puts: %d\n", stats.Puts)
fmt.Printf("Hits: %d\n", stats.Hits)
fmt.Printf("Misses: %d\n", stats.Misses)
fmt.Printf("Allocs: %d\n", stats.Allocs)

// Reset statistics
pool.ResetStats()
```

## Implementation Details

This package uses several Go features:

- `//go:inline` directives on frequently called methods
- `clear()` builtin for slice zeroing
- `min()` builtin for comparisons
- Range over integers for loops
- `unsafe.String()` for string conversions
- `sync.Map` for concurrent pool access

## Example: HTTP Response Buffer

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Get buffer from pool
    buf := kbuffer.Get(4096)
    defer kbuffer.Put(buf)

    // Build response in buffer
    n := copy(buf, []byte("HTTP/1.1 200 OK\r\n"))
    n += copy(buf[n:], []byte("Content-Type: text/plain\r\n\r\n"))
    n += copy(buf[n:], []byte("Hello, World!"))

    // Write response
    w.Write(buf[:n])
}
```

## Example: Data Processing Pipeline

```go
func processData(input []byte) ([]byte, error) {
    // Get buffer sized for input
    buf := kbuffer.GetBuffer(len(input) * 2)
    defer kbuffer.PutBuffer(buf)

    // Process data
    for i, b := range input {
        if err := buf.WriteByte(b ^ 0xFF); err != nil {
            return nil, err
        }
        if i < len(input)-1 {
            buf.WriteByte(',')
        }
    }

    // Return copy of processed data
    result := make([]byte, buf.Len())
    copy(result, buf.Bytes())
    return result, nil
}
```

## Thread Safety

- **BufferPool**: All operations are thread-safe
- **Buffer**: Individual Buffer instances are NOT thread-safe. Use one Buffer
  per goroutine or synchronize access

## Best Practices

1. **Always return buffers to pool** using `defer` to prevent memory leaks
2. **Use appropriate sizes** - the pool will round up to the next power of 2
3. **Pre-warm pools** in init() for frequently used sizes
4. **Use TryWrite** in hot paths to avoid error checking overhead
5. **Configure clearing** (`SetClearOnPut()`) based on security requirements
6. **Monitor statistics** to tune pool configuration

## Testing

```bash
# Run tests
go test ./pkg/kernel/kbuffer/...

# Check coverage
go test -cover ./pkg/kernel/kbuffer/...

# Generate coverage report
go test -coverprofile=coverage.out ./pkg/kernel/kbuffer/...
go tool cover -html=coverage.out
```

## License

Part of the Kitsunium SDK - see main LICENSE file.
