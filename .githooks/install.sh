#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

# Script to install git hooks

HOOKS_DIR=".githooks"

# Ensure we're inside a Git repository
if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "Error: this script must be run from inside a Git repository." >&2
  exit 1
fi

# Validate hooks directory exists
if [ ! -d "$HOOKS_DIR" ]; then
  echo "Error: hooks directory '$HOOKS_DIR' not found. Run this from the repo root." >&2
  exit 1
fi

# Ensure hook scripts are executable (idempotent)
chmod +x "$HOOKS_DIR"/* 2>/dev/null || true

# Set git config to use our hooks directory (local scope)
git config --local core.hooksPath "$HOOKS_DIR"

echo "✅ Git hooks installed from $HOOKS_DIR"
echo "Active hooks:"
echo "  - commit-msg: Prevents commits with AI assistant mentions"
echo "  - pre-push: Validates all commits before pushing"