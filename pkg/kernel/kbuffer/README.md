# kbuffer

Byte buffer implementations with explicit thread safety selection for
kernel-level operations.

## Package Design

The kbuffer package provides buffer implementations requiring explicit safety
choice:

- **Unsafe buffers**: Single-threaded access only, no synchronization overhead
- **Safe buffers**: Thread-safe with mutex protection for concurrent access
- **Sharded variants**: Distribute writes across multiple buffers to reduce
  contention

All buffers and pools must be explicitly instantiated. No global instances are
provided - the application or framework decides on instance management.

## Thread Safety Selection

### Explicit Safety Requirement

Every buffer creation requires an explicit choice between safe and unsafe
variants:

```go
// Single-threaded context: use unsafe buffer
buffer := kbuffer.NewUnsafeBuffer(1024)

// Multi-threaded context: use safe buffer
buffer := kbuffer.NewSafeBuffer(1024)

// High contention: use sharded safe buffer
buffer := kbuffer.NewSafeShardedBuffer(4096, 16) // 16 shards
```

### Buffer Operations

```go
// Write data
n, err := buf.Write([]byte("hello"))
buf.WriteString(" world")
buf.WriteByte('!')

// Read data
data := buf.Bytes()     // Get contents as []byte
str := buf.String()     // Get contents as string

// Buffer management
buf.Reset()             // Reset to empty, keep capacity
buf.Clear()             // Reset and zero memory
buf.Grow(1024)          // Ensure additional capacity
available := buf.Available() // Get remaining capacity
```

### Instance Management

#### No Global State

- All buffers and pools require explicit instantiation
- No global pools or buffers provided
- Application/framework manages instance lifecycle
- Allows for proper resource control and testing

#### Buffer Pooling

Applications can create pools for reduced allocations:

```go
// Application-managed pool instance
pool := kbuffer.NewPool()

// Get raw byte slice from pool
buf := pool.Get(1024)
// Use the buffer...
pool.Put(buf) // Return to pool

// Get Buffer interface from pool
buffer := pool.GetBuffer(1024)
// Use the buffer...
pool.PutBuffer(buffer) // Return to pool

// Configure pool behavior
pool.SetClearOnPut(true)      // Security: clear on return
pool.SetMaxSize(4 << 20)       // Limit: max 4MB per buffer
```

## Implementation Variants

### Unsafe Buffer (`NewUnsafeBuffer`)

- **Usage**: Single-threaded contexts only
- **Synchronization**: None
- **Access**: Will panic if accessed from multiple goroutines (development
  builds)
- **Typical throughput**: 1-2 GB/s write speed

### Safe Buffer (`NewSafeBuffer`)

- **Usage**: Multi-threaded contexts with low to moderate contention
- **Synchronization**: Single RWMutex for entire buffer
- **Access**: Thread-safe for concurrent operations
- **Typical throughput**: 200-500 MB/s write speed under contention

### Unsafe Sharded Buffer (`NewUnsafeShardedBuffer`)

- **Usage**: Single-threaded contexts with large data streams
- **Synchronization**: None (sharding for cache locality only)
- **Access**: Will panic if accessed from multiple goroutines (development
  builds)
- **Typical throughput**: 1.5-2.5 GB/s write speed

### Safe Sharded Buffer (`NewSafeShardedBuffer`)

- **Usage**: Multi-threaded contexts with high write contention
- **Synchronization**: Per-shard RWMutex
- **Access**: Thread-safe with reduced contention through sharding
- **Typical throughput**: 500 MB/s - 1 GB/s write speed under high contention

## Build Configuration

### Development Builds (default)

- Goroutine safety checks enabled for unsafe buffers
- Detects concurrent access violations
- Panics with diagnostic message on safety violations
- Minimal overhead through sampling (1 in 512 operations)

### Production Builds

- Build tag: `unsafe_no_check`
- All safety checks compiled out
- Zero overhead for safety checking
- Responsibility for correct usage lies with the application

## Technical Implementation

### Memory Layout

- Cache-line aligned structures (64 bytes)
- Padding to prevent false sharing between goroutines
- Careful field ordering for hot-path optimization
- Direct unsafe pointer operations for zero-copy where possible

### Buffer Growth Strategy

- Small buffers (<64KB): 2x growth factor
- Medium buffers (64KB-1MB): 1.5x growth factor
- Large buffers (>1MB): 1.25x growth factor
- Maximum single allocation: 16MB

### Pool Design

- Size-classed pools from 64 bytes to 4MB
- Power-of-2 size classes for fast selection
- Per-CPU pre-warming to reduce startup latency
- Automatic buffer clearing optional for security

## Usage Guidelines

### Single-Threaded Contexts

- Use unsafe variants for zero synchronization overhead
- Development builds will detect accidental concurrent access
- Suitable for request-scoped buffers in single-threaded handlers
- Typical use: protocol parsing, serialization

### Multi-Threaded Contexts

- Use safe variants for automatic synchronization
- Consider sharded variants when write contention exceeds 10%
- Shard count typically 4x CPU cores for optimal distribution
- Typical use: logging, shared output buffers

### Memory Pressure

- Buffers retain capacity after Reset() for reuse
- Call Clear() to zero memory for security-sensitive data
- Use pools for frequently allocated/deallocated buffers
- Pool size classes reduce fragmentation

## Testing

### Unit Tests

```bash
# Run with safety checks (default)
bazel test //pkg/kernel/kbuffer:kbuffer_test

# Run specific test
bazel test //pkg/kernel/kbuffer:kbuffer_test --test_filter=TestUnsafeBuffer
```

### Benchmarks

```bash
# Run benchmarks without safety checks
bazel test //pkg/kernel/kbuffer:kbuffer_bench_test --test_arg=-test.bench=. \
  --define gotags=unsafe_no_check

# Run specific benchmark
bazel test //pkg/kernel/kbuffer:kbuffer_bench_test \
  --test_arg=-test.bench=BenchmarkUnsafeBuffer
```

## Build Tags

- `unsafe_no_check`: Removes goroutine safety checks for production builds
