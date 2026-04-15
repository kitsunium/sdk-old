# Kitsunium SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/kitsunium/sdk.svg)](https://pkg.go.dev/github.com/kitsunium/sdk)
[![codecov](https://codecov.io/gh/kitsunium/sdk/branch/main/graph/badge.svg)](https://codecov.io/gh/kitsunium/sdk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CI](https://github.com/kitsunium/sdk/workflows/CI/badge.svg)](https://github.com/kitsunium/sdk/actions)

A high-performance Go SDK providing essential building blocks for modern
applications with a focus on efficiency, reliability, and developer experience.

## 🚀 Features

### Core Packages

#### 🔧 **Kernel Components**

- **pool**: Buffer pool management with zero-allocation operations
- **cache**: Advanced caching with LRU, sharded, and atomic cache
  implementations
- **errs**: Comprehensive error handling with stack traces and error registry
- **files**: File system utilities with optimized file operations

#### 📦 **Core Utilities**

- **Config Management**: Multi-format configuration parser (YAML, JSON, TOML,
  XML, INI)
- **Config Normalization**: Automatic configuration key normalization
- **Performance Optimized**: Extensive benchmarking and performance tuning

## 📊 Performance

The SDK is extensively benchmarked for optimal performance. Run benchmarks with:

```bash
make bench                    # Benchmark current commit
make bench <commit>          # Benchmark specific commit
make bench/compare           # Compare with main branch
make bench/compare <c1> <c2>  # Compare two commits
```

## 🛠️ Installation

```bash
go get github.com/kitsunium/sdk
```

## 📖 Usage

### Buffer Pool Example

```go
import "github.com/kitsunium/sdk/pkg/kernel/pool"

// Create a buffer pool
pool := pool.NewPool(1024)

// Get a buffer
buf := pool.Get()
defer pool.Put(buf)

// Use the buffer
buf.WriteString("Hello, World!")
data := buf.Bytes()
```

### LRU Cache Example

```go
import "github.com/kitsunium/sdk/pkg/kernel/cache"

// Create an LRU cache with 1000 entries
cache := cache.NewLRU(1000)

// Set a value
cache.Set("key", "value")

// Get a value
if val, ok := cache.Get("key"); ok {
    fmt.Println(val)
}
```

### Error Handling Example

```go
import "github.com/kitsunium/sdk/pkg/kernel/errs"

// Define custom errors
var (
    ErrNotFound = errs.New(404, "NOT_FOUND", "Resource not found")
    ErrInternal = errs.New(500, "INTERNAL", "Internal server error")
)

// Use errors with context
err := ErrNotFound.WithDetail("user_id", userID)
if errs.Is(err, ErrNotFound) {
    // Handle not found error
}
```

### Configuration Parser Example

```go
import "github.com/kitsunium/sdk/pkg/core/config/parser"

// Parse YAML configuration
config, err := parser.YAML.LoadFile("config.yaml")

// Parse JSON configuration
data, err := parser.JSON.LoadBytes(jsonBytes)

// Parse from environment variables
envConfig, err := parser.ENV.Load("APP_")
```

## 🏗️ Development

### Prerequisites

- Go 1.21+
- Bazel (optional, for build system)
- Make

### Building

```bash
make build              # Build all packages
make test              # Run tests
make test/coverage     # Run tests with coverage
make quality/validate  # Run all quality checks
```

### Code Quality

```bash
make quality/lint      # Run linters
make quality/format    # Format code
make quality/security  # Run security analysis
make quality/fix       # Auto-fix issues
```

### Git Hooks

Install Git hooks for automatic code quality checks:

```bash
make hooks/install
```

## 📦 Package Structure

```text
sdk/
├── pkg/
│   ├── kernel/          # Core kernel packages
│   │   ├── pool/        # Buffer pool management
│   │   ├── cache/       # Caching implementations
│   │   ├── errs/        # Error handling
│   │   ├── kfs/         # File system utilities
│   │   └── config/      # Configuration management
│   └── core/            # Core utilities
│       └── config/      # Configuration parsers
│           ├── parser/  # Multi-format parsers
│           └── normalize/ # Key normalization
└── scripts/             # Development scripts
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📈 Benchmarks

The SDK includes comprehensive benchmarks for all performance-critical
components:

- **Buffer Operations**: Optimized for minimal allocations
- **Cache Operations**: Sub-microsecond access times
- **Error Handling**: Zero-allocation error creation
- **Config Parsing**: Fast multi-format parsing

View the latest benchmark results:

```bash
make bench/list        # List saved benchmarks
make bench/compare     # Compare performance
```

## 🛡️ Development Setup

### Git Hooks

This project uses Git hooks to maintain code quality and enforce standards.
Install them with:

```bash
# Install git hooks (required for contributors)
bash .githooks/install.sh
```

The hooks will:

- **pre-commit**: Format code automatically
- **commit-msg**: Validate conventional commit format
- **pre-push**: Block AI-related content

To temporarily bypass hooks (not recommended):

```bash
git commit --no-verify  # Skip pre-commit
ALLOW_AI_MENTIONS=1 git push  # Skip AI content check
```

## 📄 License

This project is licensed under the Apache License 2.0 - see the
[LICENSE](LICENSE) file for details.

## 🔗 Links

- [Documentation](https://pkg.go.dev/github.com/kitsunium/sdk)
- [GitHub Repository](https://github.com/kitsunium/sdk)
- [Issue Tracker](https://github.com/kitsunium/sdk/issues)

## ⭐ Support

If you find this project useful, please consider giving it a star on GitHub!

---

Made with ❤️ by the Kitsunium Team
