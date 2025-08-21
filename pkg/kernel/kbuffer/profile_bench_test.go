package kbuffer

import (
	"bytes"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
)

// BenchmarkWithCPUProfile runs benchmarks with CPU profiling enabled
func BenchmarkWithCPUProfile(b *testing.B) {
	// Create CPU profile file
	f, err := os.Create("cpu.prof")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	// Start CPU profiling
	if err := pprof.StartCPUProfile(f); err != nil {
		b.Fatal(err)
	}
	defer pprof.StopCPUProfile()

	// Run actual benchmark
	b.Run("Buffer_Write", func(b *testing.B) {
		buf := NewBuffer(4096)
		data := make([]byte, 256)
		rand.Read(data)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			for j := 0; j < 16; j++ {
				buf.Write(data)
			}
		}
	})
}

// BenchmarkWithMemProfile runs benchmarks with memory profiling
func BenchmarkWithMemProfile(b *testing.B) {
	// Run benchmark
	b.Run("Pool_GetPut", func(b *testing.B) {
		p := newPool()
		size := 1024

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := p.Get(size)
			// Simulate some work
			if len(buf) > 0 {
				buf[0] = byte(i)
			}
			p.Put(buf)
		}
	})

	// Write memory profile
	f, err := os.Create("mem.prof")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkHotPath focuses on the most critical code paths
func BenchmarkHotPath(b *testing.B) {
	b.Run("Write_Small", func(b *testing.B) {
		buf := NewBuffer(1024)
		data := []byte("hello")

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.pos = 0 // Reset without clearing
			buf.Write(data)
		}
	})

	b.Run("Write_Medium", func(b *testing.B) {
		buf := NewBuffer(8192)
		data := make([]byte, 256)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.pos = 0
			for j := 0; j < 8; j++ {
				buf.Write(data)
			}
		}
	})

	b.Run("Pool_Reuse", func(b *testing.B) {
		p := newPool()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := p.Get(256)
			buf[0] = 1
			p.Put(buf)
		}
	})
}

// BenchmarkContention tests concurrent access patterns
func BenchmarkContention(b *testing.B) {
	b.Run("Pool_Parallel", func(b *testing.B) {
		p := newPool()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := p.Get(1024)
				buf[0] = 1
				p.Put(buf)
			}
		})
	})

	b.Run("Buffer_Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			buf := NewBuffer(1024)
			data := []byte("test")
			for pb.Next() {
				buf.Reset()
				buf.Write(data)
			}
		})
	})
}

// BenchmarkComparison compares with standard library
func BenchmarkComparison(b *testing.B) {
	data := make([]byte, 256)
	rand.Read(data)

	b.Run("kbuffer", func(b *testing.B) {
		buf := NewBuffer(4096)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf.Reset()
			for j := 0; j < 10; j++ {
				buf.Write(data)
			}
			_ = buf.Bytes()
		}
	})

	b.Run("bytes.Buffer", func(b *testing.B) {
		buf := bytes.NewBuffer(make([]byte, 0, 4096))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf.Reset()
			for j := 0; j < 10; j++ {
				buf.Write(data)
			}
			_ = buf.Bytes()
		}
	})
}

// BenchmarkWorstCase tests worst-case scenarios
func BenchmarkWorstCase(b *testing.B) {
	b.Run("FragmentedWrites", func(b *testing.B) {
		buf := NewBuffer(65536)
		sizes := []int{7, 13, 29, 53, 97, 193} // Prime numbers for irregular access

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf.Reset()
			for j := 0; j < 100; j++ {
				size := sizes[j%len(sizes)]
				data := make([]byte, size)
				buf.Write(data)
			}
		}
	})

	b.Run("PoolThrashing", func(b *testing.B) {
		p := newPool()
		sizes := []int{64, 256, 1024, 4096, 16384}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Get buffers of different sizes
			bufs := make([][]byte, len(sizes))
			for j, size := range sizes {
				bufs[j] = p.Get(size)
			}
			// Return in different order
			for j := len(bufs) - 1; j >= 0; j-- {
				p.Put(bufs[j])
			}
		}
	})
}

// BenchmarkOptimizationOpportunities looks for optimization opportunities
func BenchmarkOptimizationOpportunities(b *testing.B) {
	b.Run("InlinedWrite", func(b *testing.B) {
		buf := NewBuffer(1024)
		data := []byte("test")

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Direct access to avoid function call overhead
			buf.pos = 0
			copy(buf.data[buf.pos:], data)
			buf.pos += int32(len(data))
		}
	})

	b.Run("BoundsCheckElimination", func(b *testing.B) {
		buf := NewBuffer(1024)
		data := make([]byte, 64)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf.pos = 0
			// Hint to compiler that bounds are safe
			if int(buf.pos)+len(data) <= len(buf.data) {
				copy(buf.data[buf.pos:int(buf.pos)+len(data)], data)
				buf.pos += int32(len(data))
			}
		}
	})
}
