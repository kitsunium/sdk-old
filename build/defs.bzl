"""Common build definitions for the SDK."""

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

def sdk_go_library(**kwargs):
    """Standard Go library for SDK packages."""
    go_library(**kwargs)

def sdk_go_test(name, srcs, deps = [], tags = [], **kwargs):
    """
    Standard Go test with race detection and proper settings.
    Used for all unit tests with goroutines and panic detection.
    """
    go_test(
        name = name,
        srcs = srcs,
        deps = deps,
        race = "on",
        tags = tags + ["unit"],
        **kwargs
    )

def sdk_go_benchmark(name, srcs, deps = [], tags = [], **kwargs):
    """
    Standard Go benchmark for performance testing.
    No race detection, optimized for performance measurement.
    Used for both single-core and multi-core benchmarks.
    """
    go_test(
        name = name,
        srcs = srcs,
        deps = deps,
        race = "off",
        tags = tags + ["benchmark"],
        **kwargs
    )