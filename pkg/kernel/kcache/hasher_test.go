package kcache

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

// TestFNVHasher tests the FNV hasher implementation
func TestFNVHasher(t *testing.T) {
	h := newFNVHasher()

	// Test all integer types
	testCases := []struct {
		name string
		key  any
	}{
		// String types
		{"string", "test"},
		{"empty string", ""},

		// Byte slices
		{"bytes", []byte("bytes")},
		{"empty bytes", []byte{}},
		{"nil bytes", []byte(nil)},

		// Signed integers
		{"int", int(42)},
		{"int8", int8(42)},
		{"int16", int16(42)},
		{"int32", int32(42)},
		{"int64", int64(42)},

		// Unsigned integers
		{"uint", uint(42)},
		{"uint8", uint8(42)},
		{"uint16", uint16(42)},
		{"uint32", uint32(42)},
		{"uint64", uint64(42)},

		// Float types
		{"float32", float32(3.14)},
		{"float64", float64(3.14)},
		{"float32 zero", float32(0)},
		{"float64 zero", float64(0)},
		{"float32 NaN", float32(math.NaN())},
		{"float64 NaN", math.NaN()},

		// Boolean
		{"bool true", true},
		{"bool false", false},

		// Nil
		{"nil", nil},

		// Complex types (uses hashInterface)
		{"struct", struct{ ID int }{42}},
		{"pointer", new(int)},
		{"slice", []int{1, 2, 3}},
		{"nil slice", []int(nil)},
		{"array", [3]int{1, 2, 3}},
		{"map", map[string]int{"a": 1}},
		{"nil map", map[string]int(nil)},
		{"complex", complex(1, 2)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash := h.Hash(tc.key)
			// Most values should produce non-zero hash
			if tc.key != nil && hash == 0 {
				t.Errorf("Zero hash for %v", tc.key)
			}

			// Test consistency
			hash2 := h.Hash(tc.key)
			if hash != hash2 {
				t.Errorf("Inconsistent hash for %v", tc.key)
			}

			// Test equality
			if !h.Equal(tc.key, tc.key) {
				t.Errorf("Equal failed for %v", tc.key)
			}
		})
	}

	// Test Equal method with different types
	equalTests := []struct {
		name  string
		a, b  any
		equal bool
	}{
		// Same types, equal values
		{"strings equal", "test", "test", true},
		{"strings diff", "test", "other", false},
		{"bytes equal", []byte("test"), []byte("test"), true},
		{"bytes diff", []byte("test"), []byte("other"), false},
		{"int equal", 42, 42, true},
		{"int diff", 42, 43, false},
		{"int32 equal", int32(42), int32(42), true},
		{"int32 diff", int32(42), int32(43), false},
		{"int64 equal", int64(42), int64(42), true},
		{"uint equal", uint(42), uint(42), true},
		{"uint32 equal", uint32(42), uint32(42), true},
		{"uint64 equal", uint64(42), uint64(42), true},
		{"float32 equal", float32(3.14), float32(3.14), true},
		{"float32 diff", float32(3.14), float32(2.71), false},
		{"float64 equal", 3.14, 3.14, true},
		{"float32 NaN", float32(math.NaN()), float32(math.NaN()), true},
		{"float64 NaN", math.NaN(), math.NaN(), true},
		{"bool equal", true, true, true},
		{"bool diff", true, false, false},
		{"nil both", nil, nil, true},
		{"nil one", nil, 42, false},
		{"nil other", 42, nil, false},

		// Different types
		{"string vs int", "42", 42, false},
		{"int vs int32", 42, int32(42), false},
		{"float32 vs float64", float32(3.14), 3.14, false},

		// Functions
		{"same function", TestFNVHasher, TestFNVHasher, true},

		// Structs (uses DeepEqual)
		{"struct equal", struct{ X int }{1}, struct{ X int }{1}, true},
		{"struct diff", struct{ X int }{1}, struct{ X int }{2}, false},
	}

	for _, tt := range equalTests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Equal(tt.a, tt.b)
			if result != tt.equal {
				t.Errorf("Equal(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.equal)
			}
		})
	}

	// Test function equality
	fn1 := func() {}
	fn2 := func() {}
	if !h.Equal(fn1, fn1) {
		t.Error("Same function should be equal")
	}
	if h.Equal(fn1, fn2) {
		t.Error("Different functions should not be equal")
	}

	// Test nil functions
	var nilFn func()
	var nilFn2 func()
	if !h.Equal(nilFn, nilFn2) {
		t.Error("Nil functions should be equal")
	}

	// Test nil vs non-nil function
	if h.Equal(nilFn, fn1) {
		t.Error("Nil and non-nil functions should not be equal")
	}
	if h.Equal(fn1, nilFn) {
		t.Error("Non-nil and nil functions should not be equal")
	}

	// Test channel equality
	ch1 := make(chan int)
	ch2 := make(chan int)
	if !h.Equal(ch1, ch1) {
		t.Error("Same channel should be equal")
	}
	if h.Equal(ch1, ch2) {
		t.Error("Different channels should not be equal")
	}

	// Test nil channels
	var nilCh chan int
	var nilCh2 chan int
	if !h.Equal(nilCh, nilCh2) {
		t.Error("Nil channels should be equal")
	}

	// Test nil vs non-nil channel
	if h.Equal(nilCh, ch1) {
		t.Error("Nil and non-nil channels should not be equal")
	}
	if h.Equal(ch1, nilCh) {
		t.Error("Non-nil and nil channels should not be equal")
	}

	// Test mixed types (function vs channel)
	if h.Equal(fn1, ch1) {
		t.Error("Function and channel should not be equal")
	}
}

// TestXXHasher tests the XXHash hasher implementation
func TestXXHasher(t *testing.T) {
	h := &xxHasher{}

	// Test string hashing
	hash1 := h.Hash("test string")
	hash2 := h.Hash("test string")
	if hash1 != hash2 {
		t.Error("XXHasher: same string produced different hashes")
	}

	// Test different strings
	hash3 := h.Hash("different")
	if hash1 == hash3 {
		t.Error("XXHasher: different strings produced same hash")
	}

	// Test Equal method
	if !h.Equal("test", "test") {
		t.Error("XXHasher: Equal returned false for same strings")
	}
	if h.Equal("test", "different") {
		t.Error("XXHasher: Equal returned true for different strings")
	}

	// Test with various types
	testCases := []interface{}{
		42,
		3.14,
		true,
		[]byte("bytes"),
		struct{ ID int }{42},
		complex(1, 2),
		math.NaN(),
		nil,
	}

	for _, tc := range testCases {
		hash := h.Hash(tc)
		// nil might produce 0 hash, but check others
		if tc != nil {
			// Special check for NaN
			if f, ok := tc.(float64); ok && math.IsNaN(f) {
				// NaN is special, skip zero check
			} else if hash == 0 {
				t.Errorf("XXHasher: zero hash for %v", tc)
			}
		}
		if !h.Equal(tc, tc) {
			t.Errorf("XXHasher: Equal failed for %v", tc)
		}
	}

	// Test xxHashString
	strHash := xxHashString("test")
	if strHash == 0 {
		t.Error("xxHashString returned 0")
	}

	// Test xxHashBytes
	bytesHash := xxHashBytes([]byte("test"))
	if bytesHash == 0 {
		t.Error("xxHashBytes returned 0")
	}
}

// TestCityHasher tests the CityHash hasher implementation
func TestCityHasher(t *testing.T) {
	h := &cityHasher{}

	// Test string hashing
	hash1 := h.Hash("test string")
	hash2 := h.Hash("test string")
	if hash1 != hash2 {
		t.Error("CityHasher: same string produced different hashes")
	}

	// Test different strings
	hash3 := h.Hash("different")
	if hash1 == hash3 {
		t.Error("CityHasher: different strings produced same hash")
	}

	// Test Equal method
	if !h.Equal("test", "test") {
		t.Error("CityHasher: Equal returned false for same strings")
	}
	if h.Equal("test", "different") {
		t.Error("CityHasher: Equal returned true for different strings")
	}

	// Test with various types
	testCases := []interface{}{
		42,
		3.14,
		false,
		[]byte("bytes"),
		complex(1, 2),
		nil,
	}

	for _, tc := range testCases {
		hash := h.Hash(tc)
		// nil might produce 0 hash
		if tc != nil && hash == 0 {
			t.Errorf("CityHasher: zero hash for %v", tc)
		}
		if !h.Equal(tc, tc) {
			t.Errorf("CityHasher: Equal failed for %v", tc)
		}
	}

	// Test cityHashString
	strHash := cityHashString("test")
	if strHash == 0 {
		t.Error("cityHashString returned 0")
	}

	// Test cityHashBytes
	bytesHash := cityHashBytes([]byte("test"))
	if bytesHash == 0 {
		t.Error("cityHashBytes returned 0")
	}
}

// TestHashInterface tests the hashInterface function with various types
func TestHashInterface(t *testing.T) {
	h := newFNVHasher()

	// Test struct with unexported fields
	type privateStruct struct {
		Public  string
		private int // Can't access via reflection
	}

	s1 := privateStruct{Public: "test", private: 42}
	hash1 := h.Hash(s1)
	hash2 := h.Hash(s1)
	if hash1 != hash2 {
		t.Error("Struct with private fields produces inconsistent hash")
	}

	// Test nested structures
	type nested struct {
		Inner struct {
			Value int
		}
	}
	n := nested{}
	n.Inner.Value = 42
	hashNested := h.Hash(n)
	if hashNested == 0 {
		t.Error("Nested struct produced zero hash")
	}

	// Test interface containing different types
	var iface interface{} = struct{ X int }{42}
	hashIface := h.Hash(iface)
	if hashIface == 0 {
		t.Error("Interface value produced zero hash")
	}

	// Test map with multiple entries
	m := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}
	hashMap := h.Hash(m)
	if hashMap == 0 {
		t.Error("Map produced zero hash")
	}

	// Test pointer to nil
	var p *int
	hashNilPtr := h.Hash(p)
	if hashNilPtr != FNVOffsetBasis {
		t.Error("Nil pointer should produce FNVOffsetBasis")
	}

	// Test pointer to value
	val := 42
	pVal := &val
	hashPtr := h.Hash(pVal)
	if hashPtr == 0 {
		t.Error("Pointer to value produced zero hash")
	}

	// Test function and channel through interface
	fn := func() {}
	ch := make(chan int)

	hashFn := h.Hash(fn)
	hashCh := h.Hash(ch)

	if hashFn == 0 {
		t.Error("Function produced zero hash")
	}
	if hashCh == 0 {
		t.Error("Channel produced zero hash")
	}

	// Test nil function and channel
	var nilFn func()
	var nilCh chan int

	hashNilFn := h.Hash(nilFn)
	hashNilCh := h.Hash(nilCh)

	if hashNilFn != FNVOffsetBasis {
		t.Error("Nil function should produce FNVOffsetBasis")
	}
	if hashNilCh != FNVOffsetBasis {
		t.Error("Nil channel should produce FNVOffsetBasis")
	}

	// Test custom type that can't be handled (falls back to default case in hashInterface)
	type customType struct {
		unexported int
	}
	custom := customType{unexported: 42}
	hashCustom := h.Hash(custom)
	if hashCustom == 0 {
		t.Error("Custom type produced zero hash")
	}

	// Test slices with various elements
	sliceInt := []int{1, 2, 3}
	hashSlice := h.Hash(sliceInt)
	if hashSlice == 0 {
		t.Error("Int slice produced zero hash")
	}

	// Test nil slice
	var nilSlice []int
	hashNilSlice := h.Hash(nilSlice)
	if hashNilSlice != FNVOffsetBasis {
		t.Error("Nil slice should produce FNVOffsetBasis")
	}

	// Test array
	arr := [3]int{1, 2, 3}
	hashArr := h.Hash(arr)
	if hashArr == 0 {
		t.Error("Array produced zero hash")
	}

	// Test nil map
	var nilMap map[string]int
	hashNilMap := h.Hash(nilMap)
	if hashNilMap != FNVOffsetBasis {
		t.Error("Nil map should produce FNVOffsetBasis")
	}
}

// TestBytesEqual tests the bytesEqual function
func TestBytesEqual(t *testing.T) {
	tests := []struct {
		name  string
		a, b  []byte
		equal bool
	}{
		// Small slices (<=16 bytes, uses fast path)
		{"empty", []byte{}, []byte{}, true},
		{"single byte equal", []byte{1}, []byte{1}, true},
		{"single byte diff", []byte{1}, []byte{2}, false},
		{"small equal", []byte("test"), []byte("test"), true},
		{"small diff", []byte("test"), []byte("rest"), false},
		{"16 bytes equal", make([]byte, 16), make([]byte, 16), true},

		// Different lengths
		{"diff len", []byte("a"), []byte("ab"), false},

		// Large slices (>16 bytes, uses bytesEqualLarge)
		{"17 bytes equal", make([]byte, 17), make([]byte, 17), true},
		{"large equal", make([]byte, 100), make([]byte, 100), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill with same data if they should be equal
			if tt.equal && len(tt.a) > 4 {
				for i := range tt.a {
					tt.a[i] = byte(i)
					tt.b[i] = byte(i)
				}
			}

			result := bytesEqual(tt.a, tt.b)
			if result != tt.equal {
				t.Errorf("bytesEqual(%d bytes, %d bytes) = %v, want %v",
					len(tt.a), len(tt.b), result, tt.equal)
			}
		})
	}

	// Test specific small slice scenarios
	for size := 1; size <= 16; size++ {
		a := make([]byte, size)
		b := make([]byte, size)

		// Fill with same data
		for i := range a {
			a[i] = byte(i)
			b[i] = byte(i)
		}

		if !bytesEqual(a, b) {
			t.Errorf("bytesEqual failed for size %d", size)
		}

		// Make them different at various positions
		if size > 0 {
			b[0] = 255
			if bytesEqual(a, b) {
				t.Errorf("bytesEqual should return false for different bytes at start (size %d)", size)
			}
			b[0] = a[0]

			b[size-1] = 255
			if bytesEqual(a, b) {
				t.Errorf("bytesEqual should return false for different bytes at end (size %d)", size)
			}
		}
	}
}

// TestBytesEqualLarge tests the bytesEqualLarge function for large byte slices
func TestBytesEqualLarge(t *testing.T) {
	// Test various sizes that exercise different code paths
	testSizes := []int{
		0,    // Empty
		7,    // Less than 8 bytes (remainder only)
		8,    // Exactly 8 bytes
		15,   // 8 + 7 remainder
		16,   // 2 * 8 bytes
		23,   // 2 * 8 + 7 remainder
		31,   // Just below typical boundary
		32,   // Typical boundary
		63,   // 7 * 8 + 7 remainder
		64,   // 8 * 8 bytes
		100,  // Mixed
		1024, // Large
	}

	for _, size := range testSizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			a := make([]byte, size)
			b := make([]byte, size)

			// Fill with same data
			for i := range a {
				a[i] = byte(i % 256)
				b[i] = byte(i % 256)
			}

			// Test equal slices
			if !bytesEqualLarge(a, b) {
				t.Errorf("bytesEqualLarge: returned false for equal slices of size %d", size)
			}

			// Test difference at various positions
			if size > 0 {
				// Difference at start
				oldStart := b[0]
				b[0] = byte((int(a[0]) + 1) % 256)
				if bytesEqualLarge(a, b) {
					t.Errorf("bytesEqualLarge: returned true for different slices (diff at start, size %d)", size)
				}
				b[0] = oldStart

				// Difference in middle
				oldMid := b[size/2]
				b[size/2] = byte((int(a[size/2]) + 1) % 256)
				if bytesEqualLarge(a, b) {
					t.Errorf("bytesEqualLarge: returned true for different slices (diff in middle, size %d)", size)
				}
				b[size/2] = oldMid

				// Difference at end
				oldEnd := b[size-1]
				b[size-1] = byte((int(a[size-1]) + 1) % 256)
				if bytesEqualLarge(a, b) {
					t.Errorf("bytesEqualLarge: returned true for different slices (diff at end, size %d)", size)
				}
				b[size-1] = oldEnd

				// Test difference in remainder part (for sizes not divisible by 8)
				if size > 8 && size%8 != 0 {
					// Difference in the last few bytes that are handled by remainder loop
					idx := size - 3
					if idx >= 0 {
						oldRem := b[idx]
						b[idx] = byte((int(a[idx]) + 1) % 256)
						if bytesEqualLarge(a, b) {
							t.Errorf("bytesEqualLarge: returned true for different slices (diff in remainder, size %d)", size)
						}
						b[idx] = oldRem
					}
				}
			}
		})
	}

	// Note: bytesEqualLarge assumes lengths are already checked (done in bytesEqual)
	// So we don't test different lengths here as it would cause panic

	// Test nil cases
	if !bytesEqualLarge(nil, nil) {
		t.Error("bytesEqualLarge: failed for nil slices")
	}

	// Test empty slices
	if !bytesEqualLarge([]byte{}, []byte{}) {
		t.Error("bytesEqualLarge: failed for empty slices")
	}

	// Test misaligned data
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}

	// Compare subslices starting at various offsets
	for offset := 0; offset < 8; offset++ {
		end := 64 + offset
		if end <= len(data) {
			sub := data[offset:end]
			if !bytesEqualLarge(sub, sub) {
				t.Errorf("bytesEqualLarge: failed for misaligned slice at offset %d", offset)
			}
		}
	}
}

// TestIdentityHasher tests the identity hasher implementation
func TestIdentityHasher(t *testing.T) {
	h := &identityHasher{}

	// Test with uint64 keys
	var key1 uint64 = 42
	var key2 uint64 = 100

	hash1 := h.Hash(key1)
	if hash1 != key1 {
		t.Errorf("identityHasher.Hash(uint64) = %d, want %d", hash1, key1)
	}

	hash2 := h.Hash(key2)
	if hash2 != key2 {
		t.Errorf("identityHasher.Hash(uint64) = %d, want %d", hash2, key2)
	}

	// Test with non-uint64 keys (should use FNV fallback)
	strKey := "test"
	strHash := h.Hash(strKey)
	if strHash == 0 {
		t.Error("identityHasher.Hash(string) returned 0")
	}

	intKey := 42
	intHash := h.Hash(intKey)
	if intHash == 0 {
		t.Error("identityHasher.Hash(int) returned 0")
	}

	// Test Equal method
	if !h.Equal(key1, key1) {
		t.Error("identityHasher.Equal failed for same uint64")
	}

	if h.Equal(key1, key2) {
		t.Error("identityHasher.Equal returned true for different uint64")
	}

	if !h.Equal(strKey, strKey) {
		t.Error("identityHasher.Equal failed for same string")
	}

	if h.Equal(strKey, "different") {
		t.Error("identityHasher.Equal returned true for different strings")
	}
}

// TestHasherDistribution tests hash distribution quality
func TestHasherDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping distribution test in short mode")
	}

	hashers := []Hasher{
		newFNVHasher(),
		&xxHasher{},
		&cityHasher{},
	}

	names := []string{"FNV", "XXHash", "CityHash"}

	for idx, h := range hashers {
		t.Run(names[idx], func(t *testing.T) {
			const numKeys = 10000
			const numBuckets = 256

			buckets := make([]int, numBuckets)

			// Hash many keys
			for i := 0; i < numKeys; i++ {
				hash := h.Hash(i)
				bucket := int(hash % numBuckets)
				buckets[bucket]++
			}

			// Check distribution
			expectedPerBucket := numKeys / numBuckets
			maxDeviation := float64(expectedPerBucket) * 0.25 // Allow 25% deviation

			badBuckets := 0
			for bucket, count := range buckets {
				deviation := float64(count - expectedPerBucket)
				if deviation < 0 {
					deviation = -deviation
				}
				if deviation > maxDeviation {
					badBuckets++
					if badBuckets <= 5 { // Only log first 5 bad buckets
						t.Logf("Bucket %d has %d items, expected ~%d (deviation %.1f%%)",
							bucket, count, expectedPerBucket, deviation/float64(expectedPerBucket)*100)
					}
				}
			}

			// Allow up to 10% of buckets to be outside the deviation
			maxBadBuckets := numBuckets / 10
			if badBuckets > maxBadBuckets {
				t.Errorf("%s: %d buckets (%.1f%%) have poor distribution",
					names[idx], badBuckets, float64(badBuckets)/float64(numBuckets)*100)
			}
		})
	}
}

// BenchmarkHashers benchmarks different hasher implementations
func BenchmarkHashers(b *testing.B) {
	hashers := []struct {
		name   string
		hasher Hasher
	}{
		{"FNV", newFNVHasher()},
		{"XXHash", &xxHasher{}},
		{"CityHash", &cityHasher{}},
	}

	testData := []interface{}{
		"short",
		"medium length string for testing",
		"very long string that contains a lot of text to test the hasher performance on larger inputs that might be common in real world usage scenarios",
		42,
		3.14159,
		[]byte("byte slice data"),
	}

	for _, h := range hashers {
		for _, data := range testData {
			name := h.name + "/" + reflect.TypeOf(data).String()
			b.Run(name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_ = h.hasher.Hash(data)
				}
			})
		}
	}
}
