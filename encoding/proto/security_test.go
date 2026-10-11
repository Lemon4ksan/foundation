// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/lemon4ksan/foundation/encoding/proto"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

// TestRepro_WireBytes_LengthPrefixOverflowPanic verifies that a length prefix near MaxUint64
// does not cause integer wraparound or slice indexing panic.
func TestRepro_WireBytes_LengthPrefixOverflowPanic(t *testing.T) {
	buf := []byte{0x0a} // Tag 1, WireBytes
	buf = binary.AppendUvarint(buf, math.MaxUint64)

	for _, err := range proto.Scan(buf) {
		require.ErrorIs(t, err, proto.ErrTruncated)
	}
}

// TestRepro_VarintOverflow_MoreThan10Bytes verifies that varints > 10 bytes return ErrVarintOverflow.
func TestRepro_VarintOverflow_MoreThan10Bytes(t *testing.T) {
	// 11 bytes with 0x80 continuation bit
	buf11 := make([]byte, 11)
	for i := range buf11 {
		buf11[i] = 0x80
	}
	for _, err := range proto.Scan(buf11) {
		require.ErrorIs(t, err, proto.ErrVarintOverflow)
	}

	// 10 bytes where 10th byte has continuation bit
	buf10 := make([]byte, 10)
	for i := range buf10 {
		buf10[i] = 0x80
	}
	for _, err := range proto.Scan(buf10) {
		require.ErrorIs(t, err, proto.ErrVarintOverflow)
	}

	// 10 bytes where 10th byte has bit 1 set (overflows 64-bit uint)
	bufOverflowVal := make([]byte, 10)
	for i := range 9 {
		bufOverflowVal[i] = 0x80
	}
	bufOverflowVal[9] = 0x02 // bit 1 set, > 1
	for _, err := range proto.Scan(bufOverflowVal) {
		require.ErrorIs(t, err, proto.ErrVarintOverflow)
	}
}

// TestRepro_SkipField_WireBytesOverflow verifies that SkipField with MaxUint64 length prefix
// returns ErrTruncated and does not return (0, nil) which would cause an infinite loop.
func TestRepro_SkipField_WireBytesOverflow(t *testing.T) {
	var payload []byte
	payload = binary.AppendUvarint(payload, math.MaxUint64)

	n, err := proto.SkipField(payload, proto.WireBytes)
	require.ErrorIs(t, err, proto.ErrTruncated)
	assert.Equal(t, 0, n)
}

// TestRepro_Tag_OverflowAndInvalidFieldNum verifies that tags > MaxUint32 or with fieldNum > MaxFieldNumber
// are rejected with ErrInvalidTag.
func TestRepro_Tag_OverflowAndInvalidFieldNum(t *testing.T) {
	// (1 << 35) + 8: in low 32 bits, this is 8 (field 1, wire 0)
	var buf []byte
	buf = binary.AppendUvarint(buf, (uint64(1)<<35)+8)
	buf = binary.AppendUvarint(buf, 42)

	for _, err := range proto.Scan(buf) {
		require.ErrorIs(t, err, proto.ErrInvalidTag)
	}
}

// TestSecurity_VarintOverflowLocations tests overflow detection across all syntactic positions.
func TestSecurity_VarintOverflowLocations(t *testing.T) {
	overflowVarint := make([]byte, 11)
	for i := range overflowVarint {
		overflowVarint[i] = 0x80
	}

	// 1. Tag is overflow varint
	for _, err := range proto.Scan(overflowVarint) {
		require.ErrorIs(t, err, proto.ErrVarintOverflow)
	}

	// 2. WireVarint value is overflow varint
	tagVarint := proto.AppendTag(nil, 1, proto.WireVarint)
	bufVarint := append(tagVarint, overflowVarint...)
	for _, err := range proto.Scan(bufVarint) {
		require.ErrorIs(t, err, proto.ErrVarintOverflow)
	}

	// 3. WireBytes length is overflow varint
	tagBytes := proto.AppendTag(nil, 1, proto.WireBytes)
	bufBytes := append(tagBytes, overflowVarint...)
	for _, err := range proto.Scan(bufBytes) {
		require.ErrorIs(t, err, proto.ErrVarintOverflow)
	}

	// 4. SkipField on WireVarint with overflow
	n, err := proto.SkipField(overflowVarint, proto.WireVarint)
	require.ErrorIs(t, err, proto.ErrVarintOverflow)
	assert.Equal(t, 0, n)

	// 5. SkipField on WireBytes with overflow length
	n, err = proto.SkipField(overflowVarint, proto.WireBytes)
	require.ErrorIs(t, err, proto.ErrVarintOverflow)
	assert.Equal(t, 0, n)

	// 6. Packed varints with overflow
	for _, err := range proto.ScanPackedVarints(overflowVarint) {
		require.ErrorIs(t, err, proto.ErrVarintOverflow)
	}
}

// TestSecurity_TruncatedBuffers tests all truncation edge cases.
func TestSecurity_TruncatedBuffers(t *testing.T) {
	// Truncated incomplete varint (continuation bit set but buffer ends)
	truncatedVarint := []byte{0x80, 0x80, 0x80}
	for _, err := range proto.Scan(truncatedVarint) {
		require.ErrorIs(t, err, proto.ErrTruncated)
	}

	// Truncated WireFixed32 (< 4 bytes)
	bufFixed32 := proto.AppendTag(nil, 1, proto.WireFixed32)
	bufFixed32 = append(bufFixed32, 0x01, 0x02)
	for _, err := range proto.Scan(bufFixed32) {
		require.ErrorIs(t, err, proto.ErrTruncated)
	}
	n, err := proto.SkipField([]byte{0x01, 0x02}, proto.WireFixed32)
	require.ErrorIs(t, err, proto.ErrTruncated)
	assert.Equal(t, 0, n)

	// Truncated WireFixed64 (< 8 bytes)
	bufFixed64 := proto.AppendTag(nil, 1, proto.WireFixed64)
	bufFixed64 = append(bufFixed64, 0x01, 0x02, 0x03, 0x04)
	for _, err := range proto.Scan(bufFixed64) {
		require.ErrorIs(t, err, proto.ErrTruncated)
	}
	n, err = proto.SkipField([]byte{0x01, 0x02, 0x03, 0x04}, proto.WireFixed64)
	require.ErrorIs(t, err, proto.ErrTruncated)
	assert.Equal(t, 0, n)

	// Truncated WireBytes payload (length declared 10, only 3 present)
	var bufBytes []byte
	bufBytes = proto.AppendTag(bufBytes, 1, proto.WireBytes)
	bufBytes = binary.AppendUvarint(bufBytes, 10)
	bufBytes = append(bufBytes, []byte{1, 2, 3}...)
	for _, err := range proto.Scan(bufBytes) {
		require.ErrorIs(t, err, proto.ErrTruncated)
	}
	skipN, skipErr := proto.SkipField(bufBytes[1:], proto.WireBytes)
	require.ErrorIs(t, skipErr, proto.ErrTruncated)
	assert.Equal(t, 0, skipN)
}

// TestSecurity_InvalidTagsAndWireTypes validates field number limits and invalid wire types.
func TestSecurity_InvalidTagsAndWireTypes(t *testing.T) {
	// Zero tag (tag == 0 -> fieldNum == 0)
	for _, err := range proto.Scan([]byte{0x00}) {
		require.ErrorIs(t, err, proto.ErrInvalidTag)
	}

	// Field num 0 with wire types 1, 2, 5
	for _, wt := range []proto.WireType{proto.WireFixed64, proto.WireBytes, proto.WireFixed32} {
		for _, err := range proto.Scan([]byte{byte(wt)}) {
			require.ErrorIs(t, err, proto.ErrInvalidTag)
		}
	}

	// Field num > MaxFieldNumber (536,870,911)
	var bigFieldBuf []byte
	bigFieldBuf = binary.AppendUvarint(bigFieldBuf, uint64(proto.MaxFieldNumber+1)<<3)
	bigFieldBuf = binary.AppendUvarint(bigFieldBuf, 1)
	for _, err := range proto.Scan(bigFieldBuf) {
		require.ErrorIs(t, err, proto.ErrInvalidTag)
	}

	// Invalid wire types 3, 4, 6, 7
	for _, wt := range []proto.WireType{proto.WireStartGroup, proto.WireEndGroup, 6, 7} {
		tag := proto.Tag(1, wt)
		tagBuf := binary.AppendUvarint(nil, uint64(tag))
		for _, err := range proto.Scan(tagBuf) {
			require.ErrorIs(t, err, proto.ErrInvalidWireType)
		}

		n, err := proto.SkipField([]byte{0x01, 0x02}, wt)
		require.ErrorIs(t, err, proto.ErrInvalidWireType)
		assert.Equal(t, 0, n)
	}
}

// TestSecurity_DeeplyNestedMessages tests iterative parsing of deeply nested sub-messages.
func TestSecurity_DeeplyNestedMessages(t *testing.T) {
	// Build 100 levels of nested length-delimited messages
	payload := proto.AppendVarint(nil, 1, 42) // inner payload: field 1 = 42

	for range 100 {
		var outer []byte
		outer = proto.AppendBytes(outer, 1, payload)
		payload = outer
	}

	// Iteratively peel back each layer
	curr := payload
	for range 100 {
		seenField := false
		for f, err := range proto.Scan(curr) {
			require.NoError(t, err)
			assert.Equal(t, uint32(1), f.Num)
			assert.Equal(t, proto.WireBytes, f.Type)
			curr = f.Message()
			seenField = true
		}
		require.True(t, seenField)
	}

	// Innermost payload is WireVarint = 42
	seenInner := false
	for f, err := range proto.Scan(curr) {
		require.NoError(t, err)
		assert.Equal(t, uint32(1), f.Num)
		assert.Equal(t, proto.WireVarint, f.Type)
		assert.Equal(t, uint64(42), f.Varint())
		seenInner = true
	}
	require.True(t, seenInner)
}

// TestSecurity_PackedRepeatedRoundTrip verifies packed repeated encode/decode.
func TestSecurity_PackedRepeatedRoundTrip(t *testing.T) {
	// 1. Packed varints
	varintOrig := []uint64{0, 1, 127, 128, 16383, 16384, math.MaxUint32, math.MaxUint64}
	var packedBuf []byte
	packedBuf = proto.AppendPackedVarints(packedBuf, 1, varintOrig)

	for f, err := range proto.Scan(packedBuf) {
		require.NoError(t, err)
		assert.Equal(t, uint32(1), f.Num)
		assert.Equal(t, proto.WireBytes, f.Type)

		var decoded []uint64
		for v, vErr := range f.PackedVarints() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, varintOrig, decoded)
	}

	// 2. Packed fixed32
	f32Orig := []uint32{0, 1, 1000, 0xDEADBEEF, math.MaxUint32}
	var f32Buf []byte
	f32Buf = proto.AppendPackedFixed32(f32Buf, 2, f32Orig)

	for f, err := range proto.Scan(f32Buf) {
		require.NoError(t, err)
		assert.Equal(t, uint32(2), f.Num)
		var decoded []uint32
		for v, vErr := range f.PackedFixed32() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, f32Orig, decoded)
	}

	// 3. Packed fixed64
	f64Orig := []uint64{0, 1, 1000, 0x1122334455667788, math.MaxUint64}
	var f64Buf []byte
	f64Buf = proto.AppendPackedFixed64(f64Buf, 3, f64Orig)

	for f, err := range proto.Scan(f64Buf) {
		require.NoError(t, err)
		assert.Equal(t, uint32(3), f.Num)
		var decoded []uint64
		for v, vErr := range f.PackedFixed64() {
			require.NoError(t, vErr)
			decoded = append(decoded, v)
		}
		assert.Equal(t, f64Orig, decoded)
	}

	// 4. Packed Sint32 / Sint64
	s32Orig := []int32{0, -1, 1, -100, 100, math.MinInt32, math.MaxInt32}
	var s32Buf []byte
	s32Buf = proto.AppendPackedSint32(s32Buf, 4, s32Orig)
	for f, err := range proto.Scan(s32Buf) {
		require.NoError(t, err)
		var decoded []int32
		for v, vErr := range f.PackedVarints() {
			require.NoError(t, vErr)
			decoded = append(decoded, proto.DecodeZigZag32(uint32(v)))
		}
		assert.Equal(t, s32Orig, decoded)
	}
}

// TestSecurity_ZeroAllocUnderStress asserts 0 heap allocations during scan operations.
func TestSecurity_ZeroAllocUnderStress(t *testing.T) {
	var buf []byte
	buf = proto.AppendVarint(buf, 1, 12345)
	buf = proto.AppendFixed32(buf, 2, 0xABCD1234)
	buf = proto.AppendFixed64(buf, 3, 0x1234567890ABCDEF)
	buf = proto.AppendString(buf, 4, "zero-alloc-stress-test")
	buf = proto.AppendBytes(buf, 5, []byte{1, 2, 3, 4, 5})
	buf = proto.AppendPackedVarints(buf, 6, []uint64{10, 20, 30, 40})

	// Scan allocs
	scanAllocs := testing.AllocsPerRun(1000, func() {
		for f, err := range proto.Scan(buf) {
			if err != nil {
				t.Fatalf("scan error: %v", err)
			}
			_ = f.Num
			_ = f.Type
			_ = f.Raw
			_ = f.Data
		}
	})
	assert.Equal(t, float64(0), scanAllocs)

	// SkipField allocs
	skipAllocs := testing.AllocsPerRun(1000, func() {
		_, _ = proto.SkipField(buf, proto.WireVarint)
	})
	assert.Equal(t, float64(0), skipAllocs)
}

// TestSecurity_CircularAndRecursiveMessages tests recursive traversal against circular payloads,
// deep nesting, and Billion Laughs exponential entity expansion attacks.
func TestSecurity_CircularAndRecursiveMessages(t *testing.T) {
	// 1. Deep nesting exceeding DefaultMaxDepth (64)
	inner := proto.AppendVarint(nil, 1, 42)
	for range 70 {
		inner = proto.AppendMessage(nil, 1, inner)
	}

	err := proto.ScanMessage(inner, 64, func(depth int, f proto.Field) (bool, error) {
		return true, nil
	})
	require.ErrorIs(t, err, proto.ErrMaxDepthExceeded)

	// 2. Truncated submessage inside recursion
	var truncatedInner []byte
	truncatedInner = proto.AppendTag(truncatedInner, 1, proto.WireBytes)
	truncatedInner = binary.AppendUvarint(truncatedInner, 100) // length 100, but 0 payload bytes
	wrapped := proto.AppendMessage(nil, 1, truncatedInner)

	err = proto.ScanMessage(wrapped, 64, func(depth int, f proto.Field) (bool, error) {
		return true, nil
	})
	require.ErrorIs(t, err, proto.ErrTruncated)

	// 3. Varint overflow submessage inside recursion
	overflowBytes := make([]byte, 11)
	for i := range overflowBytes {
		overflowBytes[i] = 0x80
	}
	wrappedOverflow := proto.AppendMessage(nil, 1, overflowBytes)
	err = proto.ScanMessage(wrappedOverflow, 64, func(depth int, f proto.Field) (bool, error) {
		return true, nil
	})
	require.ErrorIs(t, err, proto.ErrVarintOverflow)

	// 4. Billion Laughs / Exponential DAG Bomb Defense
	// Build a branching payload where each level contains 2 references to child
	leaf := proto.AppendVarint(nil, 1, 99)
	for range 20 { // 2^20 = 1,048,576 operations if unbudgeted!
		var branch []byte
		branch = proto.AppendMessage(branch, 1, leaf)
		branch = proto.AppendMessage(branch, 2, leaf)
		leaf = branch
	}

	// Traversal with budget must terminate with ErrRecursionBudgetExceeded rather than hanging
	err = proto.ScanMessage(leaf, 64, func(depth int, f proto.Field) (bool, error) {
		return true, nil
	})
	require.ErrorIs(t, err, proto.ErrRecursionBudgetExceeded)
}
