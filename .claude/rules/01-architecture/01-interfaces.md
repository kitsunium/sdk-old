# Interface Design Patterns

## Purpose

Define clear, minimal, and performant interface contracts for kernel packages.

## When to Use

- Creating new public APIs
- Defining component boundaries
- Establishing performance contracts

## Rules

### 1. Minimal Interface Principle

Keep interfaces as small as possible, typically 1-3 methods.

**Good:**

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}
```

**Bad:**

```go
type Storage interface {
    Read([]byte) (int, error)
    Write([]byte) (int, error)
    Close() error
    Flush() error
    Seek(int64, int) (int64, error)
    // Too many methods - split into smaller interfaces
}
```

### 2. Performance Documentation

Document performance requirements directly in interfaces.

**Good:**

```go
// Processor provides high-performance processing operations.
//
// Performance requirements:
//   - Get: <100ns typical, O(1), zero allocations
//   - Put: <150ns typical, O(1), zero allocations
//   - Reset: <50ns typical, O(1)
type Processor interface {
    // Get returns a resource of size n.
    // Must complete in O(1) with zero allocations.
    Get(n int) []byte

    // Put returns the resource to the manager.
    // Must complete in O(1) with zero allocations.
    Put(b []byte)
}
```

### 3. Error Handling

Define error variables at package level, not in interfaces.

**Good:**

```go
var (
    ErrSizeTooLarge = errors.New("foo: requested size exceeds maximum")
    ErrManagerExhausted = errors.New("foo: manager exhausted")
)

type Manager interface {
    Get(size int) ([]byte, error) // Returns ErrSizeTooLarge or ErrManagerExhausted
}
```

## Do's and Don'ts

### Do's

- ✅ Start with interfaces before implementation
- ✅ Document performance requirements
- ✅ Keep interfaces focused on single responsibility
- ✅ Use standard library interfaces when applicable
- ✅ Define errors at package level

### Don'ts

- ❌ Create interfaces with more than 5 methods
- ❌ Mix different concerns in one interface
- ❌ Forget performance documentation
- ❌ Return concrete types from interface methods
- ❌ Define interfaces in separate packages from implementation

## Related Documents

- [02-structs.md](02-structs.md) - Implementing efficient structs
- [../02-implementation/01-safe-unsafe-pattern.md](../02-implementation/01-safe-unsafe-pattern.md) - Implementing interfaces
- [../04-conventions/02-documentation.md](../04-conventions/02-documentation.md) - Documentation standards
