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

func TestProto_RoundTrip(t *testing.T) {
	t.Parallel()

	var buf []byte
	buf = proto.AppendVarint(buf, 1, 1337)
	buf = proto.AppendSint32(buf, 2, -42)
	buf = proto.AppendSint64(buf, 3, -999999999)
	buf = proto.AppendBool(buf, 4, true)
	buf = proto.AppendFixed32(buf, 5, 0x12345678)
	buf = proto.AppendFixed64(buf, 6, 0x1122334455667788)
	buf = proto.AppendFloat32(buf, 7, 3.14)
	buf = proto.AppendFloat64(buf, 8, 2.718281828)
	buf = proto.AppendString(buf, 9, "zero-alloc-proto")
	buf = proto.AppendBytes(buf, 10, []byte{0xDE, 0xAD, 0xBE, 0xEF})

	seen := make(map[uint32]bool)

	for f, err := range proto.Scan(buf) {
		require.NoError(t, err)
		seen[f.Num] = true

		switch f.Num {
		case 1:
			assert.Equal(t, uint64(1337), f.Varint())
		case 2:
			assert.Equal(t, int32(-42), f.Sint32())
		case 3:
			assert.Equal(t, int64(-999999999), f.Sint64())
		case 4:
			assert.True(t, f.Bool())
		case 5:
			assert.Equal(t, uint32(0x12345678), f.Fixed32())
		case 6:
			assert.Equal(t, uint64(0x1122334455667788), f.Fixed64())
		case 7:
			assert.InDelta(t, float32(3.14), f.Float32(), 0.001)
		case 8:
			assert.InDelta(t, 2.718281828, f.Float64(), 0.00001)
		case 9:
			assert.Equal(t, "zero-alloc-proto", f.String())
		case 10:
			assert.Equal(t, []byte{0xDE, 0xAD, 0xBE, 0xEF}, f.Bytes())
		}
	}

	assert.Equal(t, 10, len(seen))
}

func TestProto_SkipField(t *testing.T) {
	t.Parallel()

	var buf []byte
	buf = proto.AppendVarint(buf, 1, 100)
	buf = proto.AppendFixed32(buf, 2, 200)
	buf = proto.AppendFixed64(buf, 3, 300)
	buf = proto.AppendString(buf, 4, "skip-me")

	// Skip field 1 (tag + varint)
	tagBytes := 1
	n, err := proto.SkipField(buf[tagBytes:], proto.WireVarint)
	require.NoError(t, err)
	assert.Greater(t, n, 0)
}

func BenchmarkScan_ZeroAlloc(b *testing.B) {
	var buf []byte
	buf = proto.AppendVarint(buf, 1, 123456)
	buf = proto.AppendString(buf, 2, "benchmark-payload-data")
	buf = proto.AppendFixed64(buf, 3, 987654321)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for f, err := range proto.Scan(buf) {
			if err != nil || f.Num == 0 {
				b.Fatal("unexpected scan error")
			}
		}
	}
}
