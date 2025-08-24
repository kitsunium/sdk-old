package kerror

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
)

func BenchmarkDefine(b *testing.B) {
	// Skip Define benchmarks as they pollute the registry
	b.Skip("Skipping Define benchmarks to avoid registry pollution")
}

func BenchmarkNew(b *testing.B) {
	err := Define(KConfig{
		Package: "bench",
		Code:    1001,
		Message: "benchmark error",
	})

	b.Run("Basic", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = err.New()
		}
	})

	b.Run("WithDetails", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = err.New().WithDetails(map[string]any{"detail1": "value1", "detail2": i})
		}
	})

	b.Run("WithDetail", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = err.New().WithDetail("user_id", 123).WithDetail("request_id", "abc123")
		}
	})

	b.Run("WithTags", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = err.New().WithTags(map[string]string{"network": "net", "timeout": "yes", "retry": "yes"})
		}
	})

	b.Run("Complete", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = err.New().
				WithDetails(map[string]any{"detail1": "value1"}).
				WithDetail("user_id", 123).
				WithTags(map[string]string{"network": "net", "timeout": "yes"})
		}
	})
}

func BenchmarkWrap(b *testing.B) {
	err := Define(KConfig{
		Package: "bench",
		Code:    2001,
		Message: "wrap benchmark error",
	})
	cause := errors.New("underlying error")

	b.Run("Basic", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = err.Wrap(cause)
		}
	})

	b.Run("Wrapf", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = err.Wrapf(cause, "custom message %d", i)
		}
	})

	b.Run("WithDetails", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = err.Wrap(cause).WithDetails(map[string]any{"key": "value"})
		}
	})
}

func BenchmarkInstanceOperations(b *testing.B) {
	err := Define(KConfig{
		Package: "bench",
		Code:    3001,
		Message: "instance benchmark error",
	})
	inst := err.New()

	b.Run("Error", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = inst.Error()
		}
	})

	b.Run("KErrorString", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = inst.KError().String()
		}
	})

	b.Run("MarshalJSON", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = inst.MarshalJSON()
		}
	})
}

func BenchmarkRegistry(b *testing.B) {
	// Pre-populate registry
	for i := 0; i < 100; i++ {
		_ = Define(KConfig{
			Package: "registry_bench",
			Code:    i,
			Message: fmt.Sprintf("error %d", i),
		})
	}

	b.Run("GetError", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = GetError(uint32(i%100 + 1))
		}
	})

	b.Run("GetErrorByPackageCode", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = GetErrorByPackageCode("registry_bench", i%100)
		}
	})
}

func BenchmarkResult(b *testing.B) {

	b.Run("NewResult_Ok", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewResult[int](42, true)
		}
	})

	b.Run("NewResult_Err", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var zero int
			_ = NewResult[int](zero, false)
		}
	})

	b.Run("CheckOk", func(b *testing.B) {
		result := NewResult[int](42, true)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = result.Ok
		}
	})

	b.Run("CheckErr", func(b *testing.B) {
		var zero int
		result := NewResult[int](zero, false)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = !result.Ok
		}
	})

	b.Run("Unwrap", func(b *testing.B) {
		result := NewResult[int](42, true)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = result.Unwrap()
		}
	})

	b.Run("UnwrapOr", func(b *testing.B) {
		var zero int
		result := NewResult[int](zero, false)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = result.UnwrapOr(99)
		}
	})
}

func BenchmarkStackTrace(b *testing.B) {
	err := Define(KConfig{
		Package: "bench",
		Code:    5001,
		Message: "stack trace benchmark error",
	})

	b.Run("CaptureStackTrace", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = err.New() // This captures stack trace internally
		}
	})

	b.Run("FormatStackTrace", func(b *testing.B) {
		inst := err.New()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = inst.StackTrace()
		}
	})
}

func BenchmarkConcurrentOperations(b *testing.B) {
	err := Define(KConfig{
		Package: "bench",
		Code:    6001,
		Message: "concurrent benchmark error",
	})

	b.Run("ConcurrentNew", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = err.New()
			}
		})
	})

	b.Run("ConcurrentWrap", func(b *testing.B) {
		cause := errors.New("cause")
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = err.Wrap(cause)
			}
		})
	})

	b.Run("ConcurrentRegistry", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_, _ = GetError(uint32(i%100 + 1))
				i++
			}
		})
	})
}

func BenchmarkErrorComparison(b *testing.B) {
	err1 := Define(KConfig{
		Package: "bench",
		Code:    7001,
		Message: "error 1",
	})
	err2 := Define(KConfig{
		Package: "bench",
		Code:    7002,
		Message: "error 2",
	})
	inst1 := err1.New()
	_ = err2.New() // inst2 not used in benchmarks

	b.Run("Is_Same", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = errors.Is(inst1, err1)
		}
	})

	b.Run("Is_Different", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = errors.Is(inst1, err2)
		}
	})

	b.Run("As", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var target *Instance
			_ = errors.As(inst1, &target)
		}
	})
}

func BenchmarkMemoryAllocation(b *testing.B) {
	err := Define(KConfig{
		Package: "bench",
		Code:    8001,
		Message: "memory benchmark error",
	})

	b.Run("New", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = err.New()
		}
	})

	b.Run("NewWithDetails", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = err.New().WithDetails(map[string]any{"key1": "value1", "key2": "value2"})
		}
	})

	b.Run("NewWithLargeDetails", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			details := make(map[string]interface{})
			for j := 0; j < 10; j++ {
				details[strconv.Itoa(j)] = fmt.Sprintf("value_%d_%d", i, j)
			}
			inst := err.New()
			inst = inst.WithDetails(details)
		}
	})
}

func BenchmarkMetrics(b *testing.B) {
	// Metrics are enabled by default in config
	err := Define(KConfig{
		Package: "bench_metrics",
		Code:    9001,
		Message: "metrics benchmark error",
	})

	b.Run("CreateInstance", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = err.New()
		}
	})
}

func BenchmarkCallerPackageDetection(b *testing.B) {
	// Skip to avoid registry pollution
	b.Skip("Skipping caller package detection benchmarks to avoid registry pollution")
}

func BenchmarkLargeRegistry(b *testing.B) {
	// Pre-populate with many errors
	packages := []string{"pkg1", "pkg2", "pkg3", "pkg4", "pkg5"}
	for _, pkg := range packages {
		for i := 0; i < 100; i++ {
			_ = Define(KConfig{
				Package: pkg,
				Code:    i,
				Message: fmt.Sprintf("%s error %d", pkg, i),
			})
		}
	}

	b.Run("RandomLookup", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pkg := packages[i%len(packages)]
			code := i % 100
			_, _ = GetErrorByPackageCode(pkg, code)
		}
	})
}

func BenchmarkCachePerformance(b *testing.B) {
	// Test the internal cache performance
	err := Define(KConfig{
		Package: "cache_bench",
		Code:    10001,
		Message: "cache benchmark error",
	})

	// Pre-warm cache
	for i := 0; i < 100; i++ {
		_ = err.New()
	}

	b.Run("CachedOperations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = getCallerPackage() // This uses internal cache
		}
	})
}

func BenchmarkParallelDefineAndUse(b *testing.B) {
	// Skip to avoid registry pollution
	b.Skip("Skipping parallel define benchmarks to avoid registry pollution")
}