// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"errors"
	"math"
	"testing"

	"github.com/lemon4ksan/foundation/encoding/proto"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestVarint_EdgeCases(t *testing.T) {
	t.Parallel()

	testVals := []uint64{
		0, 1, 2, 127, 128, 129,
		255, 256, 16383, 16384,
		(1 << 21) - 1, 1 << 21,
		(1 << 28) - 1, 1 << 28,
		(1 << 35) - 1, 1 << 35,
		(1 << 42) - 1, 1 << 42,
		(1 << 49) - 1, 1 << 49,
		(1 << 56) - 1, 1 << 56,
		(1 << 63) - 1, 1 << 63,
		math.MaxUint32,
		math.MaxUint64,
	}

	for _, val := range testVals {
		sz := proto.SizeVarint(val)
		encoded := proto.AppendUvarint(nil, val)
		assert.Equal(t, sz, len(encoded))

		// PutUvarint test
		buf := make([]byte, sz)
		n := proto.PutUvarint(buf, val)
		assert.Equal(t, sz, n)
		assert.Equal(t, encoded, buf)

		// DecodeUvarint test
		decoded, read, err := proto.DecodeUvarint(buf)
		require.NoError(t, err)
		assert.Equal(t, sz, read)
		assert.Equal(t, val, decoded)

		// Uvarint compatibility helper test
		uvVal, uvRead := proto.Uvarint(buf)
		assert.Equal(t, sz, uvRead)
		assert.Equal(t, val, uvVal)
	}
}

func TestVarint_Errors(t *testing.T) {
	t.Parallel()

	// Empty buffer
	_, _, err := proto.DecodeUvarint(nil)
	assert.True(t, errors.Is(err, proto.ErrTruncated))

	val, n := proto.Uvarint(nil)
	assert.Equal(t, uint64(0), val)
	assert.Equal(t, 0, n)

	// Truncated multi-byte varint
	truncated := []byte{0x80, 0x80}
	_, _, err = proto.DecodeUvarint(truncated)
	assert.True(t, errors.Is(err, proto.ErrTruncated))

	// Overflow varint: 10 bytes with byte 9 > 1
	overflow := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02}
	_, _, err = proto.DecodeUvarint(overflow)
	assert.True(t, errors.Is(err, proto.ErrVarintOverflow))

	val, n = proto.Uvarint(overflow)
	assert.Equal(t, uint64(0), val)
	assert.Equal(t, -1, n)
}

func TestPackedVarints_RoundTrip(t *testing.T) {
	t.Parallel()

	// Empty slice
	buf := proto.AppendPackedVarints(nil, 1, nil)
	assert.Equal(t, 0, len(buf))

	// Diverse varints
	vals := []uint64{0, 1, 42, 127, 128, 300, 16384, 1000000, math.MaxUint32, math.MaxUint64}
	buf = proto.AppendPackedVarints(buf, 2, vals)
	require.NotEmpty(t, buf)

	// Scan through message fields
	scanned := false
	for f, err := range proto.Scan(buf) {
		require.NoError(t, err)
		assert.Equal(t, uint32(2), f.Num)
		assert.Equal(t, proto.WireBytes, f.Type)
		scanned = true

		var decoded []uint64
		for v, vErr := range f.PackedVarints() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, vals, decoded)
	}
	assert.True(t, scanned)

	// Direct decode into slice
	dst, err := proto.DecodePackedVarints(buf[2:], nil) // skip tag + length
	require.NoError(t, err)
	assert.Equal(t, vals, dst)
}

func TestPackedFixed32_RoundTrip(t *testing.T) {
	t.Parallel()

	// Empty
	buf := proto.AppendPackedFixed32(nil, 1, nil)
	assert.Equal(t, 0, len(buf))

	vals := []uint32{0, 1, 1337, 0x12345678, math.MaxUint32}
	buf = proto.AppendPackedFixed32(buf, 3, vals)
	require.NotEmpty(t, buf)

	scanned := false
	for f, err := range proto.Scan(buf) {
		require.NoError(t, err)
		assert.Equal(t, uint32(3), f.Num)
		scanned = true

		var decoded []uint32
		for v, vErr := range f.PackedFixed32() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, vals, decoded)

		// Direct decode
		direct, dErr := proto.DecodePackedFixed32(f.Bytes(), nil)
		require.NoError(t, dErr)
		assert.Equal(t, vals, direct)
	}
	assert.True(t, scanned)
}

func TestPackedFixed64_RoundTrip(t *testing.T) {
	t.Parallel()

	buf := proto.AppendPackedFixed64(nil, 1, nil)
	assert.Equal(t, 0, len(buf))

	vals := []uint64{0, 1, 0x0102030405060708, 0xAABBCCDDEEFF0011, math.MaxUint64}
	buf = proto.AppendPackedFixed64(buf, 4, vals)

	scanned := false
	for f, err := range proto.Scan(buf) {
		require.NoError(t, err)
		assert.Equal(t, uint32(4), f.Num)
		scanned = true

		var decoded []uint64
		for v, vErr := range f.PackedFixed64() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, vals, decoded)

		direct, dErr := proto.DecodePackedFixed64(f.Bytes(), nil)
		require.NoError(t, dErr)
		assert.Equal(t, vals, direct)
	}
	assert.True(t, scanned)
}

func TestPackedFloat32_RoundTrip(t *testing.T) {
	t.Parallel()

	buf := proto.AppendPackedFloat32(nil, 1, nil)
	assert.Equal(t, 0, len(buf))

	vals := []float32{
		0.0,
		-0.0,
		1.0,
		-1.0,
		3.14159,
		float32(math.Pi),
		float32(math.SmallestNonzeroFloat32),
		float32(math.MaxFloat32),
	}
	buf = proto.AppendPackedFloat32(buf, 5, vals)

	scanned := false
	for f, err := range proto.Scan(buf) {
		require.NoError(t, err)
		assert.Equal(t, uint32(5), f.Num)
		scanned = true

		var decoded []float32
		for v, vErr := range f.PackedFloat32() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, vals, decoded)

		direct, dErr := proto.DecodePackedFloat32(f.Bytes(), nil)
		require.NoError(t, dErr)
		assert.Equal(t, vals, direct)
	}
	assert.True(t, scanned)
}

func TestPackedFloat64_RoundTrip(t *testing.T) {
	t.Parallel()

	buf := proto.AppendPackedFloat64(nil, 1, nil)
	assert.Equal(t, 0, len(buf))

	vals := []float64{0.0, -0.0, 1.0, -1.0, math.Pi, math.E, math.SmallestNonzeroFloat64, math.MaxFloat64}
	buf = proto.AppendPackedFloat64(buf, 6, vals)

	scanned := false
	for f, err := range proto.Scan(buf) {
		require.NoError(t, err)
		assert.Equal(t, uint32(6), f.Num)
		scanned = true

		var decoded []float64
		for v, vErr := range f.PackedFloat64() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, vals, decoded)

		direct, dErr := proto.DecodePackedFloat64(f.Bytes(), nil)
		require.NoError(t, dErr)
		assert.Equal(t, vals, direct)
	}
	assert.True(t, scanned)
}

func TestPackedSint32_RoundTrip(t *testing.T) {
	t.Parallel()

	buf := proto.AppendPackedSint32(nil, 1, nil)
	assert.Equal(t, 0, len(buf))

	vals := []int32{0, 1, -1, 42, -42, 1000, -1000, math.MinInt32, math.MaxInt32}
	buf = proto.AppendPackedSint32(buf, 7, vals)

	scanned := false
	for f, err := range proto.Scan(buf) {
		require.NoError(t, err)
		assert.Equal(t, uint32(7), f.Num)
		scanned = true

		var decoded []int32
		for v, vErr := range f.PackedSint32() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, vals, decoded)

		direct, dErr := proto.DecodePackedSint32(f.Bytes(), nil)
		require.NoError(t, dErr)
		assert.Equal(t, vals, direct)
	}
	assert.True(t, scanned)
}

func TestPackedSint64_RoundTrip(t *testing.T) {
	t.Parallel()

	buf := proto.AppendPackedSint64(nil, 1, nil)
	assert.Equal(t, 0, len(buf))

	vals := []int64{0, 1, -1, 1000000, -1000000, math.MinInt64, math.MaxInt64}
	buf = proto.AppendPackedSint64(buf, 8, vals)

	scanned := false
	for f, err := range proto.Scan(buf) {
		require.NoError(t, err)
		assert.Equal(t, uint32(8), f.Num)
		scanned = true

		var decoded []int64
		for v, vErr := range f.PackedSint64() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, vals, decoded)

		direct, dErr := proto.DecodePackedSint64(f.Bytes(), nil)
		require.NoError(t, dErr)
		assert.Equal(t, vals, direct)
	}
	assert.True(t, scanned)
}

func TestPackedBools_RoundTrip(t *testing.T) {
	t.Parallel()

	buf := proto.AppendPackedBools(nil, 1, nil)
	assert.Equal(t, 0, len(buf))

	vals := []bool{true, false, true, true, false, false, true, true, true, false}
	buf = proto.AppendPackedBools(buf, 9, vals)

	scanned := false
	for f, err := range proto.Scan(buf) {
		require.NoError(t, err)
		assert.Equal(t, uint32(9), f.Num)
		scanned = true

		var decoded []bool
		for v, vErr := range f.PackedBools() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, vals, decoded)

		direct, dErr := proto.DecodePackedBools(f.Bytes(), nil)
		require.NoError(t, dErr)
		assert.Equal(t, vals, direct)
	}
	assert.True(t, scanned)
}

func TestPackedInt32_Int64_Uint32_Uint64(t *testing.T) {
	t.Parallel()

	// Int32
	i32Vals := []int32{-100, -1, 0, 1, 100, math.MinInt32, math.MaxInt32}
	buf32 := proto.AppendPackedInt32(nil, 1, i32Vals)
	for f, err := range proto.Scan(buf32) {
		require.NoError(t, err)
		var decoded []int32
		for v, vErr := range f.PackedInt32() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, i32Vals, decoded)
	}

	// Int64
	i64Vals := []int64{-500, -1, 0, 1, 500, math.MinInt64, math.MaxInt64}
	buf64 := proto.AppendPackedInt64(nil, 2, i64Vals)
	for f, err := range proto.Scan(buf64) {
		require.NoError(t, err)
		var decoded []int64
		for v, vErr := range f.PackedInt64() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, i64Vals, decoded)
	}

	// Uint32
	u32Vals := []uint32{0, 10, 100, 1000, math.MaxUint32}
	bufu32 := proto.AppendPackedUint32(nil, 3, u32Vals)
	for f, err := range proto.Scan(bufu32) {
		require.NoError(t, err)
		var decoded []uint32
		for v, vErr := range f.PackedUint32() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, u32Vals, decoded)
	}

	// Sfixed32 and Sfixed64
	sf32Vals := []int32{-10, -5, 0, 5, 10}
	bufsf32 := proto.AppendPackedSfixed32(nil, 4, sf32Vals)
	for f, err := range proto.Scan(bufsf32) {
		require.NoError(t, err)
		var decoded []uint32
		for v, vErr := range f.PackedFixed32() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, len(sf32Vals), len(decoded))
	}

	sf64Vals := []int64{-1000, -500, 0, 500, 1000}
	bufsf64 := proto.AppendPackedSfixed64(nil, 5, sf64Vals)
	for f, err := range proto.Scan(bufsf64) {
		require.NoError(t, err)
		var decoded []uint64
		for v, vErr := range f.PackedFixed64() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, len(sf64Vals), len(decoded))
	}
}

func TestPacked_TruncatedErrors(t *testing.T) {
	t.Parallel()

	// Truncated Fixed32 (3 bytes)
	data32 := []byte{1, 2, 3}
	for _, err := range proto.ScanPackedFixed32(data32) {
		assert.True(t, errors.Is(err, proto.ErrTruncated))
	}
	_, dErr := proto.DecodePackedFixed32(data32, nil)
	assert.True(t, errors.Is(dErr, proto.ErrTruncated))

	// Truncated Fixed64 (7 bytes)
	data64 := []byte{1, 2, 3, 4, 5, 6, 7}
	for _, err := range proto.ScanPackedFixed64(data64) {
		assert.True(t, errors.Is(err, proto.ErrTruncated))
	}
	_, dErr64 := proto.DecodePackedFixed64(data64, nil)
	assert.True(t, errors.Is(dErr64, proto.ErrTruncated))

	// Truncated Float32
	for _, err := range proto.ScanPackedFloat32(data32) {
		assert.True(t, errors.Is(err, proto.ErrTruncated))
	}
	_, df32Err := proto.DecodePackedFloat32(data32, nil)
	assert.True(t, errors.Is(df32Err, proto.ErrTruncated))

	// Truncated Float64
	for _, err := range proto.ScanPackedFloat64(data64) {
		assert.True(t, errors.Is(err, proto.ErrTruncated))
	}
	_, df64Err := proto.DecodePackedFloat64(data64, nil)
	assert.True(t, errors.Is(df64Err, proto.ErrTruncated))

	// Truncated Varints
	dataVarint := []byte{0x80}
	for _, err := range proto.ScanPackedVarints(dataVarint) {
		assert.True(t, errors.Is(err, proto.ErrTruncated))
	}
	_, dvarErr := proto.DecodePackedVarints(dataVarint, nil)
	assert.True(t, errors.Is(dvarErr, proto.ErrTruncated))
}

// -----------------------------------------------------------------------------
// BENCHMARKS: Proving 0 B/op, 0 allocs/op, and > 2 GB/s wire throughput
// -----------------------------------------------------------------------------

func BenchmarkVarint_Decode_SingleByte(b *testing.B) {
	const count = 10000
	data := make([]byte, count)
	for i := range data {
		data[i] = byte(i % 128)
	}

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		offset := 0
		n := len(data)
		for offset < n {
			val, readBytes, err := proto.DecodeUvarint(data[offset:])
			if err != nil || val > 127 {
				b.Fatal("unexpected error")
			}
			offset += readBytes
		}
	}
}

func BenchmarkVarint_Decode_MultiByte(b *testing.B) {
	const count = 5000
	var data []byte
	for i := range count {
		data = proto.AppendUvarint(data, uint64(1000+i))
	}

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		offset := 0
		n := len(data)
		for offset < n {
			val, readBytes, err := proto.DecodeUvarint(data[offset:])
			if err != nil || val < 1000 {
				b.Fatal("unexpected error")
			}
			offset += readBytes
		}
	}
}

func BenchmarkVarint_Encode_SingleByte(b *testing.B) {
	buf := make([]byte, 0, 16)
	b.SetBytes(1)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = proto.AppendUvarint(buf[:0], 42)
	}
}

func BenchmarkVarint_Encode_MultiByte(b *testing.B) {
	buf := make([]byte, 0, 16)
	b.SetBytes(3)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = proto.AppendUvarint(buf[:0], 250000)
	}
}

func BenchmarkPackedVarints_Append_SingleByte(b *testing.B) {
	const count = 1000
	vals := make([]uint64, count)
	for i := range vals {
		vals[i] = uint64(i % 120)
	}

	buf := make([]byte, 0, count*2)
	wireBytes := len(proto.AppendPackedVarints(nil, 1, vals))
	b.SetBytes(int64(wireBytes))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = proto.AppendPackedVarints(buf[:0], 1, vals)
	}
}

func BenchmarkPackedVarints_Scan_SingleByte(b *testing.B) {
	const count = 1000
	vals := make([]uint64, count)
	for i := range vals {
		vals[i] = uint64(i % 120)
	}
	buf := proto.AppendPackedVarints(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for v, err := range proto.ScanPackedVarints(payload) {
			if err != nil || v > 120 {
				b.Fatal("scan error")
			}
		}
	}
}

func BenchmarkPackedVarints_Decode_SWAR(b *testing.B) {
	const count = 1000
	vals := make([]uint64, count)
	for i := range vals {
		vals[i] = uint64(i % 120)
	}
	buf := proto.AppendPackedVarints(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	dst := make([]uint64, 0, count)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var err error
		dst, err = proto.DecodePackedVarints(payload, dst)
		if err != nil || len(dst) != count {
			b.Fatal("decode error")
		}
	}
}

func BenchmarkPackedFixed32_Append(b *testing.B) {
	const count = 1000
	vals := make([]uint32, count)
	for i := range vals {
		vals[i] = uint32(i * 1337)
	}

	buf := make([]byte, 0, count*4+16)
	wireBytes := len(proto.AppendPackedFixed32(nil, 1, vals))
	b.SetBytes(int64(wireBytes))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = proto.AppendPackedFixed32(buf[:0], 1, vals)
	}
}

func BenchmarkPackedFixed32_Scan(b *testing.B) {
	const count = 1000
	vals := make([]uint32, count)
	for i := range vals {
		vals[i] = uint32(i * 1337)
	}
	buf := proto.AppendPackedFixed32(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for v, err := range proto.ScanPackedFixed32(payload) {
			if err != nil && v == 0 {
				b.Fatal("scan error")
			}
		}
	}
}

func BenchmarkPackedFixed32_Decode(b *testing.B) {
	const count = 1000
	vals := make([]uint32, count)
	for i := range vals {
		vals[i] = uint32(i * 1337)
	}
	buf := proto.AppendPackedFixed32(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	dst := make([]uint32, count)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var err error
		dst, err = proto.DecodePackedFixed32(payload, dst)
		if err != nil || len(dst) != count {
			b.Fatal("decode error")
		}
	}
}

func BenchmarkPackedFixed64_Append(b *testing.B) {
	const count = 1000
	vals := make([]uint64, count)
	for i := range vals {
		vals[i] = uint64(i * 987654321)
	}

	buf := make([]byte, 0, count*8+16)
	wireBytes := len(proto.AppendPackedFixed64(nil, 1, vals))
	b.SetBytes(int64(wireBytes))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = proto.AppendPackedFixed64(buf[:0], 1, vals)
	}
}

func BenchmarkPackedFixed64_Scan(b *testing.B) {
	const count = 1000
	vals := make([]uint64, count)
	for i := range vals {
		vals[i] = uint64(i * 987654321)
	}
	buf := proto.AppendPackedFixed64(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for v, err := range proto.ScanPackedFixed64(payload) {
			if err != nil && v == 0 {
				b.Fatal("scan error")
			}
		}
	}
}

func BenchmarkPackedFixed64_Decode(b *testing.B) {
	const count = 1000
	vals := make([]uint64, count)
	for i := range vals {
		vals[i] = uint64(i * 987654321)
	}
	buf := proto.AppendPackedFixed64(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	dst := make([]uint64, count)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var err error
		dst, err = proto.DecodePackedFixed64(payload, dst)
		if err != nil || len(dst) != count {
			b.Fatal("decode error")
		}
	}
}

func BenchmarkPackedFloat32_Append(b *testing.B) {
	const count = 1000
	vals := make([]float32, count)
	for i := range vals {
		vals[i] = float32(i) * 1.5
	}

	buf := make([]byte, 0, count*4+16)
	wireBytes := len(proto.AppendPackedFloat32(nil, 1, vals))
	b.SetBytes(int64(wireBytes))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = proto.AppendPackedFloat32(buf[:0], 1, vals)
	}
}

func BenchmarkPackedFloat32_Scan(b *testing.B) {
	const count = 1000
	vals := make([]float32, count)
	for i := range vals {
		vals[i] = float32(i) * 1.5
	}
	buf := proto.AppendPackedFloat32(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for v, err := range proto.ScanPackedFloat32(payload) {
			if err != nil && v == 0 {
				b.Fatal("scan error")
			}
		}
	}
}

func BenchmarkPackedFloat64_Append(b *testing.B) {
	const count = 1000
	vals := make([]float64, count)
	for i := range vals {
		vals[i] = float64(i) * 2.718
	}

	buf := make([]byte, 0, count*8+16)
	wireBytes := len(proto.AppendPackedFloat64(nil, 1, vals))
	b.SetBytes(int64(wireBytes))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = proto.AppendPackedFloat64(buf[:0], 1, vals)
	}
}

func BenchmarkPackedFloat64_Scan(b *testing.B) {
	const count = 1000
	vals := make([]float64, count)
	for i := range vals {
		vals[i] = float64(i) * 2.718
	}
	buf := proto.AppendPackedFloat64(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for v, err := range proto.ScanPackedFloat64(payload) {
			if err != nil && v == 0 {
				b.Fatal("scan error")
			}
		}
	}
}

func BenchmarkPackedBools_Append(b *testing.B) {
	const count = 1000
	vals := make([]bool, count)
	for i := range vals {
		vals[i] = (i % 2) == 0
	}

	buf := make([]byte, 0, count+16)
	wireBytes := len(proto.AppendPackedBools(nil, 1, vals))
	b.SetBytes(int64(wireBytes))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = proto.AppendPackedBools(buf[:0], 1, vals)
	}
}

func BenchmarkPackedBools_Scan(b *testing.B) {
	const count = 1000
	vals := make([]bool, count)
	for i := range vals {
		vals[i] = (i % 2) == 0
	}
	buf := proto.AppendPackedBools(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, err := range proto.ScanPackedBools(payload) {
			if err != nil {
				b.Fatal("scan error")
			}
		}
	}
}

func BenchmarkPackedSint32_Append(b *testing.B) {
	const count = 1000
	vals := make([]int32, count)
	for i := range vals {
		if i%2 == 0 {
			vals[i] = int32(i % 60)
		} else {
			vals[i] = -int32(i % 60)
		}
	}

	buf := make([]byte, 0, count*2)
	wireBytes := len(proto.AppendPackedSint32(nil, 1, vals))
	b.SetBytes(int64(wireBytes))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = proto.AppendPackedSint32(buf[:0], 1, vals)
	}
}

func BenchmarkPackedSint32_Scan(b *testing.B) {
	const count = 1000
	vals := make([]int32, count)
	for i := range vals {
		if i%2 == 0 {
			vals[i] = int32(i % 60)
		} else {
			vals[i] = -int32(i % 60)
		}
	}
	buf := proto.AppendPackedSint32(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, err := range proto.ScanPackedSint32(payload) {
			if err != nil {
				b.Fatal("scan error")
			}
		}
	}
}

func BenchmarkPackedSint32_Decode(b *testing.B) {
	const count = 1000
	vals := make([]int32, count)
	for i := range vals {
		if i%2 == 0 {
			vals[i] = int32(i % 60)
		} else {
			vals[i] = -int32(i % 60)
		}
	}
	buf := proto.AppendPackedSint32(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	dst := make([]int32, 0, count)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var err error
		dst, err = proto.DecodePackedSint32(payload, dst)
		if err != nil || len(dst) != count {
			b.Fatal("decode error")
		}
	}
}

func BenchmarkPackedSint64_Append(b *testing.B) {
	const count = 1000
	vals := make([]int64, count)
	for i := range vals {
		if i%2 == 0 {
			vals[i] = int64(i % 60)
		} else {
			vals[i] = -int64(i % 60)
		}
	}

	buf := make([]byte, 0, count*2)
	wireBytes := len(proto.AppendPackedSint64(nil, 1, vals))
	b.SetBytes(int64(wireBytes))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = proto.AppendPackedSint64(buf[:0], 1, vals)
	}
}

func BenchmarkPackedSint64_Scan(b *testing.B) {
	const count = 1000
	vals := make([]int64, count)
	for i := range vals {
		if i%2 == 0 {
			vals[i] = int64(i % 60)
		} else {
			vals[i] = -int64(i % 60)
		}
	}
	buf := proto.AppendPackedSint64(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, err := range proto.ScanPackedSint64(payload) {
			if err != nil {
				b.Fatal("scan error")
			}
		}
	}
}

func BenchmarkPackedSint64_Decode(b *testing.B) {
	const count = 1000
	vals := make([]int64, count)
	for i := range vals {
		if i%2 == 0 {
			vals[i] = int64(i % 60)
		} else {
			vals[i] = -int64(i % 60)
		}
	}
	buf := proto.AppendPackedSint64(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	dst := make([]int64, 0, count)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var err error
		dst, err = proto.DecodePackedSint64(payload, dst)
		if err != nil || len(dst) != count {
			b.Fatal("decode error")
		}
	}
}

func BenchmarkPackedFloat32_Decode(b *testing.B) {
	const count = 1000
	vals := make([]float32, count)
	for i := range vals {
		vals[i] = float32(i) * 1.5
	}
	buf := proto.AppendPackedFloat32(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	dst := make([]float32, count)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var err error
		dst, err = proto.DecodePackedFloat32(payload, dst)
		if err != nil || len(dst) != count {
			b.Fatal("decode error")
		}
	}
}

func BenchmarkPackedFloat64_Decode(b *testing.B) {
	const count = 1000
	vals := make([]float64, count)
	for i := range vals {
		vals[i] = float64(i) * 2.718
	}
	buf := proto.AppendPackedFloat64(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	dst := make([]float64, count)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var err error
		dst, err = proto.DecodePackedFloat64(payload, dst)
		if err != nil || len(dst) != count {
			b.Fatal("decode error")
		}
	}
}

func BenchmarkPackedBools_Decode(b *testing.B) {
	const count = 1000
	vals := make([]bool, count)
	for i := range vals {
		vals[i] = (i % 2) == 0
	}
	buf := proto.AppendPackedBools(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	dst := make([]bool, 0, count)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var err error
		dst, err = proto.DecodePackedBools(payload, dst)
		if err != nil || len(dst) != count {
			b.Fatal("decode error")
		}
	}
}

func BenchmarkVarint_Decode_Batch(b *testing.B) {
	const count = 1000
	vals := make([]uint64, count)
	for i := range vals {
		vals[i] = uint64(i % 120)
	}
	buf := proto.AppendPackedVarints(nil, 1, vals)

	var payload []byte
	for f, err := range proto.Scan(buf) {
		if err != nil {
			b.Fatal(err)
		}
		payload = f.Bytes()
	}

	dst := make([]uint64, 0, count)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var err error
		dst, err = proto.DecodeVarints(payload, dst)
		if err != nil || len(dst) != count {
			b.Fatal("decode error")
		}
	}
}
