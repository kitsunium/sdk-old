package errs

import (
	"fmt"
	"testing"
)

// Benchmarks to measure performance improvements with cache

func BenchmarkGetError(b *testing.B) {
	// Setup: Register many errors
	ClearRegistry()
	defer ClearRegistry()

	// Pre-register errors for benchmark
	for i := 0; i < 1000; i++ {
		Define(KConfig{
			Code:    i,
			Message: fmt.Sprintf("Error %d", i),
			Package: "benchmark",
		})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			GetError(uint32((i % 1000) + 1))
			i++
		}
	})
}

func BenchmarkGetErrorByPackageCode(b *testing.B) {
	// Setup: Register many errors
	ClearRegistry()
	defer ClearRegistry()

	for i := 0; i < 100; i++ {
		for j := 0; j < 10; j++ {
			Define(KConfig{
				Code:    j,
				Message: fmt.Sprintf("Error %d-%d", i, j),
				Package: fmt.Sprintf("pkg%d", i),
			})
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			GetErrorByPackageCode(fmt.Sprintf("pkg%d", i%100), i%10)
			i++
		}
	})
}

func BenchmarkDefineError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Clear and redefine to avoid duplicates
		ClearRegistry()
		Define(KConfig{
			Code:    i,
			Message: fmt.Sprintf("Error %d", i),
			Package: fmt.Sprintf("pkg%d", i),
		})
	}
}

func BenchmarkValidatePackageCode(b *testing.B) {
	// Setup: Register some errors
	ClearRegistry()
	defer ClearRegistry()

	for i := 0; i < 100; i++ {
		Define(KConfig{
			Code:    i,
			Message: fmt.Sprintf("Error %d", i),
			Package: "benchmark",
		})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Mix of existing and non-existing codes
			if i%2 == 0 {
				ValidatePackageCode("benchmark", i%100)
			} else {
				ValidatePackageCode("benchmark", 1000+i)
			}
			i++
		}
	})
}

func BenchmarkGetCallerPackage(b *testing.B) {
	// Clear cache to measure caching performance
	ClearRegistry()
	defer ClearRegistry()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			getCallerPackage()
		}
	})
}

func BenchmarkListErrors(b *testing.B) {
	// Setup: Register many errors
	ClearRegistry()
	defer ClearRegistry()

	for i := 0; i < 1000; i++ {
		Define(KConfig{
			Code:    i,
			Message: fmt.Sprintf("Error %d", i),
			Package: fmt.Sprintf("pkg%d", i%10),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ListErrors()
	}
}

func BenchmarkListPackages(b *testing.B) {
	// Setup: Register errors in many packages
	ClearRegistry()
	defer ClearRegistry()

	for i := 0; i < 100; i++ {
		Define(KConfig{
			Code:    1,
			Message: "Error",
			Package: fmt.Sprintf("pkg%d", i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ListPackages()
	}
}

func BenchmarkConcurrentRegistry(b *testing.B) {
	// Benchmark concurrent access to registry
	ClearRegistry()
	defer ClearRegistry()

	// Pre-populate
	for i := 0; i < 100; i++ {
		Define(KConfig{
			Code:    i,
			Message: fmt.Sprintf("Error %d", i),
			Package: "concurrent",
		})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 4 {
			case 0:
				GetError(uint32((i % 100) + 1))
			case 1:
				GetErrorByPackageCode("concurrent", i%100)
			case 2:
				ValidatePackageCode("concurrent", 1000+i)
			case 3:
				ListPackageCodes("concurrent")
			}
			i++
		}
	})
}
