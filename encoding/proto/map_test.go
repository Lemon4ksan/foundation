// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/encoding/proto"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestMap_RoundTrip(t *testing.T) {
	t.Parallel()

	var msg []byte
	msg = proto.AppendMapEntryStringString(msg, 1, "greeting", "hello world")
	msg = proto.AppendMapEntryStringInt64(msg, 1, "count", 42)
	msg = proto.AppendMapEntryInt64String(msg, 1, 100, "hundred")
	msg = proto.AppendMapEntryVarintVarint(msg, 1, 1337, 7331)
	msg = proto.AppendMapEntryStringBytes(msg, 1, "raw", []byte{0xCA, 0xFE})

	// Test ScanMapEntries iterator
	entries := make([]proto.MapEntry, 0, 5)
	for entry, err := range proto.ScanMapEntries(msg, 1) {
		require.NoError(t, err)
		entries = append(entries, entry)
	}

	require.Equal(t, 5, len(entries))

	// Entry 0: string -> string
	assert.Equal(t, "greeting", entries[0].Key.String())
	assert.Equal(t, "hello world", entries[0].Value.String())

	// Entry 1: string -> int64
	assert.Equal(t, "count", entries[1].Key.String())
	assert.Equal(t, int64(42), entries[1].Value.Int64())

	// Entry 2: int64 -> string
	assert.Equal(t, int64(100), entries[2].Key.Int64())
	assert.Equal(t, "hundred", entries[2].Value.String())

	// Entry 3: varint -> varint
	assert.Equal(t, uint64(1337), entries[3].Key.Varint())
	assert.Equal(t, uint64(7331), entries[3].Value.Varint())

	// Entry 4: string -> bytes
	assert.Equal(t, "raw", entries[4].Key.String())
	assert.Equal(t, []byte{0xCA, 0xFE}, entries[4].Value.Bytes())

	// Test ForEachMapEntry
	count := 0
	err := proto.ForEachMapEntry(msg, 1, func(k, v proto.Field) error {
		count++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	// Test Field.MapEntry()
	for f, err := range proto.Scan(msg) {
		require.NoError(t, err)
		k, v, mErr := f.MapEntry()
		require.NoError(t, mErr)
		assert.Equal(t, uint32(1), k.Num)
		assert.Equal(t, uint32(2), v.Num)
	}
}

func TestMap_GenericAppend(t *testing.T) {
	t.Parallel()

	kField := proto.Fixed32Field(1, 0x12345678)
	vField := proto.Fixed64Field(2, 0xabcdef0123456789)

	var buf []byte
	buf = proto.AppendMapEntry(buf, 10, kField, vField)

	for entry, err := range proto.ScanMap(buf, 10) {
		require.NoError(t, err)
		assert.Equal(t, uint32(0x12345678), entry.Key.Fixed32())
		assert.Equal(t, uint64(0xabcdef0123456789), entry.Value.Fixed64())
	}
}

func TestMap_OrderingAndDuplicates(t *testing.T) {
	t.Parallel()

	// Value before key in payload:
	var payload []byte
	payload = proto.AppendString(payload, 2, "value-first")
	payload = proto.AppendString(payload, 1, "key-second")

	k, v, err := proto.ScanMapEntry(payload)
	require.NoError(t, err)
	assert.Equal(t, "key-second", k.String())
	assert.Equal(t, "value-first", v.String())

	// Duplicate fields: last one wins
	var dupPayload []byte
	dupPayload = proto.AppendString(dupPayload, 1, "key-initial")
	dupPayload = proto.AppendString(dupPayload, 2, "val-initial")
	dupPayload = proto.AppendString(dupPayload, 1, "key-final")
	dupPayload = proto.AppendString(dupPayload, 2, "val-final")

	k, v, err = proto.ScanMapEntry(dupPayload)
	require.NoError(t, err)
	assert.Equal(t, "key-final", k.String())
	assert.Equal(t, "val-final", v.String())

	// Unknown field in entry ignored
	var unknownPayload []byte
	unknownPayload = proto.AppendString(unknownPayload, 1, "valid-key")
	unknownPayload = proto.AppendVarint(unknownPayload, 99, 12345) // unknown field
	unknownPayload = proto.AppendString(unknownPayload, 2, "valid-val")

	k, v, err = proto.ScanMapEntry(unknownPayload)
	require.NoError(t, err)
	assert.Equal(t, "valid-key", k.String())
	assert.Equal(t, "valid-val", v.String())
}

func TestMap_Errors(t *testing.T) {
	t.Parallel()

	// Truncated buffer
	_, _, err := proto.ScanMapEntry([]byte{0x0a, 0x05, 'a', 'b'})
	require.ErrorIs(t, err, proto.ErrTruncated)

	// Invalid wire type
	f := proto.VarintField(1, 100)
	_, _, err = f.MapEntry()
	require.ErrorIs(t, err, proto.ErrInvalidWireType)
}

func TestSubmessage_Recursion(t *testing.T) {
	t.Parallel()

	// Build nested hierarchy:
	// Root -> Sub1 (field 10) -> Sub2 (field 20) -> Sub3 (field 30) -> Leaf field 1 = 999
	var sub3 []byte
	sub3 = proto.AppendVarint(sub3, 1, 999)
	sub3 = proto.AppendString(sub3, 2, "deep-leaf")

	var sub2 []byte
	sub2 = proto.AppendMessage(sub2, 30, sub3)
	sub2 = proto.AppendInt32(sub2, 5, 55)

	var sub1 []byte
	sub1 = proto.AppendMessage(sub1, 20, sub2)

	var root []byte
	root = proto.AppendMessage(root, 10, sub1)
	root = proto.AppendString(root, 1, "root-name")

	// Walk with ScanMessage
	depthsVisited := make(map[int]int)
	fieldsSeen := make([]uint32, 0)

	err := proto.ScanMessage(root, 64, func(depth int, f proto.Field) (bool, error) {
		depthsVisited[depth]++
		fieldsSeen = append(fieldsSeen, f.Num)
		if f.Type == proto.WireBytes && (f.Num == 10 || f.Num == 20 || f.Num == 30) {
			return true, nil
		}
		return false, nil
	})
	require.NoError(t, err)

	// Root has field 10 and field 1 (depth 0)
	assert.Equal(t, 2, depthsVisited[0])
	// Sub1 has field 20 (depth 1)
	assert.Equal(t, 1, depthsVisited[1])
	// Sub2 has field 30 and field 5 (depth 2)
	assert.Equal(t, 2, depthsVisited[2])
	// Sub3 has field 1 and field 2 (depth 3)
	assert.Equal(t, 2, depthsVisited[3])

	// Test Field.Scan() submessage iterator
	sub1Field, found, err := proto.FindField(root, 10)
	require.NoError(t, err)
	require.True(t, found)

	sub2Count := 0
	for subField, sErr := range sub1Field.Scan() {
		require.NoError(t, sErr)
		if subField.Num == 20 {
			sub2Count++
		}
	}
	assert.Equal(t, 1, sub2Count)

	// Test Field.ScanMessage
	subRecurseCount := 0
	err = sub1Field.ScanMessage(64, func(depth int, f proto.Field) (bool, error) {
		subRecurseCount++
		return false, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, subRecurseCount)

	// Test WalkMessage
	walked := 0
	err = proto.WalkMessage(root, func(depth int, f proto.Field) bool {
		return f.Num == 10
	}, func(depth int, f proto.Field) error {
		walked++
		return nil
	})
	require.NoError(t, err)
	assert.Greater(t, walked, 0)

	// Test ErrStop early exit
	visitedBeforeStop := 0
	err = proto.ScanMessage(root, 64, func(depth int, f proto.Field) (bool, error) {
		visitedBeforeStop++
		if f.Num == 10 {
			return false, proto.ErrStop
		}
		return false, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, visitedBeforeStop)

	// Test ErrSkipSubmessage
	skippedDeeper := true
	err = proto.ScanMessage(root, 64, func(depth int, f proto.Field) (bool, error) {
		if f.Num == 10 {
			return false, proto.ErrSkipSubmessage
		}
		if depth > 0 {
			skippedDeeper = false
		}
		return false, nil
	})
	require.NoError(t, err)
	assert.True(t, skippedDeeper)

	// Test max depth exceeded
	err = proto.ScanMessage(root, 2, func(depth int, f proto.Field) (bool, error) {
		if f.Type == proto.WireBytes {
			return true, nil
		}
		return false, nil
	})
	require.ErrorIs(t, err, proto.ErrMaxDepthExceeded)
}

func TestLookup_FindLastField(t *testing.T) {
	t.Parallel()

	var msg []byte
	msg = proto.AppendVarint(msg, 1, 10)
	msg = proto.AppendVarint(msg, 1, 20)
	msg = proto.AppendVarint(msg, 1, 30)

	f, found, err := proto.FindField(msg, 1)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, uint64(10), f.Varint())

	lastF, found, err := proto.FindLastField(msg, 1)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, uint64(30), lastF.Varint())
}

func TestLookup_SkipGroup(t *testing.T) {
	t.Parallel()

	var groupBuf []byte
	groupBuf = proto.AppendTag(groupBuf, 4, proto.WireStartGroup)
	groupBuf = proto.AppendVarint(groupBuf, 1, 42)
	groupBuf = proto.AppendTag(groupBuf, 4, proto.WireEndGroup)

	n, err := proto.SkipTagAndField(groupBuf)
	require.NoError(t, err)
	assert.Equal(t, len(groupBuf), n)
}


func TestLookup_FindFieldAndTypedExtractors(t *testing.T) {
	t.Parallel()

	var msg []byte
	msg = proto.AppendVarint(msg, 1, 12345)
	msg = proto.AppendInt32(msg, 2, -50)
	msg = proto.AppendInt64(msg, 3, -100000000)
	msg = proto.AppendUint32(msg, 4, 999)
	msg = proto.AppendUint64(msg, 5, 88888888)
	msg = proto.AppendSint32(msg, 6, -123)
	msg = proto.AppendSint64(msg, 7, -456789)
	msg = proto.AppendBool(msg, 8, true)
	msg = proto.AppendFixed32(msg, 9, 0x11223344)
	msg = proto.AppendFixed64(msg, 10, 0xaabbccddeeff0011)
	msg = proto.AppendFloat32(msg, 11, 1.25)
	msg = proto.AppendFloat64(msg, 12, 2.5)
	msg = proto.AppendString(msg, 13, "lookup-string")
	msg = proto.AppendBytes(msg, 14, []byte{1, 2, 3, 4})

	// FindField
	f, found, err := proto.FindField(msg, 1)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, uint64(12345), f.Varint())

	// Not found
	_, found, err = proto.FindField(msg, 999)
	require.NoError(t, err)
	assert.False(t, found)

	// Typed extractors
	v, found, err := proto.FindVarint(msg, 1)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, uint64(12345), v)

	i32, found, err := proto.FindInt32(msg, 2)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, int32(-50), i32)

	i64, found, err := proto.FindInt64(msg, 3)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, int64(-100000000), i64)

	u32, found, err := proto.FindUint32(msg, 4)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, uint32(999), u32)

	u64, found, err := proto.FindUint64(msg, 5)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, uint64(88888888), u64)

	s32, found, err := proto.FindSint32(msg, 6)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, int32(-123), s32)

	s64, found, err := proto.FindSint64(msg, 7)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, int64(-456789), s64)

	b, found, err := proto.FindBool(msg, 8)
	require.NoError(t, err)
	require.True(t, found)
	assert.True(t, b)

	fx32, found, err := proto.FindFixed32(msg, 9)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, uint32(0x11223344), fx32)

	fx64, found, err := proto.FindFixed64(msg, 10)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, uint64(0xaabbccddeeff0011), fx64)

	fl32, found, err := proto.FindFloat32(msg, 11)
	require.NoError(t, err)
	require.True(t, found)
	assert.InDelta(t, float32(1.25), fl32, 0.0001)

	fl64, found, err := proto.FindFloat64(msg, 12)
	require.NoError(t, err)
	require.True(t, found)
	assert.InDelta(t, 2.5, fl64, 0.00001)

	s, found, err := proto.FindString(msg, 13)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "lookup-string", s)

	by, found, err := proto.FindBytes(msg, 14)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, []byte{1, 2, 3, 4}, by)

	// Wire type mismatch error
	_, _, err = proto.FindVarint(msg, 13) // field 13 is WireBytes
	require.ErrorIs(t, err, proto.ErrInvalidWireType)

	_, _, err = proto.FindString(msg, 1) // field 1 is WireVarint
	require.ErrorIs(t, err, proto.ErrInvalidWireType)
}

func TestLookup_FindNested(t *testing.T) {
	t.Parallel()

	var inner []byte
	inner = proto.AppendString(inner, 5, "nested-value")
	inner = proto.AppendVarint(inner, 6, 4242)

	var outer []byte
	outer = proto.AppendMessage(outer, 2, inner)
	outer = proto.AppendVarint(outer, 1, 100)

	// FindNestedField
	f, found, err := proto.FindNestedField(outer, 2, 5)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "nested-value", f.String())

	// FindNestedString
	str, found, err := proto.FindNestedString(outer, 2, 5)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "nested-value", str)

	// FindNestedVarint
	val, found, err := proto.FindNestedVarint(outer, 2, 6)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, uint64(4242), val)

	// Nested path not found
	_, found, err = proto.FindNestedField(outer, 2, 99)
	require.NoError(t, err)
	assert.False(t, found)

	// Intermediate field not a submessage
	_, _, err = proto.FindNestedField(outer, 1, 5)
	require.ErrorIs(t, err, proto.ErrInvalidWireType)
}

func TestLookup_SkipTagAndField(t *testing.T) {
	t.Parallel()

	var buf []byte
	buf = proto.AppendVarint(buf, 1, 100)
	buf = proto.AppendString(buf, 2, "hello")
	buf = proto.AppendFixed32(buf, 3, 42)
	buf = proto.AppendFixed64(buf, 4, 84)

	offset := 0
	for offset < len(buf) {
		n, err := proto.SkipTagAndField(buf[offset:])
		require.NoError(t, err)
		assert.Greater(t, n, 0)
		offset += n
	}
	assert.Equal(t, len(buf), offset)
}

// -----------------------------------------------------------------------------
// BENCHMARKS (0-alloc contracts)
// -----------------------------------------------------------------------------

func BenchmarkScanMapEntry_ZeroAlloc(b *testing.B) {
	var payload []byte
	payload = proto.AppendString(payload, 1, "device_id")
	payload = proto.AppendString(payload, 2, "sensor-alpha-001")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		k, v, err := proto.ScanMapEntry(payload)
		if err != nil || k.Num != 1 || v.Num != 2 {
			b.Fatal("unexpected map entry scan error")
		}
	}
}

func BenchmarkScanMapEntries_ZeroAlloc(b *testing.B) {
	var msg []byte
	for i := 0; i < 8; i++ {
		msg = proto.AppendMapEntryStringString(msg, 1, "k", "v")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		count := 0
		for entry, err := range proto.ScanMapEntries(msg, 1) {
			if err != nil {
				b.Fatal(err)
			}
			if entry.Key.Num == 1 {
				count++
			}
		}
		if count != 8 {
			b.Fatal("wrong count")
		}
	}
}

func BenchmarkAppendMapEntryStringString_ZeroAlloc(b *testing.B) {
	buf := make([]byte, 0, 128)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		dst := proto.AppendMapEntryStringString(buf[:0], 1, "sensor_key", "sensor_val")
		if len(dst) == 0 {
			b.Fatal("empty dst")
		}
	}
}

func BenchmarkSubmessageRecursion_ZeroAlloc(b *testing.B) {
	var sub2 []byte
	sub2 = proto.AppendVarint(sub2, 1, 42)
	sub2 = proto.AppendString(sub2, 2, "leaf")

	var sub1 []byte
	sub1 = proto.AppendMessage(sub1, 2, sub2)

	var root []byte
	root = proto.AppendMessage(root, 1, sub1)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var leafVal uint64
		err := proto.ScanMessage(root, 64, func(depth int, f proto.Field) (bool, error) {
			if depth < 2 && f.Type == proto.WireBytes {
				return true, nil
			}
			if f.Num == 1 {
				leafVal = f.Varint()
			}
			return false, nil
		})
		if err != nil || leafVal != 42 {
			b.Fatal("unexpected recursion result")
		}
	}
}

func BenchmarkFieldScan_ZeroAlloc(b *testing.B) {
	var sub []byte
	sub = proto.AppendVarint(sub, 1, 100)
	sub = proto.AppendString(sub, 2, "field-scan")

	var root []byte
	root = proto.AppendMessage(root, 5, sub)

	f, found, err := proto.FindField(root, 5)
	if err != nil || !found {
		b.Fatal("not found")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		count := 0
		for subField, sErr := range proto.Scan(f.Message()) {
			if sErr != nil {
				b.Fatal(sErr)
			}
			count += int(subField.Num)
		}
		if count != 3 {
			b.Fatal("wrong count")
		}
	}
}

func BenchmarkFindField_ZeroAlloc(b *testing.B) {
	var msg []byte
	msg = proto.AppendVarint(msg, 1, 100)
	msg = proto.AppendString(msg, 2, "benchmark")
	msg = proto.AppendFixed64(msg, 3, 200)
	msg = proto.AppendVarint(msg, 4, 300)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		f, found, err := proto.FindField(msg, 3)
		if err != nil || !found || f.Num != 3 {
			b.Fatal("find field error")
		}
	}
}

func BenchmarkFindString_ZeroAlloc(b *testing.B) {
	var msg []byte
	msg = proto.AppendVarint(msg, 1, 100)
	msg = proto.AppendString(msg, 2, "fast-string-extraction")
	msg = proto.AppendFixed64(msg, 3, 200)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		s, found, err := proto.FindString(msg, 2)
		if err != nil || !found || len(s) == 0 {
			b.Fatal("find string error")
		}
	}
}

func BenchmarkFindVarint_ZeroAlloc(b *testing.B) {
	var msg []byte
	msg = proto.AppendString(msg, 1, "test")
	msg = proto.AppendVarint(msg, 2, 999999)
	msg = proto.AppendFixed64(msg, 3, 200)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		v, found, err := proto.FindVarint(msg, 2)
		if err != nil || !found || v != 999999 {
			b.Fatal("find varint error")
		}
	}
}

func BenchmarkFindNestedField_ZeroAlloc(b *testing.B) {
	var sub2 []byte
	sub2 = proto.AppendString(sub2, 5, "nested-value")

	var sub1 []byte
	sub1 = proto.AppendMessage(sub1, 2, sub2)

	var root []byte
	root = proto.AppendMessage(root, 1, sub1)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		f, found, err := proto.FindNestedField(root, 1, 2, 5)
		if err != nil || !found || f.Num != 5 {
			b.Fatal("find nested field error")
		}
	}
}

func BenchmarkThroughput_FindField(b *testing.B) {
	var msg []byte
	for i := 1; i <= 32; i++ {
		msg = proto.AppendString(msg, uint32(i), "throughput-payload-block-data-segment")
	}
	target := uint32(32)

	b.SetBytes(int64(len(msg)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		f, found, err := proto.FindField(msg, target)
		if err != nil || !found || f.Num != target {
			b.Fatal("find field throughput error")
		}
	}
}

func BenchmarkThroughput_ScanMapEntry(b *testing.B) {
	var entry []byte
	entry = proto.AppendString(entry, 1, "urn:foundation:service:telemetry:cluster:region:zone:node:instance")
	entry = proto.AppendString(entry, 2, "metric_value_timeseries_dataset_zero_allocation_protobuf_stream")

	b.SetBytes(int64(len(entry)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		k, v, err := proto.ScanMapEntry(entry)
		if err != nil || k.Num != 1 || v.Num != 2 {
			b.Fatal("map entry scan throughput error")
		}
	}
}

