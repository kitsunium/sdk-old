#!/bin/bash

# Script to set up git hooks for the project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Setting up Git hooks for the project...${NC}"
echo ""

# Get the project root directory
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOOKS_DIR="$PROJECT_ROOT/.githooks"

# Check if .githooks directory exists
if [ ! -d "$HOOKS_DIR" ]; then
    echo -e "${RED}Error: .githooks directory not found at $HOOKS_DIR${NC}"
    exit 1
fi

# Configure git to use our hooks directory
echo "Configuring Git to use $HOOKS_DIR..."
git config core.hooksPath "$HOOKS_DIR"

# Make all hooks executable
echo "Making hooks executable..."
chmod +x "$HOOKS_DIR"/*

# List available hooks
echo ""
echo -e "${GREEN}✓ Git hooks configured successfully!${NC}"
echo ""
echo "Available hooks:"
for hook in "$HOOKS_DIR"/*; do
    if [ -f "$hook" ]; then
        basename "$hook" | sed 's/^/  • /'
    fi
done

echo ""
echo -e "${YELLOW}Features enabled:${NC}"
echo "  • t.Skip() detection: Prevents commits with t.Skip() in test files"
echo "  • Auto-formatting: Formats Go files before commit"
echo "  • Gazelle integration: Updates BUILD.bazel files automatically"
echo "  • Main branch protection: Prevents direct commits to main branch"
echo ""
echo -e "${BLUE}To disable hooks temporarily, use:${NC}"
echo "  git commit --no-verify"
echo ""
echo -e "${BLUE}To reset hooks to system default:${NC}"
echo "  git config --unset core.hooksPath"
echo ""