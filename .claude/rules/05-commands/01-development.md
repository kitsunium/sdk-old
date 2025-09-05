# Development Commands for Kernel Packages

## Purpose

Provide essential commands for efficient local development, testing, and iteration on kernel packages.

## Quick Development Cycle

### Basic Commands

```bash
# Quick test current package
go test .

# Test with coverage
go test -cover .

# Run specific test
go test -run TestWidget_Write

# Quick benchmark
go test -bench=Widget_Write -benchmem

# Build and check
go build ./...
```

### Watch Mode Development

```bash
# Install watch tool
go install github.com/cosmtrek/air@latest

# Create .air.toml
cat > .air.toml << 'EOF'
root = "."
testdata_dir = "testdata"

[build]
  cmd = "go test -cover ./pkg/kernel/foo"
  bin = ""
  include_ext = ["go", "tpl", "tmpl", "html"]
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  delay = 1000
  stop_on_error = false
EOF

# Run with auto-reload
air
```

### Fast Iteration Script

```bash
#!/bin/bash
# dev.sh - Fast development iteration

PACKAGE=${1:-./pkg/kernel/foo}

clear
echo "Testing $PACKAGE..."
go test -short $PACKAGE

if [ $? -eq 0 ]; then
    echo -e "\n✅ Tests passed, running benchmarks..."
    go test -bench=. -benchmem -benchtime=1s $PACKAGE
else
    echo -e "\n❌ Tests failed"
    exit 1
fi
```

## Code Generation

### Generate Mocks

```bash
# Install mockgen
go install github.com/golang/mock/mockgen@latest

# Generate mocks for interfaces
mockgen -source=interface.go -destination=mocks/mock_widget.go -package=mocks

# With go:generate directive
//go:generate mockgen -source=interface.go -destination=mocks/mock_widget.go
go generate ./...
```

### Generate Benchmarks

```bash
# Script to generate benchmark from tests
#!/bin/bash
# gen_bench.sh

cat > benchmark_generated_test.go << 'EOF'
// Code generated; DO NOT EDIT.
package foo

import "testing"

EOF

# Convert test functions to benchmarks
grep -h "^func Test" *_test.go | while read -r line; do
    name=$(echo "$line" | sed 's/func Test/func Benchmark/' | sed 's/(t \*testing.T)/(b \*testing.B)/')
    echo "$name {
    for i := 0; i < b.N; i++ {
        // Benchmark code here
    }
}" >> benchmark_generated_test.go
done
```

## Debugging

### Debug Build

```bash
# Build with debug symbols
go build -gcflags="all=-N -l" ./pkg/kernel/foo

# With race detector
go build -race ./pkg/kernel/foo

# With escape analysis
go build -gcflags="-m=2" ./pkg/kernel/foo 2>&1 | grep escape
```

### Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug test
dlv test ./pkg/kernel/foo -- -test.run TestWidget_Write

# Debug commands
(dlv) break TestWidget_Write
(dlv) continue
(dlv) print buf
(dlv) step
(dlv) next
```

### Trace Execution

```bash
# Generate trace
go test -trace=trace.out ./pkg/kernel/foo

# View trace
go tool trace trace.out
```

## Profiling

### CPU Profiling

```bash
# Generate CPU profile
go test -cpuprofile=cpu.prof -bench=.

# Analyze
go tool pprof cpu.prof
(pprof) top
(pprof) list Widget.Write
(pprof) web
```

### Memory Profiling

```bash
# Generate memory profile
go test -memprofile=mem.prof -bench=.

# Analyze allocations
go tool pprof -alloc_space mem.prof

# Analyze in-use memory
go tool pprof -inuse_space mem.prof
```

### Block Profiling

```bash
# Enable block profiling in test
import _ "runtime/pprof"

# Generate profile
go test -blockprofile=block.prof -bench=.

# Analyze
go tool pprof block.prof
```

## Local Validation

### Pre-commit Checks

```bash
#!/bin/bash
# pre-commit.sh

set -e

echo "Running pre-commit checks..."

# Format check
if ! go fmt ./... | grep -q .; then
    echo "✅ Format check passed"
else
    echo "❌ Format check failed - run 'go fmt ./...'"
    exit 1
fi

# Vet check
if go vet ./...; then
    echo "✅ Vet check passed"
else
    echo "❌ Vet check failed"
    exit 1
fi

# Test with race detector
if go test -race -short ./...; then
    echo "✅ Race test passed"
else
    echo "❌ Race test failed"
    exit 1
fi

# Coverage check
COVERAGE=$(go test -cover ./... | grep -oP '\d+\.\d+(?=%)')
for cov in $COVERAGE; do
    if (( $(echo "$cov < 95.0" | bc -l) )); then
        echo "❌ Coverage $cov% below 95%"
        exit 1
    fi
done
echo "✅ Coverage check passed"

echo "All checks passed!"
```

## Environment Setup

### Development Environment Variables

```bash
# Enable verbose testing
export GOTEST_VERBOSE=1

# Set test timeout
export GOTEST_TIMEOUT=30s

# Enable race detector by default
export GORACE="history_size=7"

# CPU profiling
export GOGC=100
export GOMAXPROCS=4
```

### Makefile for Development

```makefile
# Makefile
.PHONY: test bench cover clean dev

PACKAGE := ./pkg/kernel/foo

test:
	go test -v $(PACKAGE)

test-race:
	go test -race $(PACKAGE)

bench:
	go test -bench=. -benchmem $(PACKAGE)

cover:
	go test -coverprofile=coverage.out $(PACKAGE)
	go tool cover -html=coverage.out

clean:
	rm -f *.prof *.out *.test

dev:
	@while true; do \
		clear; \
		go test -short $(PACKAGE); \
		echo "Press Ctrl+C to stop, any key to rerun..."; \
		read -n 1; \
	done
```

## IDE Integration

### VS Code Tasks

```json
// .vscode/tasks.json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Test Current Package",
      "type": "shell",
      "command": "go test -v -cover ${fileDirname}",
      "group": {
        "kind": "test",
        "isDefault": true
      }
    },
    {
      "label": "Benchmark Current Package",
      "type": "shell",
      "command": "go test -bench=. -benchmem ${fileDirname}"
    },
    {
      "label": "Check Coverage",
      "type": "shell",
      "command": "go test -coverprofile=/tmp/cover.out ${fileDirname} && go tool cover -html=/tmp/cover.out"
    }
  ]
}
```

### GoLand Run Configurations

```xml
<!-- .idea/runConfigurations/Test_with_Coverage.xml -->
<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="Test with Coverage" type="GoTestRunConfiguration">
    <module name="project" />
    <working_directory value="$PROJECT_DIR$" />
    <go_parameters value="-i" />
    <framework value="gotest" />
    <kind value="PACKAGE" />
    <package value="github.com/org/project/pkg/kernel/foo" />
    <directory value="$PROJECT_DIR$" />
    <filePath value="$PROJECT_DIR$" />
    <coverage enabled="true" />
    <method v="2" />
  </configuration>
</component>
```

## Quick Fixes

### Common Issues

```bash
# Clear test cache
go clean -testcache

# Update dependencies
go mod tidy
go mod download

# Fix imports
goimports -w .

# Update vendor
go mod vendor
```

## Do's and Don'ts

### Do's

- ✅ Use short test flag for quick iteration
- ✅ Profile regularly during development
- ✅ Automate repetitive tasks
- ✅ Use watch mode for TDD
- ✅ Validate before committing

### Don'ts

- ❌ Don't skip tests for speed
- ❌ Don't ignore race detector warnings
- ❌ Don't commit without format check
- ❌ Don't develop without coverage visibility

## Related Documents

- [02-validation.md](02-validation.md) - Full validation suite
- [03-production-builds.md](03-production-builds.md) - Production builds
- [../03-testing/01-unit-tests.md](../03-testing/01-unit-tests.md) - Testing practices
