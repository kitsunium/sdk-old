# kerror - Advanced Error Management Package

A high-performance, thread-safe error management system for Go applications with built-in support for HTTP status codes, stack traces, metrics, and context propagation.

## Features

- **Unique Error Constants**: Define package-scoped error constants with automatic ID generation
- **HTTP Status Integration**: Built-in HTTP status code support for web applications
- **Stack Trace Capture**: Optional stack trace capture for debugging
- **Context Propagation**: Pass errors through context with trace/span ID support
- **Metrics Collection**: Pluggable metrics system for error monitoring
- **Zero-Allocation Design**: Object pooling for high-performance scenarios
- **Thread-Safe**: All operations are safe for concurrent use
- **JSON Serialization**: Full JSON marshal/unmarshal support

## Installation

```go
import "github.com/kitsunium/sdk/pkg/kernel/kerror"
```

## Quick Start

### Define Error Constants

```go
package mypackage

import "github.com/kitsunium/sdk/pkg/kernel/kerror"

var (
    ErrNotFound = kerror.Define(kerror.KConfig{
        Code:    404,
        Message: "resource not found",
    })
    
    ErrUnauthorized = kerror.Define(kerror.KConfig{
        Code:    401,
        Message: "unauthorized access",
    })
    
    ErrInternalServer = kerror.Define(kerror.KConfig{
        Code:    500,
        Message: "internal server error",
    })
)
```

### Create Error Instances

```go
// Simple error
err := ErrNotFound.New()

// With formatted message
err := ErrNotFound.Newf("user %s not found", userID)

// Wrap existing error
err := ErrInternalServer.Wrap(dbError)

// Wrap with message
err := ErrInternalServer.Wrapf(dbError, "failed to query user %s", userID)
```

### Add Context and Metadata

```go
err := ErrNotFound.New().
    WithTag("user_id", userID).
    WithTag("resource", "profile").
    WithDetail("query", queryParams).
    WithContext(ctx)
```

## API Reference

### Configuration

#### `GlobalConfig` Structure

```go
type GlobalConfig struct {
    EnableStackTrace  bool   // Capture stack traces (default: false)
    EnableMetrics     bool   // Enable metrics collection (default: false)
    MaxInstancePool   int    // Max pooled instances (default: 1000)
    DefaultPackage    string // Default package name (default: "unknown")
    MaxTags           int    // Max tags per instance (default: 50)
    MaxDetails        int    // Max details per instance (default: 100)
    MaxTagKeyLen      int    // Max tag key length (default: 100)
    MaxTagValueLen    int    // Max tag value length (default: 1000)
    StackTraceDepth   int    // Max stack frames (default: 32)
    EnableValidation  bool   // Enable input validation (default: true)
}
```

#### `Configure(cfg GlobalConfig)`
Set global configuration for the kerror package.

#### `GetConfig() GlobalConfig`
Get current global configuration.

### Error Definition

#### `Define(config KConfig) KError`
Define a new error constant. Automatically detects package name if not provided.

#### `KConfig` Structure
```go
type KConfig struct {
    Package string // Optional package name (auto-detected if empty)
    Code    int    // HTTP status code (also used as exit code)
    Message string // Optional message (auto-filled from HTTP status if empty)
}
```

### KError Methods

#### `New() *Instance`
Create a new error instance.

#### `Newf(format string, args ...any) *Instance`
Create a new error instance with formatted message.

#### `Wrap(cause error) *Instance`
Wrap an existing error.

#### `Wrapf(cause error, format string, args ...any) *Instance`
Wrap an existing error with formatted message.

#### `ID() uint32`
Get the unique error ID.

#### `Package() string`
Get the package name.

#### `Code() int`
Get the error code (HTTP status).

#### `Message() string`
Get the default message.

#### `Is(target error) bool`
Check if error matches target (implements errors.Is).

### Instance Methods

#### Context Methods

- `WithContext(ctx context.Context) *Instance` - Attach context
- `Context() context.Context` - Get attached context

#### Metadata Methods

- `WithTag(key, value string) *Instance` - Add a tag
- `WithTags(tags map[string]string) *Instance` - Add multiple tags
- `Tag(key string) (string, bool)` - Get a tag value
- `Tags() map[string]string` - Get all tags

#### Detail Methods

- `WithDetail(key string, value any) *Instance` - Add a detail
- `WithDetails(details map[string]any) *Instance` - Add multiple details
- `Detail(key string) (any, bool)` - Get a detail value
- `Details() map[string]any` - Get all details

#### Stack Trace Methods

- `CaptureStack(skip int) *Instance` - Manually capture stack trace
- `StackTrace() string` - Get formatted stack trace

#### Error Interface Methods

- `Error() string` - Get error message (implements error interface)
- `Unwrap() error` - Get wrapped error (implements errors.Unwrap)
- `Is(target error) bool` - Check error equality (implements errors.Is)
- `As(target any) bool` - Type assertion (implements errors.As)

#### Utility Methods

- `KError() KError` - Get underlying KError
- `Package() string` - Get package name
- `Code() int` - Get error code
- `Release()` - Return instance to pool for reuse
- `OTelAttributes() map[string]any` - Get OpenTelemetry attributes
- `MarshalJSON() ([]byte, error)` - JSON serialization

### Registry Functions

#### `GetError(id uint32) (*KError, bool)`
Retrieve a registered error by ID.

#### `GetErrorByPackageCode(pkg string, code int) (*KError, bool)`
Retrieve a registered error by package and code.

#### `ListErrors() []KError`
Get all registered errors.

#### `ListPackageCodes(pkg string) []int`
Get all error codes for a package.

#### `ListPackages() []string`
Get all packages with defined errors.

#### `ValidatePackageCode(pkg string, code int) error`
Check if a code is already used in a package.

#### `ClearRegistry()`
Clear the error registry (useful for testing).

### Context Functions

#### `FromContext(ctx context.Context) (*Instance, bool)`
Extract an error instance from context.

#### `ToContext(ctx context.Context, inst *Instance) context.Context`
Add an error instance to context.

#### `ExtractTraceID(ctx context.Context) string`
Extract trace ID from context (integrate with your tracing library).

#### `ExtractSpanID(ctx context.Context) string`
Extract span ID from context (integrate with your tracing library).

### Metrics

#### `SetMetricsCollector(collector MetricsCollector)`
Set a custom metrics collector.

#### `MetricsCollector` Interface
```go
type MetricsCollector interface {
    RecordErrorDefinition(pkg string, code int)
    RecordErrorInstance(pkg string, code int)
    RecordErrorWrapped(pkg string, code int)
}
```

#### `GetMetricsSnapshot() map[string]any`
Get current metrics snapshot (only with SimpleMetrics).

## Advanced Usage

### Enable Stack Traces

```go
kerror.Configure(kerror.GlobalConfig{
    EnableStackTrace: true,
    StackTraceDepth:  20,
})
```

### Custom Metrics Collector

```go
type PrometheusCollector struct {
    // ... prometheus counters
}

func (p *PrometheusCollector) RecordErrorDefinition(pkg string, code int) {
    // Record in Prometheus
}

// Set custom collector
kerror.SetMetricsCollector(&PrometheusCollector{})
```

### Error Propagation Through Context

```go
// In handler
err := ErrNotFound.New().WithTag("resource", "user")
ctx := kerror.ToContext(ctx, err)

// In downstream service
if err, ok := kerror.FromContext(ctx); ok {
    // Handle error from context
}
```

### Performance Optimization

```go
// Reuse instances with Release()
err := ErrNotFound.New()
defer err.Release() // Returns to pool for reuse

// Configure pool size
kerror.Configure(kerror.GlobalConfig{
    MaxInstancePool: 5000,
})
```

## Thread Safety

All operations in the kerror package are thread-safe:
- Error definition and registration
- Instance creation and modification
- Registry queries
- Metrics collection
- Configuration changes

## Performance Considerations

- **Object Pooling**: Instances are pooled to reduce GC pressure
- **Zero-Allocation**: String building uses pools
- **Lock-Free Operations**: Uses sync.Map for concurrent access
- **Lazy Initialization**: Stack traces captured only when needed
- **Cached Lookups**: Package names and HTTP status texts are cached

## Best Practices

1. **Define errors at package level**: Create error constants as package variables
2. **Use appropriate HTTP codes**: Leverage standard HTTP status codes
3. **Add context**: Use tags and details for debugging information
4. **Release instances**: Call Release() when done for better performance
5. **Enable metrics in production**: Monitor error patterns and frequencies
6. **Use structured logging**: JSON serialization provides structured output