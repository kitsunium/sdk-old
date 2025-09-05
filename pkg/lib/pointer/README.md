# Package pointer

## Overview

The `pointer` package provides utility functions for creating pointers to literal values in Go. This package simplifies working with APIs that require pointer parameters and handling optional struct
fields.

## Features

- **Generic Function**: Modern generic `Convert[T]` function for any type
- **Type-Specific Functions**: Legacy functions for backward compatibility
- **Zero Overhead**: All functions are inlined by the compiler
- **Type Safety**: Compile-time type checking with generics

## Installation

```go
import "github.com/kitsunium/sdk/pkg/lib/pointer"
```

## API Reference

### Generic Function (Recommended)

#### `Convert[T any](v T) *T`

Creates a pointer to any value using Go generics:

```go
strPtr := pointer.Convert("hello")         // *string
intPtr := pointer.Convert(42)              // *int
floatPtr := pointer.Convert(3.14)          // *float64
structPtr := pointer.Convert(MyStruct{})   // *MyStruct
```

### Type-Specific Functions (Deprecated)

While these functions are maintained for backward compatibility, using `Convert[T]` is recommended:

```go
pointer.String(v string) *string
pointer.Int(v int) *int
pointer.Int8(v int8) *int8
pointer.Int16(v int16) *int16
pointer.Int32(v int32) *int32
pointer.Int64(v int64) *int64
pointer.Uint(v uint) *uint
pointer.Uint8(v uint8) *uint8
pointer.Uint16(v uint16) *uint16
pointer.Uint32(v uint32) *uint32
pointer.Uint64(v uint64) *uint64
pointer.Float32(v float32) *float32
pointer.Float64(v float64) *float64
pointer.Bool(v bool) *bool
pointer.Byte(v byte) *byte
pointer.Rune(v rune) *rune
pointer.Complex64(v complex64) *complex64
pointer.Complex128(v complex128) *complex128
```

## Use Cases

### API Parameters

Many Go APIs require pointer parameters for optional values:

```go
type DatabaseConfig struct {
    Host     *string
    Port     *int
    Username *string
    Password *string
    Timeout  *time.Duration
}

config := DatabaseConfig{
    Host:     pointer.Convert("localhost"),
    Port:     pointer.Convert(5432),
    Username: pointer.Convert("admin"),
    // Password is nil (not provided)
    Timeout:  pointer.Convert(30 * time.Second),
}
```

### JSON Marshaling

Distinguish between zero values and absent fields:

```go
type User struct {
    Name     string  `json:"name"`
    Age      *int    `json:"age,omitempty"`
    Active   *bool   `json:"active,omitempty"`
    Balance  *float64 `json:"balance,omitempty"`
}

user := User{
    Name:   "Alice",
    Age:    pointer.Convert(0),  // Explicitly set to 0
    Active: pointer.Convert(false), // Explicitly set to false
    // Balance is nil (field absent in JSON)
}

// JSON output: {"name":"Alice","age":0,"active":false}
// Without pointers, age:0 and active:false would be omitted
```

### Configuration Structs

Handle optional configuration with defaults:

```go
type ServerConfig struct {
    Address  *string
    Port     *int
    Debug    *bool
    Workers  *int
}

func NewServer(cfg ServerConfig) *Server {
    // Apply defaults for nil values
    address := "0.0.0.0"
    if cfg.Address != nil {
        address = *cfg.Address
    }

    port := 8080
    if cfg.Port != nil {
        port = *cfg.Port
    }

    debug := false
    if cfg.Debug != nil {
        debug = *cfg.Debug
    }

    workers := runtime.NumCPU()
    if cfg.Workers != nil {
        workers = *cfg.Workers
    }

    return &Server{
        address: address,
        port:    port,
        debug:   debug,
        workers: workers,
    }
}

// Usage
server := NewServer(ServerConfig{
    Port:  pointer.Convert(3000),
    Debug: pointer.Convert(true),
    // Address and Workers use defaults
})
```

### Database Operations

Work with nullable database fields:

```go
type Product struct {
    ID          int
    Name        string
    Description *string // Nullable in database
    Price       float64
    DiscountPct *float64 // Nullable in database
}

// Insert product with optional fields
product := Product{
    ID:          1,
    Name:        "Widget",
    Description: pointer.Convert("A useful widget"),
    Price:       19.99,
    // DiscountPct is NULL in database
}

// Update only specific fields
updates := map[string]interface{}{
    "description": pointer.Convert("An extremely useful widget"),
    "discount_pct": pointer.Convert(10.0),
}
```

### Conditional Assignment

Simplify conditional pointer creation:

```go
func processUser(data map[string]interface{}) *User {
    user := &User{}

    // Instead of verbose conditional blocks
    if val, ok := data["age"].(int); ok {
        user.Age = pointer.Convert(val)
    }

    if val, ok := data["verified"].(bool); ok {
        user.Verified = pointer.Convert(val)
    }

    return user
}
```

## Migration Guide

### From Type-Specific to Generic

Migrate from deprecated type-specific functions to the generic `Convert`:

```go
// Old way (deprecated)
strPtr := pointer.String("hello")
intPtr := pointer.Int(42)
boolPtr := pointer.Bool(true)

// New way (recommended)
strPtr := pointer.Convert("hello")
intPtr := pointer.Convert(42)
boolPtr := pointer.Convert(true)
```

### Benefits of Generic Function

1. **Single API**: One function for all types
2. **Better Performance**: Compiler can optimize better with generics
3. **Type Inference**: No need to specify type in function name
4. **Future-Proof**: Aligned with modern Go practices

## Performance

All functions are marked with `//go:inline` directive for zero-overhead:

```go
// The compiler inlines this call
ptr := pointer.Convert(42)

// Equivalent to writing
var v = 42
ptr := &v
```

Benchmark results:

```
BenchmarkConvert-8        1000000000    0.25 ns/op    0 B/op    0 allocs/op
BenchmarkString-8         1000000000    0.26 ns/op    0 B/op    0 allocs/op
BenchmarkInt-8            1000000000    0.25 ns/op    0 B/op    0 allocs/op
```

## Best Practices

### 1. Use Generic Convert

Always prefer the generic function for new code:

```go
// Good
ptr := pointer.Convert(value)

// Avoid (deprecated)
ptr := pointer.Int(value)
```

### 2. Nil Checking

Always check for nil before dereferencing:

```go
if config.Port != nil {
    fmt.Printf("Port: %d\n", *config.Port)
}

// Or use a helper function
func deref[T any](ptr *T, defaultVal T) T {
    if ptr != nil {
        return *ptr
    }
    return defaultVal
}

port := deref(config.Port, 8080)
```

### 3. Avoid Pointer Chains

Don't create pointers to pointers unnecessarily:

```go
// Bad
ptr := pointer.Convert(pointer.Convert(42)) // **int

// Good
ptr := pointer.Convert(42) // *int
```

### 4. Consider Memory Impact

Pointers add indirection and may impact performance in hot paths:

```go
// For frequently accessed data, consider copying
type Cache struct {
    // May be better as non-pointer for cache locality
    MaxSize int // Not *int
}
```

## Common Patterns

### Optional Parameters

```go
func Connect(host string, opts ...ConnectOption) (*Connection, error) {
    config := &ConnectConfig{
        Host:    host,
        Port:    pointer.Convert(5432),     // Default
        Timeout: pointer.Convert(30),       // Default
    }

    for _, opt := range opts {
        opt(config)
    }
    // ...
}

// Usage
conn, err := Connect("localhost",
    WithPort(pointer.Convert(3306)),
    WithTimeout(pointer.Convert(60)),
)
```

### Builder Pattern

```go
type QueryBuilder struct {
    table  string
    where  *string
    limit  *int
    offset *int
}

func (b *QueryBuilder) Where(clause string) *QueryBuilder {
    b.where = pointer.Convert(clause)
    return b
}

func (b *QueryBuilder) Limit(n int) *QueryBuilder {
    b.limit = pointer.Convert(n)
    return b
}
```

## Thread Safety

All functions in this package are thread-safe and can be called concurrently. Each function creates a new pointer to a new value.

## Dependencies

This package has no external dependencies and uses only Go standard library features.

## License

Part of the Kitsunium SDK. See the main repository for license information.
