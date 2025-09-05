# Naming Conventions for Kernel Packages

## Purpose

Establish consistent naming conventions that promote code clarity, maintainability, and Go idiomatic practices in kernel packages.

## Package Naming

### Kernel Package Names

- **Pattern**: `k{function}` (lowercase, no underscores)
- **Examples**: `kbuffer`, `kcache`, `kpool`, `kqueue`
- **Rationale**: `k` prefix identifies kernel-level optimized packages

```go
// Good
package kbuffer
package kcache

// Bad
package buffer      // Missing k prefix
package k_buffer    // No underscores
package KBuffer     // Not lowercase
```

## File Naming

### Implementation Files

```
interface.go         # Interfaces (always first)
constants.go         # Constants and enums
errors.go           # Error definitions
options.go          # Configuration options
buffer.go           # Main type implementation
safe_buffer.go      # Safe implementation variant
unsafe_buffer.go    # Unsafe implementation variant
pool.go             # Pooling implementation
sharded.go          # Sharded/concurrent variant
global.go           # Global instance management
```

### Test Files

```
buffer_test.go              # Unit tests for buffer.go
safe_buffer_test.go         # Tests for safe implementation
kbuffer_bench_test.go       # Consolidated benchmarks
kbuffer_integration_test.go # Integration tests
```

## Type Naming

### Interfaces

```go
// Suffix with -er when possible
type Reader interface { }
type Writer interface { }
type Closer interface { }

// Use descriptive names when -er doesn't fit
type Buffer interface { }
type Pool interface { }
```

### Structs

```go
// Public structs: PascalCase
type SafeBuffer struct { }
type UnsafeBuffer struct { }

// Private structs: camelCase
type bufferShard struct { }
type poolEntry struct { }
```

### Type Aliases

```go
// Meaningful names that indicate purpose
type ErrorCode int32
type BufferState uint32
type ShardIndex uint64
```

## Function and Method Naming

### Constructors

```go
// New{Type} pattern
func NewBuffer(size int) Buffer { }
func NewSafeBuffer(size int) *SafeBuffer { }
func NewUnsafeBuffer(size int) *UnsafeBuffer { }

// Private constructors: lowercase
func newBufferShard() *bufferShard { }
```

### Methods

```go
// PascalCase for exported methods
func (b *Buffer) Write([]byte) (int, error) { }
func (b *Buffer) Read([]byte) (int, error) { }
func (b *Buffer) Reset() { }

// camelCase for private methods
func (b *Buffer) ensureCapacity(int) { }
func (b *Buffer) growIfNeeded() { }
```

### Getters and Setters

```go
// No Get prefix for getters
func (b *Buffer) Len() int { }      // Not GetLen()
func (b *Buffer) Cap() int { }      // Not GetCap()

// Set prefix for setters
func (b *Buffer) SetSize(int) { }
func (b *Buffer) SetTimeout(time.Duration) { }
```

## Variable Naming

### Constants

```go
// All caps with underscores for exported constants
const (
    DEFAULT_SIZE = 4096
    MAX_BUFFER_SIZE = 1 << 20
)

// CamelCase for unexported constants
const (
    defaultTimeout = 5 * time.Second
    maxRetries = 3
)

// Enum-style constants
type State int
const (
    StateIdle State = iota
    StateReading
    StateWriting
    StateClosed
)
```

### Variables

```go
// Short names in small scopes
for i := 0; i < len(data); i++ { }

// Descriptive names in larger scopes
var bufferPool *sync.Pool
var globalInstance *Buffer

// Common abbreviations
buf  // buffer
err  // error
ctx  // context
req  // request
resp // response
msg  // message
```

### Receiver Names

```go
// Single letter or short abbreviation
func (b *Buffer) Write() { }       // b for Buffer
func (p *Pool) Get() { }            // p for Pool
func (sb *SafeBuffer) Lock() { }   // sb for SafeBuffer
```

## Error Naming

### Error Variables

```go
// Err prefix for sentinel errors
var (
    ErrBufferFull = errors.New("buffer full")
    ErrClosed = errors.New("closed")
    ErrTimeout = errors.New("timeout")
)
```

### Error Types

```go
// Error suffix for error types
type ValidationError struct { }
type TimeoutError struct { }
type ConfigError struct { }
```

## Option Naming

### Functional Options

```go
// With prefix for option functions
func WithSize(size int) Option { }
func WithTimeout(d time.Duration) Option { }
func WithPooling(enabled bool) Option { }
```

## Special Naming Patterns

### Build Tags

```go
// Lowercase with underscores
//go:build linux && amd64
//go:build !race && !debug
//go:build integration
```

### Compiler Directives

```go
//go:inline
//go:noinline
//go:nosplit
//go:noescape
//go:linkname
```

### Internal Packages

```
pkg/kernel/kbuffer/internal/cache/   # Internal caching
pkg/kernel/kbuffer/internal/unsafe/  # Unsafe utilities
```

## Acronym Handling

### Common Acronyms

```go
// Keep acronyms uppercase
ID   // not Id
URL  // not Url
API  // not Api
HTTP // not Http
SQL  // not Sql
JSON // not Json
XML  // not Xml

// In names
userID      // not userId
apiURL      // not apiUrl
HTTPClient  // not HttpClient
```

## Do's and Don'ts

### Do's

- ✅ Use meaningful, pronounceable names
- ✅ Keep names short but descriptive
- ✅ Use consistent naming throughout package
- ✅ Follow Go conventions
- ✅ Use common abbreviations
- ✅ Name tests descriptively

### Don'ts

- ❌ Don't use Hungarian notation
- ❌ Don't use underscores in Go names (except constants)
- ❌ Don't abbreviate unnecessarily
- ❌ Don't use generic names like "data" or "info"
- ❌ Don't use negative boolean names (use IsValid, not IsNotValid)

## Test Naming

### Test Functions

```go
// Test{Type}_{Method}_{Scenario}
func TestBuffer_Write_Success(t *testing.T) { }
func TestBuffer_Write_BufferFull(t *testing.T) { }
func TestBuffer_Write_ConcurrentAccess(t *testing.T) { }

// Benchmark naming
func BenchmarkBuffer_Write(b *testing.B) { }
func BenchmarkSafeVsUnsafe(b *testing.B) { }
```

### Table-Driven Test Cases

```go
tests := []struct {
    name     string  // Descriptive test case name
    input    []byte
    expected int
    wantErr  bool
}{
    {
        name:     "empty input",
        input:    []byte{},
        expected: 0,
        wantErr:  false,
    },
}
```

## Related Documents

- [02-documentation.md](02-documentation.md) - Documentation standards
- [03-code-organization.md](03-code-organization.md) - Code structure
- [../01-architecture/03-file-organization.md](../01-architecture/03-file-organization.md) - File naming
