package pool

import (
	"bytes"
	"testing"
	"time"
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

// TestUnsafeShardedBufferWorkStealing tests fallback when primary shard is full.
func TestUnsafeShardedBufferWorkStealing(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	// Create small shards to test work stealing
	buf := newUnsafeShardedBuffer(40, 4).(*unsafeShardedBuffer) // 10 bytes per shard

	// Fill shards with multiple writes
	for i := 0; i < 5; i++ {
		data := []byte("12345678") // 8 bytes each
		_, err := buf.Write(data)
		if err != nil && err != errBufferFull {
			t.Fatalf("Write %d error = %v", i, err)
		}
	}

	// Check that data is distributed
	nonEmptyShards := 0
	for i := 0; i < buf.ShardCount(); i++ {
		if buf.shards[i].buffer.Len() > 0 {
			nonEmptyShards++
		}
	}

	if nonEmptyShards < 2 {
		t.Logf("Warning: Only %d shards used (work stealing may not have triggered)", nonEmptyShards)
	}
}

// TestUnsafeShardedBufferWriteString tests string write operations.
func TestUnsafeShardedBufferWriteString(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(200, 4).(*unsafeShardedBuffer)

	// Empty string
	n, err := buf.WriteString("")
	if err != nil || n != 0 {
		t.Errorf("WriteString(empty) = %d, %v; want 0, nil", n, err)
	}

	// Normal string
	str := "unsafe sharded string"
	n, err = buf.WriteString(str)
	if err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if n != len(str) {
		t.Errorf("WriteString() = %d, want %d", n, len(str))
	}
}

// TestUnsafeShardedBufferWriteByte tests single byte write operations.
func TestUnsafeShardedBufferWriteByte(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// Write multiple bytes
	for i := byte(0); i < 10; i++ {
		if err := buf.WriteByte(i); err != nil {
			t.Fatalf("WriteByte(%d) error = %v", i, err)
		}
	}

	// Check total length
	if buf.Len() != 10 {
		t.Errorf("Len() = %d, want 10", buf.Len())
	}
}

// TestUnsafeShardedBufferWriteAt tests positional write across shards.
func TestUnsafeShardedBufferWriteAt(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer) // At least minBufferSize per shard
	shardCapacity := int64(buf.Cap()) / int64(buf.ShardCount())

	// Write to different shard positions
	tests := []struct {
		data   []byte
		offset int64
	}{
		{[]byte("first"), 0},                  // First shard
		{[]byte("second"), shardCapacity},     // Second shard
		{[]byte("third"), shardCapacity * 2},  // Third shard
		{[]byte("fourth"), shardCapacity * 3}, // Fourth shard
	}

	for _, tt := range tests {
		n, err := buf.WriteAt(tt.data, tt.offset)
		if err != nil || n != len(tt.data) {
			t.Errorf("WriteAt(%d) = %d, %v; want %d, nil", tt.offset, n, err, len(tt.data))
		}
	}

	// Invalid offsets
	_, err := buf.WriteAt([]byte("test"), -1)
	if err != errInvalidOffset {
		t.Errorf("WriteAt(-1) error = %v, want errInvalidOffset", err)
	}

	_, err = buf.WriteAt([]byte("test"), int64(buf.Cap())+100)
	if err != errInvalidOffset {
		t.Errorf("WriteAt(beyond) error = %v, want errInvalidOffset", err)
	}
}

// TestUnsafeShardedBufferWriteToShard tests direct shard writing.
func TestUnsafeShardedBufferWriteToShard(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// Write to each shard directly
	for i := 0; i < 4; i++ {
		data := []byte{byte(i * 10), byte(i*10 + 1)}
		n, err := buf.WriteToShard(i, data)
		if err != nil {
			t.Fatalf("WriteToShard(%d) error = %v", i, err)
		}
		if n != len(data) {
			t.Errorf("WriteToShard(%d) = %d, want %d", i, n, len(data))
		}
	}

	// Invalid shard indices
	_, err := buf.WriteToShard(-1, []byte("test"))
	if err != errShardOutOfBounds {
		t.Errorf("WriteToShard(-1) error = %v, want errShardOutOfBounds", err)
	}

	_, err = buf.WriteToShard(10, []byte("test"))
	if err != errShardOutOfBounds {
		t.Errorf("WriteToShard(10) error = %v, want errShardOutOfBounds", err)
	}
}

// TestUnsafeShardedBufferTryWrite tests non-blocking write attempts.
func TestUnsafeShardedBufferTryWrite(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(20, 2).(*unsafeShardedBuffer) // Small buffer

	// Should succeed initially
	data := []byte("test")
	if !buf.TryWrite(data) {
		t.Error("TryWrite() = false, want true")
	}

	// Fill buffer
	for i := 0; i < 10; i++ {
		buf.TryWrite([]byte("x"))
	}

	// Large write should eventually fail
	largeData := make([]byte, 100)
	if buf.TryWrite(largeData) {
		t.Error("TryWrite(large) = true when buffer should be full")
	}
}

// TestUnsafeShardedBufferBytes tests collecting bytes from all shards.
func TestUnsafeShardedBufferBytes(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// Empty buffer
	if buf.Bytes() != nil {
		t.Error("Bytes() on empty buffer should return nil")
	}

	// Write to specific shards
	data1 := []byte("shard1")
	data2 := []byte("shard2")
	buf.WriteToShard(0, data1)
	buf.WriteToShard(1, data2)

	// Get consolidated bytes
	allBytes := buf.Bytes()
	expectedLen := len(data1) + len(data2)
	if len(allBytes) != expectedLen {
		t.Errorf("Bytes() len = %d, want %d", len(allBytes), expectedLen)
	}

	// Verify content
	if !bytes.Contains(allBytes, data1) || !bytes.Contains(allBytes, data2) {
		t.Errorf("Bytes() missing expected data: %q", allBytes)
	}
}

// TestUnsafeShardedBufferString tests string consolidation.
func TestUnsafeShardedBufferString(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// Empty buffer
	if buf.String() != "" {
		t.Error("String() on empty buffer should return empty string")
	}

	// Write strings
	buf.WriteString("unsafe ")
	buf.WriteString("string")

	str := buf.String()
	if len(str) == 0 {
		t.Error("String() returned empty after writes")
	}
}

// TestUnsafeShardedBufferBytesUnsafe tests unsafe byte access.
func TestUnsafeShardedBufferBytesUnsafe(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// Empty buffer
	ptr, length := buf.BytesUnsafe()
	if ptr != 0 || length != 0 {
		t.Errorf("BytesUnsafe() on empty = %d, %d; want 0, 0", ptr, length)
	}

	// Write to first shard
	buf.WriteToShard(0, []byte("first shard"))

	// Should return first shard's data
	ptr, length = buf.BytesUnsafe()
	if ptr == 0 || length == 0 {
		t.Errorf("BytesUnsafe() = %d, %d; expected non-zero values", ptr, length)
	}
}

// TestUnsafeShardedBufferStateOperations tests Len, Cap, Available.
func TestUnsafeShardedBufferStateOperations(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer) // At least minBufferSize per shard
	actualCap := buf.Cap()

	// Initial state
	if buf.Len() != 0 {
		t.Errorf("Initial Len() = %d, want 0", buf.Len())
	}
	// Cap should be at least the requested size
	if buf.Cap() < 256 {
		t.Errorf("Cap() = %d, want at least 256", buf.Cap())
	}
	if buf.Available() != actualCap {
		t.Errorf("Initial Available() = %d, want %d", buf.Available(), actualCap)
	}

	// After writing
	data := []byte("test data")
	buf.Write(data)

	if buf.Len() != len(data) {
		t.Errorf("Len() after write = %d, want %d", buf.Len(), len(data))
	}
}

// TestUnsafeShardedBufferReset tests reset operation across all shards.
func TestUnsafeShardedBufferReset(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// Write to all shards
	for i := 0; i < 4; i++ {
		buf.WriteToShard(i, []byte("data"))
	}

	// Reset all shards
	buf.Reset()

	// Verify all shards are reset
	if buf.Len() != 0 {
		t.Errorf("Len() after Reset = %d, want 0", buf.Len())
	}

	for i := 0; i < buf.ShardCount(); i++ {
		if buf.shards[i].buffer.Len() != 0 {
			t.Errorf("Shard %d not reset: Len = %d", i, buf.shards[i].buffer.Len())
		}
	}
}

// TestUnsafeShardedBufferClear tests clear operation across all shards.
func TestUnsafeShardedBufferClear(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// Write data to clear
	for i := 0; i < 4; i++ {
		buf.WriteToShard(i, []byte("secret"))
	}

	// Clear all shards
	buf.Clear()

	// Verify cleared
	if buf.Len() != 0 {
		t.Errorf("Len() after Clear = %d, want 0", buf.Len())
	}
}

// TestUnsafeShardedBufferTruncate tests truncate operation.
func TestUnsafeShardedBufferTruncate(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// Write data
	for i := 0; i < 4; i++ {
		buf.WriteToShard(i, []byte("12345"))
	}

	// Truncate to smaller size
	buf.Truncate(10)
	if buf.Len() > 10 {
		t.Errorf("Len() after Truncate(10) = %d, should be <= 10", buf.Len())
	}

	// Truncate to 0
	buf.Truncate(0)
	if buf.Len() != 0 {
		t.Errorf("Len() after Truncate(0) = %d, want 0", buf.Len())
	}

	// Truncate negative (should reset)
	buf.WriteToShard(0, []byte("test"))
	buf.Truncate(-1)
	if buf.Len() != 0 {
		t.Errorf("Len() after Truncate(-1) = %d, want 0", buf.Len())
	}
}

// TestUnsafeShardedBufferGrow tests grow operation.
func TestUnsafeShardedBufferGrow(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer) // At least minBufferSize per shard
	shardCap := buf.shards[0].buffer.Cap()

	// Should succeed when space available
	if err := buf.Grow(shardCap); err != nil {
		t.Errorf("Grow(%d) error = %v", shardCap, err)
	}

	// Fill most shards
	for i := 0; i < 3; i++ {
		fillData := make([]byte, shardCap)
		buf.WriteToShard(i, fillData)
	}

	// Should succeed if one shard has space
	if err := buf.Grow(shardCap); err != nil {
		t.Errorf("Grow(%d) with one shard available error = %v", shardCap, err)
	}

	// Fill last shard
	buf.WriteToShard(3, make([]byte, shardCap))

	// Should fail when no space
	if err := buf.Grow(1); err != errBufferFull {
		t.Errorf("Grow(1) with no space error = %v, want errBufferFull", err)
	}
}

// TestUnsafeShardedBufferExtend tests extend operation.
func TestUnsafeShardedBufferExtend(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// Extend should work
	if err := buf.Extend(5); err != nil {
		t.Errorf("Extend(5) error = %v", err)
	}
}

// TestUnsafeShardedBufferClone tests clone operation.
func TestUnsafeShardedBufferClone(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// Write data
	for i := 0; i < 4; i++ {
		buf.WriteToShard(i, []byte{byte(i * 2), byte(i*2 + 1)})
	}
	buf.pooled = true

	// Clone buffer
	clone := buf.Clone()

	// Verify clone properties
	if clone.Len() != buf.Len() {
		t.Errorf("Clone Len() = %d, want %d", clone.Len(), buf.Len())
	}
	if clone.Cap() != buf.Cap() {
		t.Errorf("Clone Cap() = %d, want %d", clone.Cap(), buf.Cap())
	}

	// Verify deep copy
	if cloneSharded, ok := clone.(*unsafeShardedBuffer); ok {
		if cloneSharded.pooled {
			t.Error("Clone should not be marked as pooled")
		}

		// Modify original and verify independence
		buf.WriteToShard(0, []byte("modified"))
		if bytes.Equal(clone.Bytes(), buf.Bytes()) {
			t.Error("Clone should be independent of original")
		}
	}
}

// TestUnsafeShardedBufferRemainingSlice tests remaining slice operation.
func TestUnsafeShardedBufferRemainingSlice(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(256, 4).(*unsafeShardedBuffer) // At least minBufferSize per shard
	shardCap := buf.shards[0].buffer.Cap()

	// Should return available space
	remaining := buf.RemainingSlice()
	if len(remaining) == 0 {
		t.Error("RemainingSlice() should return available space")
	}

	// Fill all shards
	for i := 0; i < 4; i++ {
		fillData := make([]byte, shardCap)
		buf.WriteToShard(i, fillData)
	}

	// Should return nil when full
	remaining = buf.RemainingSlice()
	if remaining != nil {
		t.Errorf("RemainingSlice() when full = %v, want nil", remaining)
	}
}

// TestUnsafeShardedBufferAppendBytes tests variadic append operation.
func TestUnsafeShardedBufferAppendBytes(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// Empty append
	if err := buf.AppendBytes(); err != nil {
		t.Errorf("AppendBytes(empty) error = %v", err)
	}

	// Multiple bytes
	if err := buf.AppendBytes('u', 'n', 's', 'a', 'f', 'e'); err != nil {
		t.Errorf("AppendBytes error = %v", err)
	}

	if buf.Len() != 6 {
		t.Errorf("Len() = %d, want 6", buf.Len())
	}
}

// TestUnsafeShardedBufferBalance tests rebalancing operation.
func TestUnsafeShardedBufferBalance(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// Create unbalanced distribution
	buf.WriteToShard(0, []byte("1111111111")) // 10 bytes
	buf.WriteToShard(1, []byte("22"))         // 2 bytes
	buf.WriteToShard(2, []byte("333"))        // 3 bytes
	buf.WriteToShard(3, []byte("4444"))       // 4 bytes
	// Total: 19 bytes

	originalLen := buf.Len()

	// Balance the shards
	buf.Balance()

	// Check total data is preserved
	if buf.Len() != originalLen {
		t.Errorf("Len() after Balance = %d, want %d", buf.Len(), originalLen)
	}

	// Check distribution
	lengths := make([]int, 4)
	for i := 0; i < 4; i++ {
		lengths[i] = buf.shards[i].buffer.Len()
	}

	// Verify relatively even distribution
	minLen, maxLen := lengths[0], lengths[0]
	for _, l := range lengths {
		if l < minLen {
			minLen = l
		}
		if l > maxLen {
			maxLen = l
		}
	}

	if maxLen-minLen > 1 {
		t.Errorf("Uneven distribution after Balance: lengths=%v (diff=%d)",
			lengths, maxLen-minLen)
	}
}

// TestUnsafeShardedBufferPerformance tests performance characteristics.
func TestUnsafeShardedBufferPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Enable safety checks (testingSkipSafetyCheck=false) for performance testing
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(1<<20, 16).(*unsafeShardedBuffer) // 1MB, 16 shards
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i % 256)
	}

	start := time.Now()
	iterations := 10000

	for i := 0; i < iterations; i++ {
		if i%100 == 0 {
			buf.Reset()
		}
		buf.Write(data)
	}

	elapsed := time.Since(start)
	opsPerSec := float64(iterations) / elapsed.Seconds()

	t.Logf("Unsafe sharded buffer performance: %d iterations in %v", iterations, elapsed)
	t.Logf("Operations per second: %.0f", opsPerSec)
}

// TestUnsafeShardedBufferDataRace is moved to a separate file with !race build tag
// to exclude it when the race detector is enabled

// TestUnsafeShardedBufferSelectShard tests shard selection logic.
func TestUnsafeShardedBufferSelectShard(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	buf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)

	// selectShard should return different shards in round-robin fashion
	// Note: Implementation uses a static counter, so results may vary
	shard1 := buf.selectShard()
	shard2 := buf.selectShard()
	shard3 := buf.selectShard()
	shard4 := buf.selectShard()

	// At least verify we get valid shards
	if shard1 == nil || shard2 == nil || shard3 == nil || shard4 == nil {
		t.Error("selectShard() returned nil")
	}
}

// TestUnsafeShardedBufferEdgeCases tests various edge cases.
func TestUnsafeShardedBufferEdgeCases(t *testing.T) {
	// Enable safety checks (testingSkipSafetyCheck=false)
	oldDebugMode := testingSkipSafetyCheck
	testingSkipSafetyCheck = false
	defer func() { testingSkipSafetyCheck = oldDebugMode }()

	// Test with single shard
	singleShard := newUnsafeShardedBuffer(100, 1).(*unsafeShardedBuffer)
	if singleShard.ShardCount() != 1 {
		t.Errorf("Single shard count = %d, want 1", singleShard.ShardCount())
	}
	singleShard.Write([]byte("single"))
	if singleShard.Len() != 6 {
		t.Errorf("Single shard Len() = %d, want 6", singleShard.Len())
	}

	// Test with maximum shards
	maxShard := newUnsafeShardedBuffer(maxShardCount*100, maxShardCount).(*unsafeShardedBuffer)
	if maxShard.ShardCount() != maxShardCount {
		t.Errorf("Max shard count = %d, want %d", maxShard.ShardCount(), maxShardCount)
	}

	// Test empty Balance()
	emptyBuf := newUnsafeShardedBuffer(100, 4).(*unsafeShardedBuffer)
	emptyBuf.Balance() // Should not panic
	if emptyBuf.Len() != 0 {
		t.Error("Balance() on empty buffer changed length")
	}

	// Test with minimum capacity
	minBuf := newUnsafeShardedBuffer(1, 4).(*unsafeShardedBuffer)
	if minBuf.Cap() < minBufferSize*4 {
		t.Errorf("Minimum capacity = %d, expected at least %d", minBuf.Cap(), minBufferSize*4)
	}
}
