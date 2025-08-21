#!/bin/bash

# Script to install git hooks

HOOKS_DIR=".githooks"

# Set git config to use our hooks directory
git config core.hooksPath "$HOOKS_DIR"

echo "✅ Git hooks installed from $HOOKS_DIR"
echo "Active hooks:"
echo "  - commit-msg: Prevents commits with AI assistant mentions"
echo "  - pre-push: Validates all commits before pushing"