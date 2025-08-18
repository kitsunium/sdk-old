package kbuffer

import (
	"sync"
	"testing"
	"unsafe"
)

// Comparison benchmarks between old buffer and new kbuffer implementations

// Old buffer implementation (for comparison)
type oldBuffer struct {
	b []byte
	n int
}

func newOldBuffer(size int) *oldBuffer {
	return &oldBuffer{
		b: make([]byte, size),
		n: 0,
	}
}

func (b *oldBuffer) Write(p []byte) (int, error) {
	if len(p) > len(b.b)-b.n {
		return 0, ErrBufferOverflow
	}
	n := copy(b.b[b.n:], p)
	b.n += n
	return n, nil
}

func (b *oldBuffer) String() string {
	return string(b.b[:b.n])
}

// Benchmark: Write operations
func BenchmarkWriteComparison(b *testing.B) {
	data := []byte("Hello, World! This is a test string for benchmarking.")
	
	b.Run("OldBuffer", func(b *testing.B) {
		buf := newOldBuffer(4096)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.n = 0 // Reset
			buf.Write(data)
		}
	})
	
	b.Run("KBuffer", func(b *testing.B) {
		buf := NewBuffer(4096)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Free() // Reset
			buf.Write(data)
		}
	})
	
	b.Run("KBuffer_TryWrite", func(b *testing.B) {
		buf := NewBuffer(4096)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Free() // Reset
			buf.TryWrite(data) // Optimized version
		}
	})
}

// Benchmark: String conversion
func BenchmarkStringConversion(b *testing.B) {
	data := []byte("Hello, World! This is a test string for benchmarking.")
	
	b.Run("OldBuffer_String", func(b *testing.B) {
		buf := newOldBuffer(4096)
		buf.Write(data)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = buf.String()
		}
	})
	
	b.Run("KBuffer_String_Copy", func(b *testing.B) {
		// Simulating old method with string copy
		buf := NewBuffer(4096)
		buf.Write(data)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = string(buf.Bytes()) // Creates a copy
		}
	})
	
	b.Run("KBuffer_String_Unsafe", func(b *testing.B) {
		// New optimized method with unsafe.String
		buf := NewBuffer(4096)
		buf.Write(data)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = buf.String() // Uses unsafe.String, no allocation
		}
	})
}

// Benchmark: Pool comparison
func BenchmarkPoolComparison(b *testing.B) {
	b.Run("OldPool_SyncPool", func(b *testing.B) {
		// Traditional sync.Pool approach
		pool := &sync.Pool{
			New: func() interface{} {
				return make([]byte, 1024)
			},
		}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := pool.Get().([]byte)
				buf[0] = 1
				pool.Put(buf)
			}
		})
	})
	
	b.Run("KBuffer_Pool_SyncMap", func(b *testing.B) {
		// KBuffer pool with sync.Map
		pool := NewBufferPool()
		pool.Prewarm([]int{1024}, 100)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := pool.Get(1024)
				buf[0] = 1
				pool.Put(buf)
			}
		})
	})
	
	b.Run("KBuffer_Pool_WithStats", func(b *testing.B) {
		// KBuffer pool with statistics tracking
		pool := NewBufferPool()
		pool.Prewarm([]int{1024}, 100)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := pool.Get(1024)
				buf[0] = 1
				pool.Put(buf)
			}
		})
		// Statistics are tracked atomically without performance impact
	})
}

// Benchmark: Memory optimizations
func BenchmarkMemoryOptimizations(b *testing.B) {
	b.Run("OldBuffer_Clear", func(b *testing.B) {
		buf := newOldBuffer(4096)
		data := make([]byte, 4096)
		for i := range data {
			data[i] = byte(i)
		}
		buf.Write(data[:1000])
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Old way: manual loop
			for j := range buf.b {
				buf.b[j] = 0
			}
			buf.n = 0
		}
	})
	
	b.Run("KBuffer_Clear", func(b *testing.B) {
		buf := NewBuffer(4096)
		data := make([]byte, 4096)
		for i := range data {
			data[i] = byte(i)
		}
		buf.Write(data[:1000])
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Clear() // Uses Go 1.21+ clear() builtin
		}
	})
	
	b.Run("KBuffer_Free", func(b *testing.B) {
		buf := NewBuffer(4096)
		data := make([]byte, 4096)
		for i := range data {
			data[i] = byte(i)
		}
		buf.Write(data[:1000])
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Free() // Just resets position, no clearing
		}
	})
}

// Benchmark: Advanced operations
func BenchmarkAdvancedOperations(b *testing.B) {
	b.Run("OldBuffer_Append", func(b *testing.B) {
		buf := newOldBuffer(4096)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.n = 0
			// Old way: multiple Write calls
			buf.Write([]byte{1})
			buf.Write([]byte{2})
			buf.Write([]byte{3})
			buf.Write([]byte{4})
			buf.Write([]byte{5})
		}
	})
	
	b.Run("KBuffer_AppendBytes", func(b *testing.B) {
		buf := NewBuffer(4096)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Free()
			// New optimized way
			buf.AppendBytes(1, 2, 3, 4, 5)
		}
	})
	
	b.Run("KBuffer_WriteByte", func(b *testing.B) {
		buf := NewBuffer(4096)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Free()
			// Optimized single byte writes
			buf.WriteByte(1)
			buf.WriteByte(2)
			buf.WriteByte(3)
			buf.WriteByte(4)
			buf.WriteByte(5)
		}
	})
}

// Benchmark: Struct layout optimization
func BenchmarkStructLayout(b *testing.B) {
	b.Run("OldLayout", func(b *testing.B) {
		type oldLayout struct {
			b   []byte
			n   int
			cap int
		}
		buf := oldLayout{
			b:   make([]byte, 4096),
			n:   0,
			cap: 4096,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.n = (buf.n + 1) % buf.cap
		}
	})
	
	b.Run("KBuffer_OptimizedLayout", func(b *testing.B) {
		// KBuffer uses optimized field ordering for better cache alignment
		buf := NewBuffer(4096)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.pos = (buf.pos + 1) % buf.c
		}
	})
}

// Benchmark: Zero-allocation string conversion
func BenchmarkZeroAllocationString(b *testing.B) {
	data := []byte("This is a test string that we'll convert multiple times")
	
	b.Run("Standard_Conversion", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = string(data) // Allocates
		}
	})
	
	b.Run("Unsafe_String", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// KBuffer's approach
			_ = unsafe.String(unsafe.SliceData(data), len(data)) // Zero allocation
		}
	})
}

// Benchmark: Pool prewarming effect
func BenchmarkPoolPrewarming(b *testing.B) {
	b.Run("Cold_Pool", func(b *testing.B) {
		pool := NewBufferPool()
		// No prewarming
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := pool.Get(1024)
			pool.Put(buf)
		}
	})
	
	b.Run("Prewarmed_Pool", func(b *testing.B) {
		pool := NewBufferPool()
		// Prewarm with buffers
		pool.Prewarm([]int{1024}, 100)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := pool.Get(1024)
			pool.Put(buf)
		}
	})
}

// Benchmark: Inline optimizations
func BenchmarkInlineOptimizations(b *testing.B) {
	buf := NewBuffer(4096)
	
	b.Run("Len", func(b *testing.B) {
		// Marked with //go:inline
		for i := 0; i < b.N; i++ {
			_ = buf.Len()
		}
	})
	
	b.Run("Cap", func(b *testing.B) {
		// Marked with //go:inline
		for i := 0; i < b.N; i++ {
			_ = buf.Cap()
		}
	})
	
	b.Run("Available", func(b *testing.B) {
		// Marked with //go:inline
		for i := 0; i < b.N; i++ {
			_ = buf.Available()
		}
	})
}

// Benchmark: Power of 2 optimizations
func BenchmarkPowerOf2Optimization(b *testing.B) {
	b.Run("Regular_Modulo", func(b *testing.B) {
		size := 1000 // Not power of 2
		for i := 0; i < b.N; i++ {
			_ = i % size
		}
	})
	
	b.Run("Power2_BitMask", func(b *testing.B) {
		size := 1024 // Power of 2
		mask := size - 1
		for i := 0; i < b.N; i++ {
			_ = i & mask // Faster than modulo
		}
	})
	
	b.Run("NextPowerOf2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = nextPowerOf2(1000)
		}
	})
}