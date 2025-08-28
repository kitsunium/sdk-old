package kbuffer

import (
	"bytes"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewSafeShardedBufferCreation tests safe sharded buffer creation.
func TestNewSafeShardedBufferCreation(t *testing.T) {
	tests := []struct {
		name       string
		capacity   int
		shards     int
		wantShards int // Expected shard count (rounded to power of 2)
		wantCap    int // Approximate expected capacity
	}{
		{"zero capacity and shards", 0, 0, defaultShardCount, defaultBufferSize},
		{"negative capacity", -100, 4, 4, minBufferSize * 4},
		{"negative shards", 1024, -1, defaultShardCount, 1024},
		{"normal 4 shards", 1024, 4, 4, 1024},
		{"3 shards rounds to 4", 1024, 3, 4, 1024},
		{"7 shards rounds to 8", 1024, 7, 8, 1024},
		{"16 shards", 2048, 16, 16, 2048},
		{"excessive shards", 1024, 1000, maxShardCount, minBufferSize * maxShardCount},
		{"excessive capacity", maxBufferSize + 1000, 4, 4, maxBufferSize},
		{"with options", 512, 2, 2, 512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []Option{}
			if tt.name == "with options" {
				opts = append(opts, func(b Buffer) error { return nil })
			}

			buf := newSafeShardedBuffer(tt.capacity, tt.shards, opts...).(*safeShardedBuffer)

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

// TestSafeShardedBufferWrite tests write operations across shards.
func TestSafeShardedBufferWrite(t *testing.T) {
	buf := newSafeShardedBuffer(1024, 4).(*safeShardedBuffer)

	// Test empty write
	n, err := buf.Write([]byte{})
	if err != nil || n != 0 {
		t.Errorf("Write(empty) = %d, %v; want 0, nil", n, err)
	}

	// Test normal write
	data := []byte("hello sharded world")
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

// TestSafeShardedBufferWorkStealing tests fallback to other shards when primary is full.
func TestSafeShardedBufferWorkStealing(t *testing.T) {
	// Create small shards to test work stealing
	// Note: each shard will have at least minBufferSize capacity
	buf := newSafeShardedBuffer(256, 4).(*safeShardedBuffer) // 64 bytes min per shard

	// Write directly to specific shards to test work stealing
	// First fill shard 0
	shardCap := buf.shards[0].buffer.Cap()
	fillData := make([]byte, shardCap)
	_, err := buf.WriteToShard(0, fillData)
	if err != nil {
		t.Fatalf("WriteToShard(0) error = %v", err)
	}

	// Now try writing through normal Write which should select shard 0 first
	// but fall back to another shard
	testData := []byte("work-stealing-test")
	_, err = buf.Write(testData)
	if err != nil {
		t.Fatalf("Write after shard 0 full error = %v", err)
	}

	// Check that data is distributed across shards
	nonEmptyShards := 0
	for i := 0; i < buf.ShardCount(); i++ {
		if buf.shards[i].buffer.Len() > 0 {
			nonEmptyShards++
		}
	}

	if nonEmptyShards < 2 {
		t.Errorf("Data not distributed across shards: %d shards used", nonEmptyShards)
	}
}

// TestSafeShardedBufferWriteString tests string write operations.
func TestSafeShardedBufferWriteString(t *testing.T) {
	buf := newSafeShardedBuffer(200, 4).(*safeShardedBuffer)

	// Empty string
	n, err := buf.WriteString("")
	if err != nil || n != 0 {
		t.Errorf("WriteString(empty) = %d, %v; want 0, nil", n, err)
	}

	// Normal string
	str := "sharded string test"
	n, err = buf.WriteString(str)
	if err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if n != len(str) {
		t.Errorf("WriteString() = %d, want %d", n, len(str))
	}

	// Verify total length
	if buf.Len() != len(str) {
		t.Errorf("Len() = %d, want %d", buf.Len(), len(str))
	}
}

// TestSafeShardedBufferWriteByte tests single byte write operations.
func TestSafeShardedBufferWriteByte(t *testing.T) {
	buf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)

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

// TestSafeShardedBufferWriteAt tests positional write across shards.
func TestSafeShardedBufferWriteAt(t *testing.T) {
	buf := newSafeShardedBuffer(256, 4).(*safeShardedBuffer) // Will have at least minBufferSize per shard
	shardCapacity := int64(buf.Cap()) / int64(buf.ShardCount())

	// Write to first shard
	data1 := []byte("first")
	n, err := buf.WriteAt(data1, 0)
	if err != nil || n != len(data1) {
		t.Errorf("WriteAt(0) = %d, %v; want %d, nil", n, err, len(data1))
	}

	// Write to second shard
	data2 := []byte("second")
	n, err = buf.WriteAt(data2, shardCapacity)
	if err != nil || n != len(data2) {
		t.Errorf("WriteAt(%d) = %d, %v; want %d, nil", shardCapacity, n, err, len(data2))
	}

	// Write to third shard
	data3 := []byte("third")
	n, err = buf.WriteAt(data3, shardCapacity*2)
	if err != nil || n != len(data3) {
		t.Errorf("WriteAt(%d) = %d, %v; want %d, nil", shardCapacity*2, n, err, len(data3))
	}

	// Invalid offset (beyond capacity)
	_, err = buf.WriteAt([]byte("test"), int64(buf.Cap())+100)
	if err != errInvalidOffset {
		t.Errorf("WriteAt(beyond) error = %v, want errInvalidOffset", err)
	}

	// Invalid offset (negative)
	_, err = buf.WriteAt([]byte("test"), -1)
	if err != errInvalidOffset {
		t.Errorf("WriteAt(-1) error = %v, want errInvalidOffset", err)
	}
}

// TestSafeShardedBufferWriteToShard tests direct shard writing.
func TestSafeShardedBufferWriteToShard(t *testing.T) {
	buf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)

	// Write to specific shards
	for i := 0; i < 4; i++ {
		data := []byte{byte(i), byte(i)}
		n, err := buf.WriteToShard(i, data)
		if err != nil {
			t.Fatalf("WriteToShard(%d) error = %v", i, err)
		}
		if n != len(data) {
			t.Errorf("WriteToShard(%d) = %d, want %d", i, n, len(data))
		}
	}

	// Invalid shard index (negative)
	_, err := buf.WriteToShard(-1, []byte("test"))
	if err != errShardOutOfBounds {
		t.Errorf("WriteToShard(-1) error = %v, want errShardOutOfBounds", err)
	}

	// Invalid shard index (too large)
	_, err = buf.WriteToShard(10, []byte("test"))
	if err != errShardOutOfBounds {
		t.Errorf("WriteToShard(10) error = %v, want errShardOutOfBounds", err)
	}
}

// TestSafeShardedBufferTryWrite tests non-blocking write attempts.
func TestSafeShardedBufferTryWrite(t *testing.T) {
	buf := newSafeShardedBuffer(20, 2).(*safeShardedBuffer) // Small buffer

	// Should succeed initially
	data := []byte("test")
	if !buf.TryWrite(data) {
		t.Error("TryWrite() = false, want true")
	}

	// Fill buffer
	for i := 0; i < 10; i++ {
		buf.TryWrite([]byte("x"))
	}

	// Eventually should fail when full
	largeDat := make([]byte, 100)
	if buf.TryWrite(largeDat) {
		t.Error("TryWrite(large) = true when buffer should be full")
	}
}

// TestSafeShardedBufferBytes tests collecting bytes from all shards.
func TestSafeShardedBufferBytes(t *testing.T) {
	buf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)

	// Empty buffer
	if buf.Bytes() != nil {
		t.Error("Bytes() on empty buffer should return nil")
	}

	// Write to multiple shards
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

	// Verify content (order depends on shard order)
	combined := string(allBytes)
	if !bytes.Contains(allBytes, data1) || !bytes.Contains(allBytes, data2) {
		t.Errorf("Bytes() missing expected data: %q", combined)
	}
}

// TestSafeShardedBufferString tests string consolidation.
func TestSafeShardedBufferString(t *testing.T) {
	buf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)

	// Empty buffer
	if buf.String() != "" {
		t.Error("String() on empty buffer should return empty string")
	}

	// Write strings to shards
	buf.WriteString("hello ")
	buf.WriteString("world")

	str := buf.String()
	if len(str) != 11 { // "hello " + "world"
		t.Errorf("String() len = %d, want 11", len(str))
	}
}

// TestSafeShardedBufferBytesUnsafe tests unsafe byte access.
func TestSafeShardedBufferBytesUnsafe(t *testing.T) {
	buf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)

	// Empty buffer
	ptr, length := buf.BytesUnsafe()
	if ptr != 0 || length != 0 {
		t.Errorf("BytesUnsafe() on empty = %d, %d; want 0, 0", ptr, length)
	}

	// Write to first shard
	buf.WriteToShard(0, []byte("first shard data"))

	// Should return first shard's data
	ptr, length = buf.BytesUnsafe()
	if ptr == 0 || length == 0 {
		t.Errorf("BytesUnsafe() = %d, %d; expected non-zero values", ptr, length)
	}
}

// TestSafeShardedBufferStateOperations tests Len, Cap, Available.
func TestSafeShardedBufferStateOperations(t *testing.T) {
	buf := newSafeShardedBuffer(256, 4).(*safeShardedBuffer) // Will have at least minBufferSize per shard
	actualCap := buf.Cap()

	// Initial state
	if buf.Len() != 0 {
		t.Errorf("Initial Len() = %d, want 0", buf.Len())
	}
	// Cap should be at least the requested size
	if buf.Cap() < 256 {
		t.Errorf("Cap() = %d, want at least 256", buf.Cap())
	}
	initialAvailable := buf.Available()
	if initialAvailable != actualCap {
		t.Errorf("Initial Available() = %d, want %d", initialAvailable, actualCap)
	}

	// After writing
	data := []byte("test data")
	buf.Write(data)

	if buf.Len() != len(data) {
		t.Errorf("Len() after write = %d, want %d", buf.Len(), len(data))
	}
	if buf.Cap() != actualCap {
		t.Errorf("Cap() after write = %d, want %d", buf.Cap(), actualCap)
	}
	if buf.Available() >= initialAvailable {
		t.Errorf("Available() after write = %d, should be less than %d",
			buf.Available(), initialAvailable)
	}
}

// TestSafeShardedBufferReset tests reset operation across all shards.
func TestSafeShardedBufferReset(t *testing.T) {
	buf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)

	// Write to all shards
	for i := 0; i < 4; i++ {
		buf.WriteToShard(i, []byte("data"))
	}

	// Verify data exists
	if buf.Len() == 0 {
		t.Error("Buffer should have data before reset")
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

// TestSafeShardedBufferClear tests clear operation across all shards.
func TestSafeShardedBufferClear(t *testing.T) {
	buf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)

	// Write sensitive data to all shards
	for i := 0; i < 4; i++ {
		buf.WriteToShard(i, []byte("sensitive"))
	}

	// Clear all shards
	buf.Clear()

	// Verify all shards are cleared
	if buf.Len() != 0 {
		t.Errorf("Len() after Clear = %d, want 0", buf.Len())
	}

	for i := 0; i < buf.ShardCount(); i++ {
		if buf.shards[i].buffer.Len() != 0 {
			t.Errorf("Shard %d not cleared: Len = %d", i, buf.shards[i].buffer.Len())
		}
	}
}

// TestSafeShardedBufferTruncate tests truncate operation.
func TestSafeShardedBufferTruncate(t *testing.T) {
	buf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)

	// Write data to shards
	for i := 0; i < 4; i++ {
		buf.WriteToShard(i, []byte("12345"))
	}

	originalLen := buf.Len()

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

	// Truncate to negative (should reset)
	buf.WriteToShard(0, []byte("test"))
	buf.Truncate(-5)
	if buf.Len() != 0 {
		t.Errorf("Len() after Truncate(-5) = %d, want 0", buf.Len())
	}

	// Test remainder distribution
	buf.Reset()
	for i := 0; i < 4; i++ {
		buf.WriteToShard(i, []byte("1234567890")) // 10 bytes each
	}
	buf.Truncate(15) // Should distribute total of 15 bytes across shards

	totalLen := 0
	maxShardSize := 0
	for i := 0; i < 4; i++ {
		shardLen := buf.shards[i].buffer.Len()
		totalLen += shardLen
		if shardLen > maxShardSize {
			maxShardSize = shardLen
		}
	}

	// Verify total length equals 15
	if totalLen != 15 {
		t.Errorf("Total length = %d, expected 15", totalLen)
	}

	// Verify no shard is unreasonably large (should be roughly 15/4 = ~4 bytes each)
	expectedMaxPerShard := 15/4 + 1 // Allow for rounding
	if maxShardSize > expectedMaxPerShard {
		t.Errorf("Max shard size = %d, expected <= %d", maxShardSize, expectedMaxPerShard)
	}

	_ = originalLen // Avoid unused variable warning
}

// TestSafeShardedBufferGrow tests grow operation.
func TestSafeShardedBufferGrow(t *testing.T) {
	buf := newSafeShardedBuffer(256, 4).(*safeShardedBuffer) // At least minBufferSize per shard
	shardCap := buf.shards[0].buffer.Cap()

	// Should succeed when space available
	if err := buf.Grow(shardCap); err != nil {
		t.Errorf("Grow(%d) error = %v", shardCap, err)
	}

	// Fill most shards
	for i := 0; i < 3; i++ {
		fillData := make([]byte, shardCap)
		buf.WriteToShard(i, fillData) // Fill 3 shards completely
	}

	// Should still succeed if one shard has space
	if err := buf.Grow(shardCap); err != nil {
		t.Errorf("Grow(%d) with one shard available error = %v", shardCap, err)
	}

	// Fill last shard
	buf.WriteToShard(3, make([]byte, shardCap))

	// Should fail when no shard has enough space
	if err := buf.Grow(1); err != errBufferFull {
		t.Errorf("Grow(1) with no space error = %v, want errBufferFull", err)
	}
}

// TestSafeShardedBufferExtend tests extend operation.
func TestSafeShardedBufferExtend(t *testing.T) {
	buf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)

	// Get initial state of affinity shard
	affinityShard := buf.selectShard()
	initialLen := affinityShard.buffer.Len()
	initialCap := affinityShard.buffer.Cap()

	// Extend should work on affinity shard
	extendSize := 5
	if err := buf.Extend(extendSize); err != nil {
		t.Errorf("Extend(%d) error = %v", extendSize, err)
	}

	// Verify that the affinity shard's length increased
	newLen := affinityShard.buffer.Len()
	if newLen != initialLen+extendSize {
		t.Errorf("Shard length after Extend = %d, want %d",
			newLen, initialLen+extendSize)
	}

	// Verify available space decreased accordingly
	available := affinityShard.buffer.Available()
	expectedAvailable := initialCap - newLen
	if available != expectedAvailable {
		t.Errorf("Available space = %d, want %d", available, expectedAvailable)
	}

	// Write directly to the affinity shard (not through WriteAt which uses global sharding)
	testData := []byte("test")
	n, err := affinityShard.buffer.Write(testData)
	if err != nil {
		t.Errorf("Write to affinity shard error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write wrote %d bytes, want %d", n, len(testData))
	}

	// Verify the data was written correctly
	finalLen := affinityShard.buffer.Len()
	expectedFinalLen := newLen + len(testData)
	if finalLen != expectedFinalLen {
		t.Errorf("Final shard length = %d, expected %d", finalLen, expectedFinalLen)
	}

	// Verify the buffer contains the test data
	shardData := affinityShard.buffer.Bytes()
	if len(shardData) < newLen+len(testData) {
		t.Errorf("Shard data too short: %d bytes, expected at least %d",
			len(shardData), newLen+len(testData))
	} else {
		// Data should be at the position after the extended region
		writtenData := shardData[newLen : newLen+len(testData)]
		if !bytes.Equal(writtenData, testData) {
			t.Errorf("Written data mismatch: got %v, want %v", writtenData, testData)
		}
	}
}

// TestSafeShardedBufferClone tests clone operation.
func TestSafeShardedBufferClone(t *testing.T) {
	buf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)

	// Write data to shards
	for i := 0; i < 4; i++ {
		buf.WriteToShard(i, []byte{byte(i), byte(i)})
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

	// Verify it's a deep copy
	if cloneSharded, ok := clone.(*safeShardedBuffer); ok {
		if cloneSharded.pooled {
			t.Error("Clone should not be marked as pooled")
		}

		// Verify shard independence
		buf.WriteToShard(0, []byte("new"))
		if bytes.Equal(clone.Bytes(), buf.Bytes()) {
			t.Error("Clone should be independent of original")
		}
	} else {
		t.Error("Clone is not a safeShardedBuffer")
	}
}

// TestSafeShardedBufferRemainingSlice tests remaining slice operation.
func TestSafeShardedBufferRemainingSlice(t *testing.T) {
	buf := newSafeShardedBuffer(256, 4).(*safeShardedBuffer) // At least minBufferSize per shard
	shardCap := buf.shards[0].buffer.Cap()

	// Should return slice from first available shard
	remaining := buf.RemainingSlice()
	if remaining == nil || len(remaining) == 0 {
		t.Error("RemainingSlice() should return available space")
	}

	// Fill all shards
	for i := 0; i < 4; i++ {
		fillData := make([]byte, shardCap)
		buf.WriteToShard(i, fillData)
	}

	// Should return nil when all full
	remaining = buf.RemainingSlice()
	if remaining != nil {
		t.Errorf("RemainingSlice() when full = %v, want nil", remaining)
	}
}

// TestSafeShardedBufferAppendBytes tests variadic append operation.
func TestSafeShardedBufferAppendBytes(t *testing.T) {
	buf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)

	// Empty append
	if err := buf.AppendBytes(); err != nil {
		t.Errorf("AppendBytes(empty) error = %v", err)
	}

	// Single byte
	if err := buf.AppendBytes('a'); err != nil {
		t.Errorf("AppendBytes('a') error = %v", err)
	}

	// Multiple bytes
	if err := buf.AppendBytes('b', 'c', 'd', 'e', 'f'); err != nil {
		t.Errorf("AppendBytes multiple error = %v", err)
	}

	if buf.Len() != 6 {
		t.Errorf("Len() = %d, want 6", buf.Len())
	}
}

// TestSafeShardedBufferBalance tests rebalancing operation.
func TestSafeShardedBufferBalance(t *testing.T) {
	buf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)

	// Create unbalanced distribution
	buf.WriteToShard(0, []byte("111111111111")) // 12 bytes in shard 0
	buf.WriteToShard(1, []byte("22"))           // 2 bytes in shard 1
	buf.WriteToShard(2, []byte(""))             // 0 bytes in shard 2
	buf.WriteToShard(3, []byte("3333"))         // 4 bytes in shard 3
	// Total: 18 bytes

	originalData := buf.Bytes()
	originalLen := buf.Len()

	// Balance the shards
	buf.Balance()

	// Check total data is preserved
	if buf.Len() != originalLen {
		t.Errorf("Len() after Balance = %d, want %d", buf.Len(), originalLen)
	}

	balancedData := buf.Bytes()
	if len(balancedData) != len(originalData) {
		t.Errorf("Data length changed after Balance: %d -> %d",
			len(originalData), len(balancedData))
	}

	// Check distribution is more even
	// Expected: 18 bytes / 4 shards = 4.5 bytes per shard
	// So we expect distribution like: 5, 5, 4, 4
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

// TestSafeShardedBufferConcurrentAccess tests thread-safe concurrent operations.
func TestSafeShardedBufferConcurrentAccess(t *testing.T) {
	buf := newSafeShardedBuffer(10000, 8) // Large buffer with 8 shards
	const goroutines = 50
	const operations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent writes from multiple goroutines
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			data := []byte{byte(id)}

			for j := 0; j < operations; j++ {
				// Mix of operations
				switch j % 5 {
				case 0:
					buf.Write(data)
				case 1:
					buf.WriteString("str")
				case 2:
					buf.WriteByte(byte(j))
				case 3:
					_ = buf.Len()
				case 4:
					_ = buf.Available()
				}

				if j%20 == 0 {
					runtime.Gosched() // Encourage context switching
				}
			}
		}(i)
	}

	wg.Wait()

	// If we get here without panic/deadlock, concurrent access is safe
	t.Logf("Concurrent test completed: Len=%d, Cap=%d, Shards=%d",
		buf.Len(), buf.Cap(), buf.ShardCount())
}

// TestSafeShardedBufferStress performs stress testing.
func TestSafeShardedBufferStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	buf := newSafeShardedBuffer(1<<20, 16) // 1MB buffer with 16 shards
	const goroutines = 100
	const duration = 2 * time.Second

	stop := make(chan struct{})
	var totalOps atomic.Uint64

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Start workers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			data := make([]byte, 100)
			for j := range data {
				data[j] = byte(id + j)
			}

			for {
				select {
				case <-stop:
					return
				default:
					// Perform random operations
					op := totalOps.Add(1) % 10
					switch op {
					case 0, 1, 2:
						buf.Write(data[:10])
					case 3:
						buf.WriteString("stress")
					case 4:
						buf.WriteByte(byte(op))
					case 5:
						buf.TryWrite(data[:5])
					case 6:
						_ = buf.Len()
					case 7:
						_ = buf.Bytes()
					case 8:
						if totalOps.Load()%100 == 0 {
							buf.Reset()
						}
					case 9:
						buf.WriteToShard(id%buf.ShardCount(), data[:2])
					}
				}
			}
		}(i)
	}

	// Let it run
	time.Sleep(duration)
	close(stop)
	wg.Wait()

	ops := totalOps.Load()
	opsPerSec := float64(ops) / duration.Seconds()

	t.Logf("Stress test: %d operations in %v (%.0f ops/sec)",
		ops, duration, opsPerSec)
	t.Logf("Final state: Len=%d, Cap=%d", buf.Len(), buf.Cap())
}

// TestGetGoroutineID tests the goroutine ID extraction.
func TestGetGoroutineID(t *testing.T) {
	// Get ID from current goroutine
	id1 := getGoroutineID()
	if id1 == 0 {
		t.Error("getGoroutineID() returned 0")
	}

	// Get ID from same goroutine again
	id2 := getGoroutineID()
	if id1 != id2 {
		t.Errorf("getGoroutineID() not consistent: %d != %d", id1, id2)
	}

	// Get ID from different goroutine
	done := make(chan uint32)
	go func() {
		done <- getGoroutineID()
	}()

	id3 := <-done
	if id3 == id1 {
		t.Errorf("Different goroutines returned same ID: %d", id3)
	}
}

// TestNextPowerOf2 tests the power of 2 rounding function.
func TestNextPowerOf2(t *testing.T) {
	tests := []struct {
		input    uint32
		expected uint32
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{7, 8},
		{8, 8},
		{9, 16},
		{15, 16},
		{16, 16},
		{17, 32},
		{1000, 1024},
	}

	for _, tt := range tests {
		result := nextPowerOf2(tt.input)
		if result != tt.expected {
			t.Errorf("nextPowerOf2(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

// TestSafeShardedBufferEdgeCases tests various edge cases.
func TestSafeShardedBufferEdgeCases(t *testing.T) {
	// Test with single shard (degenerates to regular buffer)
	singleShard := newSafeShardedBuffer(100, 1).(*safeShardedBuffer)
	if singleShard.ShardCount() != 1 {
		t.Errorf("Single shard count = %d, want 1", singleShard.ShardCount())
	}
	singleShard.Write([]byte("single"))
	if singleShard.Len() != 6 {
		t.Errorf("Single shard Len() = %d, want 6", singleShard.Len())
	}

	// Test with maximum shards
	maxShard := newSafeShardedBuffer(maxShardCount*100, maxShardCount).(*safeShardedBuffer)
	if maxShard.ShardCount() != maxShardCount {
		t.Errorf("Max shard count = %d, want %d", maxShard.ShardCount(), maxShardCount)
	}

	// Test empty Balance() operation
	emptyBuf := newSafeShardedBuffer(100, 4).(*safeShardedBuffer)
	emptyBuf.Balance() // Should not panic
	if emptyBuf.Len() != 0 {
		t.Error("Balance() on empty buffer changed length")
	}
}
