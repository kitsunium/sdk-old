# Errors Package

Native Go error management system for the Kitsunium SDK with HTTP and exit code support.

## Overview

The errors package provides a comprehensive error management system that extends Go's native error interface with:

- **HTTP status codes** for web services
- **Exit codes** for CLI applications
- **Error tagging** for categorization
- **Error details** for additional context
- **Error wrapping** for error chains
- **Thread-safe** operations
- **Error registry** for centralized management

## Features

- Native Go error interface implementation
- HTTP and exit code support
- Error wrapping and unwrapping (Go 1.13+ compatible)
- Tag-based categorization
- Key-value detail storage
- Immutable error creation with builder pattern
- Thread-safe concurrent operations
- Standard HTTP errors pre-defined
- 98.3% test coverage

## Basic Usage

### Creating Errors

```go
import (
    "net/http"
    "github.com/kitsunium/sdk/pkg/kernel/errors"
)

// Create a simple error
err := errors.New(http.StatusNotFound, 1, "user not found")

// Create with formatted message
err := errors.Newf(http.StatusBadRequest, 1, "invalid id: %d", userID)

// Create with tags
err := errors.New(http.StatusInternalServerError, 1, "database error", "db", "critical")

// Access error properties
fmt.Println(err.HTTPCode())  // 404
fmt.Println(err.ExitCode())  // 1
fmt.Println(err.Message())   // "user not found"
fmt.Println(err.Error())     // "user not found"
```

### Using Standard Errors

Pre-defined standard errors are available for common HTTP status codes:

```go
// Use standard errors
return errors.ErrNotFound           // 404 Not Found
return errors.ErrBadRequest         // 400 Bad Request
return errors.ErrUnauthorized       // 401 Unauthorized
return errors.ErrForbidden          // 403 Forbidden
return errors.ErrInternal           // 500 Internal Server Error
return errors.ErrConflict           // 409 Conflict
return errors.ErrUnprocessable      // 422 Unprocessable Entity
return errors.ErrTooManyRequests    // 429 Too Many Requests
return errors.ErrServiceUnavailable // 503 Service Unavailable
```

### Error Wrapping

Wrap existing errors with SDK error information:

```go
// Wrap a standard error
fileErr := os.Open("config.json")
if fileErr != nil {
    return errors.Wrap(fileErr, http.StatusInternalServerError, 1, "failed to load configuration")
}

// Wrap with formatted message
dbErr := db.Connect()
if dbErr != nil {
    return errors.Wrapf(dbErr, http.StatusServiceUnavailable, 1, 
        "failed to connect to %s database", dbName)
}

// Access the underlying error
wrapped := errors.Wrap(originalErr, http.StatusInternalServerError, 1, "wrapper")
fmt.Println(wrapped.Cause())  // originalErr
fmt.Println(wrapped.Unwrap()) // originalErr (Go 1.13+ compatible)
```

### Error Tags

Use tags to categorize errors:

```go
// Create error with tags
err := errors.New(http.StatusInternalServerError, 1, "database error", "db", "critical")

// Add tags (modifies in place)
err.AddTag("retry", "notification")

// Remove tags (modifies in place)
err.RemoveTag("retry")

// Create new error with additional tags (immutable)
newErr := err.WithTag("urgent")

// Check for tags
if err.HasTag("critical") {
    // Send alert
}

// Get all tags
tags := err.Tags() // []string{"db", "critical", "notification"}
```

### Error Details

Add structured context to errors:

```go
// Add single detail (returns new error)
err := errors.New(http.StatusBadRequest, 1, "validation failed").
    WithDetail("field", "email").
    WithDetail("value", "invalid@").
    WithDetail("line", 42)

// Add multiple details at once
details := map[string]interface{}{
    "user_id": 123,
    "action":  "delete",
    "ip":      "192.168.1.1",
}
err = err.WithDetails(details)

// Access details
if val, ok := err.GetDetail("field"); ok {
    fmt.Printf("Invalid field: %v\n", val)
}

// Get all details
allDetails := err.Details()
for key, value := range allDetails {
    fmt.Printf("%s: %v\n", key, value)
}
```

## Advanced Usage

### Error Chains

Build error chains with causes:

```go
// Create base error
baseErr := errors.New(http.StatusInternalServerError, 1, "database connection failed")

// Add cause
errWithCause := baseErr.WithCause(sql.ErrConnDone)

// Error message includes cause
fmt.Println(errWithCause.Error()) 
// Output: "database connection failed: sql: connection is already closed"
```

### Error Comparison

Use Go 1.13+ error comparison:

```go
var ErrUserNotFound = errors.New(http.StatusNotFound, 1, "user not found")

// Later in code
if errors.Is(err, ErrUserNotFound) {
    // Handle user not found
}

// Type assertion
var sdkErr *errors.Error
if errors.As(err, &sdkErr) {
    fmt.Printf("HTTP Code: %d\n", sdkErr.HTTPCode())
}
```

### HTTP Helpers

Convenient methods for HTTP error handling:

```go
err := errors.New(http.StatusBadRequest, 1, "bad request")

// Check error type
if err.IsClientError() {  // true for 4xx codes
    // Client made an error
}

if err.IsServerError() {  // true for 5xx codes
    // Server error occurred
}

// Get HTTP status text
text := errors.HTTPStatusText(http.StatusNotFound) // "Not Found"
```

### Error Registry

Manage errors centrally:

```go
// Errors are automatically registered when created
err1 := errors.New(http.StatusNotFound, 1, "user not found")
err2 := errors.New(http.StatusNotFound, 1, "post not found")

// List all registered errors
allErrors := errors.ListErrors()
for message, err := range allErrors {
    fmt.Printf("%s: HTTP %d, Exit %d\n", 
        message, err.HTTPCode(), err.ExitCode())
}

// Retrieve a registered error
if err, ok := errors.GetError("user not found"); ok {
    // Use the registered error
}

// Clear registry (useful for testing)
errors.ClearRegistry()
```

## Builder Pattern

Create complex errors using method chaining:

```go
err := errors.New(http.StatusBadRequest, 1, "validation failed").
    WithTag("validation", "user-input").
    WithDetail("field", "email").
    WithDetail("reason", "invalid format").
    WithCause(originalErr)

// All methods return a new error instance (immutable)
```

## Thread Safety

The error package is designed for concurrent use:

```go
var sharedErr = errors.New(http.StatusInternalServerError, 1, "shared error")

// Safe concurrent operations
go func() {
    sharedErr.AddTag("goroutine1")
    _ = sharedErr.HasTag("test")
}()

go func() {
    sharedErr.AddTag("goroutine2")
    _ = sharedErr.Details()
}()
```

## HTTP Handler Example

```go
func userHandler(w http.ResponseWriter, r *http.Request) {
    user, err := getUserByID(r.URL.Query().Get("id"))
    if err != nil {
        var sdkErr *errors.Error
        if errors.As(err, &sdkErr) {
            http.Error(w, sdkErr.Error(), sdkErr.HTTPCode())
            return
        }
        // Fallback for non-SDK errors
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    // ... handle success
}
```

## CLI Application Example

```go
func main() {
    if err := run(); err != nil {
        var sdkErr *errors.Error
        if errors.As(err, &sdkErr) {
            fmt.Fprintf(os.Stderr, "Error: %s\n", sdkErr.Error())
            
            // Log details if in debug mode
            if debug {
                for key, val := range sdkErr.Details() {
                    fmt.Fprintf(os.Stderr, "  %s: %v\n", key, val)
                }
            }
            
            os.Exit(sdkErr.ExitCode())
        }
        // Fallback for non-SDK errors
        fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
        os.Exit(1)
    }
}
```

## Best Practices

1. **Use standard errors** when possible for consistency
2. **Add meaningful tags** for error categorization and monitoring
3. **Include relevant details** for debugging but avoid sensitive data
4. **Wrap external errors** to maintain error chain information
5. **Use appropriate HTTP codes** that match the error condition
6. **Set meaningful exit codes** for CLI applications (0=success, 1=general error, 2+=specific errors)
7. **Create immutable errors** with `WithTag()`, `WithDetail()`, `WithCause()` when sharing across goroutines
8. **Use in-place modifications** with `AddTag()`, `RemoveTag()` for local error handling

## Migration from Old System

The new system maintains backward compatibility:

```go
// Old code still works
err := errors.New(404, "not found")  // Uses 404 as both HTTP and exit code
code := err.Code()                   // Returns uint16(404) for compatibility

// New recommended approach
err := errors.New(http.StatusNotFound, 1, "not found")
httpCode := err.HTTPCode()  // 404
exitCode := err.ExitCode()  // 1
```

## Testing

```bash
# Run tests
go test ./pkg/kernel/errors/...

# Check coverage (98.3%)
go test -cover ./pkg/kernel/errors/...

# Run specific tests
go test -run TestErrorWrapping ./pkg/kernel/errors/...
```

## Performance

- Zero allocations for error property access
- Efficient tag storage using map[string]struct{}
- Thread-safe operations with minimal lock contention
- Immutable operations create new instances for safety

## License

Part of the Kitsunium SDK - see main LICENSE file.