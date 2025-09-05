# Package normalize

## Overview

The `normalize` package provides high-performance utilities for normalizing configuration keys and values in Go applications. It's designed to handle various configuration formats and ensure
consistent key naming and value formatting across different configuration sources.

## Features

- **Key Normalization**: Convert configuration keys to a consistent lowercase dot-notation format
- **Value Normalization**: Clean configuration values by trimming whitespace and quotes
- **Map Flattening**: Transform nested configuration structures into flat key-value pairs
- **Zero-Allocation Design**: Uses lookup tables and unsafe operations for optimal performance
- **Array Support**: Handle arrays in configuration with indexed notation

## Installation

```go
import "github.com/kitsunium/sdk/pkg/core/config/normalize"
```

## API Reference

### Functions

#### `Key(key string) string`

Normalizes configuration keys by:

- Converting uppercase letters to lowercase
- Replacing underscores with dots
- Preserving the original string if no transformation is needed

```go
normalized := normalize.Key("DATABASE_URL")  // Returns: "database.url"
normalized := normalize.Key("Redis_Host")    // Returns: "redis.host"
normalized := normalize.Key("api.key")       // Returns: "api.key" (unchanged)
```

#### `Value(value string) string`

Normalizes configuration values by:

- Trimming leading and trailing whitespace
- Removing matching surrounding quotes (single or double)
- Returning empty string for whitespace-only values

```go
clean := normalize.Value("  'localhost'  ")  // Returns: "localhost"
clean := normalize.Value(`"quoted"`)         // Returns: "quoted"
clean := normalize.Value("  \n\t  ")         // Returns: ""
```

#### `Map(input map[string]any) map[string]string`

Flattens nested configuration maps into a single-level map with dot-notation keys:

```go
input := map[string]any{
    "database": map[string]any{
        "host": "localhost",
        "port": 5432,
        "options": map[string]any{
            "ssl": true,
            "timeout": 30,
        },
    },
    "servers": []any{"server1", "server2"},
}

flat := normalize.Map(input)
// Results in:
// flat["database.host"] = "localhost"
// flat["database.port"] = "5432"
// flat["database.options.ssl"] = "true"
// flat["database.options.timeout"] = "30"
// flat["servers.0"] = "server1"
// flat["servers.1"] = "server2"
```

#### `StringToBytesSafe(s string) []byte`

Safely converts a string to a byte slice with allocation:

```go
bytes := normalize.StringToBytesSafe("hello")
// Safe to modify bytes without affecting the original string
```

#### `BytesToStringSafe(b []byte) string`

Safely converts a byte slice to a string with allocation:

```go
str := normalize.BytesToStringSafe([]byte("hello"))
// Safe to modify the original byte slice without affecting str
```

## Use Cases

### Configuration File Processing

Perfect for processing configuration from various sources (environment variables, YAML, JSON, etc.):

```go
// Environment variables often use UPPER_SNAKE_CASE
envKey := "DATABASE_CONNECTION_POOL_SIZE"
normalizedKey := normalize.Key(envKey)  // "database.connection.pool.size"

// Values might have extra formatting
envValue := " '10' "
normalizedValue := normalize.Value(envValue)  // "10"
```

### Multi-Source Configuration Merging

When merging configurations from different sources with different naming conventions:

```go
// From environment
envConfig := map[string]string{
    "APP_NAME": "MyApp",
    "DB_HOST": "localhost",
}

// From JSON file
jsonConfig := map[string]any{
    "app": map[string]any{
        "name": "MyApp",
        "version": "1.0.0",
    },
}

// Normalize and merge
normalized := make(map[string]string)
for k, v := range envConfig {
    normalized[normalize.Key(k)] = normalize.Value(v)
}

flatJson := normalize.Map(jsonConfig)
for k, v := range flatJson {
    normalized[k] = v
}
```

## Performance Considerations

- **Lookup Tables**: Uses pre-computed 256-byte lookup tables for O(1) character transformations
- **Unsafe Operations**: Employs unsafe pointer arithmetic for zero-allocation string processing
- **Capacity Pre-allocation**: Estimates required capacity for output maps to minimize reallocations
- **String Builder Reuse**: Reuses string builders during recursive flattening operations

## Thread Safety

All functions in this package are thread-safe and can be called concurrently. The lookup tables are initialized once at startup and are read-only thereafter.

## Dependencies

This package has no external dependencies beyond the Go standard library.

## License

Part of the Kitsunium SDK. See the main repository for license information.
