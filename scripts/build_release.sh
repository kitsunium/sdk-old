#!/usr/bin/env bash

# Build release script for Kitsunium SDK
# This script builds the SDK in production mode for release

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
MODE="prod"
VERBOSE=false

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build the Kitsunium SDK for release

Options:
    -m, --mode MODE     Build mode (dev or prod, default: prod)
    -v, --verbose       Enable verbose output
    -h, --help          Show this help message

Examples:
    $0                  # Build in production mode
    $0 --mode dev       # Build in development mode (with safety checks)
    $0 --verbose        # Build with verbose output
EOF
    exit 1
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -m|--mode)
            MODE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo -e "${RED}Error: Unknown option: $1${NC}" >&2
            usage
            ;;
    esac
done

# Validate mode
if [[ "$MODE" != "dev" && "$MODE" != "prod" ]]; then
    echo -e "${RED}Error: Invalid mode: $MODE (must be 'dev' or 'prod')${NC}" >&2
    exit 1
fi

# Print build configuration
echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}   Kitsunium SDK Release Build${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""
echo -e "Mode: ${GREEN}$MODE${NC}"

if [[ "$MODE" == "prod" ]]; then
    echo -e "Safety checks: ${RED}DISABLED${NC} (maximum performance)"
    echo -e "Optimization: ${GREEN}MAXIMUM${NC}"
else
    echo -e "Safety checks: ${GREEN}ENABLED${NC}"
    echo -e "Optimization: ${GREEN}HIGH${NC} (99% of production)"
fi

echo ""
echo -e "${YELLOW}Building...${NC}"

# Set build command
BUILD_CMD="bazel build //... --config=$MODE"

if [[ "$VERBOSE" == true ]]; then
    BUILD_CMD="$BUILD_CMD --verbose_failures --show_progress_rate_limit=5"
fi

# Execute build
if eval $BUILD_CMD; then
    echo ""
    echo -e "${GREEN}✅ Build successful!${NC}"
    echo ""
    echo "Build artifacts are available in bazel-bin/"
    
    # Show important build info
    if [[ "$MODE" == "prod" ]]; then
        echo ""
        echo -e "${YELLOW}⚠️  Production build notes:${NC}"
        echo "  - Safety checks are DISABLED (unsafe_no_check tag active)"
        echo "  - UnsafeBuffer and UnsafeShardedBuffer have NO concurrent access protection"
        echo "  - Use this build ONLY after thorough testing"
    else
        echo ""
        echo -e "${GREEN}✓ Development build notes:${NC}"
        echo "  - Safety checks are ENABLED"
        echo "  - Concurrent access will trigger panics to prevent data corruption"
        echo "  - Performance is ~99% of production mode"
    fi
else
    echo ""
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

# Optional: Run tests
echo ""
read -p "Run tests? (y/N) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Running tests in $MODE mode...${NC}"
    if bazel test //... --config=$MODE --test_output=errors; then
        echo -e "${GREEN}✅ All tests passed!${NC}"
    else
        echo -e "${RED}❌ Some tests failed!${NC}"
        exit 1
    fi
fi

echo ""
echo -e "${GREEN}Build complete!${NC}"