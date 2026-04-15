# Parser Package

Package parser provides configuration parsing utilities for various file formats
and sources.

## Overview

The parser package offers a unified interface for parsing configuration from
multiple sources including JSON, YAML, TOML, INI, XML files, environment
variables, and command-line arguments. All parsers normalize keys and values
using the normalize package and return a flattened map[string]string
representation.

## Supported Formats

### JSON Parser

```go
parser := NewJSON("config.json")
config, err := parser.Load()
```

- Supports .json extension
- Flattens nested objects with dot notation
- Arrays are indexed (e.g., `array.0`, `array.1`)

### YAML Parser

```go
parser := NewYAML("config.yaml")
config, err := parser.Load()
```

- Supports .yaml and .yml extensions
- Handles all YAML data types
- Supports anchors and aliases

### TOML Parser

```go
parser := NewTOML("config.toml")
config, err := parser.Load()
```

- Supports .toml extension
- Handles tables, arrays, and inline tables
- Preserves TOML date/time types as strings

### XML Parser

```go
parser := NewXML("config.xml")
config, err := parser.Load()
```

- Supports .xml extension
- Attributes stored as `element.attribute`
- Repeated elements indexed automatically
- Handles CDATA sections

### INI Parser

```go
parser := NewINI("config.ini")
config, err := parser.Load()
```

- Supports .ini, .cfg, and .conf extensions
- Section keys prefixed with section name
- Supports both `=` and `:` separators
- Comments with `#` or `;` prefixes

### Environment Variables

```go
parser := NewENV("APP_")
config, err := parser.Load()
```

- Optional prefix filtering
- Strips prefix from keys when filtered
- Normalizes keys (uppercase to lowercase, underscore to dot)

### Command-Line Arguments

```go
parser := NewARGS(true) // Skip first arg (program name)
config, err := parser.Load()
```

- Supports `--key=value` and `--key value` formats
- Boolean flags set to "true"
- Single and double dash prefixes

## Common Methods

All parsers implement these methods:

- `Type() string` - Returns the parser type identifier
- `Load() (map[string]string, error)` - Loads and parses from the configured
  source
- `LoadReader(r io.Reader) (map[string]string, error)` - Parses from an
  io.Reader
- `LoadBytes(data []byte) (map[string]string, error)` - Parses from byte slice

## Error Handling

The package uses the errs package for structured error handling. Common errors
include:

- `ErrInvalidExtension` - File has wrong extension
- `ErrFileNotFound` - File doesn't exist
- `ErrReadFailed` - IO error during read
- `ErrJSONParse`, `ErrYAMLParse`, etc. - Format-specific parse errors

## Usage Examples

### Loading a JSON Configuration

```go
package main

import (
    "fmt"
    "log"
    "github.com/kitsunium/sdk/pkg/core/config/parser"
)

func main() {
    p := parser.NewJSON("config.json")
    config, err := p.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Access nested values with dot notation
    dbHost := config["database.host"]
    dbPort := config["database.port"]
    fmt.Printf("Database: %s:%s\n", dbHost, dbPort)
}
```

### Parsing Environment Variables with Prefix

```go
p := parser.NewENV("MYAPP_")
config, err := p.Load()
// MYAPP_DATABASE_URL becomes "database.url" in config
```

### Parsing Command-Line Arguments

```go
p := parser.NewARGS(true) // Skip program name
config, err := p.Load()
// --database-url=localhost becomes {"database.url": "localhost"}
// --verbose becomes {"verbose": "true"}
```

### Loading from Reader

```go
file, err := os.Open("config.yaml")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

p := parser.NewYAML("")
config, err := p.LoadReader(file)
```

## Parser Options

Some parsers support options via the `ParserOption` type:

```go
parser := NewJSON("config.json",
    WithBufferSize(16384),
    WithPool(false),
)
```

Available options:

- `WithBufferSize(size int)` - Set buffer size for reading
- `WithPool(enabled bool)` - Enable/disable buffer pooling

## Key Normalization

All parsers automatically normalize keys:

- Uppercase letters become lowercase
- Underscores become dots
- `DATABASE_URL` → `database.url`

## Value Processing

Values are normalized to:

- Trim surrounding whitespace
- Remove matching quotes
- Convert non-string types to string representation
