package pointer_test

import (
	"math/cmplx"
	"testing"

	"github.com/kistunium/sdk/pkg/lib/pointer"
	"github.com/stretchr/testify/assert"
)

func TestPointerFunctions(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		value := "test"
		ptr := pointer.String(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for string pointer")
	})

	t.Run("Int", func(t *testing.T) {
		value := 42
		ptr := pointer.Int(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for int pointer")
	})

	t.Run("Int8", func(t *testing.T) {
		value := int8(42)
		ptr := pointer.Int8(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for int8 pointer")
	})

	t.Run("Int16", func(t *testing.T) {
		value := int16(42)
		ptr := pointer.Int16(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for int16 pointer")
	})

	t.Run("Int32", func(t *testing.T) {
		value := int32(42)
		ptr := pointer.Int32(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for int32 pointer")
	})

	t.Run("Int64", func(t *testing.T) {
		value := int64(42)
		ptr := pointer.Int64(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for int64 pointer")
	})

	t.Run("Uint", func(t *testing.T) {
		value := uint(42)
		ptr := pointer.Uint(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for uint pointer")
	})

	t.Run("Uint8", func(t *testing.T) {
		value := uint8(42)
		ptr := pointer.Uint8(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for uint8 pointer")
	})

	t.Run("Uint16", func(t *testing.T) {
		value := uint16(42)
		ptr := pointer.Uint16(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for uint16 pointer")
	})

	t.Run("Uint32", func(t *testing.T) {
		value := uint32(42)
		ptr := pointer.Uint32(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for uint32 pointer")
	})

	t.Run("Uint64", func(t *testing.T) {
		value := uint64(42)
		ptr := pointer.Uint64(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for uint64 pointer")
	})

	t.Run("Float32", func(t *testing.T) {
		value := float32(42.42)
		ptr := pointer.Float32(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for float32 pointer")
	})

	t.Run("Float64", func(t *testing.T) {
		value := float64(42.42)
		ptr := pointer.Float64(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for float64 pointer")
	})

	t.Run("Bool", func(t *testing.T) {
		value := true
		ptr := pointer.Bool(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for bool pointer")
	})

	t.Run("Byte", func(t *testing.T) {
		value := byte(42)
		ptr := pointer.Byte(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for byte pointer")
	})

	t.Run("Rune", func(t *testing.T) {
		value := rune('a')
		ptr := pointer.Rune(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for rune pointer")
	})

	t.Run("Complex64", func(t *testing.T) {
		value := complex64(1 + 2i)
		ptr := pointer.Complex64(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for complex64 pointer")
	})

	t.Run("Complex128", func(t *testing.T) {
		value := complex128(cmplx.Sqrt(-5 + 12i))
		ptr := pointer.Complex128(value)
		assert.NotNil(t, ptr, "pointer should not be nil")
		assert.Equal(t, value, *ptr, "value mismatch for complex128 pointer")
	})
}
