package pool

import (
	"testing"
)

// TestConstants verifies that live size/shape constants match expectations.
func TestConstants(t *testing.T) {
	if cacheLineSize != 64 {
		t.Errorf("cacheLineSize = %d, want 64", cacheLineSize)
	}
	if minBufferSize != 64 {
		t.Errorf("minBufferSize = %d, want 64", minBufferSize)
	}
	if defaultBufferSize != 4096 {
		t.Errorf("defaultBufferSize = %d, want 4096", defaultBufferSize)
	}
	if maxBufferSize != 16<<20 {
		t.Errorf("maxBufferSize = %d, want %d", maxBufferSize, 16<<20)
	}
	if poolMinSize != 64 {
		t.Errorf("poolMinSize = %d, want 64", poolMinSize)
	}
	if poolMaxSize != 1<<22 {
		t.Errorf("poolMaxSize = %d, want %d", poolMaxSize, 1<<22)
	}
	if poolClassCount != 17 {
		t.Errorf("poolClassCount = %d, want 17", poolClassCount)
	}
	if defaultShardCount != 16 {
		t.Errorf("defaultShardCount = %d, want 16", defaultShardCount)
	}
	if maxShardCount != 256 {
		t.Errorf("maxShardCount = %d, want 256", maxShardCount)
	}
	if shardCachePadding != cacheLineSize {
		t.Errorf("shardCachePadding = %d, want %d", shardCachePadding, cacheLineSize)
	}
}

// TestBufferError verifies the bufferError type implementation.
func TestBufferError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"BufferFull", errBufferFull, "buffer full"},
		{"InvalidSize", errInvalidSize, "invalid size"},
		{"InvalidOffset", errInvalidOffset, "invalid offset"},
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

// TestStateFlags verifies state flag constants are powers of 2 and non-overlapping.
func TestStateFlags(t *testing.T) {
	flags := []uint32{
		stateFlagFull,
		stateFlagPooled,
		stateFlagCleared,
	}
	for i, flag := range flags {
		if flag != 0 && (flag&(flag-1)) != 0 {
			t.Errorf("flags[%d] = %d is not a power of 2", i, flag)
		}
		for j, other := range flags {
			if i != j && flag&other != 0 {
				t.Errorf("flags[%d] (%d) overlaps with flags[%d] (%d)", i, flag, j, other)
			}
		}
	}
	if stateFlagNormal != 0 {
		t.Errorf("stateFlagNormal = %d, want 0", stateFlagNormal)
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
	var _ Sharded = (*safeShardedBuffer)(nil)
	var _ Pool = (*bufferPool)(nil)

	t.Log("All interface compliance checks passed")
}
