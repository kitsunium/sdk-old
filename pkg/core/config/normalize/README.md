# Normalize Package

Package normalize provides configuration key and value normalization utilities.

## Overview

The normalize package offers functions to transform configuration keys and values into a consistent format, ensuring uniformity across different configuration sources.

## API

### Key Normalization

```go
func Key(key string) string
```

Normalizes a configuration key by:
- Converting uppercase letters to lowercase
- Replacing underscores with dots

Examples:
- `DATABASE_URL` → `database.url`
- `Redis_Host` → `redis.host`
- `already.lowercase` → `already.lowercase`

### Value Normalization

```go
func Value(value string) string
```

Normalizes a configuration value by:
- Trimming whitespace (space, tab, newline, carriage return)
- Removing matching quotes (single or double)

Examples:
- `"  trimmed  "` → `"trimmed"`
- `"'quoted'"` → `"quoted"`
- `"\r\n  Windows  \r\n"` → `"Windows"`

### Map Flattening

```go
func Map(input map[string]any) map[string]string
```

Flattens nested map structures into dot-notation keys. Supports:
- Nested maps (recursively flattened)
- Arrays (indexed with dot notation)
- String values (normalized)
- nil values (converted to empty string)
- Other types (converted via fmt.Sprintf)

Examples:
```go
// Input
{"db": {"host": "localhost", "port": 5432}}

// Output
{"db.host": "localhost", "db.port": "5432"}

// Input with arrays
{"servers": ["a", "b"]}

// Output
{"servers.0": "a", "servers.1": "b"}
```

### Utility Functions

```go
func StringToBytes(s string) []byte
func BytesToString(b []byte) string
```

Zero-allocation conversion functions between strings and byte slices.

⚠️ **Warning**: These functions use unsafe operations. The returned values share memory with the input. Do not modify the byte slice after calling `BytesToString` or modify the string data after calling `StringToBytes`.

## Usage Example

```go
package main

import (
    "fmt"
    "github.com/kitsunium/sdk/pkg/core/config/normalize"
)

func main() {
    // Normalize a key
    key := normalize.Key("DATABASE_URL")
    fmt.Println(key) // "database.url"
    
    // Normalize a value
    value := normalize.Value("  'localhost'  ")
    fmt.Println(value) // "localhost"
    
    // Flatten a nested map
    config := map[string]any{
        "database": map[string]any{
            "host": "localhost",
            "port": 5432,
        },
    }
    flat := normalize.Map(config)
    fmt.Println(flat["database.host"]) // "localhost"
    fmt.Println(flat["database.port"]) // "5432"
}
```

## Implementation Details

The package uses lookup tables for character transformations and unsafe operations for string/byte conversions to minimize allocations and maximize efficiency.