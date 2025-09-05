# Concurrency Detection for Kernel Packages

## Purpose

Detect and prevent concurrent access to non-thread-safe implementations, providing runtime safety checks and clear error messages when concurrency violations occur.

## When to Use

- Implementing unsafe/fast variants that aren't thread-safe
- Debugging concurrency issues in production
- Validating single-threaded assumptions
- Protecting critical sections from race conditions

## Core Detection Strategies

### 1. Goroutine ID Tracking

```go
// goroutine_check.go
//go:build !production
// +build !production

package kbuffer

import (
    "fmt"
    "runtime"
    "sync/atomic"
    "unsafe"
)

// UnsafeBuffer with goroutine tracking
type UnsafeBuffer struct {
    // Goroutine tracking
    ownerGID int64 // Goroutine ID that owns this buffer

    // Regular fields
    data []byte
    pos  int
}

// getGoroutineID extracts current goroutine ID
//go:nosplit
func getGoroutineID() int64 {
    b := make([]byte, 64)
    b = b[:runtime.Stack(b, false)]
    // Extract goroutine ID from stack trace
    // Format: "goroutine 1234 [..."
    for i := 10; i < len(b); i++ {
        if b[i] == ' ' {
            id := int64(0)
            for j := 10; j < i; j++ {
                id = id*10 + int64(b[j]-'0')
            }
            return id
        }
    }
    return -1
}

// checkOwnership verifies single goroutine access
func (b *UnsafeBuffer) checkOwnership() {
    gid := getGoroutineID()

    // First access - claim ownership
    if atomic.LoadInt64(&b.ownerGID) == 0 {
        if !atomic.CompareAndSwapInt64(&b.ownerGID, 0, gid) {
            // Lost race to another goroutine
            actualOwner := atomic.LoadInt64(&b.ownerGID)
            panic(fmt.Sprintf("concurrent access detected: goroutine %d and %d", gid, actualOwner))
        }
        return
    }

    // Verify same goroutine
    if atomic.LoadInt64(&b.ownerGID) != gid {
        panic(fmt.Sprintf("concurrent access detected: owned by goroutine %d, accessed by %d",
            b.ownerGID, gid))
    }
}

// Write with ownership check
func (b *UnsafeBuffer) Write(p []byte) (int, error) {
    b.checkOwnership()
    // Actual write implementation
    n := copy(b.data[b.pos:], p)
    b.pos += n
    return n, nil
}
```

### 2. Production Build (No Checks)

```go
// goroutine_check_prod.go
//go:build production
// +build production

package kbuffer

// No-op in production for zero overhead
func (b *UnsafeBuffer) checkOwnership() {}
```

### 3. Atomic Access Counter

```go
// concurrent_detector.go
package kbuffer

import (
    "fmt"
    "sync/atomic"
)

// ConcurrentDetector tracks concurrent access
type ConcurrentDetector struct {
    accessCount int32
}

// Enter marks entry to critical section
func (d *ConcurrentDetector) Enter() {
    if !atomic.CompareAndSwapInt32(&d.accessCount, 0, 1) {
        // Already being accessed
        count := atomic.LoadInt32(&d.accessCount)
        panic(fmt.Sprintf("concurrent access detected: %d goroutines", count))
    }
}

// Exit marks exit from critical section
func (d *ConcurrentDetector) Exit() {
    if !atomic.CompareAndSwapInt32(&d.accessCount, 1, 0) {
        panic("exit without enter or concurrent modification")
    }
}

// Usage in buffer
type MonitoredBuffer struct {
    ConcurrentDetector
    data []byte
}

func (b *MonitoredBuffer) Write(p []byte) (int, error) {
    b.Enter()
    defer b.Exit()

    // Safe to proceed - single access guaranteed
    return copy(b.data, p), nil
}
```

### 4. Mutex-Based Detection

```go
// mutex_detector.go
package kbuffer

import (
    "sync"
    "sync/atomic"
)

// MutexDetector uses trylock for detection
type MutexDetector struct {
    mu          sync.Mutex
    locked      int32
    violations  int64
}

// TryEnter attempts to enter critical section
func (d *MutexDetector) TryEnter() bool {
    if !d.mu.TryLock() {
        atomic.AddInt64(&d.violations, 1)
        return false
    }
    atomic.StoreInt32(&d.locked, 1)
    return true
}

// MustEnter panics on concurrent access
func (d *MutexDetector) MustEnter() {
    if !d.TryEnter() {
        violations := atomic.LoadInt64(&d.violations)
        panic(fmt.Sprintf("concurrent access violation #%d", violations))
    }
}

// Exit releases the lock
func (d *MutexDetector) Exit() {
    atomic.StoreInt32(&d.locked, 0)
    d.mu.Unlock()
}
```

### 5. Channel-Based Detection

```go
// channel_detector.go
package kbuffer

// ChannelDetector uses channels for detection
type ChannelDetector struct {
    ch chan struct{}
}

func NewChannelDetector() *ChannelDetector {
    return &ChannelDetector{
        ch: make(chan struct{}, 1),
    }
}

// Enter using channel
func (d *ChannelDetector) Enter() {
    select {
    case d.ch <- struct{}{}:
        // Acquired access
    default:
        panic("concurrent access detected via channel")
    }
}

// Exit releases channel
func (d *ChannelDetector) Exit() {
    select {
    case <-d.ch:
        // Released
    default:
        panic("exit without enter")
    }
}
```

## Build Tag Strategy

### Development vs Production

```go
// Development build includes all checks
// go build -tags="!production"

// Production build removes checks
// go build -tags="production"

// Race detector build uses safe implementation
// go build -race
```

### Conditional Compilation Files

```
buffer.go                 # Common interface
buffer_safe.go           # Safe implementation
buffer_unsafe.go         # Unsafe implementation
buffer_unsafe_dev.go     # Development checks (build: !production)
buffer_unsafe_prod.go    # No checks (build: production)
buffer_unsafe_race.go    # Race detector fallback (build: race)
```

## Advanced Detection Patterns

### 1. Stack Trace Analysis

```go
// stack_detector.go
package kbuffer

import (
    "bytes"
    "fmt"
    "runtime"
    "sync"
)

// StackDetector tracks access patterns via stack traces
type StackDetector struct {
    mu     sync.Mutex
    stacks map[string]int
}

func (d *StackDetector) RecordAccess() {
    // Capture stack trace
    buf := make([]byte, 4096)
    n := runtime.Stack(buf, false)
    stack := string(buf[:n])

    d.mu.Lock()
    defer d.mu.Unlock()

    if d.stacks == nil {
        d.stacks = make(map[string]int)
    }

    d.stacks[stack]++

    // Detect if multiple different stacks are accessing
    if len(d.stacks) > 1 {
        panic(fmt.Sprintf("concurrent access from %d different call sites", len(d.stacks)))
    }
}
```

### 2. Time-Based Detection

```go
// time_detector.go
package kbuffer

import (
    "sync/atomic"
    "time"
)

// TimeDetector detects overlapping access via timing
type TimeDetector struct {
    lastAccess int64 // Unix nano
    duration   int64 // Expected operation duration
}

func (d *TimeDetector) Enter(expectedDuration time.Duration) {
    now := time.Now().UnixNano()
    last := atomic.LoadInt64(&d.lastAccess)

    // Check if previous operation should have completed
    if last > 0 && now < last+int64(expectedDuration) {
        overlap := time.Duration(last + int64(expectedDuration) - now)
        panic(fmt.Sprintf("concurrent access detected: operations overlap by %v", overlap))
    }

    atomic.StoreInt64(&d.lastAccess, now)
}
```

### 3. Debug Mode with Logging

```go
// debug_detector.go
//go:build debug
// +build debug

package kbuffer

import (
    "log"
    "runtime"
    "sync/atomic"
)

// DebugDetector logs all access for analysis
type DebugDetector struct {
    accessID uint64
}

func (d *DebugDetector) LogAccess(operation string) {
    id := atomic.AddUint64(&d.accessID, 1)

    // Get caller info
    _, file, line, _ := runtime.Caller(1)

    // Get goroutine ID
    gid := getGoroutineID()

    log.Printf("[ACCESS #%d] Operation: %s, GID: %d, Location: %s:%d",
        id, operation, gid, file, line)
}
```

## Runtime Panic Messages

### Informative Panic Format

```go
func panicConcurrentAccess(details ...interface{}) {
    // Capture stack traces from all goroutines
    buf := make([]byte, 1<<20) // 1MB
    n := runtime.Stack(buf, true) // all=true for all goroutines

    msg := fmt.Sprintf(`
FATAL: Concurrent Access Violation
===================================
This buffer does not support concurrent access.
Use the thread-safe version or add external synchronization.

Details: %v

All Goroutines:
%s
`, details, buf[:n])

    panic(msg)
}
```

## Testing Concurrent Access

### 1. Deliberate Race Test

```go
func TestConcurrentAccessPanics(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping concurrent access test in short mode")
    }

    buf := newUnsafeBuffer(1024)
    data := []byte("test")

    // This should panic
    var wg sync.WaitGroup
    panicCount := int32(0)

    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            defer func() {
                if r := recover(); r != nil {
                    atomic.AddInt32(&panicCount, 1)
                }
            }()

            buf.Write(data)
        }()
    }

    wg.Wait()

    if atomic.LoadInt32(&panicCount) == 0 {
        t.Error("expected panic on concurrent access")
    }
}
```

### 2. Sequential Access Test

```go
func TestSequentialAccessWorks(t *testing.T) {
    buf := newUnsafeBuffer(1024)

    // Sequential access should work
    for i := 0; i < 100; i++ {
        data := []byte(fmt.Sprintf("data-%d", i))
        _, err := buf.Write(data)
        if err != nil {
            t.Fatalf("sequential write failed: %v", err)
        }
    }
}
```

## Do's and Don'ts

### Do's

- ✅ Use build tags to remove checks in production
- ✅ Provide clear panic messages with debugging info
- ✅ Test both concurrent and sequential access
- ✅ Document thread-safety guarantees clearly
- ✅ Use atomic operations for detection
- ✅ Include goroutine information in panics
- ✅ Offer thread-safe alternatives

### Don'ts

- ❌ Don't leave detection enabled in production
- ❌ Don't use expensive detection methods
- ❌ Don't silently ignore concurrent access
- ❌ Don't use global state for detection
- ❌ Don't forget to reset detectors in pooled objects
- ❌ Don't rely only on race detector

## Performance Impact

### Benchmark Detection Overhead

```go
func BenchmarkDetectionOverhead(b *testing.B) {
    b.Run("NoDetection", func(b *testing.B) {
        // Production build - no checks
    })

    b.Run("WithDetection", func(b *testing.B) {
        // Development build - with checks
    })

    b.Run("AtomicDetection", func(b *testing.B) {
        // Atomic counter based
    })

    b.Run("MutexDetection", func(b *testing.B) {
        // Mutex based
    })
}
```

### Expected Overhead

- No detection: 0ns
- Atomic detection: ~1-2ns
- Mutex detection: ~15-25ns
- Stack trace detection: ~500ns

## Related Documents

- [01-safe-unsafe-pattern.md](01-safe-unsafe-pattern.md) - Safe/unsafe implementations
- [03-memory-optimization.md](03-memory-optimization.md) - Memory techniques
- [../03-testing/01-unit-tests.md](../03-testing/01-unit-tests.md) - Testing strategies
- [../05-commands/02-validation.md](../05-commands/02-validation.md) - Validation commands
