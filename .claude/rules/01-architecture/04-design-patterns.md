# Design Patterns for Kernel Packages

## Purpose

Implement proven design patterns optimized for performance, flexibility, and maintainability in high-performance kernel packages.

## When to Use

- Designing new kernel package APIs
- Implementing configuration systems
- Creating extensible components
- Managing object lifecycles
- Building concurrent systems

## Core Patterns

### 1. Functional Options Pattern

**Purpose**: Provide flexible, extensible configuration without breaking changes.

**Rationale**: Allows optional parameters, maintains backward compatibility, enables default values.

**Implementation**:

```go
// options.go
package kbuffer

// Option configures a Buffer
type Option func(*options) error

// Internal options struct
type options struct {
    size      int
    pooled    bool
    sharded   bool
    shardCount int
}

// Default options
func defaultOptions() *options {
    return &options{
        size:      4096,
        pooled:    true,
        sharded:   false,
        shardCount: 16,
    }
}

// WithSize sets the buffer size
func WithSize(size int) Option {
    return func(o *options) error {
        if size <= 0 || size > MaxSize {
            return fmt.Errorf("invalid size: %d", size)
        }
        o.size = size
        return nil
    }
}

// WithPooling enables/disables object pooling
func WithPooling(enabled bool) Option {
    return func(o *options) error {
        o.pooled = enabled
        return nil
    }
}

// WithSharding enables sharded implementation
func WithSharding(shards int) Option {
    return func(o *options) error {
        if shards <= 0 || shards > 1024 {
            return fmt.Errorf("invalid shard count: %d", shards)
        }
        o.sharded = true
        o.shardCount = shards
        return nil
    }
}
```

**Usage**:

```go
// buffer.go
func NewBuffer(opts ...Option) (Buffer, error) {
    // Start with defaults
    o := defaultOptions()

    // Apply options
    for _, opt := range opts {
        if err := opt(o); err != nil {
            return nil, fmt.Errorf("invalid option: %w", err)
        }
    }

    // Create appropriate implementation
    if o.sharded {
        return newShardedBuffer(o), nil
    }
    if o.pooled {
        return newPooledBuffer(o), nil
    }
    return newSimpleBuffer(o), nil
}

// Client usage
buf, err := NewBuffer(
    WithSize(8192),
    WithSharding(32),
    WithPooling(true),
)
```

### 2. Builder Pattern (Performance-Optimized)

**Purpose**: Construct complex objects step-by-step with compile-time safety.

**Implementation**:

```go
// builder.go
package kbuffer

// Builder constructs Buffer instances
type Builder struct {
    size      int
    pooled    bool
    sharded   bool
    shardCount int
    err       error
}

// NewBuilder creates a new builder
func NewBuilder() *Builder {
    return &Builder{
        size: DefaultSize,
    }
}

// Size sets buffer size (fluent)
func (b *Builder) Size(size int) *Builder {
    if b.err != nil {
        return b
    }
    if size <= 0 || size > MaxSize {
        b.err = fmt.Errorf("invalid size: %d", size)
        return b
    }
    b.size = size
    return b
}

// Sharded enables sharding
func (b *Builder) Sharded(count int) *Builder {
    if b.err != nil {
        return b
    }
    b.sharded = true
    b.shardCount = count
    return b
}

// Build creates the buffer
func (b *Builder) Build() (Buffer, error) {
    if b.err != nil {
        return nil, b.err
    }

    // Create implementation based on configuration
    if b.sharded {
        return newShardedBuffer(b.toOptions()), nil
    }
    return newSimpleBuffer(b.toOptions()), nil
}

// Usage
buf, err := NewBuilder().
    Size(8192).
    Sharded(32).
    Build()
```

### 3. Object Pool Pattern (Lock-Free)

**Purpose**: Reuse objects to eliminate allocation overhead.

**Implementation**:

```go
// pool.go
package kbuffer

import (
    "sync"
    "sync/atomic"
)

// Pool manages reusable buffers
type Pool struct {
    // Use sync.Pool for automatic sizing
    pool sync.Pool

    // NEVER include statistics, metrics, or monitoring
    // Zero-overhead principle: no tracking of any kind
}

// NewPool creates a new buffer pool
func NewPool(size int) *Pool {
    return &Pool{
        pool: sync.Pool{
            New: func() interface{} {
                atomic.AddUint64(&p.news, 1)
                return &Buffer{
                    data: make([]byte, 0, size),
                }
            },
        },
    }
}

// Get retrieves a buffer from pool
//go:inline
func (p *Pool) Get() *Buffer {
    atomic.AddUint64(&p.gets, 1)
    buf := p.pool.Get().(*Buffer)
    buf.Reset() // Ensure clean state
    return buf
}

// Put returns buffer to pool
//go:inline
func (p *Pool) Put(buf *Buffer) {
    if buf == nil {
        return
    }
    atomic.AddUint64(&p.puts, 1)

    // Clear sensitive data
    for i := range buf.data {
        buf.data[i] = 0
    }
    buf.data = buf.data[:0]

    p.pool.Put(buf)
}

// Global pool for package-level convenience
var globalPool = NewPool(DefaultSize)

// GetBuffer gets a buffer from global pool
func GetBuffer() *Buffer {
    return globalPool.Get()
}

// PutBuffer returns buffer to global pool
func PutBuffer(buf *Buffer) {
    globalPool.Put(buf)
}
```

### 4. Singleton Pattern (Thread-Safe)

**Purpose**: Ensure single instance with lazy initialization.

**Implementation**:

```go
// global.go
package kbuffer

import (
    "sync"
    "sync/atomic"
)

var (
    instance atomic.Value
    initOnce sync.Once
)

// GetGlobalBuffer returns the global buffer instance
func GetGlobalBuffer() Buffer {
    initOnce.Do(func() {
        buf, _ := NewBuffer(
            WithSize(DefaultSize),
            WithSharding(runtime.NumCPU()),
        )
        instance.Store(buf)
    })
    return instance.Load().(Buffer)
}

// SetGlobalBuffer sets a custom global buffer
func SetGlobalBuffer(buf Buffer) {
    instance.Store(buf)
}
```

### 5. Strategy Pattern (Interface-Based)

**Purpose**: Select algorithms at runtime based on conditions.

**Implementation**:

```go
// strategy.go
package kbuffer

// WriteStrategy defines write behavior
type WriteStrategy interface {
    Write(buf *Buffer, data []byte) (int, error)
}

// DirectWrite writes directly to buffer
type DirectWrite struct{}

func (s DirectWrite) Write(buf *Buffer, data []byte) (int, error) {
    // Direct memory copy
    return copy(buf.data, data), nil
}

// CompressedWrite compresses before writing
type CompressedWrite struct {
    level int
}

func (s CompressedWrite) Write(buf *Buffer, data []byte) (int, error) {
    // Compress then write
    compressed := compress(data, s.level)
    return copy(buf.data, compressed), nil
}

// Buffer with strategy
type StrategyBuffer struct {
    data     []byte
    strategy WriteStrategy
}

func (b *StrategyBuffer) Write(data []byte) (int, error) {
    return b.strategy.Write(b, data)
}

// Select strategy based on size
func NewAdaptiveBuffer(size int) *StrategyBuffer {
    buf := &StrategyBuffer{
        data: make([]byte, size),
    }

    if size > LargeBufferThreshold {
        buf.strategy = CompressedWrite{level: 6}
    } else {
        buf.strategy = DirectWrite{}
    }

    return buf
}
```

### 6. Sharding Pattern (Concurrency)

**Purpose**: Reduce contention through data partitioning.

**Implementation**:

```go
// sharded.go
package kbuffer

import (
    "runtime"
    "sync/atomic"
)

// ShardedBuffer reduces contention via sharding
type ShardedBuffer struct {
    shards   []*BufferShard
    shardMask uint64
    counter  uint64
}

// BufferShard is a single shard
type BufferShard struct {
    _     [64]byte // Cache line padding
    mu    sync.Mutex
    data  []byte
    _     [64]byte // Cache line padding
}

// NewShardedBuffer creates sharded buffer
func NewShardedBuffer(size int) *ShardedBuffer {
    shardCount := uint64(runtime.NumCPU() * 2)
    // Round to power of 2
    shardCount = nextPowerOfTwo(shardCount)

    sb := &ShardedBuffer{
        shards:    make([]*BufferShard, shardCount),
        shardMask: shardCount - 1,
    }

    shardSize := size / int(shardCount)
    for i := range sb.shards {
        sb.shards[i] = &BufferShard{
            data: make([]byte, 0, shardSize),
        }
    }

    return sb
}

// getShard returns shard for current goroutine
//go:inline
func (sb *ShardedBuffer) getShard() *BufferShard {
    // Use atomic counter for distribution
    idx := atomic.AddUint64(&sb.counter, 1)
    return sb.shards[idx&sb.shardMask]
}

// Write to sharded buffer
func (sb *ShardedBuffer) Write(data []byte) (int, error) {
    shard := sb.getShard()
    shard.mu.Lock()
    n := copy(shard.data[len(shard.data):cap(shard.data)], data)
    shard.data = shard.data[:len(shard.data)+n]
    shard.mu.Unlock()
    return n, nil
}
```

### 7. Copy-on-Write Pattern

**Purpose**: Optimize read-heavy workloads with lazy copying.

**Implementation**:

```go
// cow.go
package kbuffer

import (
    "sync/atomic"
    "unsafe"
)

// COWBuffer implements copy-on-write semantics
type COWBuffer struct {
    data atomic.Value // *cowData
}

type cowData struct {
    bytes    []byte
    refCount int32
}

// Read performs zero-copy read
func (c *COWBuffer) Read() []byte {
    d := c.data.Load().(*cowData)
    atomic.AddInt32(&d.refCount, 1)
    defer atomic.AddInt32(&d.refCount, -1)
    return d.bytes
}

// Write creates copy if shared
func (c *COWBuffer) Write(p []byte) {
    d := c.data.Load().(*cowData)

    if atomic.LoadInt32(&d.refCount) > 1 {
        // Copy on write
        newData := &cowData{
            bytes:    make([]byte, len(d.bytes)),
            refCount: 1,
        }
        copy(newData.bytes, d.bytes)
        c.data.Store(newData)
        d = newData
    }

    // Perform write
    copy(d.bytes, p)
}
```

## Pattern Selection Guide

| Pattern            | Use When                    | Performance Impact      |
| ------------------ | --------------------------- | ----------------------- |
| Functional Options | Need flexible configuration | Minimal (compile-time)  |
| Builder            | Complex object construction | Minimal                 |
| Object Pool        | Frequent allocations        | High improvement        |
| Singleton          | Single shared instance      | Depends on contention   |
| Strategy           | Runtime algorithm selection | Interface call overhead |
| Sharding           | High concurrency            | Reduces contention      |
| Copy-on-Write      | Read-heavy workloads        | Optimizes reads         |

## Do's and Don'ts

### Do's

- ✅ Use functional options for public APIs
- ✅ Implement builders for complex objects
- ✅ Pool frequently allocated objects
- ✅ Shard data for concurrent access
- ✅ Use strategies for pluggable algorithms
- ✅ Document pattern usage and rationale

### Don'ts

- ❌ Don't over-engineer with unnecessary patterns
- ❌ Don't use patterns that add overhead without benefit
- ❌ Don't mix incompatible patterns
- ❌ Don't forget to benchmark pattern impact
- ❌ Don't use global state without synchronization

## Performance Considerations

### Pattern Overhead

```go
// Benchmark different patterns
func BenchmarkPatterns(b *testing.B) {
    b.Run("Direct", benchDirect)
    b.Run("FunctionalOptions", benchFunctionalOptions)
    b.Run("Builder", benchBuilder)
    b.Run("Pool", benchPool)
    b.Run("Sharded", benchSharded)
}
```

### Memory Impact

- Functional Options: ~0 bytes (stack allocated)
- Builder: ~64 bytes (single allocation)
- Pool: Amortized 0 allocations
- Sharding: N \* shard overhead

## Related Documents

- [01-interfaces.md](01-interfaces.md) - Interface patterns
- [02-structs.md](02-structs.md) - Struct design
- [../02-implementation/01-safe-unsafe-pattern.md](../02-implementation/01-safe-unsafe-pattern.md) - Implementation strategies
- [../02-implementation/03-memory-optimization.md](../02-implementation/03-memory-optimization.md) - Memory patterns
