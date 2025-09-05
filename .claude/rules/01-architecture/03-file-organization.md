# File Organization for Kernel Packages

## Purpose

Establish consistent file structure and naming conventions that promote maintainability, discoverability, and separation of concerns in kernel packages.

## When to Use

- Creating new kernel packages
- Refactoring existing packages
- Adding new functionality to packages
- Organizing test and benchmark files

## Core Package Structure

### Standard Layout

```
pkg/kernel/kpackage/
├── interface.go            # Public API contracts (ALWAYS FIRST)
├── constants.go            # Package constants and enums
├── errors.go              # Error types and variables
├── options.go             # Configuration options
├── doc.go                 # Package documentation
│
├── buffer.go              # Main implementation (one type per file)
├── buffer_test.go         # Unit tests for buffer.go
│
├── pool.go                # Object pool implementation
├── pool_test.go           # Unit tests for pool.go
│
├── safe_buffer.go         # Safe implementation
├── safe_buffer_test.go    # Tests for safe implementation
│
├── unsafe_buffer.go       # Unsafe optimized version
├── unsafe_buffer_test.go  # Tests for unsafe implementation
│
├── sharded.go             # Sharded implementation for concurrency
├── sharded_test.go        # Tests for sharded implementation
│
├── global.go              # Global instance management
├── global_test.go         # Tests for global instance
│
├── kpackage_bench_test.go # Consolidated benchmarks
├── kpackage_integration_test.go # Integration tests
│
├── internal/              # Internal packages (not exported)
│   ├── cache/            # Internal caching logic
│   └── utils/            # Internal utilities
│
├── testdata/             # Test fixtures and data
│   ├── golden/          # Golden files for tests
│   └── fixtures/        # Test fixtures
│
└── BUILD.bazel          # Bazel build configuration
```

## File Naming Rules

### 1. One Type Per File Rule

**Rule**: Each struct type gets its own file named after the type (lowercase).

**Rationale**: Improves code navigation, reduces merge conflicts, enforces single responsibility.

**Good Example**:

```go
// file: buffer.go
package kbuffer

type Buffer struct {
    // Buffer implementation
}

// All Buffer methods in same file
func NewBuffer() *Buffer { }
func (b *Buffer) Write([]byte) (int, error) { }
func (b *Buffer) Read([]byte) (int, error) { }
```

**Bad Example**:

```go
// DON'T: Multiple types in one file
// file: types.go
type Buffer struct { }
type Pool struct { }
type Config struct { }
```

### 2. Test File Pairing

**Rule**: Every implementation file has a corresponding `_test.go` file.

**Rationale**: Maintains test locality and makes missing tests obvious.

**Structure**:

```
buffer.go          → buffer_test.go
safe_buffer.go     → safe_buffer_test.go
unsafe_buffer.go   → unsafe_buffer_test.go
pool.go           → pool_test.go
```

### 3. Special Files

#### interface.go

**Purpose**: Define all public interfaces and type contracts.

```go
package kbuffer

// Buffer defines the contract for buffer operations
type Buffer interface {
    Read([]byte) (int, error)
    Write([]byte) (int, error)
    Reset()
}

// Pool defines object pooling interface
type Pool interface {
    Get() interface{}
    Put(interface{})
}
```

#### constants.go

**Purpose**: Package-level constants, enums, and configuration values.

```go
package kbuffer

const (
    // DefaultSize is the default buffer size
    DefaultSize = 4096

    // MaxSize is the maximum allowed buffer size
    MaxSize = 1 << 20 // 1MB
)

// State represents buffer state
type State uint32

const (
    StateIdle State = iota
    StateReading
    StateWriting
    StateClosed
)
```

#### errors.go

**Purpose**: Custom error types and sentinel errors.

```go
package kbuffer

import "errors"

var (
    // ErrBufferFull is returned when buffer capacity is exceeded
    ErrBufferFull = errors.New("buffer: full")

    // ErrClosed is returned on operations after Close
    ErrClosed = errors.New("buffer: closed")
)

// ValidationError represents input validation failure
type ValidationError struct {
    Field string
    Value interface{}
    Err   error
}
```

#### options.go

**Purpose**: Configuration options using functional options pattern.

```go
package kbuffer

// Option configures a Buffer
type Option func(*options)

type options struct {
    size     int
    pooled   bool
    sharded  bool
}

// WithSize sets buffer size
func WithSize(size int) Option {
    return func(o *options) {
        o.size = size
    }
}
```

#### doc.go

**Purpose**: Package-level documentation.

```go
// Package kbuffer provides high-performance, zero-allocation buffer
// implementations optimized for kernel-level operations.
//
// The package offers both safe and unsafe implementations, with the
// unsafe version providing 30-50% better performance through careful
// use of unsafe operations and memory management.
//
// Example:
//
//	buf := kbuffer.NewBuffer(kbuffer.WithSize(4096))
//	defer buf.Close()
//
//	n, err := buf.Write(data)
//	if err != nil {
//	    return err
//	}
package kbuffer
```

### 4. Implementation Files

#### Safe Implementation Pattern

```go
// safe_buffer.go
package kbuffer

import "sync"

// SafeBuffer provides thread-safe buffer operations
type SafeBuffer struct {
    mu   sync.RWMutex
    data []byte
    size int
}

// NewSafeBuffer creates a new thread-safe buffer
func NewSafeBuffer(size int) *SafeBuffer {
    return &SafeBuffer{
        data: make([]byte, 0, size),
        size: size,
    }
}
```

#### Unsafe Implementation Pattern

```go
// unsafe_buffer.go
//go:build !race
// +build !race

package kbuffer

import "unsafe"

// UnsafeBuffer provides maximum performance through unsafe operations
type UnsafeBuffer struct {
    data uintptr // Raw pointer for zero-copy operations
    len  int
    cap  int
}

// NewUnsafeBuffer creates a high-performance buffer
func NewUnsafeBuffer(size int) *UnsafeBuffer {
    // Implementation with unsafe optimizations
}
```

### 5. Test Organization

#### Unit Tests

```go
// buffer_test.go
package kbuffer

import "testing"

func TestBuffer_Write(t *testing.T) {
    t.Parallel()
    // Test implementation
}

func TestBuffer_Read(t *testing.T) {
    t.Parallel()
    // Test implementation
}
```

#### Benchmarks (Consolidated)

```go
// kbuffer_bench_test.go
package kbuffer

import "testing"

func BenchmarkSafeBuffer_Write(b *testing.B) {
    // Benchmark safe implementation
}

func BenchmarkUnsafeBuffer_Write(b *testing.B) {
    // Benchmark unsafe implementation
}

func BenchmarkComparison(b *testing.B) {
    b.Run("Safe", func(b *testing.B) {
        // Safe implementation
    })
    b.Run("Unsafe", func(b *testing.B) {
        // Unsafe implementation
    })
}
```

## Directory Structure Rules

### 1. Internal Packages

**Rule**: Use `internal/` for unexported helper packages.

```
kbuffer/
├── internal/
│   ├── cache/      # Internal caching logic
│   ├── pool/       # Internal pooling
│   └── unsafe/     # Unsafe utilities
```

### 2. Test Data

**Rule**: Use `testdata/` for test fixtures, following Go conventions.

```
kbuffer/
├── testdata/
│   ├── golden/     # Golden test files
│   │   ├── write_output.golden
│   │   └── read_output.golden
│   └── fixtures/   # Input test data
│       ├── large_file.bin
│       └── small_file.txt
```

### 3. No Nested Public Packages

**Rule**: Kernel packages should be flat, not nested.

**Good**:

```
pkg/kernel/kbuffer/
pkg/kernel/kcache/
pkg/kernel/kpool/
```

**Bad**:

```
pkg/kernel/kbuffer/cache/  # Don't nest public packages
pkg/kernel/kbuffer/pool/   # Use separate kernel packages
```

## Import Organization

### Import Grouping

```go
package kbuffer

import (
    // Standard library
    "errors"
    "fmt"
    "sync"
    "unsafe"

    // Third-party packages
    "github.com/stretchr/testify/assert"

    // Internal packages
    "github.com/org/project/internal/utils"

    // Same-module packages
    "github.com/org/project/pkg/kernel/kpool"
)
```

## Do's and Don'ts

### Do's

- ✅ One type per file (with all its methods)
- ✅ Test file for every implementation file
- ✅ Consolidate benchmarks in `*_bench_test.go`
- ✅ Use `internal/` for unexported helpers
- ✅ Group imports by category
- ✅ Use descriptive file names
- ✅ Keep package structure flat

### Don'ts

- ❌ Don't mix multiple types in one file
- ❌ Don't create deeply nested package structures
- ❌ Don't use generic names like `utils.go` or `helpers.go`
- ❌ Don't put tests in separate packages (use same package)
- ❌ Don't create circular dependencies
- ❌ Don't export internal implementation details

## Build Tags and Conditional Compilation

### Platform-Specific Files

```
buffer_linux.go      # Linux-specific implementation
buffer_darwin.go     # macOS-specific implementation
buffer_windows.go    # Windows-specific implementation
```

### Build Tag Organization

```go
//go:build !race && amd64
// +build !race,amd64

// File compiled only for AMD64 without race detector
```

## Related Documents

- [01-interfaces.md](01-interfaces.md) - Interface definitions
- [02-structs.md](02-structs.md) - Struct implementation
- [../04-conventions/01-naming-conventions.md](../04-conventions/01-naming-conventions.md) - Naming rules
- [../03-testing/01-unit-tests.md](../03-testing/01-unit-tests.md) - Test organization
