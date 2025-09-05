# Error Handling for Kernel Packages

## Purpose

Implement efficient, performant error handling strategies that provide clear diagnostics while maintaining zero-allocation principles in hot paths.

## When to Use

- Designing error types for kernel packages
- Implementing validation logic
- Creating error chains and wrapping
- Optimizing error paths for performance
- Building diagnostic and debugging capabilities

## Error Type Design

### 1. Sentinel Errors

```go
// errors.go
package kbuffer

import "errors"

// Sentinel errors for common conditions
var (
    // ErrBufferFull indicates buffer capacity exceeded
    ErrBufferFull = errors.New("buffer: full")

    // ErrBufferEmpty indicates no data available
    ErrBufferEmpty = errors.New("buffer: empty")

    // ErrClosed indicates operation on closed resource
    ErrClosed = errors.New("buffer: closed")

    // ErrInvalidSize indicates size parameter out of range
    ErrInvalidSize = errors.New("buffer: invalid size")

    // ErrConcurrentAccess indicates unsafe concurrent access
    ErrConcurrentAccess = errors.New("buffer: concurrent access detected")
)

// IsFull checks if error is buffer full
//go:inline
func IsFull(err error) bool {
    return errors.Is(err, ErrBufferFull)
}

// IsEmpty checks if error is buffer empty
//go:inline
func IsEmpty(err error) bool {
    return errors.Is(err, ErrBufferEmpty)
}
```

### 2. Custom Error Types

```go
// error_types.go
package kbuffer

import (
    "fmt"
    "runtime"
)

// ValidationError for input validation failures
type ValidationError struct {
    Field   string
    Value   interface{}
    Message string

    // Stack trace for debugging (only in debug builds)
    stack []uintptr
}

// Error implements error interface
func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %v (%s)",
        e.Field, e.Value, e.Message)
}

// WithStack adds stack trace (debug builds only)
//go:build debug
func (e *ValidationError) WithStack() *ValidationError {
    e.stack = make([]uintptr, 32)
    n := runtime.Callers(2, e.stack)
    e.stack = e.stack[:n]
    return e
}

// OperationError for operation failures
type OperationError struct {
    Op     string // Operation name
    Kind   string // Error kind
    Err    error  // Underlying error

    // Performance metrics
    Elapsed int64 // Nanoseconds
}

func (e *OperationError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s %s: %v", e.Op, e.Kind, e.Err)
    }
    return fmt.Sprintf("%s %s", e.Op, e.Kind)
}

// Unwrap for error chain
func (e *OperationError) Unwrap() error {
    return e.Err
}
```

### 3. Error Codes (Zero Allocation)

```go
// error_codes.go
package kbuffer

// ErrorCode represents error as integer (zero allocation)
type ErrorCode int32

const (
    ErrNone ErrorCode = iota
    ErrFull
    ErrEmpty
    ErrClosed
    ErrInvalidInput
    ErrOutOfBounds
    ErrConcurrent
    ErrCorrupted
)

// String returns error description
func (e ErrorCode) String() string {
    switch e {
    case ErrNone:
        return "no error"
    case ErrFull:
        return "buffer full"
    case ErrEmpty:
        return "buffer empty"
    case ErrClosed:
        return "buffer closed"
    case ErrInvalidInput:
        return "invalid input"
    case ErrOutOfBounds:
        return "index out of bounds"
    case ErrConcurrent:
        return "concurrent access"
    case ErrCorrupted:
        return "data corrupted"
    default:
        return "unknown error"
    }
}

// Error implements error interface
func (e ErrorCode) Error() string {
    return e.String()
}

// IsError checks if code represents error
//go:inline
func (e ErrorCode) IsError() bool {
    return e != ErrNone
}
```

## Performance-Optimized Error Handling

### 1. Error-Free Fast Path

```go
// fast_path.go
package kbuffer

// WriteResult combines result and error code (no allocation)
type WriteResult struct {
    N    int
    Code ErrorCode
}

// FastWrite returns result struct instead of (int, error)
//go:inline
func (b *Buffer) FastWrite(p []byte) WriteResult {
    // Fast path - no error allocation
    if b.closed {
        return WriteResult{0, ErrClosed}
    }

    if len(p) > b.available() {
        return WriteResult{0, ErrFull}
    }

    n := copy(b.data[b.pos:], p)
    b.pos += n
    return WriteResult{n, ErrNone}
}

// Usage avoids error allocation in success case
result := buf.FastWrite(data)
if result.Code != ErrNone {
    return 0, result.Code // Convert to error only when needed
}
```

### 2. Error Pooling

```go
// error_pool.go
package kbuffer

import "sync"

// ErrorPool reuses error objects
var errorPool = sync.Pool{
    New: func() interface{} {
        return &OperationError{}
    },
}

// GetError retrieves pooled error
func GetError(op, kind string, err error) *OperationError {
    e := errorPool.Get().(*OperationError)
    e.Op = op
    e.Kind = kind
    e.Err = err
    e.Elapsed = 0
    return e
}

// PutError returns error to pool
func PutError(e *OperationError) {
    if e == nil {
        return
    }
    // Clear fields
    e.Op = ""
    e.Kind = ""
    e.Err = nil
    e.Elapsed = 0
    errorPool.Put(e)
}

// Usage with defer
func (b *Buffer) OperationWithPooledError() error {
    err := GetError("Write", "failed", nil)
    defer PutError(err)

    // Operation logic
    if problem {
        err.Err = ErrBufferFull
        return err
    }

    return nil // Pool will reclaim err
}
```

### 3. Lazy Error Construction

```go
// lazy_error.go
package kbuffer

// LazyError delays string formatting until needed
type LazyError struct {
    format string
    args   []interface{}
}

// Error formats message only when called
func (e *LazyError) Error() string {
    return fmt.Sprintf(e.format, e.args...)
}

// Errorf creates lazy error
func Errorf(format string, args ...interface{}) error {
    return &LazyError{
        format: format,
        args:   args,
    }
}

// Usage - formatting happens only if error is printed
if size > maxSize {
    return Errorf("size %d exceeds maximum %d", size, maxSize)
}
```

## Validation Strategies

### 1. Inline Validation

```go
// validation.go
package kbuffer

// ValidateSize checks size constraints (inline-friendly)
//go:inline
func ValidateSize(size int) error {
    if size <= 0 {
        return ErrInvalidSize
    }
    if size > MaxBufferSize {
        return ErrBufferFull
    }
    return nil
}

// ValidateRange checks bounds (inline-friendly)
//go:inline
func ValidateRange(offset, length, capacity int) error {
    if offset < 0 || length < 0 {
        return ErrInvalidInput
    }
    if offset+length > capacity {
        return ErrOutOfBounds
    }
    return nil
}
```

### 2. Batch Validation

```go
// batch_validation.go
package kbuffer

// ValidationResult holds multiple validation errors
type ValidationResult struct {
    errors []error
}

// Add appends error if not nil
func (vr *ValidationResult) Add(err error) {
    if err != nil {
        vr.errors = append(vr.errors, err)
    }
}

// AddIf conditionally adds error
func (vr *ValidationResult) AddIf(condition bool, err error) {
    if condition {
        vr.errors = append(vr.errors, err)
    }
}

// Err returns combined error or nil
func (vr *ValidationResult) Err() error {
    switch len(vr.errors) {
    case 0:
        return nil
    case 1:
        return vr.errors[0]
    default:
        return &MultiError{errors: vr.errors}
    }
}

// MultiError combines multiple errors
type MultiError struct {
    errors []error
}

func (m *MultiError) Error() string {
    // Format multiple errors
    var b strings.Builder
    b.WriteString("multiple errors:")
    for i, err := range m.errors {
        fmt.Fprintf(&b, "\n  [%d] %v", i+1, err)
    }
    return b.String()
}
```

## Error Wrapping and Context

### 1. Efficient Wrapping

```go
// wrapping.go
package kbuffer

// WrapError adds context without allocation in success case
func WrapError(op string, err error) error {
    if err == nil {
        return nil
    }

    // Check if already wrapped to avoid double wrapping
    if _, ok := err.(*OperationError); ok {
        return err
    }

    return &OperationError{
        Op:  op,
        Err: err,
    }
}

// ChainError creates error chain
func ChainError(errors ...error) error {
    var first error
    for _, err := range errors {
        if err != nil {
            if first == nil {
                first = err
            } else {
                return &MultiError{errors: errors}
            }
        }
    }
    return first
}
```

### 2. Context Propagation

```go
// context_errors.go
package kbuffer

import "context"

// ContextError includes context information
type ContextError struct {
    Ctx context.Context
    Op  string
    Err error
}

func (e *ContextError) Error() string {
    if deadline, ok := e.Ctx.Deadline(); ok {
        return fmt.Sprintf("%s: %v (deadline: %v)", e.Op, e.Err, deadline)
    }
    return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// WithContext wraps error with context
func WithContext(ctx context.Context, op string, err error) error {
    if err == nil {
        return nil
    }

    // Check context cancellation
    if ctx.Err() != nil {
        return fmt.Errorf("%s: %w (context: %v)", op, err, ctx.Err())
    }

    return &ContextError{
        Ctx: ctx,
        Op:  op,
        Err: err,
    }
}
```

## Panic vs Error

### When to Panic

```go
// panic_conditions.go
package kbuffer

// MustWrite panics on error (for invariant violations)
func (b *Buffer) MustWrite(p []byte) int {
    n, err := b.Write(p)
    if err != nil {
        panic(fmt.Sprintf("buffer write failed: %v", err))
    }
    return n
}

// AssertInvariant panics if invariant violated
//go:inline
func AssertInvariant(condition bool, msg string) {
    if !condition {
        panic("invariant violation: " + msg)
    }
}

// CheckBounds panics on bounds violation
//go:inline
func CheckBounds(index, length int) {
    if index < 0 || index >= length {
        panic(fmt.Sprintf("index %d out of bounds [0:%d)", index, length))
    }
}
```

### Recovery Strategies

```go
// recovery.go
package kbuffer

// SafeExecute recovers from panics
func SafeExecute(fn func() error) (err error) {
    defer func() {
        if r := recover(); r != nil {
            // Convert panic to error
            switch v := r.(type) {
            case error:
                err = fmt.Errorf("panic: %w", v)
            case string:
                err = fmt.Errorf("panic: %s", v)
            default:
                err = fmt.Errorf("panic: %v", v)
            }
        }
    }()

    return fn()
}
```

## Testing Error Conditions

### Error Testing Helpers

```go
// error_test.go
package kbuffer

import "testing"

// ExpectError verifies error occurred
func ExpectError(t *testing.T, err error, expected error) {
    t.Helper()

    if err == nil {
        t.Fatalf("expected error %v, got nil", expected)
    }

    if !errors.Is(err, expected) {
        t.Fatalf("expected error %v, got %v", expected, err)
    }
}

// ExpectNoError verifies no error
func ExpectNoError(t *testing.T, err error) {
    t.Helper()

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

// ExpectPanic verifies panic occurs
func ExpectPanic(t *testing.T, fn func(), message string) {
    t.Helper()

    defer func() {
        if r := recover(); r == nil {
            t.Fatalf("expected panic with message %q", message)
        }
    }()

    fn()
}
```

## Do's and Don'ts

### Do's

- ✅ Use sentinel errors for common conditions
- ✅ Design error-free fast paths
- ✅ Pool error objects in hot paths
- ✅ Use error codes for zero-allocation
- ✅ Implement lazy error construction
- ✅ Validate inputs at API boundaries
- ✅ Provide clear error messages

### Don'ts

- ❌ Don't allocate errors in hot paths
- ❌ Don't use fmt.Errorf in performance-critical code
- ❌ Don't panic for expected errors
- ❌ Don't wrap errors multiple times
- ❌ Don't ignore error handling
- ❌ Don't use generic error messages

## Performance Benchmarks

```go
func BenchmarkErrorHandling(b *testing.B) {
    b.Run("SentinelError", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            err := ErrBufferFull
            _ = err
        }
    })

    b.Run("ErrorCode", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            code := ErrFull
            _ = code
        }
    })

    b.Run("FmtErrorf", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            err := fmt.Errorf("buffer full: %d", i)
            _ = err
        }
    })

    b.Run("LazyError", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            err := Errorf("buffer full: %d", i)
            _ = err
        }
    })
}
```

## Related Documents

- [01-safe-unsafe-pattern.md](01-safe-unsafe-pattern.md) - Error handling in unsafe code
- [02-concurrency-detection.md](02-concurrency-detection.md) - Concurrent error detection
- [../03-testing/01-unit-tests.md](../03-testing/01-unit-tests.md) - Testing error conditions
- [../04-conventions/02-documentation.md](../04-conventions/02-documentation.md) - Error documentation
