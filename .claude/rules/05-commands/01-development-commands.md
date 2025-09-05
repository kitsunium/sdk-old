# Development Commands - Local Development Workflow

## Purpose

Provide essential commands for local development, testing, and debugging of kernel packages to streamline the development workflow.

## When to Use

- During active development of kernel packages
- When debugging issues locally
- For quick iteration and testing
- Before committing code changes
- During performance optimization

## Quick Start Commands

### Basic Development Flow

```bash
# 1. Create new package structure
mkdir -p pkg/kernel/knewpackage
cd pkg/kernel/knewpackage

# 2. Generate initial files
touch interface.go constants.go errors.go
touch typename.go typename_test.go
touch BUILD.bazel

# 3. Run tests while developing
go test -v

# 4. Check coverage quickly
go test -cover

# 5. Run with race detection
go test -race

# 6. Format and lint
go fmt ./...
golint ./...
```

## Testing Commands

### Unit Testing

```bash
# Run all tests in current package
go test -v

# Run specific test by name
go test -v -run TestTypeName_Method

# Run tests with short flag (skip slow tests)
go test -v -short

# Run tests with timeout
go test -v -timeout 30s

# Run tests with race detector
go test -v -race

# Run tests with coverage
go test -v -cover

# Run tests in parallel
go test -v -parallel 8

# Run tests with verbose output
go test -v -count=1  # Disable test caching
```

### Test Coverage

```bash
# Generate coverage profile
go test -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# View coverage by function
go tool cover -func=coverage.out

# Coverage for specific package
go test -cover ./pkg/kernel/kbuffer/...

# Coverage with race detection
go test -race -cover

# Set coverage mode (set, count, atomic)
go test -covermode=atomic -coverprofile=coverage.out
```

### Benchmark Testing

```bash
# Run all benchmarks
go test -bench=.

# Run specific benchmark
go test -bench=BenchmarkTypeName_Method

# Run benchmarks with memory profiling
go test -bench=. -benchmem

# Run benchmarks for specific duration
go test -bench=. -benchtime=10s

# Run benchmarks N times for stability
go test -bench=. -count=5

# Compare Safe vs Unsafe implementations
go test -bench=".*_(Safe|Unsafe)$" -benchmem

# Skip tests, run only benchmarks
go test -bench=. -run=^$
```

## Profiling Commands

### CPU Profiling

```bash
# Generate CPU profile during tests
go test -cpuprofile=cpu.prof -bench=.

# Analyze CPU profile interactively
go tool pprof cpu.prof

# Generate CPU profile flamegraph
go tool pprof -http=:8080 cpu.prof

# Top 10 CPU consumers
go tool pprof -top cpu.prof

# List specific function
go tool pprof -list=FunctionName cpu.prof

# Generate profile during specific benchmark
go test -bench=BenchmarkTypeName -cpuprofile=cpu.prof
```

### Memory Profiling

```bash
# Generate memory profile
go test -memprofile=mem.prof -bench=.

# Analyze memory allocations
go tool pprof mem.prof

# View memory profile in browser
go tool pprof -http=:8080 mem.prof

# Check for memory leaks
go test -memprofile=mem.prof -memprofilerate=1

# Analyze heap allocations
go tool pprof -alloc_space mem.prof

# Analyze in-use memory
go tool pprof -inuse_space mem.prof
```

### Trace Analysis

```bash
# Generate execution trace
go test -trace=trace.out -bench=.

# View trace in browser
go tool trace trace.out

# Generate trace for specific test
go test -trace=trace.out -run TestTypeName_Method
```

## Build Commands

### Standard Build

```bash
# Build current package
go build

# Build with specific output
go build -o myapp

# Build for production (optimizations)
go build -ldflags="-s -w"

# Build with race detector
go build -race

# Build with specific tags
go build -tags="unsafe_no_check"

# Build for different OS/Arch
GOOS=linux GOARCH=amd64 go build

# Build with debugging symbols
go build -gcflags="all=-N -l"
```

### Bazel Build

```bash
# Build with Bazel (development mode)
bazel build --config=dev //pkg/kernel/kpackage/...

# Build with Bazel (production mode)
bazel build --config=prod //pkg/kernel/kpackage/...

# Build specific target
bazel build //pkg/kernel/kbuffer:kbuffer

# Build and run tests
bazel test //pkg/kernel/kpackage:all

# Clean build cache
bazel clean

# Build with verbose output
bazel build --config=dev --verbose_failures //...
```

## Code Quality Commands

### Formatting and Linting

```bash
# Format code
go fmt ./...
gofmt -w -s .

# Run golint
golint ./...

# Run go vet
go vet ./...

# Run staticcheck
staticcheck ./...

# Run golangci-lint (comprehensive)
golangci-lint run

# Fix lint issues automatically
golangci-lint run --fix

# Run with specific linters
golangci-lint run --enable=gocyclo,goconst
```

### Code Complexity Analysis

```bash
# Check cyclomatic complexity
gocyclo -over 10 .

# Check cognitive complexity
gocognit -over 20 .

# Generate complexity report
goreporter -p ./...

# Check for inefficient code
ineffassign ./...

# Find repeated strings (candidates for constants)
goconst ./...
```

## Debugging Commands

### Debugging with Delve

```bash
# Start debugger on test
dlv test

# Debug specific test
dlv test -- -test.run TestTypeName_Method

# Start debugger on main
dlv debug

# Attach to running process
dlv attach <pid>

# Set breakpoint and run
dlv test
> break FunctionName
> continue

# Debug with arguments
dlv debug -- arg1 arg2
```

### Print Debugging

```go
// Temporary debug prints (remember to remove!)
fmt.Printf("DEBUG: value=%+v\n", value)

// Use build tags for debug code
// +build debug

package kpackage

func debugPrint(format string, args ...interface{}) {
    fmt.Printf("[DEBUG] "+format+"\n", args...)
}
```

## Dependency Management

### Module Commands

```bash
# Initialize module
go mod init

# Download dependencies
go mod download

# Add missing dependencies
go mod tidy

# Verify dependencies
go mod verify

# Update dependencies
go mod get -u ./...

# Update specific dependency
go mod get -u github.com/some/package

# View dependency graph
go mod graph

# Why is dependency included
go mod why github.com/some/package
```

## Quick Validation Scripts

### Pre-commit Validation

```bash
#!/bin/bash
# pre-commit.sh

echo "Running pre-commit checks..."

# Format check
if ! go fmt -n ./... | grep -q .; then
    echo "❌ Code needs formatting"
    exit 1
fi

# Vet check
if ! go vet ./...; then
    echo "❌ Vet check failed"
    exit 1
fi

# Test check
if ! go test -short ./...; then
    echo "❌ Tests failed"
    exit 1
fi

# Race check
if ! go test -race -short ./...; then
    echo "❌ Race condition detected"
    exit 1
fi

echo "✅ All checks passed!"
```

### Quick Test Loop

```bash
#!/bin/bash
# watch-test.sh

# Run tests on file change
while true; do
    clear
    echo "Running tests..."
    go test -v -short
    echo "Waiting for changes..."
    fswatch -1 *.go
done
```

## Environment Setup

### Development Environment Variables

```bash
# Enable race detector by default
export GORACE="history_size=7"

# Set test timeout
export GO_TEST_TIMEOUT="30s"

# Enable verbose testing
export GOTEST_FLAGS="-v"

# Set benchmark time
export GOBENCH_TIME="10s"

# Development mode
export ENV="development"

# Enable unsafe checks
export UNSAFE_CHECK="true"
```

### Aliases for Common Commands

```bash
# Add to ~/.bashrc or ~/.zshrc

# Testing aliases
alias gt="go test -v"
alias gtc="go test -v -cover"
alias gtr="go test -v -race"
alias gtb="go test -bench=. -benchmem"

# Build aliases
alias gb="go build"
alias gbd="go build -race"
alias gbp="go build -ldflags='-s -w'"

# Quality aliases
alias gf="go fmt ./..."
alias gv="go vet ./..."
alias gl="golint ./..."

# Quick validation
alias gcheck="go fmt ./... && go vet ./... && go test -short ./..."
```

## Do's

✅ **Run tests frequently** during development ✅ **Check race conditions** before committing ✅ **Profile before optimizing** performance ✅ **Use short tests** for quick feedback ✅ **Format code**
before committing ✅ **Check coverage** regularly ✅ **Use build tags** for conditional compilation ✅ **Run benchmarks** when changing algorithms ✅ **Validate with go vet** and linters ✅ **Keep
commands in scripts** for consistency

## Don'ts

❌ **Don't commit without testing** ❌ **Don't ignore race detector** warnings ❌ **Don't optimize without profiling** ❌ **Don't skip formatting** ❌ **Don't ignore lint warnings** ❌ **Don't use
production flags** in development ❌ **Don't test with -short** exclusively ❌ **Don't ignore failed tests** ❌ **Don't profile in debug mode** ❌ **Don't forget to clean** test artifacts

## Related Documents

- [02-validation-commands.md](02-validation-commands.md) - Validation and CI commands
- [03-production-builds.md](03-production-builds.md) - Production build process
- [../03-testing/01-unit-tests.md](../03-testing/01-unit-tests.md) - Testing patterns
- [../03-testing/02-benchmarks.md](../03-testing/02-benchmarks.md) - Benchmark patterns
