# KError - Error Management for Go

Error management package for Go with metrics and distributed tracing support.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
  - [Error Definition](#error-definition)
  - [Error Instances](#error-instances)
  - [Configuration](#configuration)
  - [Registry](#registry)
  - [Context Integration](#context-integration)
  - [Metrics](#metrics)
  - [Result Type](#result-type)
- [Advanced Usage](#advanced-usage)
- [Performance](#performance)
- [Best Practices](#best-practices)

## Features

- Object pooling for instance reuse
- Generic `Result[T]` type for error handling
- Metrics collection with custom collector support
- Optional stack trace capture for debugging
- Automatic trace and span ID extraction from context
- Thread-safe operations
- Configuration options with defaults
- JSON marshal/unmarshal support
- Global registry for error management and lookup

## Installation

```bash
go get github.com/kitsunium/sdk/pkg/kernel/kerror
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/kitsunium/sdk/pkg/kernel/kerror"
)

// Define application errors
var (
    ErrNotFound = kerror.Define(kerror.KConfig{
        Code:    404,
        Message: "Resource not found",
    })

    ErrInternal = kerror.Define(kerror.KConfig{
        Code:    500,
        Message: "Internal server error",
    })
)

func main() {
    // Create error instance
    err := ErrNotFound.New()
    defer err.Release() // Return to pool

    // Add contextual information
    err.WithTag("resource", "user")
       .WithDetail("user_id", 12345)

    // Handle error
    fmt.Println(err.Error())
}
```

## API Reference

### Error Definition

#### `Define(config KConfig) KError`

Defines a new error type with the given configuration.

```go
type KConfig struct {
    Package string // Optional: package name (auto-detected if empty)
    Code    int    // Required: error code
    Message string // Optional: error message (auto-generated if empty)
}
```

```go
// Basic definition
err := kerror.Define(kerror.KConfig{
    Code:    404,
    Message: "Not found",
})

// With package
err := kerror.Define(kerror.KConfig{
    Package: "myapp",
    Code:    500,
    Message: "Server error",
})
```

### Error Instances

#### Creating Instances

```go
// Create new instance
inst := err.New()
defer inst.Release()

// Create with formatted message
inst := err.Newf("User %d not found", userID)
defer inst.Release()

// Wrap existing error
inst := err.Wrap(originalErr)
defer inst.Release()

// Wrap with formatted message
inst := err.Wrapf(originalErr, "Failed to process user %d", userID)
defer inst.Release()
```

#### Instance Methods

```go
// Tags - lightweight key-value metadata
inst.WithTag(key, value string) *Instance
inst.Tag(key string) (string, bool)
inst.Tags() map[string]string
inst.HasTag(key string) bool

// Batch tag operations
inst.BatchWithTags(tags ...struct{ Key, Value string })
inst.MapTags(fn func(key, value string) (string, string))
inst.FilterTags(fn func(key, value string) bool)
inst.MergeTags(other *Instance)

// Details - structured data
inst.WithDetail(key string, value any) *Instance
inst.Detail(key string) (any, bool)
inst.Details() map[string]any
inst.HasDetail(key string) bool
inst.WithDetails(details map[string]any) *Instance
inst.BatchWithDetails(details ...struct{ Key string; Value any })

// Type-safe detail retrieval
user, ok := kerror.DetailAs[User](inst, "user")

// Result types for tags and details
result := inst.TagResult(key string) Result[string]
result := inst.DetailResult(key string) Result[any]

// Context
inst.WithContext(ctx context.Context) *Instance
inst.Context() context.Context

// Cloning
cloned := inst.Clone() *Instance

// Error interface
inst.Error() string
inst.Is(target error) bool
inst.Unwrap() error

// Metadata
inst.ID() uint32
inst.Code() int
inst.Message() string
inst.Package() string
inst.Stack() []uintptr
inst.OTelAttributes() map[string]any

// JSON support
data, err := json.Marshal(inst)
err := json.Unmarshal(data, &inst)

// Cleanup
inst.Release() // Return to pool
```

### Configuration

#### `Configure(config GlobalConfig)`

Configures global settings for the error package.

```go
type GlobalConfig struct {
    EnableStackTrace  bool   // Capture stack traces (default: false)
    EnableMetrics     bool   // Enable metrics collection (default: false)
    EnableValidation  bool   // Validate inputs (default: true)
    MaxInstancePool   int    // Max pooled instances (default: 10000)
    DefaultPackage    string // Default package name (default: "unknown")
    MaxTags           int    // Max tags per instance (default: 100)
    MaxDetails        int    // Max details per instance (default: 100)
    MaxTagKeyLen      int    // Max tag key length (default: 128)
    MaxTagValueLen    int    // Max tag value length (default: 1024)
    StackTraceDepth   int    // Max stack frames (default: 32)
}
```

```go
kerror.Configure(kerror.GlobalConfig{
    EnableStackTrace: true,
    EnableMetrics:    true,
    MaxInstancePool:  5000,
    DefaultPackage:   "myapp",
})
```

#### `GetConfig() GlobalConfig`

Returns the current global configuration.

```go
config := kerror.GetConfig()
fmt.Printf("Stack traces enabled: %v\n", config.EnableStackTrace)
```

### Registry

#### Error Lookup

```go
// Get error by ID
err, ok := kerror.GetError(id uint32)

// Get error by package and code
err, ok := kerror.GetErrorByPackageCode(pkg string, code int)
```

#### Registry Queries

```go
// List all errors
errors := kerror.ListErrors() []KError

// List all packages
packages := kerror.ListPackages() []string

// List codes for a package
codes := kerror.ListPackageCodes(pkg string) []int

// Validate package/code combination
err := kerror.ValidatePackageCode(pkg string, code int)
```

#### Registry Management

```go
// Clear registry (useful for testing)
kerror.ClearRegistry()
```

### Context Integration

#### Store and Retrieve from Context

```go
// Store error in context
ctx := kerror.ToContext(ctx, inst)

// Retrieve from context
inst, ok := kerror.FromContext(ctx)
```

#### Trace and Span ID Extraction

```go
// Default implementations (return empty strings)
traceID := kerror.ExtractTraceID(ctx)
spanID := kerror.ExtractSpanID(ctx)

// Custom extractors can be set via internal APIs
// When WithContext is called, trace_id and span_id tags are automatically added if present
```

### Metrics

#### `SetMetricsCollector(collector MetricsCollector)`

Sets a custom metrics collector.

```go
type MetricsCollector interface {
    RecordError(pkg string, code int, tags map[string]string)
    RecordNew(pkg string, code int)
    RecordWrap(pkg string, code int)
    GetMetrics() map[string]int64
}
```

```go
// Use built-in simple metrics
collector := kerror.NewSimpleMetrics()
kerror.SetMetricsCollector(collector)

// Get metrics snapshot
metrics := kerror.GetMetrics()
fmt.Printf("Total errors: %d\n", metrics["total"])
```

### Result Type

#### Generic Result for Error Handling

```go
type Result[T any] struct {
    Value T
    Ok    bool
}
```

```go
// Create results
result := kerror.NewResult(value, ok)

// Use in functions
func findUser(id int) kerror.Result[User] {
    user, err := db.GetUser(id)
    if err != nil {
        return kerror.NewResult[User](User{}, false)
    }
    return kerror.NewResult(user, true)
}

// Handle results
result := findUser(123)
if result.Ok {
    user := result.Value
    // Use user
} else {
    // Handle error case
    defaultUser := result.UnwrapOr(User{Name: "Guest"})
}

// Unwrap methods
value := result.Unwrap()         // Panics if !Ok
value := result.UnwrapOr(defaultValue)  // Returns default if !Ok
```

## Advanced Usage

### Error Wrapping with Context

```go
func processUser(ctx context.Context, id int) error {
    user, err := getUser(id)
    if err != nil {
        return ErrInternal.Wrap(err)
            .WithContext(ctx)  // Automatically extracts trace/span IDs
            .WithTag("operation", "get_user")
            .WithDetail("user_id", id)
    }
    return nil
}
```

### Batch Operations

```go
err := ErrInternal.New()
defer err.Release()

// Add multiple tags at once
err.BatchWithTags(
    struct{ Key, Value string }{"env", "prod"},
    struct{ Key, Value string }{"version", "1.0"},
    struct{ Key, Value string }{"service", "api"},
)

// Add multiple details at once
err.BatchWithDetails(
    struct{ Key string; Value any }{"user_id", 123},
    struct{ Key string; Value any }{"request_id", "abc"},
    struct{ Key string; Value any }{"timestamp", time.Now()},
)
```

### Error Comparison

```go
// Check if error is of specific type
if err.Is(ErrNotFound) {
    // Handle not found case
}

// Get underlying cause
if cause := err.Unwrap(); cause != nil {
    // Handle wrapped error
}
```

### JSON Serialization

```go
// Marshal error instance
data, err := json.Marshal(inst)
// Output: {"id":1,"package":"myapp","code":404,"message":"Not found",...}

// Unmarshal error instance
var inst kerror.Instance
err := json.Unmarshal(data, &inst)
```

## Implementation Details

The package uses several techniques for efficiency:

- Object pooling to reuse instances
- String building with pools
- `sync.Map` for concurrent access
- Stack traces captured only when enabled
- Package names are cached

## Best Practices

1. **Define errors at package level**: Define errors as package variables for
   reuse

   ```go
   var (
       ErrNotFound = kerror.Define(kerror.KConfig{Code: 404})
       ErrInternal = kerror.Define(kerror.KConfig{Code: 500})
   )
   ```

2. **Always release instances**: Use `defer inst.Release()` after creating
   instances

   ```go
   inst := err.New()
   defer inst.Release()
   ```

3. **Use meaningful error codes**: Choose codes that make sense for your
   application

4. **Add contextual information**: Use tags for indexable data, details for
   complex data

   ```go
   inst.WithTag("user_id", userID)     // Simple string value
       .WithDetail("request", request)  // Complex object
   ```

5. **Enable metrics in production**: Monitor error rates and patterns

   ```go
   kerror.Configure(kerror.GlobalConfig{
       EnableMetrics: true,
   })
   ```

6. **Use Result type for fallible operations**: Better than returning
   `(value, error)`

   ```go
   func getUser(id int) kerror.Result[User] {
       // Implementation
   }
   ```

7. **Configure appropriately**: Tune pool sizes and limits based on load
   ```go
   kerror.Configure(kerror.GlobalConfig{
       MaxInstancePool: 5000,  // Adjust based on concurrent errors
       MaxTags:         50,     // Adjust based on tag usage
   })
   ```

## Thread Safety

All operations in this package are thread-safe:

- Error definition and registration
- Instance creation and pooling
- Tag and detail operations
- Context operations
- Metrics collection
- Registry queries

## Testing

```bash
# Run tests
go test ./pkg/kernel/kerror

# Run with coverage
go test -cover ./pkg/kernel/kerror

# Run with race detector
go test -race ./pkg/kernel/kerror

# Run benchmarks
go test -bench=. ./pkg/kernel/kerror
```

## License

See LICENSE file in the repository root.

## Contributing

Contributions are welcome! Please see CONTRIBUTING.md for details.

## Support

For issues and questions, please use the GitHub issue tracker.
