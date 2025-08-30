package kbuffer

import (
	"fmt"
	"testing"
	"unsafe"
)

// TestConstants verifies that all constant values are correctly defined.
// Tests compile-time constants for expected values and relationships.
func TestConstants(t *testing.T) {
	// Test cache line size
	if cacheLineSize != 64 {
		t.Errorf("cacheLineSize = %d, want 64", cacheLineSize)
	}

	// Test buffer size constants
	if minBufferSize != 64 {
		t.Errorf("minBufferSize = %d, want 64", minBufferSize)
	}
	if defaultBufferSize != 4096 {
		t.Errorf("defaultBufferSize = %d, want 4096", defaultBufferSize)
	}
	if maxBufferSize != 16<<20 {
		t.Errorf("maxBufferSize = %d, want %d", maxBufferSize, 16<<20)
	}
	if optimalIOSize != 65536 {
		t.Errorf("optimalIOSize = %d, want 65536", optimalIOSize)
	}

	// Test pool constants
	if poolMinSize != 64 {
		t.Errorf("poolMinSize = %d, want 64", poolMinSize)
	}
	if poolMaxSize != 1<<22 {
		t.Errorf("poolMaxSize = %d, want %d", poolMaxSize, 1<<22)
	}
	if poolClassCount != 17 {
		t.Errorf("poolClassCount = %d, want 17", poolClassCount)
	}

	// Test sharding constants
	if defaultShardCount != 16 {
		t.Errorf("defaultShardCount = %d, want 16", defaultShardCount)
	}
	if maxShardCount != 256 {
		t.Errorf("maxShardCount = %d, want 256", maxShardCount)
	}
	if shardCachePadding != cacheLineSize {
		t.Errorf("shardCachePadding = %d, want %d", shardCachePadding, cacheLineSize)
	}

	// Test atomic operation constants
	if spinLimit != 100 {
		t.Errorf("spinLimit = %d, want 100", spinLimit)
	}
	if backoffInitial != 10 {
		t.Errorf("backoffInitial = %d, want 10", backoffInitial)
	}
	if backoffMax != 10000 {
		t.Errorf("backoffMax = %d, want 10000", backoffMax)
	}

	// Test memory alignment constants
	if ptrSize != unsafe.Sizeof(uintptr(0)) {
		t.Errorf("ptrSize mismatch")
	}
	if wordSize != unsafe.Sizeof(uint(0)) {
		t.Errorf("wordSize mismatch")
	}
	if alignment16 != 16 {
		t.Errorf("alignment16 = %d, want 16", alignment16)
	}
	if alignment32 != 32 {
		t.Errorf("alignment32 = %d, want 32", alignment32)
	}
}

// TestBufferError verifies the bufferError type implementation.
// Tests that error constants work correctly without allocations.
func TestBufferError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"BufferFull", errBufferFull, "buffer full"},
		{"InvalidSize", errInvalidSize, "invalid size"},
		{"InvalidOffset", errInvalidOffset, "invalid offset"},
		{"NilBuffer", errNilBuffer, "nil buffer"},
		{"ConcurrentModification", errConcurrentModification, "concurrent modification"},
		{"ShardOutOfBounds", errShardOutOfBounds, "shard index out of bounds"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestStateFlags verifies state flag constants are powers of 2.
// Ensures bit flags don't overlap for correct bitwise operations.
func TestStateFlags(t *testing.T) {
	// Test that flags are unique powers of 2
	flags := []uint32{
		stateFlagFull,
		stateFlagLocked,
		stateFlagPooled,
		stateFlagCleared,
		stateFlagReadOnly,
	}

	for i, flag := range flags {
		// Check it's a power of 2 (has only one bit set)
		if flag != 0 && (flag&(flag-1)) != 0 {
			t.Errorf("flags[%d] = %d is not a power of 2", i, flag)
		}

		// Check no overlap with other flags
		for j, other := range flags {
			if i != j && flag&other != 0 {
				t.Errorf("flags[%d] (%d) overlaps with flags[%d] (%d)", i, flag, j, other)
			}
		}
	}

	// Test normal state is zero
	if stateFlagNormal != 0 {
		t.Errorf("stateFlagNormal = %d, want 0", stateFlagNormal)
	}
}

// TestPerformanceHints verifies performance hint constants.
// Tests that hint values are as expected for optimization.
func TestPerformanceHints(t *testing.T) {
	if likelyTrue != 1 {
		t.Errorf("likelyTrue = %d, want 1", likelyTrue)
	}
	if likelyFalse != 0 {
		t.Errorf("likelyFalse = %d, want 0", likelyFalse)
	}
	if prefetchRead != 0 {
		t.Errorf("prefetchRead = %d, want 0", prefetchRead)
	}
	if prefetchWrite != 1 {
		t.Errorf("prefetchWrite = %d, want 1", prefetchWrite)
	}
}

// TestFactoryFunctions verifies that factory functions create correct buffer types.
// Tests that safety guarantees are properly enforced.
func TestFactoryFunctions(t *testing.T) {
	t.Run("UnsafeBuffer", func(t *testing.T) {
		buf := NewUnsafeBuffer(1024)
		if buf == nil {
			t.Fatal("NewUnsafeBuffer returned nil")
		}
		// Verify it's actually an unsafe buffer
		if _, ok := buf.(*unsafeBuffer); !ok {
			t.Error("NewUnsafeBuffer did not return *unsafeBuffer")
		}
	})

	t.Run("SafeBuffer", func(t *testing.T) {
		buf := NewSafeBuffer(1024)
		if buf == nil {
			t.Fatal("NewSafeBuffer returned nil")
		}
		// Verify it's actually a safe buffer
		if _, ok := buf.(*safeBuffer); !ok {
			t.Error("NewSafeBuffer did not return *safeBuffer")
		}
	})

	t.Run("UnsafeShardedBuffer", func(t *testing.T) {
		buf := NewUnsafeShardedBuffer(1024, 4)
		if buf == nil {
			t.Fatal("NewUnsafeShardedBuffer returned nil")
		}
		// Verify it's actually an unsafe sharded buffer
		if _, ok := buf.(*unsafeShardedBuffer); !ok {
			t.Error("NewUnsafeShardedBuffer did not return *unsafeShardedBuffer")
		}
	})

	t.Run("SafeShardedBuffer", func(t *testing.T) {
		buf := NewSafeShardedBuffer(1024, 4)
		if buf == nil {
			t.Fatal("NewSafeShardedBuffer returned nil")
		}
		// Verify it's actually a safe sharded buffer
		if _, ok := buf.(*safeShardedBuffer); !ok {
			t.Error("NewSafeShardedBuffer did not return *safeShardedBuffer")
		}
	})
}

// TestGlobalPoolInterface verifies the global pool instance is accessible.
// Tests that pool singleton is properly initialized.
func TestGlobalPoolInterface(t *testing.T) {
	pool := GetGlobalPool()
	if pool == nil {
		t.Fatal("GetGlobalPool returned nil")
	}

	// Test basic pool operations
	buf := pool.GetBuffer(1024)
	if buf == nil {
		t.Fatal("Pool.GetBuffer returned nil")
	}

	// Verify buffer has expected capacity
	if buf.Cap() < 1024 {
		t.Errorf("Buffer capacity = %d, want >= 1024", buf.Cap())
	}

	// Return buffer to pool
	pool.PutBuffer(buf)
}

// TestInterfaceCompliance verifies that all buffer types implement required interfaces.
// Compile-time checks ensure interface compliance.
func TestInterfaceCompliance(t *testing.T) {
	// These are compile-time checks
	var _ Buffer = (*unsafeBuffer)(nil)
	var _ Buffer = (*safeBuffer)(nil)
	var _ Sharded = (*unsafeShardedBuffer)(nil)
	var _ Sharded = (*safeShardedBuffer)(nil)
	var _ Pool = (*bufferPool)(nil)

	// Test that buffers implement standard interfaces
	var _ Writer = (*unsafeBuffer)(nil)
	var _ Writer = (*safeBuffer)(nil)

	t.Log("All interface compliance checks passed")
}

// TestUtilityFunctions tests utility functions like min, max, and nextPowerOf2.
func TestUtilityFunctions(t *testing.T) {
	t.Run("min", func(t *testing.T) {
		tests := []struct {
			a, b int64
			want int64
		}{
			{1, 2, 1},
			{2, 1, 1},
			{0, 0, 0},
			{-1, 0, -1},
			{0, -1, -1},
			{100, 200, 100},
			{200, 100, 100},
			{int64(1 << 62), int64(1 << 61), int64(1 << 61)},
		}

		for _, tt := range tests {
			got := min(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		}
	})

	t.Run("max", func(t *testing.T) {
		tests := []struct {
			a, b int
			want int
		}{
			{1, 2, 2},
			{2, 1, 2},
			{0, 0, 0},
			{-1, 0, 0},
			{0, -1, 0},
			{100, 200, 200},
			{200, 100, 200},
		}

		for _, tt := range tests {
			got := max(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		}
	})

	t.Run("nextPowerOf2", func(t *testing.T) {
		tests := []struct {
			n    uint32
			want uint32
		}{
			{0, 1},          // Zero case
			{1, 1},          // Already power of 2
			{2, 2},          // Already power of 2
			{3, 4},          // Round up
			{4, 4},          // Already power of 2
			{5, 8},          // Round up
			{7, 8},          // Round up
			{8, 8},          // Already power of 2
			{9, 16},         // Round up
			{15, 16},        // Round up
			{16, 16},        // Already power of 2
			{17, 32},        // Round up
			{31, 32},        // Round up
			{32, 32},        // Already power of 2
			{100, 128},      // Round up
			{1000, 1024},    // Round up
			{1024, 1024},    // Already power of 2
			{1025, 2048},    // Round up
			{65535, 65536},  // Round up
			{65536, 65536},  // Already power of 2
			{65537, 131072}, // Round up
		}

		for _, tt := range tests {
			got := nextPowerOf2(tt.n)
			if got != tt.want {
				t.Errorf("nextPowerOf2(%d) = %d, want %d", tt.n, got, tt.want)
			}
		}
	})
}

// TestConcurrentFactoryFunctions tests factory functions under concurrent access.
func TestConcurrentFactoryFunctions(t *testing.T) {
	t.Run("ConcurrentUnsafeBuffer", func(t *testing.T) {
		const goroutines = 100
		const iterations = 100

		for i := 0; i < goroutines; i++ {
			go func() {
				for j := 0; j < iterations; j++ {
					buf := NewUnsafeBuffer(1024)
					if buf == nil {
						t.Error("NewUnsafeBuffer returned nil during concurrent access")
					}
					buf.Write([]byte("test"))
				}
			}()
		}
	})

	t.Run("ConcurrentSafeBuffer", func(t *testing.T) {
		const goroutines = 100
		const iterations = 100

		for i := 0; i < goroutines; i++ {
			go func() {
				for j := 0; j < iterations; j++ {
					buf := NewSafeBuffer(1024)
					if buf == nil {
						t.Error("NewSafeBuffer returned nil during concurrent access")
					}
					buf.Write([]byte("test"))
				}
			}()
		}
	})

	t.Run("ConcurrentShardedBuffers", func(t *testing.T) {
		const goroutines = 100
		const iterations = 100

		for i := 0; i < goroutines; i++ {
			go func() {
				for j := 0; j < iterations; j++ {
					// Test both safe and unsafe sharded buffers
					safeBuf := NewSafeShardedBuffer(1024, 4)
					if safeBuf == nil {
						t.Error("NewSafeShardedBuffer returned nil during concurrent access")
					}
					safeBuf.Write([]byte("test"))

					unsafeBuf := NewUnsafeShardedBuffer(1024, 4)
					if unsafeBuf == nil {
						t.Error("NewUnsafeShardedBuffer returned nil during concurrent access")
					}
					unsafeBuf.Write([]byte("test"))
				}
			}()
		}
	})
}

// TestInterfacePanicRecovery tests that the interface handles panics gracefully.
func TestInterfacePanicRecovery(t *testing.T) {
	t.Run("RecoverFromNilBuffer", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Should not panic on nil buffer operations: %v", r)
			}
		}()

		// These operations should be safe
		pool := GetGlobalPool()
		pool.PutBuffer(nil)
	})

	t.Run("RecoverFromInvalidSizes", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Should not panic on invalid sizes: %v", r)
			}
		}()

		// Test with invalid sizes
		_ = NewUnsafeBuffer(-1)
		_ = NewSafeBuffer(0)
		_ = NewUnsafeShardedBuffer(-100, 4)
		_ = NewSafeShardedBuffer(1024, 0)
		_ = NewSafeShardedBuffer(1024, -1)
		_ = NewUnsafeShardedBuffer(1024, maxShardCount+1)
	})

	t.Run("RecoverFromOverflow", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Should handle overflow gracefully: %v", r)
			}
		}()

		// Test with extremely large sizes that might overflow
		_ = NewUnsafeBuffer(1 << 62)
		_ = NewSafeShardedBuffer(1<<62, 256)
	})
}

// Benchmarks for interface functions

// BenchmarkFactoryFunctions benchmarks buffer creation.
func BenchmarkFactoryFunctions(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096, 16384}

	b.Run("NewUnsafeBuffer", func(b *testing.B) {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_ = NewUnsafeBuffer(size)
				}
			})
		}
	})

	b.Run("NewSafeBuffer", func(b *testing.B) {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_ = NewSafeBuffer(size)
				}
			})
		}
	})

	b.Run("NewUnsafeShardedBuffer", func(b *testing.B) {
		shardCounts := []int{4, 8, 16, 32}
		for _, size := range sizes {
			for _, shards := range shardCounts {
				b.Run(fmt.Sprintf("Size%d_Shards%d", size, shards), func(b *testing.B) {
					b.ResetTimer()
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						_ = NewUnsafeShardedBuffer(size, shards)
					}
				})
			}
		}
	})

	b.Run("NewSafeShardedBuffer", func(b *testing.B) {
		shardCounts := []int{4, 8, 16, 32}
		for _, size := range sizes {
			for _, shards := range shardCounts {
				b.Run(fmt.Sprintf("Size%d_Shards%d", size, shards), func(b *testing.B) {
					b.ResetTimer()
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						_ = NewSafeShardedBuffer(size, shards)
					}
				})
			}
		}
	})
}

// BenchmarkConcurrentFactoryFunctions benchmarks concurrent buffer creation.
func BenchmarkConcurrentFactoryFunctions(b *testing.B) {
	sizes := []int{256, 1024, 4096}

	b.Run("ConcurrentUnsafeBuffer", func(b *testing.B) {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						buf := NewUnsafeBuffer(size)
						buf.Write([]byte("test"))
					}
				})
			})
		}
	})

	b.Run("ConcurrentSafeBuffer", func(b *testing.B) {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						buf := NewSafeBuffer(size)
						buf.Write([]byte("test"))
					}
				})
			})
		}
	})

	b.Run("ConcurrentShardedBuffer", func(b *testing.B) {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						buf := NewSafeShardedBuffer(size, 16)
						buf.Write([]byte("test"))
					}
				})
			})
		}
	})
}

// BenchmarkUtilityFunctions benchmarks utility function performance.
func BenchmarkUtilityFunctions(b *testing.B) {
	b.Run("min", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = min(int64(i), int64(b.N))
		}
	})

	b.Run("max", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = max(i, b.N)
		}
	})

	b.Run("nextPowerOf2", func(b *testing.B) {
		values := []uint32{0, 1, 3, 7, 15, 31, 63, 127, 255, 511, 1023}
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, v := range values {
				_ = nextPowerOf2(v)
			}
		}
	})
}

// BenchmarkGlobalPool benchmarks global pool operations.
func BenchmarkGlobalPool(b *testing.B) {
	pool := GetGlobalPool()

	b.Run("GetBuffer", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = pool.GetBuffer(1024)
		}
	})

	b.Run("GetPutBuffer", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := pool.GetBuffer(1024)
			pool.PutBuffer(buf)
		}
	})

	b.Run("ConcurrentGetPutBuffer", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := pool.GetBuffer(1024)
				buf.Write([]byte("test"))
				pool.PutBuffer(buf)
			}
		})
	})
}
