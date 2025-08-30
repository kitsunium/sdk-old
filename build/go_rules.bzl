"""
Standardized Go build rules for Kitsunium SDK.

This file provides consistent macros for Go libraries, tests, and benchmarks
with proper dev/prod mode separation.
"""

load("@rules_go//go:def.bzl", "go_library", "go_test")

def kitsunium_go_library(
        name,
        srcs = None,
        importpath = None,
        deps = None,
        visibility = None,
        **kwargs):
    """
    Standardized Go library rule for Kitsunium SDK.
    
    Args:
        name: Target name
        srcs: Source files (defaults to glob(["*.go"], exclude = ["*_test.go"]))
        importpath: Import path for the library
        deps: Dependencies
        visibility: Visibility rules (defaults to //visibility:public)
        **kwargs: Additional arguments passed to go_library
    """
    if srcs == None:
        srcs = native.glob(["*.go"], exclude = ["*_test.go"])
    
    if visibility == None:
        visibility = ["//visibility:public"]
    
    # All Kitsunium libraries use these settings for consistency
    # Note: pure and static are now set via build flags in .bazelrc
    go_library(
        name = name,
        srcs = srcs,
        importpath = importpath,
        deps = deps or [],
        visibility = visibility,
        cgo = False,  # CGO disabled for all Kitsunium libraries
        **kwargs
    )

def kitsunium_go_test(
        name,
        srcs = None,
        embed = None,
        deps = None,
        size = "small",
        timeout = "short",
        exclude_patterns = None,
        **kwargs):
    """
    Standardized Go test rule for Kitsunium SDK.
    
    Unit tests ALWAYS run with:
    - Race detection enabled
    - Safety checks enabled (no gotags = dev mode)
    - Support for goroutines with panic on thread-safety violations
    
    Args:
        name: Target name
        srcs: Test source files (defaults to glob of *_test.go)
        embed: Library to embed
        deps: Test dependencies
        size: Test size (small, medium, large)
        timeout: Test timeout (short, moderate, long, eternal)
        exclude_patterns: Patterns to exclude from glob
        **kwargs: Additional arguments passed to go_test
    """
    if exclude_patterns == None:
        exclude_patterns = ["*_bench_test.go", "*_bench_multi_test.go", "*_race_test.go"]
    
    if srcs == None:
        srcs = native.glob(["*_test.go"], exclude = exclude_patterns, allow_empty = True)
    
    # Only create test target if there are test files
    if len(srcs) > 0:
        go_test(
            name = name,
            srcs = srcs,
            embed = embed or [],
            deps = deps or [],
            size = size,
            timeout = timeout,
            # TEST MODE: race detection + safety checks
            race = "on",
            # No gotags = safety checks enabled (dev mode)
            **kwargs
        )

def kitsunium_go_benchmark(
        name,
        srcs = None,
        embed = None,
        deps = None,
        **kwargs):
    """
    Standardized Go benchmark rule for Kitsunium SDK.
    
    Benchmarks ALWAYS run in production mode:
    - No race detection
    - Maximum performance (gotags = ["unsafe_no_check"])
    - Only nominal cases, no error paths
    - Suitable for both single-core and multi-core benchmarks
    
    Args:
        name: Target name
        srcs: Benchmark source files (auto-detected based on name)
        embed: Library to embed
        deps: Benchmark dependencies
        **kwargs: Additional arguments passed to go_test
    """
    # Auto-detect source files based on target name
    if srcs == None:
        if "multi" in name:
            srcs = native.glob(["*_bench_multi_test.go", "*bench*test.go"], allow_empty = True)
        else:
            srcs = native.glob(["*_bench_test.go", "*bench*test.go"], allow_empty = True)
    
    # Only create benchmark target if there are benchmark files
    if len(srcs) > 0:
        # Base benchmark arguments
        base_args = [
            "-test.run=^$$",
            "-test.bench=.",
            "-test.benchmem",
            "-test.count=1",
            # Removed hardcoded benchtime to allow override via command line
        ]
        
        # Single-core benchmarks use -test.cpu=1
        if "multi" not in name:
            base_args.append("-test.cpu=1")
        
        go_test(
            name = name,
            srcs = srcs,
            embed = embed or [],
            deps = deps or [],
            args = base_args,
            size = "large",
            timeout = "long",
            # BENCHMARK MODE: production settings only
            gotags = ["unsafe_no_check"],
            race = "off",  # NEVER enable race detector for benchmarks
            local = True,
            tags = ["benchmark", "manual"],
            **kwargs
        )

def kitsunium_go_package(
        name,
        importpath,
        deps = None,
        test_deps = None,
        has_benchmarks = True):
    """
    Complete package setup with library, tests, and benchmarks.
    
    This macro creates:
    - A library target (name)
    - A test target ("test") with race detection and safety checks
    - Benchmark targets if applicable:
      - bench: Single-core benchmark in production mode
      - bench_multi: Multi-core benchmark in production mode
    
    Args:
        name: Package name
        importpath: Import path for the library
        deps: Library dependencies
        test_deps: Additional test dependencies
        has_benchmarks: Whether to generate benchmark targets
    """
    # Create the library
    kitsunium_go_library(
        name = name,
        importpath = importpath,
        deps = deps,
    )
    
    # Create the test target (MODE 1: TEST)
    kitsunium_go_test(
        name = "test",
        embed = [":" + name],
        deps = test_deps,
    )
    
    # Create benchmark targets if needed (MODE 2: BENCHMARK)
    if has_benchmarks:
        # Single-core benchmark
        kitsunium_go_benchmark(
            name = "bench",
            embed = [":" + name],
            deps = test_deps,
        )
        
        # Multi-core benchmark
        kitsunium_go_benchmark(
            name = "bench_multi",
            embed = [":" + name],
            deps = test_deps,
        )