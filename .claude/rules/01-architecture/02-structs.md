# Struct Optimization for Kernel Packages

## Purpose

Design and implement structs with optimal memory layout, cache alignment, and minimal overhead for maximum performance in kernel packages.

## When to Use

- When implementing interfaces defined in `interface.go`
- Creating internal data structures
- Designing performance-critical types
- Managing shared state in concurrent environments

## Core Rules

### 1. Memory Layout Optimization

**Rule**: Order struct fields from largest to smallest to minimize padding.

**Rationale**: Go compiler adds padding for alignment, wasting memory if fields are poorly ordered.

**Good Example**:

```go
// Optimized: 24 bytes (no padding)
type Buffer struct {
    data   []byte    // 24 bytes (slice header)
    size   int64     // 8 bytes
    offset int64     // 8 bytes
    flags  uint32    // 4 bytes
    state  uint32    // 4 bytes
}

// Use structlayout tool to verify:
// go get -u honnef.co/go/tools/cmd/structlayout
// structlayout -json pkg/kernel/kbuffer Buffer
```

**Bad Example**:

```go
// Inefficient: 48 bytes (16 bytes of padding!)
type Buffer struct {
    state  uint32    // 4 bytes + 4 padding
    data   []byte    // 24 bytes
    flags  uint32    // 4 bytes + 4 padding
    size   int64     // 8 bytes
    offset int64     // 8 bytes
}
```

### 2. Cache Line Alignment

**Rule**: Align frequently accessed fields to CPU cache lines (64 bytes on x64).

**Rationale**: Prevents false sharing and optimizes cache utilization.

**Good Example**:

```go
// Cache-line aligned for hot path fields
type ShardedBuffer struct {
    // Hot path fields (frequently accessed together)
    _        [0]struct{}        // Ensure alignment
    data     unsafe.Pointer     // 8 bytes
    len      int64              // 8 bytes
    cap      int64              // 8 bytes
    version  uint64             // 8 bytes (for CAS operations)
    _        [32]byte           // Padding to 64 bytes

    // Cold path fields (rarely accessed)
    mu       sync.Mutex         // 8 bytes
    stats    *Statistics        // 8 bytes
    config   *Config            // 8 bytes
}

// Ensure cache line alignment
var _ = unsafe.Sizeof(ShardedBuffer{}) // Must be multiple of 64
```

### 3. Atomic Field Alignment

**Rule**: Ensure 64-bit atomic fields are 64-bit aligned on 32-bit architectures.

**Rationale**: Prevents runtime panics on 32-bit systems.

**Good Example**:

```go
type Counter struct {
    // ALWAYS put 64-bit atomic fields first
    value    uint64  // Guaranteed 64-bit aligned
    _        [56]byte // Cache line padding

    name     string
    enabled  bool
}

// Safe atomic access
func (c *Counter) Add(delta uint64) {
    atomic.AddUint64(&c.value, delta)
}
```

**Bad Example**:

```go
type Counter struct {
    name     string
    enabled  bool
    value    uint64  // DANGER: May not be 64-bit aligned on 32-bit!
}
```

### 4. Embedding for Composition

**Rule**: Use embedding to compose behaviors and reduce allocations.

**Rationale**: Avoids pointer indirection and improves cache locality.

**Good Example**:

```go
// Embed sync primitives
type SafeBuffer struct {
    sync.RWMutex // Embedded, no allocation
    data []byte
    size int
}

// Embed common fields
type baseBuffer struct {
    data []byte
    size int
    cap  int
}

type UnsafeBuffer struct {
    baseBuffer // Inherit fields
    version uint64
}
```

### 5. Zero-Value Usability

**Rule**: Design structs to be useful in their zero state when possible.

**Rationale**: Eliminates initialization overhead and simplifies usage.

**Good Example**:

```go
type Buffer struct {
    data []byte // nil slice is valid empty buffer
}

// Zero value is immediately usable
var buf Buffer
n, _ := buf.Write([]byte("hello")) // Works without initialization

func (b *Buffer) Write(p []byte) (int, error) {
    // Lazy initialization
    if b.data == nil {
        b.data = make([]byte, 0, defaultSize)
    }
    b.data = append(b.data, p...)
    return len(p), nil
}
```

### 6. Pointer vs Value Receivers

**Rule**: Use pointer receivers for large structs (>64 bytes) or mutable state.

**Rationale**: Avoids copying overhead and allows mutation.

**Good Example**:

```go
// Large struct: use pointer receiver
type LargeBuffer struct {
    data [8192]byte
    meta [256]byte
}

func (b *LargeBuffer) Reset() { // Pointer receiver
    // Modify in place, no copy
}

// Small immutable struct: value receiver acceptable
type Point struct {
    X, Y float64
}

func (p Point) Distance() float64 { // Value receiver OK
    return math.Sqrt(p.X*p.X + p.Y*p.Y)
}
```

### 7. Struct Tags for Optimization

**Rule**: Use struct tags to control behavior and optimization.

**Good Example**:

```go
type OptimizedStruct struct {
    // Disable race detector for performance-critical field
    hotPath uint64 `race:"disable"`

    // Mark as no-escape for GC optimization
    buffer []byte `go:"noescape"`

    // Cache line padding
    _ [56]byte `pad:"cacheline"`
}
```

## Memory Layout Analysis Tools

### 1. Struct Layout Tool

```bash
# Install
go install honnef.co/go/tools/cmd/structlayout@latest

# Analyze
structlayout -json pkg/kernel/kbuffer Buffer

# Optimize
structlayout-optimize -json pkg/kernel/kbuffer Buffer
```

### 2. Memory Alignment Check

```go
// Runtime alignment verification
func init() {
    var b Buffer
    if unsafe.Offsetof(b.atomicField)%8 != 0 {
        panic("atomicField not 64-bit aligned")
    }
}
```

### 3. Size Assertion

```go
// Compile-time size check
const _ = unsafe.Sizeof(Buffer{}) - 48 // Fails if size != 48
```

## Do's and Don'ts

### Do's

- ✅ Order fields by size (largest first)
- ✅ Align hot fields to cache lines
- ✅ Put atomic fields first for alignment
- ✅ Use embedding for composition
- ✅ Design for zero-value usability
- ✅ Verify layout with tools
- ✅ Document memory layout decisions
- ✅ Use `unsafe.Sizeof` for size validation

### Don'ts

- ❌ Don't randomly order struct fields
- ❌ Don't mix hot and cold fields
- ❌ Don't ignore false sharing in concurrent structs
- ❌ Don't use pointers when values suffice
- ❌ Don't forget 32-bit alignment requirements
- ❌ Don't nest mutexes in frequently copied structs

## Performance Validation

### Benchmark Memory Layout

```go
func BenchmarkStructLayout(b *testing.B) {
    b.ReportAllocs()
    b.SetBytes(int64(unsafe.Sizeof(Buffer{})))

    for i := 0; i < b.N; i++ {
        var buf Buffer
        _ = buf
    }
}
```

### Cache Miss Analysis

```bash
# Profile cache misses
go test -bench=. -cpuprofile=cpu.prof
go tool pprof -list=FunctionName cpu.prof
```

## Advanced Techniques

### 1. Tagged Union Pattern

```go
type Value struct {
    typ uint32
    _   [4]byte // Explicit padding
    val uint64  // Can hold pointer or int
}
```

### 2. Slotted Array Pattern

```go
type SlottedArray struct {
    slots [256]unsafe.Pointer // Fixed size, no allocation
    len   uint32
    cap   uint32
}
```

### 3. Bit Packing

```go
type Flags struct {
    // Pack 8 booleans into 1 byte
    isReady    bool `bit:"0"`
    isLocked   bool `bit:"1"`
    isDirty    bool `bit:"2"`
    // ... 5 more flags
}
```

## Related Documents

- [01-interfaces.md](01-interfaces.md) - Interface design
- [03-file-organization.md](03-file-organization.md) - Struct file organization
- [../02-implementation/03-memory-optimization.md](../02-implementation/03-memory-optimization.md) - Memory techniques
- [../03-testing/02-benchmarks.md](../03-testing/02-benchmarks.md) - Performance testing
