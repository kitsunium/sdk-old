package value_test

import (
	"math/cmplx"
	"testing"

	"github.com/kitsunium/sdk/pkg/lib/value"
	"github.com/stretchr/testify/assert"
)

func TestValueFunctions(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		var nilPtr *string
		v := "test"
		ptr := &v

		assert.Equal(t, "", value.String(nilPtr), "expected empty string for nil pointer")
		assert.Equal(t, v, value.String(ptr), "value mismatch for string pointer")
	})

	t.Run("Int", func(t *testing.T) {
		var nilPtr *int
		v := 42
		ptr := &v

		assert.Equal(t, 0, value.Int(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Int(ptr), "value mismatch for int pointer")
	})

	t.Run("Int8", func(t *testing.T) {
		var nilPtr *int8
		v := int8(42)
		ptr := &v

		assert.Equal(t, int8(0), value.Int8(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Int8(ptr), "value mismatch for int8 pointer")
	})

	t.Run("Int16", func(t *testing.T) {
		var nilPtr *int16
		v := int16(42)
		ptr := &v

		assert.Equal(t, int16(0), value.Int16(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Int16(ptr), "value mismatch for int16 pointer")
	})

	t.Run("Int32", func(t *testing.T) {
		var nilPtr *int32
		v := int32(42)
		ptr := &v

		assert.Equal(t, int32(0), value.Int32(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Int32(ptr), "value mismatch for int32 pointer")
	})

	t.Run("Int64", func(t *testing.T) {
		var nilPtr *int64
		v := int64(42)
		ptr := &v

		assert.Equal(t, int64(0), value.Int64(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Int64(ptr), "value mismatch for int64 pointer")
	})

	t.Run("Uint", func(t *testing.T) {
		var nilPtr *uint
		v := uint(42)
		ptr := &v

		assert.Equal(t, uint(0), value.Uint(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Uint(ptr), "value mismatch for uint pointer")
	})

	t.Run("Uint8", func(t *testing.T) {
		var nilPtr *uint8
		v := uint8(42)
		ptr := &v

		assert.Equal(t, uint8(0), value.Uint8(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Uint8(ptr), "value mismatch for uint8 pointer")
	})

	t.Run("Uint16", func(t *testing.T) {
		var nilPtr *uint16
		v := uint16(42)
		ptr := &v

		assert.Equal(t, uint16(0), value.Uint16(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Uint16(ptr), "value mismatch for uint16 pointer")
	})

	t.Run("Uint32", func(t *testing.T) {
		var nilPtr *uint32
		v := uint32(42)
		ptr := &v

		assert.Equal(t, uint32(0), value.Uint32(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Uint32(ptr), "value mismatch for uint32 pointer")
	})

	t.Run("Uint64", func(t *testing.T) {
		var nilPtr *uint64
		v := uint64(42)
		ptr := &v

		assert.Equal(t, uint64(0), value.Uint64(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Uint64(ptr), "value mismatch for uint64 pointer")
	})

	t.Run("Float32", func(t *testing.T) {
		var nilPtr *float32
		v := float32(42.42)
		ptr := &v

		assert.Equal(t, float32(0), value.Float32(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Float32(ptr), "value mismatch for float32 pointer")
	})

	t.Run("Float64", func(t *testing.T) {
		var nilPtr *float64
		v := float64(42.42)
		ptr := &v

		assert.Equal(t, float64(0), value.Float64(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Float64(ptr), "value mismatch for float64 pointer")
	})

	t.Run("Bool", func(t *testing.T) {
		var nilPtr *bool
		v := true
		ptr := &v

		assert.Equal(t, false, value.Bool(nilPtr), "expected false for nil pointer")
		assert.Equal(t, v, value.Bool(ptr), "value mismatch for bool pointer")
	})

	t.Run("Byte", func(t *testing.T) {
		var nilPtr *byte
		v := byte(42)
		ptr := &v

		assert.Equal(t, byte(0), value.Byte(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Byte(ptr), "value mismatch for byte pointer")
	})

	t.Run("Rune", func(t *testing.T) {
		var nilPtr *rune
		v := rune('a')
		ptr := &v

		assert.Equal(t, rune(0), value.Rune(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Rune(ptr), "value mismatch for rune pointer")
	})

	t.Run("Complex64", func(t *testing.T) {
		var nilPtr *complex64
		v := complex64(1 + 2i)
		ptr := &v

		assert.Equal(t, complex64(0), value.Complex64(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Complex64(ptr), "value mismatch for complex64 pointer")
	})

	t.Run("Complex128", func(t *testing.T) {
		var nilPtr *complex128
		v := complex128(cmplx.Sqrt(-5 + 12i))
		ptr := &v

		assert.Equal(t, complex128(0), value.Complex128(nilPtr), "expected 0 for nil pointer")
		assert.Equal(t, v, value.Complex128(ptr), "value mismatch for complex128 pointer")
	})
}
