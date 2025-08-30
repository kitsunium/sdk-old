package kbuffer

import (
	"testing"
)

// TestNewUnsafeShardedBufferCreation tests unsafe sharded buffer creation.
func TestNewUnsafeShardedBufferCreation(t *testing.T) {
	// Temporarily disable debug mode to avoid goroutine checks
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	tests := []struct {
		name       string
		capacity   int
		shards     int
		wantShards int // Expected shard count (rounded to power of 2)
	}{
		{"zero capacity and shards", 0, 0, defaultShardCount},
		{"negative capacity", -100, 4, 4},
		{"negative shards", 1024, -1, defaultShardCount},
		{"normal 4 shards", 1024, 4, 4},
		{"3 shards rounds to 4", 1024, 3, 4},
		{"7 shards rounds to 8", 1024, 7, 8},
		{"16 shards", 2048, 16, 16},
		{"excessive shards", 1024, 1000, maxShardCount},
		{"with options", 512, 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []Option{}
			if tt.name == "with options" {
				opts = append(opts, func(b Buffer) error { return nil })
			}

			buf := newUnsafeShardedBuffer(tt.capacity, tt.shards, opts...).(*unsafeShardedBuffer)

			if buf.ShardCount() != tt.wantShards {
				t.Errorf("ShardCount() = %d, want %d", buf.ShardCount(), tt.wantShards)
			}

			// Check that shardMask is correct for bitwise operations
			if buf.shardMask != uint32(tt.wantShards-1) {
				t.Errorf("shardMask = %d, want %d", buf.shardMask, tt.wantShards-1)
			}

			if buf.Len() != 0 {
				t.Errorf("Len() = %d, want 0", buf.Len())
			}

			// Verify all shards are initialized
			for i := 0; i < tt.wantShards; i++ {
				if buf.shards[i] == nil {
					t.Errorf("Shard %d is nil", i)
				}
				if buf.shards[i].buffer == nil {
					t.Errorf("Shard %d buffer is nil", i)
				}
			}
		})
	}
}

// TestUnsafeShardedBufferWrite tests write operations across shards.
func TestUnsafeShardedBufferWrite(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(1024, 4).(*unsafeShardedBuffer)

	// Test empty write
	n, err := buf.Write([]byte{})
	if err != nil || n != 0 {
		t.Errorf("Write(empty) = %d, %v; want 0, nil", n, err)
	}

	// Test normal write
	data := []byte("unsafe sharded write")
	n, err = buf.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() = %d, want %d", n, len(data))
	}

	// Verify data was written to some shard
	totalLen := buf.Len()
	if totalLen != len(data) {
		t.Errorf("Total Len() = %d, want %d", totalLen, len(data))
	}
}

// TestUnsafeShardedBufferAllShardsFull tests handling when all shards are full.
func TestUnsafeShardedBufferAllShardsFull(t *testing.T) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	// Create buffer with small shards
	buf := newUnsafeShardedBuffer(256, 2).(*unsafeShardedBuffer)

	// Fill all shards
	for i := 0; i < buf.ShardCount(); i++ {
		shardCap := buf.shards[i].buffer.Cap()
		fillData := make([]byte, shardCap)
		buf.WriteToShard(i, fillData)
	}

	// Now Write should fail
	_, err := buf.Write([]byte("should fail"))
	if err != errBufferFull {
		t.Errorf("Write on full buffer = %v, want %v", err, errBufferFull)
	}
}

// TestUnsafeShardedBufferWriteString tests string write operations.
func TestUnsafeShardedBufferWriteString(t *testing.T) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(512, 4).(*unsafeShardedBuffer)

	// Normal string write
	str := "unsafe string test"
	n, err := buf.WriteString(str)
	if err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if n != len(str) {
		t.Errorf("WriteString() = %d, want %d", n, len(str))
	}

	// Test all shards full
	t.Run("AllShardsFull", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 2).(*unsafeShardedBuffer)

		// Fill all shards
		for i := 0; i < buf.ShardCount(); i++ {
			shardCap := buf.shards[i].buffer.Cap()
			buf.WriteToShard(i, make([]byte, shardCap))
		}

		// WriteString should fail
		_, err := buf.WriteString("should fail")
		if err != errBufferFull {
			t.Errorf("WriteString on full buffer = %v, want %v", err, errBufferFull)
		}
	})
}

// TestUnsafeShardedBufferWriteByte tests single byte write operations.
func TestUnsafeShardedBufferWriteByte(t *testing.T) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)

	// Write multiple bytes
	for i := byte(0); i < 10; i++ {
		if err := buf.WriteByte(i); err != nil {
			t.Fatalf("WriteByte(%d) error = %v", i, err)
		}
	}

	// Test all shards full
	t.Run("AllShardsFull", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 2).(*unsafeShardedBuffer)

		// Fill all shards
		for i := 0; i < buf.ShardCount(); i++ {
			shardCap := buf.shards[i].buffer.Cap()
			buf.WriteToShard(i, make([]byte, shardCap))
		}

		// WriteByte should fail
		err := buf.WriteByte(0xFF)
		if err != errBufferFull {
			t.Errorf("WriteByte on full buffer = %v, want %v", err, errBufferFull)
		}
	})
}

// TestUnsafeShardedBufferBytesUnsafe tests BytesUnsafe function.
func TestUnsafeShardedBufferBytesUnsafe(t *testing.T) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	t.Run("EmptyBuffer", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		ptr, len := buf.BytesUnsafe()
		if ptr != 0 || len != 0 {
			t.Errorf("BytesUnsafe() on empty buffer = (%v, %d), want (0, 0)", ptr, len)
		}
	})

	t.Run("WithData", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		// Write to first shard
		data := []byte("test data")
		buf.WriteToShard(0, data)

		ptr, length := buf.BytesUnsafe()
		if ptr == 0 {
			t.Error("BytesUnsafe() returned nil pointer for non-empty buffer")
		}
		if length != len(data) {
			t.Errorf("BytesUnsafe() length = %d, want %d", length, len(data))
		}
	})

	t.Run("NoShards", func(t *testing.T) {
		// Edge case: buffer with no shards
		buf := &unsafeShardedBuffer{
			shardCount: 0,
			shards:     []*unsafeBufferShard{},
		}
		ptr, length := buf.BytesUnsafe()
		if ptr != 0 || length != 0 {
			t.Errorf("BytesUnsafe() with no shards = (%v, %d), want (0, 0)", ptr, length)
		}
	})
}

// TestUnsafeShardedBufferExtend tests extend operation.
func TestUnsafeShardedBufferExtend(t *testing.T) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)

	// Extend with negative value - the shard's Extend will handle this
	err := buf.Extend(-5)
	if err == nil {
		t.Errorf("Extend(-5) should return error, got nil")
	}

	// Normal extend
	err = buf.Extend(10)
	if err != nil {
		t.Errorf("Extend(10) error = %v", err)
	}

	// Extend when one shard is full
	buf.WriteToShard(0, make([]byte, buf.shards[0].buffer.Cap()))
	err = buf.Extend(10)
	if err != nil {
		t.Errorf("Extend with one full shard error = %v", err)
	}

	// Reset buffer and then fill all shards completely
	buf = newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
	for i := 0; i < buf.ShardCount(); i++ {
		// Fill each shard completely
		shardCap := buf.shards[i].buffer.Cap()
		buf.WriteToShard(i, make([]byte, shardCap))
	}

	// Now all shards are full, Extend should fail
	err = buf.Extend(1)
	if err != errBufferFull {
		t.Errorf("Extend on full buffer = %v, want %v", err, errBufferFull)
	}
}

// TestUnsafeShardedBufferEdgeCases tests capacity edge cases.
func TestUnsafeShardedBufferEdgeCases(t *testing.T) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	t.Run("ExcessiveCapacity", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(maxBufferSize+1000, 4).(*unsafeShardedBuffer)
		// Should be capped at maxBufferSize
		if buf.cap > uint32(maxBufferSize) {
			t.Errorf("Capacity = %d, should be capped at %d", buf.cap, maxBufferSize)
		}
	})

	t.Run("SmallCapacityWithManyShards", func(t *testing.T) {
		// Small capacity divided among many shards should still respect minBufferSize
		buf := newUnsafeShardedBuffer(100, 16).(*unsafeShardedBuffer)
		for i := 0; i < buf.ShardCount(); i++ {
			shardCap := buf.shards[i].buffer.Cap()
			if shardCap < minBufferSize {
				t.Errorf("Shard %d capacity = %d, should be at least %d", i, shardCap, minBufferSize)
			}
		}
	})
}

// TestUnsafeShardedBufferComprehensive tests all remaining methods.
func TestUnsafeShardedBufferComprehensive(t *testing.T) {
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	t.Run("WriteAt", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		data := []byte("test data")
		n, err := buf.WriteAt(data, 10)
		if err != nil {
			t.Errorf("WriteAt error = %v", err)
		}
		if n != len(data) {
			t.Errorf("WriteAt = %d, want %d", n, len(data))
		}

		// Test invalid offset
		_, err = buf.WriteAt(data, int64(buf.shardCount*100000))
		if err != errInvalidOffset {
			t.Errorf("WriteAt with large offset = %v, want %v", err, errInvalidOffset)
		}
	})

	t.Run("TryWrite", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		data := []byte("test")
		if !buf.TryWrite(data) {
			t.Error("TryWrite failed on empty buffer")
		}

		// Create a new buffer for the full test
		fullBuf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)

		// Fill all shards completely
		for i := 0; i < fullBuf.ShardCount(); i++ {
			shardCap := fullBuf.shards[i].buffer.Cap()
			fillData := make([]byte, shardCap)
			fullBuf.WriteToShard(i, fillData)
		}

		// Now all shards should be completely full
		if fullBuf.TryWrite(data) {
			t.Error("TryWrite succeeded on full buffer")
		}
	})

	t.Run("Bytes", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)

		// Empty buffer
		if bytes := buf.Bytes(); bytes != nil {
			t.Errorf("Bytes on empty = %v, want nil", bytes)
		}

		// With data
		buf.Write([]byte("test"))
		bytes := buf.Bytes()
		if len(bytes) != 4 {
			t.Errorf("Bytes length = %d, want 4", len(bytes))
		}
	})

	t.Run("String", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)

		// Empty buffer
		if s := buf.String(); s != "" {
			t.Errorf("String on empty = %q, want empty", s)
		}

		// With data
		buf.Write([]byte("hello"))
		if s := buf.String(); s != "hello" {
			t.Errorf("String = %q, want hello", s)
		}
	})

	t.Run("Cap", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		cap := buf.Cap()
		if cap <= 0 {
			t.Errorf("Cap = %d, want > 0", cap)
		}
	})

	t.Run("Available", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		initial := buf.Available()
		if initial <= 0 {
			t.Errorf("Available = %d, want > 0", initial)
		}

		buf.Write([]byte("test"))
		after := buf.Available()
		if after >= initial {
			t.Errorf("Available after write = %d, should be less than %d", after, initial)
		}
	})

	t.Run("Reset", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		buf.Write([]byte("test"))
		buf.Reset()
		if buf.Len() != 0 {
			t.Errorf("Len after Reset = %d, want 0", buf.Len())
		}
	})

	t.Run("Clear", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		buf.Write([]byte("test"))
		buf.Clear()
		if buf.Len() != 0 {
			t.Errorf("Len after Clear = %d, want 0", buf.Len())
		}
	})

	t.Run("Truncate", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		for i := 0; i < 4; i++ {
			buf.WriteToShard(i, []byte("12345"))
		}

		buf.Truncate(10)
		if buf.Len() > 10 {
			t.Errorf("Len after Truncate(10) = %d, want <= 10", buf.Len())
		}

		buf.Truncate(-1)
		if buf.Len() != 0 {
			t.Errorf("Len after Truncate(-1) = %d, want 0", buf.Len())
		}

		// Test with remainder
		for i := 0; i < 4; i++ {
			buf.WriteToShard(i, []byte("1234567890"))
		}
		buf.Truncate(15)
		totalLen := buf.Len()
		if totalLen != 15 {
			t.Errorf("Len after Truncate(15) = %d, want 15", totalLen)
		}
	})

	t.Run("Grow", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		err := buf.Grow(50)
		if err != nil {
			t.Errorf("Grow(50) = %v", err)
		}

		// Fill all shards
		for i := 0; i < buf.ShardCount(); i++ {
			buf.WriteToShard(i, make([]byte, buf.shards[i].buffer.Cap()))
		}

		err = buf.Grow(1)
		if err != errBufferFull {
			t.Errorf("Grow on full = %v, want %v", err, errBufferFull)
		}
	})

	t.Run("Clone", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		buf.Write([]byte("original"))

		clone := buf.Clone().(*unsafeShardedBuffer)
		if clone.Len() != buf.Len() {
			t.Errorf("Clone Len = %d, want %d", clone.Len(), buf.Len())
		}

		// Modify clone shouldn't affect original
		clone.Write([]byte(" modified"))
		if buf.Len() != 8 {
			t.Error("Original buffer was modified")
		}
	})

	t.Run("RemainingSlice", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)

		// Empty buffer
		remaining := buf.RemainingSlice()
		if remaining == nil {
			t.Error("RemainingSlice returned nil")
		}

		// After write
		buf.Write([]byte("test"))
		remaining = buf.RemainingSlice()
		if len(remaining) >= buf.Cap() {
			t.Error("RemainingSlice didn't account for written data")
		}
	})

	t.Run("AppendBytes", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)

		err := buf.AppendBytes('h', 'e', 'l', 'l', 'o')
		if err != nil {
			t.Errorf("AppendBytes error = %v", err)
		}

		if buf.Len() != 5 {
			t.Errorf("Len after AppendBytes = %d, want 5", buf.Len())
		}

		// Empty append
		err = buf.AppendBytes()
		if err != nil {
			t.Errorf("AppendBytes empty error = %v", err)
		}
	})

	t.Run("Balance", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)

		// Write unbalanced data
		buf.WriteToShard(0, []byte("lots of data here"))
		buf.WriteToShard(1, []byte("x"))

		buf.Balance()

		// Check that data is more evenly distributed
		// This is hard to test precisely, just ensure it doesn't panic
		if buf.Len() != 18 {
			t.Errorf("Balance changed total length: %d", buf.Len())
		}

		// Empty buffer balance
		emptyBuf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		emptyBuf.Balance() // Should not panic
	})

	t.Run("ShardCount", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		if buf.ShardCount() != 4 {
			t.Errorf("ShardCount = %d, want 4", buf.ShardCount())
		}
	})

	t.Run("SelectShard", func(t *testing.T) {
		buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer)
		// Just ensure selectShard doesn't panic
		shard := buf.selectShard()
		if shard == nil {
			t.Error("selectShard returned nil")
		}
	})
}
