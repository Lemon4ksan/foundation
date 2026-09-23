// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bin_test

import (
	"encoding/binary"
	"testing"

	"github.com/lemon4ksan/foundation/encoding/bin"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestReader_Chunks(t *testing.T) {
	t.Parallel()

	data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	t.Run("even chunking", func(t *testing.T) {
		r := bin.NewReader(data)
		var chunks [][]byte
		for chunk := range r.Chunks(2) {
			chunks = append(chunks, chunk)
		}
		require.Equal(t, 5, len(chunks))
		assert.Equal(t, []byte{1, 2}, chunks[0])
		assert.Equal(t, []byte{9, 10}, chunks[4])
	})

	t.Run("chunking with remainder", func(t *testing.T) {
		r := bin.NewReader(data)
		var chunks [][]byte
		for chunk := range r.Chunks(3) {
			chunks = append(chunks, chunk)
		}
		require.Equal(t, 3, len(chunks))
		assert.Equal(t, []byte{1, 2, 3}, chunks[0])
		assert.Equal(t, []byte{7, 8, 9}, chunks[2])
		assert.Equal(t, 1, r.Remaining())
	})

	t.Run("invalid chunk size", func(t *testing.T) {
		r := bin.NewReader(data)
		count := 0
		for range r.Chunks(0) {
			count++
		}
		assert.Equal(t, 0, count)

		for range r.Chunks(-5) {
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("early exit", func(t *testing.T) {
		r := bin.NewReader(data)
		count := 0
		for range r.Chunks(2) {
			count++
			if count == 2 {
				break
			}
		}
		assert.Equal(t, 2, count)
	})
}

func TestReader_IntegerSeqs(t *testing.T) {
	t.Parallel()

	t.Run("U8Seq", func(t *testing.T) {
		data := []byte{0x10, 0x20, 0x30, 0x40}
		r := bin.NewReader(data)
		var vals []uint8
		for v := range r.U8Seq() {
			vals = append(vals, v)
		}
		assert.Equal(t, []uint8{0x10, 0x20, 0x30, 0x40}, vals)
	})

	t.Run("U16LESeq", func(t *testing.T) {
		data := make([]byte, 6)
		binary.LittleEndian.PutUint16(data[0:2], 100)
		binary.LittleEndian.PutUint16(data[2:4], 200)
		binary.LittleEndian.PutUint16(data[4:6], 300)

		r := bin.NewReader(data)
		var vals []uint16
		for v := range r.U16LESeq() {
			vals = append(vals, v)
		}
		assert.Equal(t, []uint16{100, 200, 300}, vals)
	})

	t.Run("U32LESeq", func(t *testing.T) {
		data := make([]byte, 8)
		binary.LittleEndian.PutUint32(data[0:4], 0x12345678)
		binary.LittleEndian.PutUint32(data[4:8], 0x9abcdef0)

		r := bin.NewReader(data)
		var vals []uint32
		for v := range r.U32LESeq() {
			vals = append(vals, v)
		}
		assert.Equal(t, []uint32{0x12345678, 0x9abcdef0}, vals)
	})

	t.Run("U64LESeq", func(t *testing.T) {
		data := make([]byte, 16)
		binary.LittleEndian.PutUint64(data[0:8], 0x1122334455667788)
		binary.LittleEndian.PutUint64(data[8:16], 0x8877665544332211)

		r := bin.NewReader(data)
		var vals []uint64
		for v := range r.U64LESeq() {
			vals = append(vals, v)
		}
		assert.Equal(t, []uint64{0x1122334455667788, 0x8877665544332211}, vals)
	})

	t.Run("U16BESeq", func(t *testing.T) {
		data := make([]byte, 4)
		binary.BigEndian.PutUint16(data[0:2], 0xABCD)
		binary.BigEndian.PutUint16(data[2:4], 0x1234)

		r := bin.NewReader(data)
		var vals []uint16
		for v := range r.U16BESeq() {
			vals = append(vals, v)
		}
		assert.Equal(t, []uint16{0xABCD, 0x1234}, vals)
	})

	t.Run("U32BESeq", func(t *testing.T) {
		data := make([]byte, 8)
		binary.BigEndian.PutUint32(data[0:4], 0xAABBCCDD)
		binary.BigEndian.PutUint32(data[4:8], 0x11223344)

		r := bin.NewReader(data)
		var vals []uint32
		for v := range r.U32BESeq() {
			vals = append(vals, v)
		}
		assert.Equal(t, []uint32{0xAABBCCDD, 0x11223344}, vals)
	})

	t.Run("U64BESeq", func(t *testing.T) {
		data := make([]byte, 16)
		binary.BigEndian.PutUint64(data[0:8], 0x0102030405060708)
		binary.BigEndian.PutUint64(data[8:16], 0xF1F2F3F4F5F6F7F8)

		r := bin.NewReader(data)
		var vals []uint64
		for v := range r.U64BESeq() {
			vals = append(vals, v)
		}
		assert.Equal(t, []uint64{0x0102030405060708, 0xF1F2F3F4F5F6F7F8}, vals)
	})
}

type PacketHeader struct {
	Magic   uint16
	Length  uint16
	SeqNum  uint32
	Payload uint64
}

func TestUnmarshalSeq(t *testing.T) {
	t.Parallel()

	h1 := PacketHeader{Magic: 0xCAFE, Length: 16, SeqNum: 1, Payload: 100}
	h2 := PacketHeader{Magic: 0xBABE, Length: 16, SeqNum: 2, Payload: 200}

	t.Run("UnmarshalSeqBE", func(t *testing.T) {
		buf1, err := bin.MarshalBE(&h1, nil)
		require.NoError(t, err)
		buf2, err := bin.MarshalBE(&h2, nil)
		require.NoError(t, err)

		data := append(buf1, buf2...)

		var items []PacketHeader
		for item, err := range bin.UnmarshalSeqBE[PacketHeader](data) {
			require.NoError(t, err)
			items = append(items, item)
		}

		require.Equal(t, 2, len(items))
		assert.Equal(t, h1, items[0])
		assert.Equal(t, h2, items[1])
	})

	t.Run("UnmarshalSeqLE", func(t *testing.T) {
		buf1, err := bin.MarshalLE(&h1, nil)
		require.NoError(t, err)
		buf2, err := bin.MarshalLE(&h2, nil)
		require.NoError(t, err)

		data := append(buf1, buf2...)

		var items []PacketHeader
		for item, err := range bin.UnmarshalSeqLE[PacketHeader](data) {
			require.NoError(t, err)
			items = append(items, item)
		}

		require.Equal(t, 2, len(items))
		assert.Equal(t, h1, items[0])
		assert.Equal(t, h2, items[1])
	})

	t.Run("early exit", func(t *testing.T) {
		buf1, _ := bin.MarshalBE(&h1, nil)
		buf2, _ := bin.MarshalBE(&h2, nil)
		data := append(buf1, buf2...)

		count := 0
		for _, err := range bin.UnmarshalSeqBE[PacketHeader](data) {
			require.NoError(t, err)
			count++
			break
		}
		assert.Equal(t, 1, count)
	})

	t.Run("invalid non-struct type", func(t *testing.T) {
		for _, err := range bin.UnmarshalSeqBE[int]([]byte{1, 2, 3, 4}) {
			assert.Error(t, err)
		}
	})
}

func BenchmarkReader_Chunks_ZeroAlloc(b *testing.B) {
	data := make([]byte, 1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		r := bin.NewReader(data)
		for chunk := range r.Chunks(64) {
			if len(chunk) != 64 {
				b.Fatal("unexpected chunk size")
			}
		}
	}
}

func BenchmarkReader_U32LESeq_ZeroAlloc(b *testing.B) {
	data := make([]byte, 1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		r := bin.NewReader(data)
		for v := range r.U32LESeq() {
			if v != 0 {
				b.Fatal("unexpected val")
			}
		}
	}
}

func BenchmarkReader_U64BESeq_ZeroAlloc(b *testing.B) {
	data := make([]byte, 1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		r := bin.NewReader(data)
		for v := range r.U64BESeq() {
			if v != 0 {
				b.Fatal("unexpected val")
			}
		}
	}
}
