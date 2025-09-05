# Unit Test Patterns

## Purpose

Ensure comprehensive testing with 95% coverage minimum for kernel packages.

## When to Use

- Testing all kernel package components
- Validating both safe and unsafe implementations
- Ensuring thread-safety and error handling

## Test Structure

### Given-When-Then Pattern

```go
func TestWidget_Read_Success(t *testing.T) {
    t.Parallel() // Always use parallel for independent tests

    // Given - Setup
    widget := NewWidget(1024)
    data := []byte("test data")
    widget.Write(data)
    output := make([]byte, len(data))

    // When - Action
    n, err := widget.Read(output)

    // Then - Assert
    require.NoError(t, err)
    assert.Equal(t, len(data), n)
    assert.Equal(t, data, output[:n])
}
```

### Table-Driven Tests

```go
func TestWidget_Validation(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        size      int
        wantErr   error
        wantPanic bool
    }{
        {
            name:    "valid_size",
            size:    1024,
            wantErr: nil,
        },
        {
            name:    "zero_size",
            size:    0,
            wantErr: ErrInvalidSize,
        },
        {
            name:    "negative_size",
            size:    -1,
            wantErr: ErrInvalidSize,
        },
        {
            name:    "too_large",
            size:    MaxWidgetSize + 1,
            wantErr: ErrWidgetTooLarge,
        },
    }

    for _, tt := range tests {
        tt := tt // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            if tt.wantPanic {
                assert.Panics(t, func() {
                    NewWidget(tt.size)
                })
                return
            }

            buf, err := NewWidgetWithError(tt.size)
            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
                assert.Nil(t, buf)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, buf)
            }
        })
    }
}
```

## Testing Safe and Unsafe Versions

### Shared Test Logic

```go
// Define test interface
type WidgetTester interface {
    Read([]byte) (int, error)
    Write([]byte) (int, error)
}

// Shared test function
func testWidgetOperations(t *testing.T, newFunc func() WidgetTester) {
    t.Helper()

    widget := newFunc()

    // Test operations
    data := []byte("test")
    n, err := widget.Write(data)
    require.NoError(t, err)
    assert.Equal(t, len(data), n)

    output := make([]byte, len(data))
    n, err = widget.Read(output)
    require.NoError(t, err)
    assert.Equal(t, data, output[:n])
}

// Test safe version
func TestSafeWidget(t *testing.T) {
    t.Parallel()
    testWidgetOperations(t, func() WidgetTester {
        return NewWidget(1024)
    })
}

// Test unsafe version
func TestUnsafeWidget(t *testing.T) {
    t.Parallel()
    testWidgetOperations(t, func() WidgetTester {
        return NewUnsafeWidget(1024)
    })
}
```

### Concurrency Tests

```go
// Test safe version is thread-safe
func TestWidget_Concurrent_Safe(t *testing.T) {
    t.Parallel()

    widget := NewWidget(1024)
    iterations := 1000
    workers := 10

    var wg sync.WaitGroup
    wg.Add(workers)

    errors := make(chan error, workers*iterations)

    for w := 0; w < workers; w++ {
        go func(id int) {
            defer wg.Done()

            data := []byte(fmt.Sprintf("worker-%d", id))
            for i := 0; i < iterations; i++ {
                if _, err := widget.Write(data); err != nil {
                    errors <- err
                }

                output := make([]byte, len(data))
                if _, err := widget.Read(output); err != nil {
                    errors <- err
                }
            }
        }(w)
    }

    wg.Wait()
    close(errors)

    // Check no errors occurred
    for err := range errors {
        t.Errorf("Concurrent access failed: %v", err)
    }
}

// Test unsafe version panics on concurrent access
func TestUnsafeWidget_Concurrent_Panics(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping concurrency panic test in short mode")
    }

    widget := NewUnsafeWidget(1024)

    // Should panic when accessed concurrently
    assert.Panics(t, func() {
        var wg sync.WaitGroup
        wg.Add(2)

        for i := 0; i < 2; i++ {
            go func() {
                defer wg.Done()
                widget.Write([]byte("data"))
            }()
        }

        wg.Wait()
    })
}
```

## Coverage Requirements

### Achieving 95% Coverage

```go
// Test all paths including errors
func TestWidget_EdgeCases(t *testing.T) {
    t.Parallel()

    t.Run("nil_input", func(t *testing.T) {
        widget := NewWidget(10)
        n, err := widget.Write(nil)
        assert.NoError(t, err)
        assert.Equal(t, 0, n)
    })

    t.Run("empty_slice", func(t *testing.T) {
        widget := NewWidget(10)
        n, err := widget.Write([]byte{})
        assert.NoError(t, err)
        assert.Equal(t, 0, n)
    })

    t.Run("widget_overflow", func(t *testing.T) {
        widget := NewWidget(10)
        large := make([]byte, 20)
        _, err := widget.Write(large)
        assert.ErrorIs(t, err, ErrWidgetOverflow)
    })
}
```

### Coverage Exclusions (Must be Justified)

```go
// concurrency_check_prod.go
// +build production

// This file is excluded from coverage as it's a no-op in production
// Tested via development build in concurrency_check_test.go

func (c *concurrencyChecker) check() {
    // No-op in production
}
```

## Test Helpers

```go
// test_helpers.go

// makeTestWidget creates a widget with test data
func makeTestWidget(t *testing.T, size int, pattern byte) *Widget {
    t.Helper()

    buf := NewWidget(size)
    data := bytes.Repeat([]byte{pattern}, size)
    _, err := buf.Write(data)
    require.NoError(t, err)

    return buf
}

// assertWidgetEqual checks widget contents
func assertWidgetEqual(t *testing.T, expected, actual []byte) {
    t.Helper()

    if !bytes.Equal(expected, actual) {
        t.Errorf("Widget mismatch\nExpected: %x\nActual:   %x", expected, actual)
    }
}
```

## Do's and Don'ts

### Do's

- ✅ Use `t.Parallel()` for all independent tests
- ✅ Test both success and error paths
- ✅ Use table-driven tests for multiple cases
- ✅ Test safe and unsafe versions separately
- ✅ Include race condition tests
- ✅ Use test helpers to reduce duplication

### Don'ts

- ❌ Skip error path testing
- ❌ Ignore edge cases
- ❌ Test implementation details
- ❌ Use global state in tests
- ❌ Skip parallel execution without reason

## Related Documents

- [02-benchmarks.md](02-benchmarks.md) - Performance testing
- [04-coverage-requirements.md](04-coverage-requirements.md) - Coverage goals
- [../02-implementation/01-safe-unsafe-pattern.md](../02-implementation/01-safe-unsafe-pattern.md) - Implementation patterns
