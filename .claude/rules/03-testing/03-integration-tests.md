# Integration Testing for Kernel Packages

## Purpose

Design integration tests that validate kernel packages work correctly with other system components, handle real-world scenarios, and maintain performance under load.

## When to Use

- Testing package interactions
- Validating real-world usage patterns
- Load and stress testing
- End-to-end workflow validation
- System integration verification

## Integration Test Patterns

### Cross-Package Integration

```go
// integration_test.go
//go:build integration
// +build integration

package kbuffer_test

import (
    "testing"
    "github.com/org/project/pkg/kernel/kbuffer"
    "github.com/org/project/pkg/kernel/kpool"
    "github.com/org/project/pkg/kernel/kcache"
)

func TestBufferPoolIntegration(t *testing.T) {
    // Test buffer with pool integration
    pool := kpool.New(kpool.WithFactory(func() interface{} {
        return kbuffer.NewBuffer(1024)
    }))

    // Simulate workflow
    for i := 0; i < 1000; i++ {
        buf := pool.Get().(kbuffer.Buffer)
        data := []byte(fmt.Sprintf("test-%d", i))

        n, err := buf.Write(data)
        require.NoError(t, err)
        assert.Equal(t, len(data), n)

        buf.Reset()
        pool.Put(buf)
    }
}

func TestBufferCacheIntegration(t *testing.T) {
    cache := kcache.New(kcache.WithSize(100))

    // Store buffers in cache
    for i := 0; i < 10; i++ {
        buf := kbuffer.NewBuffer(1024)
        buf.Write([]byte(fmt.Sprintf("cached-%d", i)))
        cache.Set(fmt.Sprintf("key-%d", i), buf)
    }

    // Retrieve and verify
    for i := 0; i < 10; i++ {
        val, ok := cache.Get(fmt.Sprintf("key-%d", i))
        require.True(t, ok)

        buf := val.(kbuffer.Buffer)
        assert.Equal(t, len(fmt.Sprintf("cached-%d", i)), buf.Len())
    }
}
```

### Load Testing

```go
func TestBufferUnderLoad(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping load test")
    }

    const (
        workers   = 100
        operations = 10000
        bufferSize = 4096
    )

    buf := kbuffer.NewShardedBuffer(bufferSize * workers)

    var wg sync.WaitGroup
    errors := make(chan error, workers)

    // Start workers
    for w := 0; w < workers; w++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()

            data := make([]byte, 256)
            rand.Read(data)

            for op := 0; op < operations; op++ {
                if _, err := buf.Write(data); err != nil {
                    select {
                    case errors <- err:
                    default:
                    }
                    return
                }
            }
        }(w)
    }

    wg.Wait()
    close(errors)

    // Check for errors
    for err := range errors {
        t.Errorf("worker error: %v", err)
    }

    // Verify data integrity
    assert.Greater(t, buf.Len(), 0)
}
```

### Stress Testing

```go
func TestMemoryStress(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping stress test")
    }

    const duration = 10 * time.Second
    deadline := time.Now().Add(duration)

    var allocated int64
    var operations int64

    // Monitor memory
    go func() {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()

        for range ticker.C {
            var m runtime.MemStats
            runtime.ReadMemStats(&m)

            if m.Alloc > 100<<20 { // 100MB threshold
                t.Logf("WARNING: High memory usage: %d MB", m.Alloc>>20)
            }
        }
    }()

    // Stress test
    for time.Now().Before(deadline) {
        size := rand.Intn(65536) + 1024
        buf := kbuffer.NewBuffer(size)

        data := make([]byte, size/2)
        buf.Write(data)

        atomic.AddInt64(&allocated, int64(size))
        atomic.AddInt64(&operations, 1)

        // Occasionally trigger GC
        if operations%1000 == 0 {
            runtime.GC()
        }
    }

    t.Logf("Completed %d operations, allocated %d MB total",
        operations, allocated>>20)
}
```

### End-to-End Scenarios

```go
func TestFileProcessingScenario(t *testing.T) {
    // Real-world scenario: process large file
    testFile := "testdata/large_file.bin"

    // Create test file
    createTestFile(t, testFile, 10<<20) // 10MB
    defer os.Remove(testFile)

    // Open file
    file, err := os.Open(testFile)
    require.NoError(t, err)
    defer file.Close()

    // Process using buffer
    buf := kbuffer.NewBuffer(4096)
    reader := bufio.NewReader(file)

    var totalBytes int64
    for {
        chunk, err := reader.ReadBytes('\n')
        if err == io.EOF {
            break
        }
        require.NoError(t, err)

        n, err := buf.Write(chunk)
        require.NoError(t, err)
        totalBytes += int64(n)

        // Process buffer when full
        if buf.Len() > 3072 {
            processBuffer(buf)
            buf.Reset()
        }
    }

    // Process remaining
    if buf.Len() > 0 {
        processBuffer(buf)
    }

    assert.Equal(t, int64(10<<20), totalBytes)
}
```

### Network Integration

```go
func TestNetworkIntegration(t *testing.T) {
    // Start test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        buf := kbuffer.NewBuffer(4096)

        // Read request body
        _, err := io.Copy(buf, r.Body)
        require.NoError(t, err)

        // Echo back
        w.Write(buf.Bytes())
    }))
    defer server.Close()

    // Test client
    data := []byte("test data")
    resp, err := http.Post(server.URL, "text/plain", bytes.NewReader(data))
    require.NoError(t, err)
    defer resp.Body.Close()

    received, err := io.ReadAll(resp.Body)
    require.NoError(t, err)
    assert.Equal(t, data, received)
}
```

## Performance Integration Tests

### Throughput Testing

```go
func TestThroughput(t *testing.T) {
    sizes := []int{1024, 4096, 16384, 65536}

    for _, size := range sizes {
        t.Run(fmt.Sprintf("Size_%d", size), func(t *testing.T) {
            buf := kbuffer.NewBuffer(size * 100)
            data := make([]byte, size)

            start := time.Now()
            totalBytes := 0

            for i := 0; i < 1000; i++ {
                n, err := buf.Write(data)
                require.NoError(t, err)
                totalBytes += n

                if buf.Len() > size*90 {
                    buf.Reset()
                }
            }

            elapsed := time.Since(start)
            throughput := float64(totalBytes) / elapsed.Seconds() / (1024 * 1024)

            t.Logf("Throughput: %.2f MB/s", throughput)

            // Verify minimum throughput
            minThroughput := 100.0 // MB/s
            assert.Greater(t, throughput, minThroughput)
        })
    }
}
```

## Integration Test Helpers

### Test Data Generation

```go
func generateTestData(size int) []byte {
    data := make([]byte, size)
    rand.Read(data)
    return data
}

func createTestFile(t *testing.T, path string, size int64) {
    file, err := os.Create(path)
    require.NoError(t, err)
    defer file.Close()

    _, err = io.CopyN(file, rand.Reader, size)
    require.NoError(t, err)
}
```

### Monitoring Utilities

```go
type ResourceMonitor struct {
    t         *testing.T
    startMem  runtime.MemStats
    startTime time.Time
}

func NewResourceMonitor(t *testing.T) *ResourceMonitor {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)

    return &ResourceMonitor{
        t:         t,
        startMem:  m,
        startTime: time.Now(),
    }
}

func (rm *ResourceMonitor) Report() {
    var endMem runtime.MemStats
    runtime.ReadMemStats(&endMem)

    rm.t.Logf("Duration: %v", time.Since(rm.startTime))
    rm.t.Logf("Memory Delta: %d MB", (endMem.Alloc-rm.startMem.Alloc)>>20)
    rm.t.Logf("GC Runs: %d", endMem.NumGC-rm.startMem.NumGC)
}
```

## Do's and Don'ts

### Do's

- ✅ Use build tags for integration tests
- ✅ Test with realistic data sizes
- ✅ Monitor resource usage
- ✅ Test error conditions under load
- ✅ Verify data integrity
- ✅ Clean up test resources

### Don'ts

- ❌ Don't run integration tests in CI by default
- ❌ Don't hardcode timeouts
- ❌ Don't ignore flaky integration tests
- ❌ Don't test with only happy paths

## Related Documents

- [01-unit-tests.md](01-unit-tests.md) - Unit testing
- [02-benchmarks.md](02-benchmarks.md) - Performance testing
- [04-coverage-requirements.md](04-coverage-requirements.md) - Coverage targets
- [../05-commands/02-validation.md](../05-commands/02-validation.md) - Validation commands
