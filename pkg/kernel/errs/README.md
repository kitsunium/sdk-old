# errs

Error management package with object pooling and metrics support.

## Usage

```go
import "github.com/kitsunium/sdk/pkg/kernel/errs"
```

## Define Errors

```go
var (
    ErrNotFound = errs.Define(errs.KConfig{
        Code:    404,
        Message: "Resource not found",
    })

    ErrInternal = errs.Define(errs.KConfig{
        Code:    500,
        Message: "Internal server error",
    })
)
```

## Create Error Instances

```go
// Create instance
err := ErrNotFound.New()
defer err.Release() // Return to pool

// With formatted message
err := ErrNotFound.Newf("User %d not found", userID)

// Wrap existing error
err := ErrInternal.Wrap(originalErr)

// Add metadata
err.WithTag("resource", "user")
   .WithDetail("user_id", 12345)
```

## Instance Methods

```go
// Tags (string key-value)
err.WithTag(key, value string)
err.Tag(key string) (string, bool)
err.Tags() map[string]string

// Details (structured data)
err.WithDetail(key string, value any)
err.Detail(key string) (any, bool)
err.Details() map[string]any

// Stack trace
err.StackTrace() []string

// JSON marshaling
data, _ := json.Marshal(err)
```

## Configuration

```go
// Global configuration
errs.Configure(errs.GlobalConfig{
    EnableStack:       true,
    MaxStackDepth:     32,
    EnableValidation:  true,
    MaxInstancePool:   10000,
    DefaultPackage:    "myapp",
})
```

## Registry

```go
// Get error by package and code
err, ok := errs.GetErrorByPackageCode("myapp", 404)

// Get error instance
inst, ok := errs.GetError("myapp:404:abc123")

// List registered packages
packages := errs.ListPackages()
```

## Context Integration

```go
// Store in context
ctx = errs.WithError(ctx, err)

// Retrieve from context
if err := errs.FromContext(ctx); err != nil {
    // Handle error
}
```

## Result Type

```go
func GetUser(id int) errs.Result[*User] {
    user, err := db.FindUser(id)
    if err != nil {
        return errs.Err[*User](ErrNotFound.Wrap(err))
    }
    return errs.Ok(user)
}

result := GetUser(123)
if result.IsErr() {
    return result.Error()
}
user := result.Value()
```

## Metrics

```go
// Set custom metrics collector
errs.SetMetricsCollector(&MyCollector{})

// Record error
errs.RecordError(err.Instance())
```

## Testing

```bash
go test ./pkg/kernel/errs/...
go test -bench=. ./pkg/kernel/errs/...
```
