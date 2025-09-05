# Production Builds - Optimized Build Process

## Purpose

Define the production build process for kernel packages, including optimization flags, build tags, and deployment configurations for maximum performance.

## When to Use

- When preparing releases for production
- For performance-critical deployments
- When building optimized binaries
- For creating distributable packages
- During CI/CD production pipeline

## Build Configurations

### Development vs Production Modes

```bash
# Development Mode (with safety checks)
# - Race detection enabled
# - Panic on unsafe concurrent access
# - Debug symbols included
# - Assertions enabled
go build -tags="debug,safe_checks" ./...

# Production Mode (maximum performance)
# - No race detection
# - No panic checks (unsafe_no_check)
# - Stripped binaries
# - Optimizations enabled
go build -tags="unsafe_no_check" -ldflags="-s -w" ./...
```

### Build Tags Overview

| Tag               | Purpose                       | Usage                   |
| ----------------- | ----------------------------- | ----------------------- |
| `debug`           | Enable debug logging          | Development only        |
| `safe_checks`     | Enable safety validations     | Development only        |
| `unsafe_no_check` | Disable unsafe panic checks   | Production only         |
| `no_stats`        | Disable statistics collection | Production optimization |
| `static`          | Static linking                | Deployment              |

## Go Build Optimizations

### Compiler Optimization Flags

```bash
# Maximum optimization
go build \
  -ldflags="-s -w" \           # Strip symbols
  -trimpath \                  # Remove file paths
  -tags="unsafe_no_check" \    # Production tags
  -o app

# Advanced optimizations
go build \
  -gcflags="-m=2" \            # Inline analysis
  -gcflags="-l=4" \            # Aggressive inlining
  -gcflags="-B" \              # Disable bounds checking
  -ldflags="-s -w -X main.version=$VERSION" \
  -o app

# Static binary (no CGO)
CGO_ENABLED=0 go build \
  -ldflags="-s -w -extldflags '-static'" \
  -tags="netgo,osusergo,static_build" \
  -o app
```

### Size Optimization

```bash
# Minimize binary size
go build -ldflags="-s -w" -o app
upx --best --lzma app  # Further compression with UPX

# Check binary size
size app
strip -s app  # Additional stripping
```

## Bazel Production Builds

### Bazel Configuration (.bazelrc)

```python
# .bazelrc

# Development configuration
build:dev --compilation_mode=dbg
build:dev --strip=never
build:dev --copt=-O0
build:dev --define=gotags=debug,safe_checks

# Production configuration
build:prod --compilation_mode=opt
build:prod --strip=always
build:prod --copt=-O3
build:prod --copt=-march=native
build:prod --define=gotags=unsafe_no_check,no_stats

# Benchmark configuration
build:benchmark --compilation_mode=opt
build:benchmark --copt=-O3
build:benchmark --copt=-fno-omit-frame-pointer
build:benchmark --define=gotags=unsafe_no_check,bench

# Release configuration
build:release --config=prod
build:release --stamp
build:release --workspace_status_command=./scripts/workspace_status.sh
```

### BUILD.bazel File

```python
# BUILD.bazel

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_binary", "go_test")

go_library(
    name = "kbuffer",
    srcs = glob(["*.go"], exclude = ["*_test.go"]),
    importpath = "pkg/kernel/kbuffer",
    visibility = ["//visibility:public"],
    deps = [
        "@org_golang_x_sys//unix:go_default_library",
    ],
)

go_binary(
    name = "kbuffer_prod",
    embed = [":kbuffer"],
    goarch = "amd64",
    goos = "linux",
    pure = "on",
    static = "on",
    tags = ["unsafe_no_check"],
    visibility = ["//visibility:public"],
)

go_test(
    name = "kbuffer_test",
    srcs = glob(["*_test.go"]),
    embed = [":kbuffer"],
    race = "off",  # Disabled in production
    tags = ["unsafe_no_check"],
)
```

### Bazel Build Commands

```bash
# Production build
bazel build --config=prod //pkg/kernel/kbuffer:kbuffer_prod

# Release build with version stamping
bazel build --config=release \
  --define=VERSION=$(git describe --tags) \
  //pkg/kernel/kbuffer:kbuffer_prod

# Build all production targets
bazel build --config=prod //pkg/kernel/...

# Query production targets
bazel query 'attr(tags, "unsafe_no_check", //...)'

# Clean and rebuild
bazel clean --expunge
bazel build --config=prod //...
```

## Production Build Scripts

### Complete Build Script

```bash
#!/bin/bash
# build-prod.sh

set -e

VERSION=${1:-$(git describe --tags --always --dirty)}
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse HEAD)
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

echo "=== Production Build ==="
echo "Version: $VERSION"
echo "Commit: $GIT_COMMIT"
echo "Branch: $GIT_BRANCH"
echo "Date: $BUILD_DATE"

# Set build flags
LDFLAGS="-s -w \
  -X main.Version=$VERSION \
  -X main.BuildDate=$BUILD_DATE \
  -X main.GitCommit=$GIT_COMMIT \
  -X main.GitBranch=$GIT_BRANCH"

# Build for multiple platforms
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

for platform in "${PLATFORMS[@]}"; do
    OS=${platform%/*}
    ARCH=${platform#*/}
    OUTPUT="dist/app-${OS}-${ARCH}"

    echo "Building for $OS/$ARCH..."

    CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH go build \
        -trimpath \
        -tags="unsafe_no_check,no_stats,static_build" \
        -ldflags="$LDFLAGS" \
        -o "$OUTPUT" \
        ./cmd/app

    # Compress binary
    if command -v upx &> /dev/null; then
        upx --best --lzma "$OUTPUT"
    fi

    # Generate checksum
    sha256sum "$OUTPUT" > "$OUTPUT.sha256"

    echo "✅ Built: $OUTPUT"
done

echo "=== Build Complete ==="
ls -lh dist/
```

### Docker Production Build

```dockerfile
# Dockerfile.prod

# Build stage
FROM golang:1.20-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /build
COPY . .

# Build with production flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -trimpath \
    -tags="unsafe_no_check,no_stats,static_build" \
    -ldflags="-s -w -extldflags '-static'" \
    -o app \
    ./cmd/app

# Compress binary
RUN apk add --no-cache upx && \
    upx --best --lzma app

# Production stage
FROM scratch

COPY --from=builder /build/app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/app"]
```

### Container Build Script

```bash
#!/bin/bash
# build-container.sh

VERSION=${1:-latest}
REGISTRY=${2:-ghcr.io/org}
IMAGE_NAME="kernel-app"

echo "Building container: $REGISTRY/$IMAGE_NAME:$VERSION"

# Build multi-arch images
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    --build-arg VERSION=$VERSION \
    --tag $REGISTRY/$IMAGE_NAME:$VERSION \
    --tag $REGISTRY/$IMAGE_NAME:latest \
    --file Dockerfile.prod \
    --push \
    .

echo "✅ Container published: $REGISTRY/$IMAGE_NAME:$VERSION"
```

## Performance Validation

### Production Benchmark Script

```bash
#!/bin/bash
# benchmark-prod.sh

echo "=== Production Build Benchmarks ==="

# Build production binary
echo "Building production binary..."
go build \
    -tags="unsafe_no_check,no_stats" \
    -ldflags="-s -w" \
    -o app.prod \
    ./cmd/app

# Build development binary for comparison
echo "Building development binary..."
go build \
    -tags="debug,safe_checks" \
    -o app.dev \
    ./cmd/app

# Run benchmarks
echo "Running production benchmarks..."
./app.prod -bench > bench.prod

echo "Running development benchmarks..."
./app.dev -bench > bench.dev

# Compare results
echo "=== Performance Comparison ==="
paste bench.prod bench.dev | column -t

# Verify production is faster
prod_time=$(grep "total time" bench.prod | awk '{print $3}')
dev_time=$(grep "total time" bench.dev | awk '{print $3}')

improvement=$(echo "scale=2; (($dev_time - $prod_time) / $dev_time) * 100" | bc)
echo "Production improvement: $improvement%"

if (( $(echo "$improvement < 30" | bc -l) )); then
    echo "⚠️  Production build improvement less than 30%"
fi
```

## Release Process

### Release Build Script

```bash
#!/bin/bash
# release.sh

VERSION=$1
if [ -z "$VERSION" ]; then
    echo "Usage: ./release.sh <version>"
    exit 1
fi

echo "=== Release Build v$VERSION ==="

# Validation
echo "1. Running validation suite..."
./scripts/validate-all.sh || exit 1

# Tagging
echo "2. Creating git tag..."
git tag -a "v$VERSION" -m "Release v$VERSION"

# Build
echo "3. Building release binaries..."
./scripts/build-prod.sh $VERSION

# Generate release notes
echo "4. Generating release notes..."
git log --oneline $(git describe --tags --abbrev=0 HEAD^)..HEAD > dist/RELEASE_NOTES.md

# Create tarball
echo "5. Creating release archive..."
cd dist
tar -czf "release-v$VERSION.tar.gz" app-* *.sha256 RELEASE_NOTES.md
cd ..

echo "=== Release v$VERSION Complete ==="
echo "Files ready in dist/"
echo "Don't forget to:"
echo "  - Push tag: git push origin v$VERSION"
echo "  - Upload release to GitHub/GitLab"
echo "  - Update documentation"
```

## Deployment Configuration

### Systemd Service

```ini
# /etc/systemd/system/kernel-app.service
[Unit]
Description=Kernel Application
After=network.target

[Service]
Type=simple
User=app
Group=app
WorkingDirectory=/opt/app
ExecStart=/opt/app/kernel-app
Restart=always
RestartSec=10

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/app

# Performance
LimitNOFILE=65535
LimitNPROC=4096
TasksMax=4096

# CPU affinity for performance
CPUAffinity=0-3

[Install]
WantedBy=multi-user.target
```

### Kubernetes Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kernel-app
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      containers:
        - name: app
          image: ghcr.io/org/kernel-app:latest
          resources:
            requests:
              memory: '256Mi'
              cpu: '500m'
            limits:
              memory: '512Mi'
              cpu: '2000m'
          env:
            - name: GOMAXPROCS
              value: '2'
            - name: GOGC
              value: '100'
          securityContext:
            readOnlyRootFilesystem: true
            runAsNonRoot: true
            runAsUser: 1000
            allowPrivilegeEscalation: false
            capabilities:
              drop:
                - ALL
```

## Monitoring Production Builds

### Health Check Endpoint

```go
// healthz.go
package main

import (
    "encoding/json"
    "net/http"
    "runtime"
)

type Health struct {
    Status    string `json:"status"`
    Version   string `json:"version"`
    BuildDate string `json:"build_date"`
    GoVersion string `json:"go_version"`
    Compiler  string `json:"compiler"`
    Platform  string `json:"platform"`
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
    health := Health{
        Status:    "healthy",
        Version:   Version,
        BuildDate: BuildDate,
        GoVersion: runtime.Version(),
        Compiler:  runtime.Compiler,
        Platform:  runtime.GOOS + "/" + runtime.GOARCH,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}
```

## Do's

✅ **Use production build tags** for releases ✅ **Strip debug symbols** from binaries ✅ **Enable compiler optimizations** ✅ **Test production builds** thoroughly ✅ **Version stamp** all releases
✅ **Generate checksums** for binaries ✅ **Use static linking** when possible ✅ **Compress binaries** with UPX ✅ **Document build flags** used ✅ **Automate release process**

## Don'ts

❌ **Don't use development flags** in production ❌ **Don't include debug symbols** in production ❌ **Don't skip validation** before release ❌ **Don't build on development** machines ❌ **Don't
forget to tag** releases ❌ **Don't ignore platform-specific** optimizations ❌ **Don't enable race detector** in production ❌ **Don't use CGO** unless necessary ❌ **Don't forget security**
hardening ❌ **Don't deploy unversioned** binaries

## Related Documents

- [01-development-commands.md](01-development-commands.md) - Development builds
- [02-validation-commands.md](02-validation-commands.md) - Build validation
- [../02-implementation/04-safe-unsafe-patterns.md](../02-implementation/04-safe-unsafe-patterns.md) - Production optimizations
- [../01-architecture/03-performance-architecture.md](../01-architecture/03-performance-architecture.md) - Performance considerations
