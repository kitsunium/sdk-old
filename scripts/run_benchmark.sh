#!/bin/bash
# Wrapper script to run benchmarks directly without bazel run issues

TARGET=$1
shift

# Build the target first
bazel build $TARGET 2>&1 | grep -v "INFO:" | grep -v "Loading:" | grep -v "Analyzing:"

# Find the binary
BINARY=$(bazel cquery --output=files $TARGET 2>/dev/null)

if [ -z "$BINARY" ]; then
  echo "Error: Could not find binary for $TARGET"
  exit 1
fi

# Run the binary directly with arguments
exec $BINARY "$@"