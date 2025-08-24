# Storage

Package storage provides core storage abstractions and interfaces for the
Kitsunium SDK.

## Overview

The storage package defines foundational interfaces that can be implemented by
various storage backends including databases, file systems, and in-memory
stores. It serves as the contract between the application layer and concrete
storage implementations.

This package is designed to support multiple storage backends through a unified
interface, enabling applications to switch between different storage solutions
without changing application code.

## Installation

```bash
go get github.com/kitsunium/sdk/pkg/core/storage
```

## Usage

```go
import "github.com/kitsunium/sdk/pkg/core/storage"
```

## Architecture

The storage package provides a pluggable architecture where different storage
backends can be implemented behind common interfaces. This allows for:

- **Backend Agnostic Code**: Write application logic once, run with any storage
  backend
- **Easy Testing**: Mock storage implementations for unit tests
- **Flexible Deployment**: Switch between storage solutions based on environment
  needs
- **Scalability**: Migrate between storage solutions as requirements evolve

## Interfaces

The package defines core interfaces that storage implementations should satisfy:

### Storage Interface

```go
type Storage interface {
    // Core CRUD operations
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)

    // Lifecycle management
    Close() error
    Health(ctx context.Context) error
}
```

### Transactional Storage

For storage backends that support transactions:

```go
type TransactionalStorage interface {
    Storage

    Begin(ctx context.Context) (Transaction, error)
}

type Transaction interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte) error
    Delete(ctx context.Context, key string) error

    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}
```

## Examples

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/kitsunium/sdk/pkg/core/storage"
    "github.com/kitsunium/sdk/pkg/core/storage/memory" // Example implementation
)

func main() {
    // Create a storage instance
    store := memory.New()
    defer store.Close()

    ctx := context.Background()

    // Store data
    err := store.Set(ctx, "user:123", []byte(`{"name": "John", "email": "john@example.com"}`))
    if err != nil {
        log.Fatal(err)
    }

    // Retrieve data
    data, err := store.Get(ctx, "user:123")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("User data: %s\n", data)

    // Check existence
    exists, err := store.Exists(ctx, "user:123")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("User exists: %v\n", exists)

    // Delete data
    err = store.Delete(ctx, "user:123")
    if err != nil {
        log.Fatal(err)
    }
}
```

### With Transactions

```go
func transferData(store storage.TransactionalStorage) error {
    ctx := context.Background()

    tx, err := store.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx) // Safe to call even after commit

    // Perform multiple operations atomically
    if err := tx.Set(ctx, "account:1", []byte("100")); err != nil {
        return err
    }

    if err := tx.Set(ctx, "account:2", []byte("200")); err != nil {
        return err
    }

    // Commit the transaction
    return tx.Commit(ctx)
}
```

## Implementation Guidelines

When implementing a storage backend:

1. **Context Support**: All operations should respect context cancellation and
   timeouts
2. **Error Handling**: Return descriptive errors with context about what failed
3. **Resource Management**: Implement proper cleanup in Close() method
4. **Health Checks**: Provide meaningful health status in Health() method
5. **Concurrency Safety**: Ensure thread-safe operations if supporting
   concurrent access

### Example Implementation Skeleton

```go
type MyStorage struct {
    // Your storage implementation fields
}

func New(config Config) storage.Storage {
    return &MyStorage{
        // Initialize your storage
    }
}

func (s *MyStorage) Get(ctx context.Context, key string) ([]byte, error) {
    // Implementation
}

func (s *MyStorage) Set(ctx context.Context, key string, value []byte) error {
    // Implementation
}

func (s *MyStorage) Delete(ctx context.Context, key string) error {
    // Implementation
}

func (s *MyStorage) Exists(ctx context.Context, key string) (bool, error) {
    // Implementation
}

func (s *MyStorage) Close() error {
    // Cleanup resources
}

func (s *MyStorage) Health(ctx context.Context) error {
    // Check if storage is healthy
}
```

## Available Implementations

- `memory`: In-memory storage for testing and development
- `database`: SQL database storage adapters
- `file`: File system based storage

## Testing

```bash
# Run tests
go test ./pkg/core/storage/...

# Run with race detection
go test -race ./pkg/core/storage/...

# Benchmark tests
go test -bench=. ./pkg/core/storage/...
```

## Contributing

1. Implement the core `Storage` interface
2. Add comprehensive tests
3. Include benchmarks for performance-critical operations
4. Document any specific configuration or limitations
5. Follow the existing code style and patterns

## License

This package is part of the Kitsunium SDK and is subject to the project's
license terms.
