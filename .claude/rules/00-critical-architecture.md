# 🚨 CRITICAL ARCHITECTURE RULES - HIGHEST PRIORITY 🚨

## **THESE RULES OVERRIDE ALL OTHER RULES IN CASE OF CONFLICT**

## 1. KERNEL vs CORE Package Distinction

### KERNEL Packages (`pkg/kernel/*`)

**ABSOLUTE RULES FOR KERNEL:**

- **🚫 ZERO METRICS ALLOWED** - No telemetry, no prometheus, no opentelemetry, no stats collection whatsoever
- **🎯 HYPER-PERFORMANCE IS THE ONLY GOAL** - Every nanosecond counts
- **✅ UNSAFE CODE IS AUTHORIZED** - This is the ONLY place where unsafe is acceptable and encouraged when it provides performance benefits
- **⚡ ZERO ALLOCATIONS** - All critical paths must achieve zero heap allocations
- **🔥 PERFORMANCE > SAFETY** - We accept the risk of unsafe code for performance gains

#### Kernel Package Characteristics:

```go
// GOOD - Kernel package
package kfoo

import "unsafe"

// Direct memory manipulation for performance
type Widget struct {
    data unsafe.Pointer
    size uintptr
}

func (w *Widget) Read() []byte {
    // Zero-copy, unsafe operations
    return (*[1 << 30]byte)(w.data)[:w.size:w.size]
}
```

```go
// BAD - NEVER in kernel
package kfoo

import "github.com/prometheus/client_golang/prometheus"

var requestCount = prometheus.NewCounter(...) // 🚫 FORBIDDEN IN KERNEL
```

### CORE Packages (`pkg/core/*`)

**RULES FOR CORE:**

- **✅ METRICS ALLOWED** - Can use telemetry, prometheus, opentelemetry for business insights
- **🏢 BUSINESS LOGIC FOCUS** - Acts as wrapper/orchestrator for kernel functionality
- **🛡️ SAFETY FIRST** - NO unsafe code allowed in core packages
- **📊 OBSERVABILITY** - Should provide metrics, logging, tracing for production monitoring
- **🔄 FLEXIBILITY** - Can sacrifice some performance for maintainability and features

#### Core Package Characteristics:

```go
// GOOD - Core package
package core

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/kitsunium/sdk/pkg/kernel/kfoo"
)

var (
    requestDuration = prometheus.NewHistogramVec(...) // ✅ OK in core
    requestCount    = prometheus.NewCounter(...)      // ✅ OK in core
)

type Service struct {
    widget *kfoo.Widget // Uses kernel for performance
    // ... business logic fields
}

func (s *Service) Process() {
    timer := prometheus.NewTimer(requestDuration)
    defer timer.ObserveDuration()

    // Business logic wrapping kernel functionality
    s.widget.Read()
    requestCount.Inc()
}
```

## 2. MANDATORY Package Documentation

### **EVERY PACKAGE MUST HAVE A COMPLETE README.md**

Each package is an API that WILL be reused. Documentation is NOT optional.

#### Required README Structure:

```markdown
# Package Name

## Purpose

Clear, concise description of what this package does and why it exists.

## API Reference

### Types

- Document ALL exported types
- Include usage examples

### Functions

- Document ALL exported functions
- Include parameters, returns, errors
- Provide code examples

### Interfaces

- Document ALL interfaces
- Explain implementation requirements

## Usage Examples

Complete, runnable examples showing common use cases.

## Performance Characteristics

- Time complexity of operations
- Memory usage patterns
- Benchmark results

## Thread Safety

Explicitly state thread-safety guarantees.

## Error Handling

Document all error conditions and how to handle them.
```

**NO PACKAGE CAN BE CONSIDERED COMPLETE WITHOUT FULL README**

## 3. MANDATORY Development Workflow

### After EVERY File Creation/Modification

**IMMEDIATE MANDATORY ACTION:**

```bash
# FORMAT CODE - NON-NEGOTIABLE
# This command handles EVERYTHING: Go formatting, Bazel updates, imports, etc.
make fmt
```

### Final Package Validation

**BEFORE ANY PACKAGE IS CONSIDERED "DONE":**

```bash
# MANDATORY FINAL VALIDATION
# This single command runs the COMPLETE test suite including:
# - Unit tests
# - Race detection
# - Linting
# - Coverage checks
# - Benchmarks
# - All validations
make test
```

**THE PACKAGE IS NOT COMPLETE UNTIL:**

- ✅ 100% of tests pass
- ✅ 0 linting errors
- ✅ 0 race conditions
- ✅ Coverage >95% (>99% for kernel)
- ✅ All benchmarks meet targets
- ✅ README.md is complete

**IF ANY CHECK FAILS:**

1. Fix ALL issues immediately
2. Re-run the ENTIRE validation sequence
3. Repeat until 100% pass rate

## 4. Anti-Pattern Prevention

### When Refactoring or Improving

**MANDATORY COMPLIANCE CHECK:**

- Before ANY refactoring, re-read ALL applicable rules
- After refactoring, validate against ALL patterns
- If introducing new patterns, document why they don't violate rules

### Common Anti-Patterns to AVOID

#### In Kernel Packages:

```go
// 🚫 NEVER DO THIS IN KERNEL
func Process() {
    start := time.Now()
    defer metrics.RecordDuration(time.Since(start)) // NO METRICS!
}

// 🚫 NEVER DO THIS IN KERNEL
func Read() ([]byte, error) {
    data := make([]byte, 1024) // AVOID ALLOCATIONS!
    return data, nil
}
```

#### In Core Packages:

```go
// 🚫 NEVER DO THIS IN CORE
import "unsafe"

func Process() {
    ptr := unsafe.Pointer(&data) // NO UNSAFE IN CORE!
}
```

## 5. Decision Tree for Package Placement

```
Is this code performance-critical (nanoseconds matter)?
├─ YES → Goes in pkg/kernel/
│   ├─ Use unsafe if needed
│   ├─ Zero allocations required
│   └─ NO metrics allowed
│
└─ NO → Is this business logic or orchestration?
    ├─ YES → Goes in pkg/core/
    │   ├─ Metrics encouraged
    │   ├─ Safety over performance
    │   └─ NO unsafe allowed
    │
    └─ NO → Goes in pkg/lib/ (utilities)
```

## 6. Enforcement

**THESE RULES ARE ENFORCED BY:**

1. Git pre-commit hooks
2. CI/CD pipeline checks
3. Code review requirements
4. Automated rejection of non-compliant code

**NO EXCEPTIONS. NO NEGOTIATIONS.**

## 7. Priority Order

When rules conflict, follow this priority:

1. **THIS DOCUMENT** (00-critical-architecture.md)
2. Architecture rules (01-architecture/\*)
3. Implementation rules (02-implementation/\*)
4. Testing rules (03-testing/\*)
5. Convention rules (04-conventions/\*)
6. Command rules (05-commands/\*)

---

**⚠️ VIOLATING THESE RULES = IMMEDIATE REJECTION ⚠️**

Remember:

- Kernel = Pure performance, no compromises
- Core = Business value with observability
- Every package = Complete API with full documentation
- Quality gates = Non-negotiable, 100% compliance required
