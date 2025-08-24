# Package value

## Overview

The `value` package provides utility functions for safely dereferencing pointers
in Go. It complements the `pointer` package by offering safe extraction of
values from pointers, returning appropriate zero values for nil pointers. This
is particularly useful when working with APIs that return pointer types or
handling optional struct fields.

## Features

- **Safe Dereferencing**: Never panics on nil pointers
- **Generic Functions**: Modern generic `Convert[T]` and `ConvertOr[T]`
  functions
- **Custom Defaults**: Specify custom default values with `ConvertOr`
- **Type-Specific Functions**: Legacy functions for backward compatibility
- **Zero Overhead**: All functions are inlined by the compiler

## Installation

```go
import "github.com/kitsunium/sdk/pkg/lib/value"
```

## API Reference

### Generic Functions (Recommended)

#### `Convert[T any](ptr *T) T`

Safely dereferences a pointer, returning the zero value if nil:

```go
var strPtr *string
str := value.Convert(strPtr) // Returns "" (zero value)

intPtr := &someInt
val := value.Convert(intPtr) // Returns the value of someInt

type User struct {
    Name string
    Age  int
}
var userPtr *User
user := value.Convert(userPtr) // Returns User{} (zero value)
```

#### `ConvertOr[T any](ptr *T, defaultValue T) T`

Dereferences a pointer with a custom default value for nil:

```go
var portPtr *int
port := value.ConvertOr(portPtr, 8080) // Returns 8080

namePtr := &someName
name := value.ConvertOr(namePtr, "Anonymous") // Returns value of someName

var configPtr *Config
config := value.ConvertOr(configPtr, DefaultConfig()) // Returns DefaultConfig()
```

### Type-Specific Functions (Deprecated)

While maintained for backward compatibility, using generic functions is
recommended:

```go
value.String(ptr *string) string           // Returns "" if nil
value.Int(ptr *int) int                    // Returns 0 if nil
value.Int8(ptr *int8) int8                 // Returns 0 if nil
value.Int16(ptr *int16) int16              // Returns 0 if nil
value.Int32(ptr *int32) int32              // Returns 0 if nil
value.Int64(ptr *int64) int64              // Returns 0 if nil
value.Uint(ptr *uint) uint                 // Returns 0 if nil
value.Uint8(ptr *uint8) uint8              // Returns 0 if nil
value.Uint16(ptr *uint16) uint16           // Returns 0 if nil
value.Uint32(ptr *uint32) uint32           // Returns 0 if nil
value.Uint64(ptr *uint64) uint64           // Returns 0 if nil
value.Float32(ptr *float32) float32        // Returns 0 if nil
value.Float64(ptr *float64) float64        // Returns 0 if nil
value.Bool(ptr *bool) bool                 // Returns false if nil
value.Byte(ptr *byte) byte                 // Returns 0 if nil
value.Rune(ptr *rune) rune                 // Returns 0 if nil
value.Complex64(ptr *complex64) complex64  // Returns 0 if nil
value.Complex128(ptr *complex128) complex128 // Returns 0 if nil
```

## Use Cases

### Processing API Responses

Safely handle optional fields from API responses:

```go
type APIResponse struct {
    Status   string
    Data     *string
    ErrorMsg *string
    Count    *int
}

func processResponse(resp APIResponse) {
    // Safely extract values with zero defaults
    data := value.Convert(resp.Data)       // "" if nil
    errorMsg := value.Convert(resp.ErrorMsg) // "" if nil
    count := value.Convert(resp.Count)     // 0 if nil

    if errorMsg != "" {
        log.Printf("Error: %s", errorMsg)
        return
    }

    log.Printf("Received %d items: %s", count, data)
}
```

### Configuration with Defaults

Handle optional configuration with custom defaults:

```go
type ServerConfig struct {
    Host     *string
    Port     *int
    Timeout  *int
    MaxConns *int
}

func NewServer(cfg ServerConfig) *Server {
    return &Server{
        host:     value.ConvertOr(cfg.Host, "localhost"),
        port:     value.ConvertOr(cfg.Port, 8080),
        timeout:  value.ConvertOr(cfg.Timeout, 30),
        maxConns: value.ConvertOr(cfg.MaxConns, 100),
    }
}

// Usage
server := NewServer(ServerConfig{
    Port: pointer.Convert(3000),
    // Other fields use defaults
})
```

### Database Records

Work with nullable database fields:

```go
type User struct {
    ID        int
    Username  string
    Email     *string
    Phone     *string
    Age       *int
    Verified  *bool
}

func displayUser(user User) {
    fmt.Printf("User #%d: %s\n", user.ID, user.Username)
    fmt.Printf("Email: %s\n", value.ConvertOr(user.Email, "Not provided"))
    fmt.Printf("Phone: %s\n", value.ConvertOr(user.Phone, "Not provided"))
    fmt.Printf("Age: %d\n", value.ConvertOr(user.Age, 0))
    fmt.Printf("Verified: %v\n", value.Convert(user.Verified))
}
```

### JSON Unmarshaling

Handle optional JSON fields gracefully:

```go
type Product struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Description *string  `json:"description,omitempty"`
    Price       float64  `json:"price"`
    Discount    *float64 `json:"discount,omitempty"`
    InStock     *bool    `json:"in_stock,omitempty"`
}

func (p Product) GetDescription() string {
    return value.ConvertOr(p.Description, "No description available")
}

func (p Product) GetFinalPrice() float64 {
    discount := value.Convert(p.Discount) // 0 if nil
    return p.Price * (1 - discount/100)
}

func (p Product) IsAvailable() bool {
    return value.ConvertOr(p.InStock, true) // Default to true if not specified
}
```

### Method Chaining

Safely chain operations with pointer returns:

```go
type QueryBuilder struct {
    table   string
    where   *string
    orderBy *string
    limit   *int
}

func (q QueryBuilder) Build() string {
    query := "SELECT * FROM " + q.table

    if where := value.Convert(q.where); where != "" {
        query += " WHERE " + where
    }

    if orderBy := value.Convert(q.orderBy); orderBy != "" {
        query += " ORDER BY " + orderBy
    }

    if limit := value.Convert(q.limit); limit > 0 {
        query += fmt.Sprintf(" LIMIT %d", limit)
    }

    return query
}
```

## Combining with Pointer Package

The `value` and `pointer` packages work together seamlessly:

```go
import (
    "github.com/kitsunium/sdk/pkg/lib/pointer"
    "github.com/kitsunium/sdk/pkg/lib/value"
)

// Create pointers
config := Config{
    Host: pointer.Convert("localhost"),
    Port: pointer.Convert(8080),
}

// Safely extract values
host := value.Convert(config.Host)
port := value.ConvertOr(config.Port, 3000)

// Round-trip example
original := 42
ptr := pointer.Convert(original)
recovered := value.Convert(ptr) // 42

var nilPtr *int
safe := value.Convert(nilPtr) // 0 (no panic)
```

## Performance Comparison

Benchmark comparison with manual nil checks:

```go
// Manual nil check
func manualCheck(ptr *int) int {
    if ptr != nil {
        return *ptr
    }
    return 0
}

// Using value package
func usingValue(ptr *int) int {
    return value.Convert(ptr)
}
```

Benchmark results:

```
BenchmarkManualCheck-8     1000000000    0.28 ns/op    0 B/op    0 allocs/op
BenchmarkValueConvert-8    1000000000    0.26 ns/op    0 B/op    0 allocs/op
BenchmarkConvertOr-8       1000000000    0.27 ns/op    0 B/op    0 allocs/op
```

The compiler inlines these functions, making them as fast as manual checks.

## Best Practices

### 1. Use Generic Functions

Prefer generic functions for new code:

```go
// Good
val := value.Convert(ptr)
val := value.ConvertOr(ptr, defaultVal)

// Avoid (deprecated)
val := value.Int(ptr)
```

### 2. Choose Appropriate Defaults

Use `ConvertOr` when zero values aren't suitable:

```go
// When 0 is not a good default
timeout := value.ConvertOr(cfg.Timeout, 30) // 30 seconds default

// When empty string is not ideal
name := value.ConvertOr(user.Name, "Anonymous")

// When false might not be the intent
enabled := value.ConvertOr(feature.Enabled, true)
```

### 3. Avoid Double Pointers

Don't unnecessarily dereference pointers to pointers:

```go
// Avoid
var ptrPtr **int
val := value.Convert(value.Convert(ptrPtr)) // Confusing

// Better: handle explicitly if needed
if ptrPtr != nil && *ptrPtr != nil {
    val := **ptrPtr
}
```

### 4. Document Default Behavior

Make default values clear in your API:

```go
// LoadConfig loads configuration from file.
// Missing values use defaults:
//   - Port: 8080
//   - Timeout: 30 seconds
//   - MaxConnections: 100
func LoadConfig(path string) Config {
    // ...
}
```

## Common Patterns

### Configuration Loading

```go
func LoadConfig(data map[string]*string) Config {
    return Config{
        Host:     value.ConvertOr(data["host"], "localhost"),
        Port:     value.ConvertOr(parseIntPtr(data["port"]), 8080),
        LogLevel: value.ConvertOr(data["log_level"], "info"),
    }
}
```

### Null-Safe Getters

```go
type User struct {
    name  *string
    email *string
    age   *int
}

func (u User) GetName() string {
    return value.ConvertOr(u.name, "Unknown")
}

func (u User) GetEmail() string {
    return value.ConvertOr(u.email, "")
}

func (u User) GetAge() int {
    return value.ConvertOr(u.age, 0)
}
```

### Optional Parameters

```go
type Options struct {
    Timeout  *time.Duration
    Retries  *int
    Verbose  *bool
}

func DoWork(opts Options) {
    timeout := value.ConvertOr(opts.Timeout, 5*time.Second)
    retries := value.ConvertOr(opts.Retries, 3)
    verbose := value.Convert(opts.Verbose) // false if nil

    // Use the values...
}
```

## Migration Guide

### From Manual Checks to Value Package

Before:

```go
func getString(ptr *string) string {
    if ptr != nil {
        return *ptr
    }
    return ""
}

func getIntWithDefault(ptr *int, def int) int {
    if ptr != nil {
        return *ptr
    }
    return def
}
```

After:

```go
func getString(ptr *string) string {
    return value.Convert(ptr)
}

func getIntWithDefault(ptr *int, def int) int {
    return value.ConvertOr(ptr, def)
}
```

## Thread Safety

All functions in this package are thread-safe and can be called concurrently.
They only read from the provided pointer and don't modify any shared state.

## Zero Values Reference

Default zero values returned for nil pointers:

| Type                                | Zero Value                |
| ----------------------------------- | ------------------------- |
| string                              | ""                        |
| int, int8, int16, int32, int64      | 0                         |
| uint, uint8, uint16, uint32, uint64 | 0                         |
| float32, float64                    | 0.0                       |
| bool                                | false                     |
| byte                                | 0                         |
| rune                                | 0                         |
| complex64, complex128               | 0+0i                      |
| struct                              | All fields at zero values |
| slice                               | nil                       |
| map                                 | nil                       |
| chan                                | nil                       |
| func                                | nil                       |
| interface                           | nil                       |
| pointer                             | nil                       |

## Dependencies

This package has no external dependencies and uses only Go standard library
features.

## License

Part of the Kitsunium SDK. See the main repository for license information.
