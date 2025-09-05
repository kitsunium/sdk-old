# Validation Commands for Kernel Packages

## Purpose

Provide comprehensive validation commands to ensure kernel packages meet all quality, performance, and safety requirements before release.

## Full Validation Suite

### Master Validation Script

```bash
#!/bin/bash
# validate.sh - Complete validation suite

set -e

PACKAGE=${1:-./pkg/kernel/...}
ERRORS=0

echo "========================================="
echo "    KERNEL PACKAGE VALIDATION SUITE"
echo "========================================="

# 1. Format Check
echo -n "Format Check............"
if gofmt -l . | grep -q .; then
    echo "❌ FAILED"
    gofmt -d .
    ((ERRORS++))
else
    echo "✅ PASSED"
fi

# 2. Vet Check
echo -n "Vet Check..............."
if go vet $PACKAGE 2>&1 | grep -q .; then
    echo "❌ FAILED"
    go vet $PACKAGE
    ((ERRORS++))
else
    echo "✅ PASSED"
fi

# 3. Staticcheck
echo -n "Static Analysis........."
if staticcheck $PACKAGE 2>&1 | grep -q .; then
    echo "❌ FAILED"
    staticcheck $PACKAGE
    ((ERRORS++))
else
    echo "✅ PASSED"
fi

# 4. Unit Tests
echo -n "Unit Tests.............."
if ! go test -short $PACKAGE > /dev/null 2>&1; then
    echo "❌ FAILED"
    go test -v $PACKAGE
    ((ERRORS++))
else
    echo "✅ PASSED"
fi

# 5. Race Detection
echo -n "Race Detection.........."
if ! go test -race -short $PACKAGE > /dev/null 2>&1; then
    echo "❌ FAILED"
    go test -race $PACKAGE
    ((ERRORS++))
else
    echo "✅ PASSED"
fi

# 6. Coverage Check
echo -n "Coverage (≥95%)........."
COVERAGE=$(go test -cover $PACKAGE 2>/dev/null | grep -oP '\d+\.\d+(?=%)' | head -1)
if (( $(echo "$COVERAGE < 95.0" | bc -l) )); then
    echo "❌ FAILED ($COVERAGE%)"
    ((ERRORS++))
else
    echo "✅ PASSED ($COVERAGE%)"
fi

# 7. Benchmarks
echo -n "Benchmarks.............."
if ! go test -bench=. -benchtime=1s -run=^$ $PACKAGE > /dev/null 2>&1; then
    echo "❌ FAILED"
    ((ERRORS++))
else
    echo "✅ PASSED"
fi

# 8. Build Check
echo -n "Build Check............."
if ! go build $PACKAGE 2>&1 | grep -q .; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
    go build -v $PACKAGE
    ((ERRORS++))
fi

echo "========================================="
if [ $ERRORS -eq 0 ]; then
    echo "✅ ALL CHECKS PASSED!"
    exit 0
else
    echo "❌ $ERRORS CHECKS FAILED"
    exit 1
fi
```

## Static Analysis

### Install Analysis Tools

```bash
# Install all tools
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

### GolangCI-Lint Configuration

```yaml
# .golangci.yml
linters:
  enable:
    - govet
    - errcheck
    - staticcheck
    - gosimple
    - structcheck
    - varcheck
    - ineffassign
    - deadcode
    - typecheck
    - gosec
    - megacheck
    - misspell
    - unparam
    - prealloc
    - scopelint
    - gocritic
    - gochecknoinits
    - gochecknoglobals

linters-settings:
  govet:
    check-shadowing: true
  gosec:
    excludes:
      - G404 # Allow weak random for non-crypto use
  gocritic:
    enabled-checks:
      - rangeValCopy
      - unnamedResult
      - paramTypeCombine

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gosec
        - gochecknoglobals
```

### Run Analysis

```bash
# Run all linters
golangci-lint run ./pkg/kernel/...

# Run specific linter
golangci-lint run --disable-all -E staticcheck ./pkg/kernel/...

# With auto-fix
golangci-lint run --fix ./pkg/kernel/...
```

## Security Validation

### Security Scan

```bash
# Run gosec
gosec -fmt json -out gosec-report.json ./...

# Check for vulnerabilities
go list -json -m all | nancy sleuth

# Audit dependencies
go list -m all | go-mod-outdated
```

### Vulnerability Check Script

```bash
#!/bin/bash
# security_check.sh

echo "Security Validation"
echo "==================="

# Check for known vulnerabilities
echo -n "Vulnerability scan......"
if govulncheck ./... 2>&1 | grep -q "No vulnerabilities found"; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
    govulncheck ./...
    exit 1
fi

# Check for secrets
echo -n "Secret scan............."
if gitleaks detect --no-git 2>&1 | grep -q "no leaks found"; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
    gitleaks detect --verbose
    exit 1
fi
```

## Performance Validation

### Performance Requirements Check

```bash
#!/bin/bash
# perf_validate.sh

PACKAGE=$1

# Run benchmarks and check requirements
echo "Performance Validation"
echo "====================="

# Check unsafe vs safe performance
go test -bench=SafeVsUnsafe -benchmem $PACKAGE > bench.out

# Extract performance difference
SAFE=$(grep "BenchmarkSafe" bench.out | awk '{print $3}')
UNSAFE=$(grep "BenchmarkUnsafe" bench.out | awk '{print $3}')

if [ -n "$UNSAFE" ] && [ -n "$SAFE" ]; then
    IMPROVEMENT=$(echo "scale=2; ($SAFE - $UNSAFE) / $SAFE * 100" | bc)

    if (( $(echo "$IMPROVEMENT < 30" | bc -l) )); then
        echo "❌ Unsafe implementation improvement ${IMPROVEMENT}% < 30% requirement"
        exit 1
    else
        echo "✅ Unsafe implementation ${IMPROVEMENT}% faster"
    fi
fi

# Check zero allocations
if grep -q "0 allocs/op" bench.out; then
    echo "✅ Zero allocation targets met"
else
    echo "⚠️  Some operations allocate memory"
fi
```

### Memory Leak Detection

```bash
# Run with memory profiling
go test -memprofile=mem.prof -bench=. -benchtime=10s

# Check for leaks
go tool pprof -inuse_space mem.prof << EOF
top
list Buffer.Write
EOF

# Automated leak check
#!/bin/bash
BEFORE=$(go test -bench=. -benchtime=1s -benchmem | grep "allocs/op" | awk '{sum+=$5} END {print sum}')
sleep 5
AFTER=$(go test -bench=. -benchtime=1s -benchmem | grep "allocs/op" | awk '{sum+=$5} END {print sum}')

if [ "$AFTER" -gt "$BEFORE" ]; then
    echo "⚠️  Potential memory leak detected"
fi
```

## Compliance Validation

### License Check

```bash
# Check license headers
#!/bin/bash
# check_license.sh

LICENSE_HEADER="// Copyright (c) 2024"

for file in $(find . -name "*.go" -not -path "./vendor/*"); do
    if ! head -1 "$file" | grep -q "$LICENSE_HEADER"; then
        echo "Missing license header: $file"
        exit 1
    fi
done
```

### Documentation Check

```bash
# Check for missing documentation
golint ./... | grep -E "exported .* should have comment"

# Ensure README exists
if [ ! -f "pkg/kernel/kbuffer/README.md" ]; then
    echo "Missing README.md"
    exit 1
fi
```

## Integration Validation

### Cross-Platform Build

```bash
#!/bin/bash
# cross_platform.sh

PACKAGE=$1
PLATFORMS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64"

for platform in $PLATFORMS; do
    IFS='/' read -r -a array <<< "$platform"
    GOOS="${array[0]}"
    GOARCH="${array[1]}"

    echo -n "Building for $GOOS/$GOARCH..."
    if GOOS=$GOOS GOARCH=$GOARCH go build $PACKAGE > /dev/null 2>&1; then
        echo "✅"
    else
        echo "❌"
        exit 1
    fi
done
```

## CI/CD Validation Pipeline

### GitHub Actions Workflow

```yaml
# .github/workflows/validate.yml
name: Validation
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install tools
        run: |
          go install honnef.co/go/tools/cmd/staticcheck@latest
          go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

      - name: Format check
        run: test -z $(gofmt -l .)

      - name: Vet
        run: go vet ./...

      - name: Static analysis
        run: staticcheck ./...

      - name: Lint
        run: golangci-lint run

      - name: Test with race
        run: go test -race -coverprofile=coverage.out ./...

      - name: Coverage check
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$COVERAGE < 95.0" | bc -l) )); then
            echo "Coverage $COVERAGE% below 95%"
            exit 1
          fi

      - name: Benchmarks
        run: go test -bench=. -benchmem ./...
```

## Do's and Don'ts

### Do's

- ✅ Run full validation before commits
- ✅ Automate validation in CI/CD
- ✅ Check multiple Go versions
- ✅ Validate on all target platforms
- ✅ Monitor performance regressions
- ✅ Validate documentation completeness

### Don'ts

- ❌ Don't skip validation steps
- ❌ Don't ignore warnings
- ❌ Don't disable linters without reason
- ❌ Don't merge failing validations
- ❌ Don't bypass security checks

## Related Documents

- [01-development.md](01-development.md) - Development commands
- [03-production-builds.md](03-production-builds.md) - Production builds
- [../03-testing/04-coverage-requirements.md](../03-testing/04-coverage-requirements.md) - Coverage requirements
