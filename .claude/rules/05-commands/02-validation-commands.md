# Validation Commands - Quality Assurance and CI/CD

## Purpose

Define validation commands and procedures for ensuring code quality, performance standards, and production readiness of kernel packages.

## When to Use

- Before committing code changes
- In CI/CD pipeline stages
- During code review validation
- For release readiness checks
- When validating Safe vs Unsafe implementations

## Core Validation Suite

### Complete Validation Script

```bash
#!/bin/bash
# validate-all.sh

set -e  # Exit on error

echo "=== Running Complete Validation Suite ==="

# 1. Code formatting
echo "1. Checking code formatting..."
if ! gofmt -l . | grep -q .; then
    echo "✅ Code formatting passed"
else
    echo "❌ Code needs formatting:"
    gofmt -l .
    exit 1
fi

# 2. Linting
echo "2. Running linters..."
golangci-lint run || exit 1
echo "✅ Linting passed"

# 3. Unit tests
echo "3. Running unit tests..."
go test -v -short -parallel 8 ./... || exit 1
echo "✅ Unit tests passed"

# 4. Race detection
echo "4. Checking for race conditions..."
go test -race -short ./... || exit 1
echo "✅ Race detection passed"

# 5. Coverage check
echo "5. Checking test coverage..."
coverage=$(go test -cover ./... | grep -oP '\d+\.\d+(?=%)' | head -1)
if (( $(echo "$coverage < 95" | bc -l) )); then
    echo "❌ Coverage $coverage% is below 95% requirement"
    exit 1
fi
echo "✅ Coverage $coverage% meets requirements"

# 6. Benchmarks
echo "6. Running benchmarks..."
go test -bench=. -benchtime=1s ./... || exit 1
echo "✅ Benchmarks passed"

# 7. Build verification
echo "7. Verifying builds..."
go build ./... || exit 1
echo "✅ Build verification passed"

echo "=== ✅ ALL VALIDATIONS PASSED ==="
```

## Safe vs Unsafe Validation

### Performance Comparison Script

```bash
#!/bin/bash
# validate-safe-unsafe.sh

PACKAGE=${1:-./...}
MIN_IMPROVEMENT=30  # Minimum 30% improvement required

echo "=== Validating Safe vs Unsafe Performance ==="

# Run benchmarks and capture output
echo "Running benchmarks..."
go test -bench=".*_(Safe|Unsafe)$" -benchmem $PACKAGE > bench_results.txt

# Parse results
while IFS= read -r line; do
    if [[ $line == *"_Safe"* ]]; then
        safe_ns=$(echo $line | awk '{print $3}')
        safe_name=$(echo $line | awk '{print $1}')
    elif [[ $line == *"_Unsafe"* ]]; then
        unsafe_ns=$(echo $line | awk '{print $3}')
        unsafe_name=$(echo $line | awk '{print $1}')

        # Calculate improvement
        if [[ -n "$safe_ns" && -n "$unsafe_ns" ]]; then
            improvement=$(echo "scale=2; (($safe_ns - $unsafe_ns) / $safe_ns) * 100" | bc)

            echo "Benchmark: ${safe_name%_Safe*}"
            echo "  Safe:   $safe_ns ns/op"
            echo "  Unsafe: $unsafe_ns ns/op"
            echo "  Improvement: $improvement%"

            # Check if improvement meets requirement
            if (( $(echo "$improvement < $MIN_IMPROVEMENT" | bc -l) )); then
                echo "  ❌ Insufficient improvement (requires >$MIN_IMPROVEMENT%)"
                exit 1
            else
                echo "  ✅ Meets performance requirement"
            fi

            # Reset for next pair
            unset safe_ns safe_name
        fi
    fi
done < bench_results.txt

echo "=== ✅ All unsafe implementations justified ==="
```

### Concurrency Safety Validation

```bash
#!/bin/bash
# validate-concurrency.sh

echo "=== Validating Concurrency Safety ==="

# Test Safe version for race conditions
echo "Testing Safe implementation for races..."
if go test -race -run "TestSafe.*Concurrent" ./...; then
    echo "✅ Safe version is race-free"
else
    echo "❌ Safe version has race conditions"
    exit 1
fi

# Test Unsafe version panics on concurrent access
echo "Testing Unsafe panic behavior..."
if go test -run "TestUnsafe.*Panic" ./...; then
    echo "✅ Unsafe version correctly panics on concurrent access"
else
    echo "❌ Unsafe version doesn't panic on concurrent access"
    exit 1
fi

echo "=== ✅ Concurrency validation passed ==="
```

## Coverage Validation

### Detailed Coverage Report

```bash
#!/bin/bash
# validate-coverage.sh

MIN_COVERAGE=${1:-95}
PACKAGE=${2:-./...}

echo "=== Coverage Validation ==="
echo "Minimum required: $MIN_COVERAGE%"

# Generate coverage profile
go test -coverprofile=coverage.out $PACKAGE

# Get overall coverage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

echo "Overall coverage: $COVERAGE%"

# Check if meets minimum
if (( $(echo "$COVERAGE < $MIN_COVERAGE" | bc -l) )); then
    echo "❌ Coverage below minimum requirement"

    # Show uncovered functions
    echo -e "\n=== Uncovered Functions ==="
    go tool cover -func=coverage.out | grep -E "0\.0%"

    # Generate HTML report
    go tool cover -html=coverage.out -o coverage.html
    echo "Detailed report: coverage.html"

    exit 1
fi

echo "✅ Coverage meets requirements"

# Show coverage by package
echo -e "\n=== Coverage by Package ==="
go test -cover $PACKAGE | grep -E "coverage:"

# Critical paths check (must be 100%)
echo -e "\n=== Critical Path Coverage ==="
CRITICAL_FUNCS=(
    "NewBuffer"
    "Read"
    "Write"
    "Close"
)

for func in "${CRITICAL_FUNCS[@]}"; do
    coverage=$(go tool cover -func=coverage.out | grep "$func" | awk '{print $3}')
    if [[ "$coverage" != "100.0%" ]]; then
        echo "❌ $func: $coverage (requires 100%)"
        exit 1
    else
        echo "✅ $func: $coverage"
    fi
done

echo "=== ✅ Coverage validation passed ==="
```

## Benchmark Validation

### Performance Regression Detection

```bash
#!/bin/bash
# validate-benchmarks.sh

# Save current benchmarks
echo "=== Running Current Benchmarks ==="
go test -bench=. -benchmem -count=5 ./... > current.txt

# Compare with baseline (if exists)
if [ -f baseline.txt ]; then
    echo "=== Comparing with Baseline ==="

    # Install benchstat if needed
    which benchstat || go install golang.org/x/perf/cmd/benchstat@latest

    # Compare
    benchstat baseline.txt current.txt > comparison.txt

    # Check for regression (>10% slower)
    if grep -E "\+[0-9]{2,}\.[0-9]+%" comparison.txt; then
        echo "❌ Performance regression detected:"
        cat comparison.txt
        exit 1
    else
        echo "✅ No performance regression"
        cat comparison.txt
    fi
else
    echo "No baseline found, saving current as baseline"
    cp current.txt baseline.txt
fi

# Check allocation targets
echo -e "\n=== Checking Allocation Targets ==="
while IFS= read -r line; do
    if [[ $line == *"allocs/op"* ]]; then
        allocs=$(echo $line | awk '{print $(NF-1)}')
        name=$(echo $line | awk '{print $1}')

        # Critical paths should have 0 allocations
        if [[ $name == *"_Critical"* ]] && [[ $allocs != "0" ]]; then
            echo "❌ $name has $allocs allocations (should be 0)"
            exit 1
        fi
    fi
done < current.txt

echo "✅ Allocation targets met"
```

## Build Validation

### Multi-Platform Build Check

```bash
#!/bin/bash
# validate-builds.sh

echo "=== Multi-Platform Build Validation ==="

PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${PLATFORMS[@]}"; do
    OS=${platform%/*}
    ARCH=${platform#*/}

    echo "Building for $OS/$ARCH..."
    if GOOS=$OS GOARCH=$ARCH go build ./...; then
        echo "✅ $platform build successful"
    else
        echo "❌ $platform build failed"
        exit 1
    fi
done

echo "=== ✅ All platform builds successful ==="
```

### Bazel Build Validation

```bash
#!/bin/bash
# validate-bazel.sh

echo "=== Bazel Build Validation ==="

# Development build (with safety checks)
echo "1. Development build..."
if bazel build --config=dev //pkg/kernel/...; then
    echo "✅ Dev build passed"
else
    echo "❌ Dev build failed"
    exit 1
fi

# Production build (optimized, no checks)
echo "2. Production build..."
if bazel build --config=prod //pkg/kernel/...; then
    echo "✅ Prod build passed"
else
    echo "❌ Prod build failed"
    exit 1
fi

# Run tests in both modes
echo "3. Development tests..."
if bazel test --config=dev //pkg/kernel/...; then
    echo "✅ Dev tests passed"
else
    echo "❌ Dev tests failed"
    exit 1
fi

echo "4. Production tests..."
if bazel test --config=prod //pkg/kernel/...; then
    echo "✅ Prod tests passed"
else
    echo "❌ Prod tests failed"
    exit 1
fi

echo "=== ✅ Bazel validation complete ==="
```

## CI/CD Integration

### GitHub Actions Workflow

```yaml
# .github/workflows/validate.yml
name: Validate

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  validate:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: [1.19, 1.20]

    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go-version }}

      - name: Format Check
        run: |
          if [ -n "$(gofmt -l .)" ]; then
            echo "Code needs formatting"
            gofmt -d .
            exit 1
          fi

      - name: Lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest

      - name: Unit Tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Coverage Check
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: $coverage%"
          if (( $(echo "$coverage < 95" | bc -l) )); then
            echo "Coverage below 95%"
            exit 1
          fi

      - name: Benchmarks
        run: go test -bench=. -benchmem ./...

      - name: Safe vs Unsafe Validation
        run: ./scripts/validate-safe-unsafe.sh

      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
          fail_ci_if_error: true
```

### Pre-Push Hook

```bash
#!/bin/bash
# .git/hooks/pre-push

echo "Running pre-push validation..."

# Quick validation only
./scripts/validate-quick.sh || {
    echo "❌ Validation failed. Push aborted."
    echo "Run './scripts/validate-all.sh' for detailed report"
    exit 1
}

echo "✅ Pre-push validation passed"
```

## Success Metrics

### Validation Checklist

| Check          | Target        | Command                     |
| -------------- | ------------- | --------------------------- |
| Format         | 100% clean    | `gofmt -l .`                |
| Lint           | No errors     | `golangci-lint run`         |
| Unit Tests     | 100% pass     | `go test ./...`             |
| Race Detection | No races      | `go test -race ./...`       |
| Coverage       | >95%          | `go test -cover ./...`      |
| Benchmarks     | No regression | `benchstat old.txt new.txt` |
| Safe/Unsafe    | >30% gain     | Custom script               |
| Build          | All platforms | `GOOS=x GOARCH=y go build`  |
| Bazel Dev      | Success       | `bazel build --config=dev`  |
| Bazel Prod     | Success       | `bazel build --config=prod` |

## Do's

✅ **Run validation before pushing** code ✅ **Automate validation** in CI/CD ✅ **Check coverage** for critical paths ✅ **Validate performance** improvements ✅ **Test on multiple platforms** ✅
**Use consistent validation** scripts ✅ **Document validation** requirements ✅ **Fail fast** on validation errors ✅ **Keep validation fast** for dev workflow ✅ **Track metrics** over time

## Don'ts

❌ **Don't skip validation** for "small" changes ❌ **Don't ignore failing** tests ❌ **Don't accept low coverage** without justification ❌ **Don't merge without** CI passing ❌ **Don't disable
race** detection ❌ **Don't ignore benchmark** regressions ❌ **Don't hardcode** validation thresholds ❌ **Don't run validation** in debug mode ❌ **Don't ignore platform-specific** issues ❌ **Don't
bypass validation** in emergency fixes

## Related Documents

- [01-development-commands.md](01-development-commands.md) - Development workflow
- [03-production-builds.md](03-production-builds.md) - Production build process
- [../03-testing/04-coverage-requirements.md](../03-testing/04-coverage-requirements.md) - Coverage standards
- [../02-implementation/04-safe-unsafe-patterns.md](../02-implementation/04-safe-unsafe-patterns.md) - Safe/Unsafe validation
