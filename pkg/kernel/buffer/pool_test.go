package buffer_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/kitsunium/sdk/pkg/kernel/buffer"
	"github.com/stretchr/testify/assert"
)

func TestByteBufferPool(t *testing.T) {
	pool := buffer.NewBufferPool()

	// Tests a single get and put operation in the pool
	t.Run("GetPutSingle", func(t *testing.T) {
		size := 128
		buf := pool.Get(size)

		assert.Equal(t, 0, buf.Len(), "expected empty buffer after Get")
		assert.Equal(t, size, buf.Cap(), "buffer capacity mismatch")

		_, _ = buf.Write(make([]byte, size))
		assert.Equal(t, size, buf.Len(), "expected buffer length to match written data")

		pool.Put(buf)

		buf2 := pool.Get(size)
		assert.Equal(t, 0, buf2.Len(), "expected empty buffer after Put and Get")
		assert.Equal(t, size, buf2.Cap(), "buffer capacity mismatch after reuse")
		pool.Put(buf2)
	})

	// Tests concurrent access to the pool
	t.Run("ConcurrentAccess", func(t *testing.T) {
		concurrency := 10
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					size := 1 << uint(rand.Intn(12)+1) // Random size (power of 2 between 2 and 4 KB)
					buf := pool.Get(size)
					assert.Equal(t, size, buf.Cap(), "buffer capacity mismatch in concurrent access")
					pool.Put(buf)
				}
			}()
		}
		wg.Wait()
	})

	// Tests multiple fixed sizes
	t.Run("MixedSizes", func(t *testing.T) {
		sizes := []int{64, 256, 1024, 4096, 8192, 16384}
		for _, size := range sizes {
			buf := pool.Get(size)
			assert.Equal(t, size, buf.Cap(), "buffer capacity mismatch for size %d", size)
			pool.Put(buf)
		}
	})

	// Tests invalid sizes
	t.Run("InvalidSizes", func(t *testing.T) {
		t.Run("SizeBelowMinBitSize", func(t *testing.T) {
			assert.PanicsWithValue(t, "requested size must be a power of 2 within valid bounds", func() {
				pool.Get(1) // Size below 2^minBitSize (2)
			}, "expected panic for size below minBitSize")
		})

		t.Run("SizeAboveMaxBitSize", func(t *testing.T) {
			assert.PanicsWithValue(t, "requested size must be a power of 2 within valid bounds", func() {
				pool.Get(1 << (buffer.MaxBitSize + 1)) // Size above 2^maxBitSize
			}, "expected panic for size above maxBitSize")
		})

		t.Run("SizeNotPowerOfTwo", func(t *testing.T) {
			assert.PanicsWithValue(t, "requested size must be a power of 2 within valid bounds", func() {
				pool.Get(300) // Size not a power of 2
			}, "expected panic for size not a power of 2")
		})
	})

	// Tests valid sizes
	t.Run("ValidSizes", func(t *testing.T) {
		sizes := []int{1 << 1, 1 << 5, 1 << 10, 1 << 20} // Powers of 2 within bounds
		for _, size := range sizes {
			t.Run(fmt.Sprintf("SizeValid-%d", size), func(t *testing.T) {
				assert.NotPanics(t, func() {
					buf := pool.Get(size)
					assert.Equal(t, size, buf.Cap(), "buffer capacity mismatch for valid size")
					pool.Put(buf)
				}, "expected no panic for valid size %d", size)
			})
		}
	})

	// Tests putting invalid buffers into the pool
	t.Run("InvalidBuffersInPut", func(t *testing.T) {
		t.Run("BufferBelowMinBitSize", func(t *testing.T) {
			buf := buffer.NewBuffer(1) // Capacity below 2^MinBitSize
			assert.PanicsWithValue(t, "buffer capacity must be a power of 2 within valid bounds", func() {
				pool.Put(buf)
			}, "expected panic for buffer below MinBitSize")
		})

		t.Run("BufferAboveMaxBitSize", func(t *testing.T) {
			buf := buffer.NewBuffer(1 << (buffer.MaxBitSize + 1)) // Capacity above 2^MaxBitSize
			assert.PanicsWithValue(t, "buffer capacity must be a power of 2 within valid bounds", func() {
				pool.Put(buf)
			}, "expected panic for buffer above MaxBitSize")
		})

		t.Run("BufferNotPowerOfTwo", func(t *testing.T) {
			buf := buffer.NewBuffer(300) // Capacity not a power of 2
			assert.PanicsWithValue(t, "buffer capacity must be a power of 2 within valid bounds", func() {
				pool.Put(buf)
			}, "expected panic for buffer not a power of 2")
		})
	})
}

func BenchmarkByteBufferPoolGetPut(b *testing.B) {
	pool := buffer.NewBufferPool()
	size := 1 << 10 // 1 KB
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := pool.Get(size)
		pool.Put(buf)
	}
}

func BenchmarkByteBufferPoolConcurrent(b *testing.B) {
	pool := buffer.NewBufferPool()
	concurrency := 10
	var wg sync.WaitGroup

	b.ResetTimer()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/concurrency; j++ {
				// Ensure the size is between 2^MinBitSize and 2^(MinBitSize+11)
				size := 1 << uint(rand.Intn(12)+1) // Minimum is 2 (2^1)
				buf := pool.Get(size)
				pool.Put(buf)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkByteBufferPoolMixedSizes(b *testing.B) {
	pool := buffer.NewBufferPool()
	sizes := []int{64, 256, 1024, 4096, 8192, 16384}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		size := sizes[rand.Intn(len(sizes))]
		buf := pool.Get(size)
		pool.Put(buf)
	}
}
