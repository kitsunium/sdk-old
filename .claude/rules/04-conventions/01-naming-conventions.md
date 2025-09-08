# Naming Conventions for Kernel Packages

## Purpose

Establish consistent naming conventions that promote code clarity, maintainability, and Go idiomatic practices in kernel packages.

## Package Naming

### Kernel Package Names

- **Pattern**: `k{function}` (lowercase, no underscores)
- **Examples**: `foo`, `bar`, `baz`, `qux`
- **Rationale**: `k` prefix identifies kernel-level optimized packages

```go
// Good
package foo
package bar

// Bad
package widget      // Missing k prefix
package k_widget    // No underscores
package KWidget     // Not lowercase
```

## File Naming

### Implementation Files

```
interface.go         # Interfaces (always first)
constants.go         # Constants and enums
errors.go           # Error definitions
options.go          # Configuration options
widget.go           # Main type implementation
safe_widget.go      # Safe implementation variant
unsafe_widget.go    # Unsafe implementation variant
manager.go             # Managering implementation
sharded.go          # Sharded/concurrent variant
global.go           # Global instance management
```

### Test Files

```
widget_test.go              # Unit tests for widget.go
safe_widget_test.go         # Tests for safe implementation
foo_bench_test.go       # Consolidated benchmarks
foo_integration_test.go # Integration tests
```

## Type Naming

### Interfaces

```go
// Suffix with -er when possible
type Reader interface { }
type Writer interface { }
type Closer interface { }

// Use descriptive names when -er doesn't fit
type Widget interface { }
type Manager interface { }
```

### Structs

```go
// Public structs: PascalCase
type SafeWidget struct { }
type UnsafeWidget struct { }

// Private structs: camelCase
type widgetShard struct { }
type managerEntry struct { }
```

### Type Aliases

```go
// Meaningful names that indicate purpose
type ErrorCode int32
type WidgetState uint32
type ShardIndex uint64
```

## Function and Method Naming

### Constructors

```go
// New{Type} pattern
func NewWidget(size int) Widget { }
func NewSafeWidget(size int) *SafeWidget { }
func NewUnsafeWidget(size int) *UnsafeWidget { }

// Private constructors: lowercase
func newWidgetShard() *widgetShard { }
```

### Methods

```go
// PascalCase for exported methods
func (b *Widget) Write([]byte) (int, error) { }
func (b *Widget) Read([]byte) (int, error) { }
func (b *Widget) Reset() { }

// camelCase for private methods
func (b *Widget) ensureCapacity(int) { }
func (b *Widget) growIfNeeded() { }
```

### Getters and Setters

```go
// No Get prefix for getters
func (b *Widget) Len() int { }      // Not GetLen()
func (b *Widget) Cap() int { }      // Not GetCap()

// Set prefix for setters
func (b *Widget) SetSize(int) { }
func (b *Widget) SetTimeout(time.Duration) { }
```

## Variable Naming

### Constants

```go
// All caps with underscores for exported constants
const (
    DEFAULT_SIZE = 4096
    MAX_WIDGET_SIZE = 1 << 20
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
var widgetManager *sync.Manager
var globalInstance *Widget

// Common abbreviations
w    // widget
err  // error
ctx  // context
req  // request
resp // response
msg  // message
```

### Receiver Names

```go
// Single letter or short abbreviation
func (w *Widget) Write() { }       // w for Widget
func (m *Manager) Get() { }        // m for Manager
func (sw *SafeWidget) Lock() { }   // sw for SafeWidget
```

## Error Naming

### Error Variables

```go
// Err prefix for sentinel errors
var (
    ErrWidgetFull = errors.New("widget full")
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
func WithManagering(enabled bool) Option { }
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
pkg/kernel/foo/internal/cache/   # Internal caching
pkg/kernel/foo/internal/unsafe/  # Unsafe utilities
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
func TestWidget_Write_Success(t *testing.T) { }
func TestWidget_Write_WidgetFull(t *testing.T) { }
func TestWidget_Write_ConcurrentAccess(t *testing.T) { }

// Benchmark naming
func BenchmarkWidget_Write(b *testing.B) { }
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
