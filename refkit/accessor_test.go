// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/lemon4ksan/foundation/refkit"
	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestAsInt64(t *testing.T) {
	t.Parallel()

	// Signed integer kinds
	val, ok := refkit.AsInt64(reflect.ValueOf(int(42)))
	assert.True(t, ok)
	assert.Equal(t, int64(42), val)

	val, ok = refkit.AsInt64(reflect.ValueOf(int8(-8)))
	assert.True(t, ok)
	assert.Equal(t, int64(-8), val)

	val, ok = refkit.AsInt64(reflect.ValueOf(int16(1600)))
	assert.True(t, ok)
	assert.Equal(t, int64(1600), val)

	val, ok = refkit.AsInt64(reflect.ValueOf(int32(-32000)))
	assert.True(t, ok)
	assert.Equal(t, int64(-32000), val)

	val, ok = refkit.AsInt64(reflect.ValueOf(int64(math.MaxInt64)))
	assert.True(t, ok)
	assert.Equal(t, int64(math.MaxInt64), val)

	val, ok = refkit.AsInt64(reflect.ValueOf(int64(math.MinInt64)))
	assert.True(t, ok)
	assert.Equal(t, int64(math.MinInt64), val)

	// Unsigned integer kinds <= MaxInt64
	val, ok = refkit.AsInt64(reflect.ValueOf(uint(100)))
	assert.True(t, ok)
	assert.Equal(t, int64(100), val)

	val, ok = refkit.AsInt64(reflect.ValueOf(uint8(255)))
	assert.True(t, ok)
	assert.Equal(t, int64(255), val)

	val, ok = refkit.AsInt64(reflect.ValueOf(uint16(65535)))
	assert.True(t, ok)
	assert.Equal(t, int64(65535), val)

	val, ok = refkit.AsInt64(reflect.ValueOf(uint32(100000)))
	assert.True(t, ok)
	assert.Equal(t, int64(100000), val)

	val, ok = refkit.AsInt64(reflect.ValueOf(uint64(math.MaxInt64)))
	assert.True(t, ok)
	assert.Equal(t, int64(math.MaxInt64), val)

	val, ok = refkit.AsInt64(reflect.ValueOf(uintptr(1024)))
	assert.True(t, ok)
	assert.Equal(t, int64(1024), val)

	// Unsigned integer overflow > MaxInt64
	_, ok = refkit.AsInt64(reflect.ValueOf(uint64(math.MaxUint64)))
	assert.False(t, ok)

	_, ok = refkit.AsInt64(reflect.ValueOf(uint64(math.MaxInt64) + 1))
	assert.False(t, ok)

	// Incompatible kinds
	_, ok = refkit.AsInt64(reflect.ValueOf(3.14))
	assert.False(t, ok)

	_, ok = refkit.AsInt64(reflect.ValueOf("42"))
	assert.False(t, ok)

	_, ok = refkit.AsInt64(reflect.ValueOf(true))
	assert.False(t, ok)

	_, ok = refkit.AsInt64(reflect.Value{})
	assert.False(t, ok)
}

func TestAsUint64(t *testing.T) {
	t.Parallel()

	// Unsigned integer kinds
	val, ok := refkit.AsUint64(reflect.ValueOf(uint(42)))
	assert.True(t, ok)
	assert.Equal(t, uint64(42), val)

	val, ok = refkit.AsUint64(reflect.ValueOf(uint8(200)))
	assert.True(t, ok)
	assert.Equal(t, uint64(200), val)

	val, ok = refkit.AsUint64(reflect.ValueOf(uint16(50000)))
	assert.True(t, ok)
	assert.Equal(t, uint64(50000), val)

	val, ok = refkit.AsUint64(reflect.ValueOf(uint32(1000000)))
	assert.True(t, ok)
	assert.Equal(t, uint64(1000000), val)

	val, ok = refkit.AsUint64(reflect.ValueOf(uint64(math.MaxUint64)))
	assert.True(t, ok)
	assert.Equal(t, uint64(math.MaxUint64), val)

	val, ok = refkit.AsUint64(reflect.ValueOf(uintptr(2048)))
	assert.True(t, ok)
	assert.Equal(t, uint64(2048), val)

	// Signed integer kinds >= 0
	val, ok = refkit.AsUint64(reflect.ValueOf(int(42)))
	assert.True(t, ok)
	assert.Equal(t, uint64(42), val)

	val, ok = refkit.AsUint64(reflect.ValueOf(int64(math.MaxInt64)))
	assert.True(t, ok)
	assert.Equal(t, uint64(math.MaxInt64), val)

	// Signed integer kinds < 0
	_, ok = refkit.AsUint64(reflect.ValueOf(int(-1)))
	assert.False(t, ok)

	_, ok = refkit.AsUint64(reflect.ValueOf(int8(-128)))
	assert.False(t, ok)

	_, ok = refkit.AsUint64(reflect.ValueOf(int64(math.MinInt64)))
	assert.False(t, ok)

	// Incompatible kinds
	_, ok = refkit.AsUint64(reflect.ValueOf("123"))
	assert.False(t, ok)

	_, ok = refkit.AsUint64(reflect.ValueOf(false))
	assert.False(t, ok)

	_, ok = refkit.AsUint64(reflect.Value{})
	assert.False(t, ok)
}

func TestAsFloat64(t *testing.T) {
	t.Parallel()

	val, ok := refkit.AsFloat64(reflect.ValueOf(float32(3.14)))
	assert.True(t, ok)
	assert.InDelta(t, 3.14, val, 0.001)

	val, ok = refkit.AsFloat64(reflect.ValueOf(2.718281828459045))
	assert.True(t, ok)
	assert.Equal(t, 2.718281828459045, val)

	// Incompatible kinds
	_, ok = refkit.AsFloat64(reflect.ValueOf(123))
	assert.False(t, ok)

	_, ok = refkit.AsFloat64(reflect.ValueOf("3.14"))
	assert.False(t, ok)

	_, ok = refkit.AsFloat64(reflect.ValueOf(true))
	assert.False(t, ok)

	_, ok = refkit.AsFloat64(reflect.Value{})
	assert.False(t, ok)
}

func TestAsBool(t *testing.T) {
	t.Parallel()

	val, ok := refkit.AsBool(reflect.ValueOf(true))
	assert.True(t, ok)
	assert.True(t, val)

	val, ok = refkit.AsBool(reflect.ValueOf(false))
	assert.True(t, ok)
	assert.False(t, val)

	// Incompatible kinds
	_, ok = refkit.AsBool(reflect.ValueOf(1))
	assert.False(t, ok)

	_, ok = refkit.AsBool(reflect.ValueOf(0))
	assert.False(t, ok)

	_, ok = refkit.AsBool(reflect.ValueOf("true"))
	assert.False(t, ok)

	_, ok = refkit.AsBool(reflect.Value{})
	assert.False(t, ok)
}

func TestAsBytes(t *testing.T) {
	t.Parallel()

	// Byte slice
	raw := []byte("hello systems")
	val, ok := refkit.AsBytes(reflect.ValueOf(raw))
	assert.True(t, ok)
	assert.Equal(t, raw, val)

	// Nil byte slice
	var nilBytes []byte
	valNil, okNil := refkit.AsBytes(reflect.ValueOf(nilBytes))
	assert.True(t, okNil)
	assert.Nil(t, valNil)

	// Addressable byte array
	arr := [4]byte{'a', 'b', 'c', 'd'}
	arrVal := reflect.ValueOf(&arr).Elem()
	valArr, okArr := refkit.AsBytes(arrVal)
	assert.True(t, okArr)
	assert.Equal(t, []byte("abcd"), valArr)

	// Non-addressable byte array
	valNonAddr, okNonAddr := refkit.AsBytes(reflect.ValueOf([4]byte{'a', 'b', 'c', 'd'}))
	assert.False(t, okNonAddr)
	assert.Nil(t, valNonAddr)

	// Non-byte slice
	intSlice := []int{1, 2, 3}
	_, ok = refkit.AsBytes(reflect.ValueOf(intSlice))
	assert.False(t, ok)

	// Incompatible kinds
	_, ok = refkit.AsBytes(reflect.ValueOf("string"))
	assert.False(t, ok)

	_, ok = refkit.AsBytes(reflect.ValueOf(123))
	assert.False(t, ok)

	_, ok = refkit.AsBytes(reflect.Value{})
	assert.False(t, ok)
}

func TestAsString(t *testing.T) {
	t.Parallel()

	val, ok := refkit.AsString(reflect.ValueOf("hello"))
	assert.True(t, ok)
	assert.Equal(t, "hello", val)

	val, ok = refkit.AsString(reflect.ValueOf(""))
	assert.True(t, ok)
	assert.Equal(t, "", val)

	// Incompatible kinds
	_, ok = refkit.AsString(reflect.ValueOf([]byte("hello")))
	assert.False(t, ok)

	_, ok = refkit.AsString(reflect.ValueOf(123))
	assert.False(t, ok)

	_, ok = refkit.AsString(reflect.ValueOf(true))
	assert.False(t, ok)

	_, ok = refkit.AsString(reflect.Value{})
	assert.False(t, ok)
}

func TestAccessors_ZeroAlloc(t *testing.T) {
	intVal := reflect.ValueOf(int64(123456789))
	uintVal := reflect.ValueOf(uint64(987654321))
	floatVal := reflect.ValueOf(float64(3.14159265))
	boolVal := reflect.ValueOf(true)
	bytesVal := reflect.ValueOf([]byte("zero-alloc payload"))
	strVal := reflect.ValueOf("zero-alloc string")

	var iSink int64
	var uSink uint64
	var fSink float64
	var bSink bool
	var bySink []byte
	var sSink string

	allocsInt := testing.AllocsPerRun(1000, func() {
		v, ok := refkit.AsInt64(intVal)
		if !ok {
			t.Fatal("expected ok")
		}
		iSink = v
	})
	assert.Equal(t, float64(0), allocsInt, "AsInt64 must achieve 0 allocs/op")

	allocsUint := testing.AllocsPerRun(1000, func() {
		v, ok := refkit.AsUint64(uintVal)
		if !ok {
			t.Fatal("expected ok")
		}
		uSink = v
	})
	assert.Equal(t, float64(0), allocsUint, "AsUint64 must achieve 0 allocs/op")

	allocsFloat := testing.AllocsPerRun(1000, func() {
		v, ok := refkit.AsFloat64(floatVal)
		if !ok {
			t.Fatal("expected ok")
		}
		fSink = v
	})
	assert.Equal(t, float64(0), allocsFloat, "AsFloat64 must achieve 0 allocs/op")

	allocsBool := testing.AllocsPerRun(1000, func() {
		v, ok := refkit.AsBool(boolVal)
		if !ok {
			t.Fatal("expected ok")
		}
		bSink = v
	})
	assert.Equal(t, float64(0), allocsBool, "AsBool must achieve 0 allocs/op")

	allocsBytes := testing.AllocsPerRun(1000, func() {
		v, ok := refkit.AsBytes(bytesVal)
		if !ok {
			t.Fatal("expected ok")
		}
		bySink = v
	})
	assert.Equal(t, float64(0), allocsBytes, "AsBytes must achieve 0 allocs/op")

	allocsStr := testing.AllocsPerRun(1000, func() {
		v, ok := refkit.AsString(strVal)
		if !ok {
			t.Fatal("expected ok")
		}
		sSink = v
	})
	assert.Equal(t, float64(0), allocsStr, "AsString must achieve 0 allocs/op")

	_ = iSink
	_ = uSink
	_ = fSink
	_ = bSink
	_ = bySink
	_ = sSink
}

func BenchmarkAsInt64_Unboxed(b *testing.B) {
	val := reflect.ValueOf(int64(42))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		v, _ := refkit.AsInt64(val)
		_ = v
	}
}

func BenchmarkAsInt64_BoxedInterface(b *testing.B) {
	val := reflect.ValueOf(int64(42))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		v := val.Interface().(int64)
		_ = v
	}
}

func BenchmarkAsBytes_Unboxed(b *testing.B) {
	data := []byte("benchmark bytes extraction")
	val := reflect.ValueOf(data)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		bs, _ := refkit.AsBytes(val)
		_ = bs
	}
}
