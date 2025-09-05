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
pkg/kernel/foo/
├── interface.go                 # Public API contracts (ALWAYS FIRST)
├── constants.go                 # Package constants and enums
├── errors.go                    # Error types and variables
├── options.go                   # Configuration options
├── doc.go                       # Package documentation
│
├── bar.go                       # Main implementation (one type per file)
├── bar_test.go                  # Unit tests for bar.go
│
├── baz.go                       # Secondary implementation
├── baz_test.go                  # Unit tests for baz.go
│
├── safe_bar.go                  # Safe implementation
├── safe_bar_test.go             # Tests for safe implementation
│
├── unsafe_bar.go                # Unsafe optimized version
├── unsafe_bar_test.go           # Tests for unsafe implementation
│
├── sharded.go                   # Sharded implementation for concurrency
├── sharded_test.go              # Tests for sharded implementation
│
├── global.go                    # Global instance management
├── global_test.go               # Tests for global instance
│
├── foo_bench_test.go            # Consolidated benchmarks
├── foo_integration_test.go      # Integration tests
│
├── internal/                    # Internal packages (not exported)
│   ├── cache/                   # Internal caching logic
│   └── utils/                   # Internal utilities
│
├── testdata/                    # Test fixtures and data
│   ├── golden/                  # Golden files for tests
│   └── fixtures/                # Test fixtures
│
└── BUILD.bazel                  # Bazel build configuration
```

## File Naming Rules

### 1. One Type Per File Rule

**Rule**: Each struct type gets its own file named after the type (lowercase).

**Rationale**: Improves code navigation, reduces merge conflicts, enforces single responsibility.

**Good Example**:

```go
// file: bar.go
package foo

type Bar struct {
    // Bar implementation
}

// All Bar methods in same file
func NewBar() *Bar { }
func (b *Bar) Process([]byte) error { }
func (b *Bar) Execute() (Result, error) { }
```

**Bad Example**:

```go
// DON'T: Multiple types in one file
// file: types.go
type Foo struct { }
type Bar struct { }
type Config struct { }
```

### 2. Test File Pairing

**Rule**: Every implementation file has a corresponding `_test.go` file.

**Rationale**: Maintains test locality and makes missing tests obvious.

**Structure**:

```
bar.go          → bar_test.go
safe_bar.go     → safe_bar_test.go
unsafe_bar.go   → unsafe_bar_test.go
baz.go          → baz_test.go
```

### 3. Special Files

#### interface.go

**Purpose**: Define all public interfaces and type contracts.

```go
package foo

// Bar defines the contract for bar operations
type Bar interface {
    Process([]byte) error
    Execute() (Result, error)
    Reset()
}

// Baz defines the baz interface
type Baz interface {
    Get() interface{}
    Put(interface{})
}
```

#### constants.go

**Purpose**: Package-level constants, enums, and configuration values.

```go
package foo

const (
    // DefaultSize is the default size
    DefaultSize = 4096

    // MaxSize is the maximum allowed size
    MaxSize = 1 << 20 // 1MB
)

// State represents operational state
type State uint32

const (
    StateIdle State = iota
    StateActive
    StateProcessing
    StateClosed
)
```

#### errors.go

**Purpose**: Custom error types and sentinel errors.

```go
package foo

import "errors"

var (
    // ErrCapacityExceeded is returned when capacity is exceeded
    ErrCapacityExceeded = errors.New("foo: capacity exceeded")

    // ErrClosed is returned on operations after Close
    ErrClosed = errors.New("foo: closed")
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
package foo

// Option configures a Foo instance
type Option func(*options)

type options struct {
    size     int
    managered   bool
    sharded  bool
}

// WithSize sets the size
func WithSize(size int) Option {
    return func(o *options) {
        o.size = size
    }
}
```

#### doc.go

**Purpose**: Package-level documentation.

```go
// Package foo provides high-performance, zero-allocation
// implementations optimized for kernel-level operations.
//
// The package offers both safe and unsafe implementations, with the
// unsafe version providing 30-50% better performance through careful
// use of unsafe operations and memory management.
//
// Example:
//
// instance := foo.New(foo.WithSize(4096))
// defer instance.Close()
//
// err := instance.Process(data)
// if err != nil {
//     return err
// }
package foo
```

### 4. Implementation Files

#### Safe Implementation Pattern

```go
// safe_bar.go
package foo

import "sync"

// SafeBar provides thread-safe operations
type SafeBar struct {
    mu   sync.RWMutex
    data []byte
    size int
}

// NewSafeBar creates a new thread-safe instance
func NewSafeBar(size int) *SafeBar {
    return &SafeBar{
        data: make([]byte, 0, size),
        size: size,
    }
}
```

#### Unsafe Implementation Pattern

```go
// unsafe_bar.go
//go:build !race
// +build !race

package foo

import "unsafe"

// UnsafeBar provides maximum performance through unsafe operations
type UnsafeBar struct {
    data uintptr // Raw pointer for zero-copy operations
    len  int
    cap  int
}

// NewUnsafeBar creates a high-performance instance
func NewUnsafeBar(size int) *UnsafeBar {
    // Implementation with unsafe optimizations
}
```

### 5. Test Organization

#### Unit Tests

```go
// bar_test.go
package foo

import "testing"

func TestBar_Process(t *testing.T) {
    t.Parallel()
    // Test implementation
}

func TestBar_Execute(t *testing.T) {
    t.Parallel()
    // Test implementation
}
```

#### Benchmarks (Consolidated)

```go
// foo_bench_test.go
package foo

import "testing"

func BenchmarkSafeBar_Process(b *testing.B) {
    // Benchmark safe implementation
}

func BenchmarkUnsafeBar_Process(b *testing.B) {
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
foo/
├── internal/
│   ├── cache/      # Internal caching logic
│   ├── manager/       # Internal managering
│   └── unsafe/     # Unsafe utilities
```

### 2. Test Data

**Rule**: Use `testdata/` for test fixtures, following Go conventions.

```
foo/
├── testdata/
│   ├── golden/     # Golden test files
│   │   ├── process_output.golden
│   │   └── execute_output.golden
│   └── fixtures/   # Input test data
│       ├── large_file.bin
│       └── small_file.txt
```

### 3. No Nested Public Packages

**Rule**: Kernel packages should be flat, not nested.

**Good**:

```
pkg/kernel/foo/
pkg/kernel/bar/
pkg/kernel/baz/
```

**Bad**:

```
pkg/kernel/foo/cache/  # Don't nest public packages
pkg/kernel/foo/manager/   # Use separate kernel packages
```

## Import Organization

### Import Grouping

```go
package foo

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
    "github.com/org/project/pkg/kernel/bar"
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
bar_linux.go      # Linux-specific implementation
bar_darwin.go     # macOS-specific implementation
bar_windows.go    # Windows-specific implementation
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
