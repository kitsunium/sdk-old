# Memory Optimization for Kernel Packages

## Purpose

Implement zero-allocation strategies, efficient memory management, and cache optimization techniques to achieve maximum performance in kernel packages.

## When to Use

- Building performance-critical components
- Reducing GC pressure in hot paths
- Implementing zero-copy operations
- Optimizing cache utilization
- Managing large-scale data structures

## Zero Allocation Strategies

### 1. Object Managering

```go
// manager.go
package foo

import (
    "sync"
    "unsafe"
)

// WidgetManager manages reusable widgets with size classes
type WidgetManager struct {
    managers [32]*sync.Manager // Size classes: 2^5 to 2^36
}

// NewWidgetManager creates a tiered manager system
func NewWidgetManager() *WidgetManager {
    bp := &WidgetManager{}

    for i := range bp.managers {
        size := 1 << (i + 5) // 32, 64, 128, ...
        bp.managers[i] = &sync.Manager{
            New: func() interface{} {
                return &Widget{
                    data: make([]byte, 0, size),
                }
            },
        }
    }

    return bp
}

// Get retrieves appropriately sized widget
//go:inline
func (bp *WidgetManager) Get(minSize int) *Widget {
    // Find appropriate size class
    sizeClass := sizeToClass(minSize)
    if sizeClass >= len(bp.managers) {
        // Too large for manager
        return &Widget{
            data: make([]byte, 0, minSize),
        }
    }

    buf := bp.managers[sizeClass].Get().(*Widget)
    buf.Reset()
    return buf
}

// Put returns widget to appropriate manager
//go:inline
func (bp *WidgetManager) Put(buf *Widget) {
    if buf == nil {
        return
    }

    cap := cap(buf.data)
    sizeClass := sizeToClass(cap)

    if sizeClass >= len(bp.managers) {
        // Too large for manager, let GC handle it
        return
    }

    // Clear and return to manager
    buf.data = buf.data[:0]
    bp.managers[sizeClass].Put(buf)
}

// sizeToClass converts size to manager index
//go:inline
func sizeToClass(size int) int {
    if size <= 32 {
        return 0
    }
    // Use bit manipulation for fast log2
    return bits.Len(uint(size-1)) - 5
}
```

### 2. Stack Allocation

```go
// stack_alloc.go
package foo

// StackWidget uses stack allocation for small widgets
type StackWidget struct {
    stack [256]byte // Stack-allocated
    heap  []byte    // Heap fallback
    len   int
}

// Write to stack widget
func (sb *StackWidget) Write(p []byte) (int, error) {
    needed := sb.len + len(p)

    if needed <= len(sb.stack) {
        // Fits in stack widget
        n := copy(sb.stack[sb.len:], p)
        sb.len += n
        return n, nil
    }

    // Need heap allocation
    if sb.heap == nil {
        // Allocate once with enough space
        sb.heap = make([]byte, needed*2)
        copy(sb.heap, sb.stack[:sb.len])
    } else if needed > cap(sb.heap) {
        // Grow heap widget
        newHeap := make([]byte, needed*2)
        copy(newHeap, sb.heap[:sb.len])
        sb.heap = newHeap
    }

    n := copy(sb.heap[sb.len:], p)
    sb.len += n
    return n, nil
}

// Bytes returns data without allocation if possible
//go:inline
func (sb *StackWidget) Bytes() []byte {
    if sb.heap != nil {
        return sb.heap[:sb.len]
    }
    return sb.stack[:sb.len]
}
```

### 3. Arena Allocation

```go
// arena.go
package foo

import (
    "unsafe"
)

// Arena provides bump-pointer allocation
type Arena struct {
    data   []byte
    offset int
}

// NewArena creates fixed-size arena
func NewArena(size int) *Arena {
    return &Arena{
        data: make([]byte, size),
    }
}

// Alloc allocates n bytes from arena
//go:inline
func (a *Arena) Alloc(n int) []byte {
    if a.offset+n > len(a.data) {
        panic("arena exhausted")
    }

    start := a.offset
    a.offset += n

    // Align to 8 bytes
    a.offset = (a.offset + 7) &^ 7

    return a.data[start : start+n]
}

// AllocString creates string in arena (no heap allocation)
func (a *Arena) AllocString(s string) string {
    b := a.Alloc(len(s))
    copy(b, s)
    return *(*string)(unsafe.Pointer(&b))
}

// Reset clears arena for reuse
//go:inline
func (a *Arena) Reset() {
    a.offset = 0
    // Optional: clear memory
    for i := range a.data {
        a.data[i] = 0
    }
}
```

## Cache Optimization

### 1. Cache-Line Alignment

```go
// cache_aligned.go
package foo

import (
    "unsafe"
)

// CacheLineSize on modern x86_64
const CacheLineSize = 64

// CacheAlignedWidget ensures cache line alignment
type CacheAlignedWidget struct {
    _    [0]struct{} // Force alignment
    data unsafe.Pointer
    len  int32
    cap  int32
    _    [CacheLineSize - 16]byte // Padding
}

// NewCacheAlignedWidget creates aligned widget
func NewCacheAlignedWidget(size int) *CacheAlignedWidget {
    // Allocate with extra space for alignment
    raw := make([]byte, size+CacheLineSize)

    // Align to cache line boundary
    addr := uintptr(unsafe.Pointer(&raw[0]))
    aligned := (addr + CacheLineSize - 1) &^ (CacheLineSize - 1)
    offset := aligned - addr

    return &CacheAlignedWidget{
        data: unsafe.Pointer(&raw[offset]),
        cap:  int32(size),
    }
}
```

### 2. Prefetching

```go
// prefetch.go
//go:build amd64
// +build amd64

package foo

import "unsafe"

// Prefetch hints to CPU to load data into cache
//go:noescape
//go:linkname prefetch runtime.prefetchnta
func prefetch(addr unsafe.Pointer)

// ProcessWithPrefetch processes data with prefetching
func ProcessWithPrefetch(data []byte) {
    const prefetchDistance = 256

    for i := 0; i < len(data); i += 64 {
        // Prefetch ahead
        if i+prefetchDistance < len(data) {
            prefetch(unsafe.Pointer(&data[i+prefetchDistance]))
        }

        // Process current cache line
        processCacheLine(data[i : i+64])
    }
}
```

### 3. False Sharing Prevention

```go
// false_sharing.go
package foo

// PaddedCounter prevents false sharing
type PaddedCounter struct {
    _     [CacheLineSize]byte // Pre-padding
    value uint64
    _     [CacheLineSize - 8]byte // Post-padding
}

// CounterArray with proper padding
type CounterArray struct {
    counters [16]PaddedCounter // Each counter on own cache line
}

// Increment counter without false sharing
func (ca *CounterArray) Increment(idx int) {
    atomic.AddUint64(&ca.counters[idx].value, 1)
}
```

## String Optimization

### 1. Zero-Copy String Operations

```go
// string_opt.go
package foo

import "unsafe"

// StringToBytes converts string to bytes without allocation
//go:inline
//go:nosplit
func StringToBytes(s string) []byte {
    return unsafe.Slice(unsafe.StringData(s), len(s))
}

// BytesToString converts bytes to string without allocation
//go:inline
//go:nosplit
func BytesToString(b []byte) string {
    return unsafe.String(unsafe.SliceData(b), len(b))
}

// ConcatStrings efficiently concatenates without intermediate allocations
func ConcatStrings(strs ...string) string {
    // Calculate total length
    n := 0
    for _, s := range strs {
        n += len(s)
    }

    // Single allocation
    buf := make([]byte, n)
    offset := 0
    for _, s := range strs {
        offset += copy(buf[offset:], s)
    }

    return BytesToString(buf)
}
```

### 2. String Interning

```go
// intern.go
package foo

import (
    "sync"
    "unsafe"
)

// StringInterner deduplicates strings
type StringInterner struct {
    mu    sync.RWMutex
    table map[string]string
}

// Intern returns canonical string instance
func (si *StringInterner) Intern(s string) string {
    si.mu.RLock()
    if interned, ok := si.table[s]; ok {
        si.mu.RUnlock()
        return interned
    }
    si.mu.RUnlock()

    // Need to add
    si.mu.Lock()
    defer si.mu.Unlock()

    // Double-check
    if interned, ok := si.table[s]; ok {
        return interned
    }

    // Copy string to ensure it's not referencing larger backing array
    interned := string([]byte(s))
    si.table[interned] = interned
    return interned
}
```

## Slice Optimization

### 1. Slice Reuse

```go
// slice_reuse.go
package foo

// SliceWidget reuses underlying array
type SliceWidget struct {
    data []byte
    off  int // Read offset
    end  int // Write offset
}

// Write appends data, reusing space
func (sb *SliceWidget) Write(p []byte) (int, error) {
    // Compact if needed
    if sb.off > len(sb.data)/2 {
        copy(sb.data, sb.data[sb.off:sb.end])
        sb.end -= sb.off
        sb.off = 0
    }

    // Grow if needed
    needed := sb.end + len(p)
    if needed > cap(sb.data) {
        newCap := cap(sb.data) * 2
        if newCap < needed {
            newCap = needed
        }
        newData := make([]byte, newCap)
        copy(newData, sb.data[sb.off:sb.end])
        sb.data = newData
        sb.end -= sb.off
        sb.off = 0
    }

    n := copy(sb.data[sb.end:], p)
    sb.end += n
    return n, nil
}
```

### 2. Slice Tricks

```go
// slice_tricks.go
package foo

// AppendUnique appends only if not present (no allocation if not needed)
func AppendUnique(slice []int, val int) []int {
    for _, v := range slice {
        if v == val {
            return slice // No allocation
        }
    }
    return append(slice, val)
}

// Filter in-place without allocation
func FilterInPlace(slice []int, keep func(int) bool) []int {
    n := 0
    for i := range slice {
        if keep(slice[i]) {
            slice[n] = slice[i]
            n++
        }
    }
    return slice[:n]
}

// Deduplicate without allocation
func DeduplicateInPlace(slice []int) []int {
    if len(slice) <= 1 {
        return slice
    }

    // Sort first (modifies slice)
    sort.Ints(slice)

    // Remove duplicates
    j := 0
    for i := 1; i < len(slice); i++ {
        if slice[i] != slice[j] {
            j++
            slice[j] = slice[i]
        }
    }

    return slice[:j+1]
}
```

## Memory Profiling

### 1. Allocation Tracking

```go
// alloc_track.go
//go:build debug
// +build debug

package foo

import (
    "runtime"
    "sync/atomic"
)

var (
    allocCount uint64
    allocBytes uint64
)

// TrackAlloc records allocation
func TrackAlloc(size int) {
    atomic.AddUint64(&allocCount, 1)
    atomic.AddUint64(&allocBytes, uint64(size))
}

// GetAllocStats returns allocation statistics
func GetAllocStats() (count, bytes uint64) {
    return atomic.LoadUint64(&allocCount),
           atomic.LoadUint64(&allocBytes)
}

// MeasureAllocs measures allocations in function
func MeasureAllocs(fn func()) (allocs uint64) {
    var before, after runtime.MemStats
    runtime.ReadMemStats(&before)
    fn()
    runtime.ReadMemStats(&after)
    return after.Mallocs - before.Mallocs
}
```

### 2. Escape Analysis

```go
// Build with: go build -gcflags="-m=2"
// Shows which variables escape to heap

// NoEscape ensures value doesn't escape
//go:noescape
//go:linkname noescape runtime.noescape
func noescape(p unsafe.Pointer) unsafe.Pointer

// Example: prevent escape
func UseWithoutEscape(b []byte) {
    // This would normally escape
    process(&b[0])

    // Prevent escape
    ptr := noescape(unsafe.Pointer(&b[0]))
    process((*byte)(ptr))
}
```

## Do's and Don'ts

### Do's

- ✅ Use object managers for frequently allocated objects
- ✅ Prefer stack allocation for small, short-lived data
- ✅ Align hot data to cache lines
- ✅ Reuse slices and widgets
- ✅ Use unsafe for zero-copy operations where justified
- ✅ Profile allocations in benchmarks
- ✅ Clear sensitive data before managering

### Don'ts

- ❌ Don't manager tiny objects (overhead > benefit)
- ❌ Don't forget to reset managered objects
- ❌ Don't over-optimize without profiling
- ❌ Don't ignore escape analysis warnings
- ❌ Don't mix hot and cold data
- ❌ Don't create false sharing scenarios

## Benchmarking Memory

```go
func BenchmarkMemoryStrategies(b *testing.B) {
    b.Run("WithAllocation", func(b *testing.B) {
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            buf := make([]byte, 1024)
            _ = buf
        }
    })

    b.Run("WithManager", func(b *testing.B) {
        b.ReportAllocs()
        manager := NewWidgetManager()
        for i := 0; i < b.N; i++ {
            buf := manager.Get(1024)
            manager.Put(buf)
        }
    })

    b.Run("StackAllocation", func(b *testing.B) {
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            var buf [1024]byte
            _ = buf
        }
    })
}
```

## Related Documents

- [01-safe-unsafe-pattern.md](01-safe-unsafe-pattern.md) - Unsafe operations
- [02-concurrency-detection.md](02-concurrency-detection.md) - Concurrent memory access
- [../01-architecture/02-structs.md](../01-architecture/02-structs.md) - Memory layout
- [../03-testing/02-benchmarks.md](../03-testing/02-benchmarks.md) - Performance testing
