# Code Organization - Kernel Structure Standards

## Purpose

Define mandatory code organization patterns for kernel packages to ensure consistent structure, maintainability, and readability across the codebase.

## When to Use

- When creating new packages or files
- When organizing imports and dependencies
- When structuring types and methods
- During refactoring to improve code organization
- During code review to ensure compliance

## Package Structure

### Standard Package Layout

```
pkg/kernel/kpackage/
├── interface.go         # ALL interfaces (mandatory)
├── constants.go         # ALL constants (if any)
├── errors.go           # ALL errors (if any)
├── doc.go              # Package documentation (optional)
│
├── type1.go            # First type implementation
├── type1_test.go       # Type1 tests
├── type1_unsafe.go     # Type1 unsafe version (if applicable)
│
├── type2.go            # Second type implementation
├── type2_test.go       # Type2 tests
│
├── helper.go           # Shared helpers (if needed)
├── helper_test.go      # Helper tests
│
├── kpackage_bench_test.go  # Consolidated benchmarks
├── mocks_test.go           # Test mocks
│
├── BUILD.bazel         # Build configuration
└── README.md           # Package README (optional)
```

### File Organization Rules

1. **One primary type per file**
2. **Group related functionality together**
3. **Separate concerns clearly**
4. **Test files mirror source files**
5. **Benchmarks consolidated in one file**

## Import Organization

### Import Grouping

```go
import (
    // Standard library imports (alphabetically)
    "context"
    "errors"
    "fmt"
    "sync"
    "time"

    // Third-party imports (alphabetically)
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/zap"

    // Internal imports (alphabetically)
    "pkg/kernel/kbuffer"
    "pkg/kernel/kpool"
    "pkg/util/validation"
)
```

### Import Aliases

```go
import (
    // Use aliases for clarity when needed
    stdlog "log"
    "github.com/sirupsen/logrus"

    // Avoid dot imports except in tests
    . "github.com/onsi/ginkgo" // Only in test files

    // Use underscore for side-effects only
    _ "net/http/pprof" // Register pprof handlers
)
```

## Type Organization

### Struct Definition Order

```go
// TypeName represents... [documentation]
type TypeName struct {
    // Exported fields first (alphabetically)
    Config    Config
    Logger    Logger

    // Unexported fields next (logically grouped)
    mu        sync.RWMutex  // Guards all fields below
    data      []byte        // Internal buffer
    position  int           // Current position
    closed    bool          // Closed flag

    // Cache-line padding for performance
    _         [64]byte      // Prevent false sharing
}
```

### Method Organization

```go
// 1. Constructor(s)
func NewTypeName() *TypeName { }
func NewTypeNameWithConfig(cfg Config) *TypeName { }

// 2. Interface implementations (grouped by interface)
// io.Reader implementation
func (t *TypeName) Read(p []byte) (int, error) { }

// io.Writer implementation
func (t *TypeName) Write(p []byte) (int, error) { }

// io.Closer implementation
func (t *TypeName) Close() error { }

// 3. Exported methods (alphabetically)
func (t *TypeName) Flush() error { }
func (t *TypeName) Reset() { }
func (t *TypeName) Size() int { }

// 4. Unexported methods (alphabetically)
func (t *TypeName) checkClosed() error { }
func (t *TypeName) grow(n int) { }
func (t *TypeName) init() { }
```

## Interface Organization

### Interface File Structure (interface.go)

```go
package kpackage

// ============ Public Interfaces ============

// Reader defines the interface for reading operations.
type Reader interface {
    Read([]byte) (int, error)
}

// Writer defines the interface for writing operations.
type Writer interface {
    Write([]byte) (int, error)
}

// ReadWriter combines Reader and Writer.
type ReadWriter interface {
    Reader
    Writer
}

// ============ Internal Interfaces ============

// allocator defines internal memory allocation strategy.
type allocator interface {
    alloc(size int) []byte
    free([]byte)
}

// pool defines internal pooling behavior.
type pool interface {
    get() interface{}
    put(interface{})
}
```

## Constants Organization

### Constants File Structure (constants.go)

```go
package kpackage

// ============ Exported Constants ============

// Size constants
const (
    // MinSize is the minimum buffer size
    MinSize = 512

    // DefaultSize is the default buffer size
    DefaultSize = 4096

    // MaxSize is the maximum buffer size
    MaxSize = 1 << 20 // 1MB
)

// State constants
const (
    // StateIdle indicates idle state
    StateIdle = iota

    // StateActive indicates active processing
    StateActive

    // StateClosed indicates closed state
    StateClosed
)

// ============ Internal Constants ============

const (
    // Internal magic numbers
    magicNumber = 0xDEADBEEF

    // Internal thresholds
    growthFactor = 2
    shrinkFactor = 4
)

// ============ Computed Constants ============

const (
    // Computed at compile time
    cacheLineSize = 64
    maxUint       = ^uint(0)
    maxInt        = int(maxUint >> 1)
)
```

## Error Organization

### Errors File Structure (errors.go)

```go
package kpackage

import (
    "errors"
    "fmt"
)

// ============ Sentinel Errors ============

var (
    // ErrClosed is returned when operating on closed resource
    ErrClosed = errors.New("resource closed")

    // ErrInvalidSize is returned for invalid size parameters
    ErrInvalidSize = errors.New("invalid size")

    // ErrTimeout is returned when operation times out
    ErrTimeout = errors.New("operation timeout")
)

// ============ Error Types ============

// ValidationError represents a validation failure
type ValidationError struct {
    Field string
    Value interface{}
    Err   error
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error in field %s: %v", e.Field, e.Err)
}

func (e *ValidationError) Unwrap() error {
    return e.Err
}

// ============ Error Constructors ============

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, err error) error {
    return &ValidationError{
        Field: field,
        Value: value,
        Err:   err,
    }
}
```

## Test Organization

### Test File Structure

```go
package kpackage

import (
    "testing"
    "sync"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// ============ Test Fixtures ============

var (
    testData = []byte("test data")
    testConfig = Config{Size: 1024}
)

// ============ Unit Tests ============

func TestTypeName_Method_Success(t *testing.T) {
    t.Parallel()
    // Test implementation
}

func TestTypeName_Method_Error(t *testing.T) {
    t.Parallel()
    // Test implementation
}

// ============ Integration Tests ============

func TestTypeName_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    // Test implementation
}

// ============ Test Helpers ============

func createTestInstance(t *testing.T) *TypeName {
    t.Helper()
    // Helper implementation
}

func assertTypeEqual(t *testing.T, expected, actual *TypeName) {
    t.Helper()
    // Assertion implementation
}
```

### Benchmark Organization

```go
// kpackage_bench_test.go

package kpackage

import (
    "testing"
)

// ============ Type1 Benchmarks ============

func BenchmarkType1_Method_Safe(b *testing.B) {
    // Benchmark implementation
}

func BenchmarkType1_Method_Unsafe(b *testing.B) {
    // Benchmark implementation
}

// ============ Type2 Benchmarks ============

func BenchmarkType2_Method_Safe(b *testing.B) {
    // Benchmark implementation
}

func BenchmarkType2_Method_Unsafe(b *testing.B) {
    // Benchmark implementation
}

// ============ Comparative Benchmarks ============

func BenchmarkComparison_Safe_vs_Unsafe(b *testing.B) {
    b.Run("Safe", func(b *testing.B) {
        // Safe version
    })
    b.Run("Unsafe", func(b *testing.B) {
        // Unsafe version
    })
}

// ============ Benchmark Helpers ============

func setupBenchmark(b *testing.B) *TypeName {
    // Setup code
    b.ResetTimer()
    return instance
}
```

## Function Organization

### Function Complexity Rules

```go
// Functions should be:
// - Short (typically <50 lines)
// - Focused (single responsibility)
// - Testable (minimal dependencies)

// ✅ GOOD: Focused function
func validateInput(data []byte) error {
    if len(data) == 0 {
        return ErrEmptyInput
    }
    if len(data) > MaxSize {
        return ErrTooLarge
    }
    return nil
}

// ❌ BAD: Too complex
func processEverything(data []byte) ([]byte, error) {
    // 200 lines of mixed validation, transformation,
    // I/O, and business logic
}
```

### Early Returns Pattern

```go
// ✅ GOOD: Early returns for clarity
func process(data []byte) error {
    // Validate first, fail fast
    if err := validate(data); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    // Check preconditions
    if !isReady() {
        return ErrNotReady
    }

    // Main logic with minimal nesting
    result := transform(data)
    return store(result)
}

// ❌ BAD: Deep nesting
func processBad(data []byte) error {
    if err := validate(data); err == nil {
        if isReady() {
            result := transform(data)
            if result != nil {
                return store(result)
            }
        }
    }
    return err
}
```

## Variable Organization

### Variable Declaration Groups

```go
// Group related variables
var (
    // Configuration
    defaultTimeout = 30 * time.Second
    maxRetries    = 3

    // Pools
    bufferPool = &sync.Pool{
        New: func() interface{} {
            return make([]byte, 1024)
        },
    }

    // Metrics
    requestCount atomic.Int64
    errorCount   atomic.Int64
)

// Avoid scattered declarations
var timeout = 30 * time.Second
var retries = 3
var pool = &sync.Pool{}
```

## Do's

✅ **Group related code together** logically ✅ **Order methods consistently** (constructor, interface, public, private) ✅ **Separate concerns** into different files ✅ **Use clear section comments**
for organization ✅ **Keep files focused** on single responsibility ✅ **Order imports** by standard/third-party/internal ✅ **Place constants and errors** in dedicated files ✅ **Use consistent
spacing** between sections ✅ **Organize tests** to mirror source structure ✅ **Document organization decisions** in comments

## Don'ts

❌ **Don't mix unrelated types** in one file ❌ **Don't scatter related functions** across files ❌ **Don't create deeply nested** package structures ❌ **Don't use random import ordering** ❌ **Don't
mix tests and implementation** in same file ❌ **Don't create circular dependencies** ❌ **Don't put helpers in utils.go** (be specific) ❌ **Don't exceed 1000 lines** per file (split if needed) ❌
**Don't mix public and private** methods randomly ❌ **Don't ignore logical grouping** of related code

## Code Metrics Guidelines

| Metric              | Maximum    | Ideal       |
| ------------------- | ---------- | ----------- |
| File length         | 1000 lines | <500 lines  |
| Function length     | 50 lines   | <30 lines   |
| Function complexity | 10         | <5          |
| Package files       | 20 files   | <10 files   |
| Import depth        | 5 levels   | <3 levels   |
| Type methods        | 20 methods | <10 methods |

## Related Documents

- [01-naming-conventions.md](01-naming-conventions.md) - Naming standards
- [02-documentation-standards.md](02-documentation-standards.md) - Documentation requirements
- [../01-architecture/01-package-structure.md](../01-architecture/01-package-structure.md) - Package architecture patterns
