# 📚 Go Kernel Package Development Rules - Index

## Overview

This documentation provides comprehensive guidelines for developing high-performance Go kernel packages with a focus on safety, performance, and maintainability.

## Directory Structure

```
.claude/rules/
├── 00-index.md                    # This file - main overview
├── 01-architecture/                # Structural patterns and design
│   ├── 01-interfaces.md           # Interface design and contracts
│   ├── 02-structs.md              # Struct optimization and layout
│   ├── 03-file-organization.md    # File naming and package structure
│   └── 04-design-patterns.md      # Common architectural patterns
├── 02-implementation/              # Implementation details
│   ├── 01-safe-unsafe-pattern.md  # Dual implementation strategy
│   ├── 02-concurrency-detection.md # Runtime concurrency checks
│   ├── 03-memory-optimization.md  # Memory and cache optimization
│   └── 04-error-handling.md       # Error patterns and strategies
├── 03-testing/                     # Testing and validation
│   ├── 01-unit-tests.md           # Unit test patterns
│   ├── 02-benchmarks.md           # Benchmark patterns
│   ├── 03-integration-tests.md    # Integration testing
│   └── 04-coverage-requirements.md # Coverage goals
├── 04-conventions/                 # Code standards
│   ├── 01-naming-conventions.md   # Naming rules
│   ├── 02-documentation.md        # Documentation standards
│   └── 03-code-organization.md    # Code structure rules
└── 05-commands/                    # Development commands
    ├── 01-development.md           # Local dev commands
    ├── 02-validation.md            # Test and validation
    └── 03-production-builds.md     # Production builds
```

## Quick Start Guide

### Creating a New Kernel Package

1. **Define Architecture** → [01-architecture/01-interfaces.md](01-architecture/01-interfaces.md)
2. **Implement Safe Version** → [02-implementation/01-safe-unsafe-pattern.md](02-implementation/01-safe-unsafe-pattern.md)
3. **Add Tests** → [03-testing/01-unit-tests.md](03-testing/01-unit-tests.md)
4. **Add Benchmarks** → [03-testing/02-benchmarks.md](03-testing/02-benchmarks.md)
5. **Optimize if Needed** → [02-implementation/03-memory-optimization.md](02-implementation/03-memory-optimization.md)
6. **Validate** → [05-commands/02-validation.md](05-commands/02-validation.md)

### Decision Trees

#### Should I Create an Unsafe Version?

```
Start → Run Benchmarks → Gain > 30%?
  ├─ Yes → Document limitations → Add concurrency detection → Create unsafe version
  └─ No → Keep safe version only
```

#### What Test Coverage Do I Need?

```
Kernel Package? → Yes → 95% minimum
                └─ No → 80% minimum
```

## Key Principles

### 1. Safety First

- Always implement safe (thread-safe) version first
- Unsafe versions only when performance gain > 30%
- Runtime detection of concurrent access in development

### 2. Performance Targets

- Zero allocations after initialization
- Sub-microsecond operations typical
- CPU cache-friendly data structures

### 3. Testing Requirements

- 95%+ code coverage for kernel packages
- Comprehensive benchmarks (safe vs unsafe)
- Race condition testing mandatory

### 4. Documentation Standards

- Performance requirements in interfaces
- Memory layout documentation in structs
- Complexity guarantees in methods

## Common Commands

### Quick Development

```bash
# Run tests
go test -v -race ./pkg/kernel/kbuffer

# Run benchmarks
go test -bench=. -benchmem ./pkg/kernel/kbuffer

# Check coverage
go test -cover ./pkg/kernel/kbuffer

# Compare safe vs unsafe
go test -bench="(Safe|Unsafe)" -benchmem ./pkg/kernel/kbuffer
```

### Full Validation

```bash
# Complete validation suite
make validate-kernel PKG=kbuffer
```

## Navigation by Task

### "I want to..."

- **Design a new interface** → [01-architecture/01-interfaces.md](01-architecture/01-interfaces.md)
- **Optimize memory usage** → [02-implementation/03-memory-optimization.md](02-implementation/03-memory-optimization.md)
- **Add thread safety** → [02-implementation/01-safe-unsafe-pattern.md](02-implementation/01-safe-unsafe-pattern.md)
- **Write unit tests** → [03-testing/01-unit-tests.md](03-testing/01-unit-tests.md)
- **Add benchmarks** → [03-testing/02-benchmarks.md](03-testing/02-benchmarks.md)
- **Handle errors properly** → [02-implementation/04-error-handling.md](02-implementation/04-error-handling.md)
- **Name files correctly** → [04-conventions/01-naming-conventions.md](04-conventions/01-naming-conventions.md)
- **Document my code** → [04-conventions/02-documentation.md](04-conventions/02-documentation.md)
- **Build for production** → [05-commands/03-production-builds.md](05-commands/03-production-builds.md)

## Common Pitfalls to Avoid

1. **Creating unsafe versions without benchmarking** - Measure first!
2. **Missing race condition tests** - Always use `-race` flag
3. **Poor cache alignment** - Review [02-implementation/03-memory-optimization.md](02-implementation/03-memory-optimization.md)
4. **Inadequate documentation** - See [04-conventions/02-documentation.md](04-conventions/02-documentation.md)
5. **Mixing concerns in files** - One type per file rule
6. **Ignoring coverage requirements** - 95% is mandatory for kernel

## Success Metrics

| Metric               | Target     | Validation Command          |
| -------------------- | ---------- | --------------------------- |
| Test Coverage        | ≥95%       | `go test -cover`            |
| Benchmark Regression | <5%        | `benchstat old.txt new.txt` |
| Race Conditions      | 0          | `go test -race`             |
| Memory Allocations   | 0/op ideal | `go test -bench -benchmem`  |
| Test Duration        | <10s       | `go test -timeout 10s`      |

## Getting Help

- Start with this index for navigation
- Each document has cross-references to related topics
- Follow examples in each guide
- Use validation commands to verify compliance

## Updates and Maintenance

This documentation is maintained as part of the SDK. When adding new patterns or discovering optimizations:

1. Update the relevant rule file
2. Add examples demonstrating the pattern
3. Update cross-references if needed
4. Validate with a real implementation

Last Updated: 2024 Version: 1.0
