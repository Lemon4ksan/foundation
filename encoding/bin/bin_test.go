package bin_test

// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"encoding/binary"
	"errors"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/encoding/bin"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

type TestHeader struct {
	Magic      uint32
	Version    uint16
	Flags      uint16
	Size       uint64
	Tag        [4]byte
	Name       string `binkit:"skip"`
	IgnoreThis []byte `binkit:"-"`
}

func TestReader_Writer_Roundtrip(t *testing.T) {
	t.Parallel()

	w := bin.NewWriter(nil, 64)
	w.U8(0x7F).
		U16LE(0x1234).
		U32LE(0xCAFEBABE).
		U64LE(0x0102030405060708).
		U16BE(0x4321).
		U32BE(0xDEADBEEF).
		U64BE(0x0807060504030201).
		RawString("foundation-binkit").
		Raw([]byte{1, 2, 3, 4})

	buf := w.Bytes()
	require.NotEmpty(t, buf)

	r := bin.NewReader(buf)
	assert.Equal(t, uint8(0x7F), r.U8())
	assert.Equal(t, uint16(0x1234), r.U16LE())
	assert.Equal(t, uint32(0xCAFEBABE), r.U32LE())
	assert.Equal(t, uint64(0x0102030405060708), r.U64LE())
	assert.Equal(t, uint16(0x4321), r.U16BE())
	assert.Equal(t, uint32(0xDEADBEEF), r.U32BE())
	assert.Equal(t, uint64(0x0807060504030201), r.U64BE())
	assert.Equal(t, "foundation-binkit", r.String(len("foundation-binkit")))
	assert.Equal(t, []byte{1, 2, 3, 4}, r.Bytes(4))
	require.NoError(t, r.Err())

	// Sticky error on buffer overrun
	_ = r.U32LE()
	assert.Equal(t, bin.ErrBufferTooShort, r.Err())
}

func TestStruct_UnmarshalLE_MarshalLE(t *testing.T) {
	t.Parallel()

	orig := TestHeader{
		Magic:   0x04034B50,
		Version: 20,
		Flags:   0x08,
		Size:    1048576,
		Tag:     [4]byte{'P', 'A', 'C', 'K'},
		Name:    "ignored string",
	}

	wire, err := bin.MarshalLE(&orig, nil)
	require.NoError(t, err)
	assert.Equal(t, 20, len(wire)) // 4 + 2 + 2 + 8 + 4 = 20

	var decoded TestHeader
	err = bin.UnmarshalLE(wire, &decoded)
	require.NoError(t, err)

	assert.Equal(t, orig.Magic, decoded.Magic)
	assert.Equal(t, orig.Version, decoded.Version)
	assert.Equal(t, orig.Flags, decoded.Flags)
	assert.Equal(t, orig.Size, decoded.Size)
	assert.Equal(t, orig.Tag, decoded.Tag)
	assert.Equal(t, "", decoded.Name) // skipped
}

func TestStruct_UnmarshalBE_MarshalBE(t *testing.T) {
	t.Parallel()

	orig := TestHeader{
		Magic:   0x04034B50,
		Version: 20,
		Flags:   0x08,
		Size:    1048576,
		Tag:     [4]byte{'E', 'L', 'F', '0'},
	}

	wire, err := bin.MarshalBE(&orig, nil)
	require.NoError(t, err)
	assert.Equal(t, 20, len(wire))

	var decoded TestHeader
	err = bin.UnmarshalBE(wire, &decoded)
	require.NoError(t, err)

	assert.Equal(t, orig.Magic, decoded.Magic)
	assert.Equal(t, orig.Version, decoded.Version)
	assert.Equal(t, orig.Flags, decoded.Flags)
	assert.Equal(t, orig.Size, decoded.Size)
	assert.Equal(t, orig.Tag, decoded.Tag)
}
type AdversarialRecord struct {
	ID       uint32
	Flag     uint8
	Value    uint64
	Tag      [4]byte
	Internal int `binkit:"-"`
}

func TestReader_Adversarial_EarlyExit(t *testing.T) {
	t.Parallel()

	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}

	t.Run("Chunks early exit after 0 (first yield returns false)", func(t *testing.T) {
		r := bin.NewReader(data)
		count := 0
		r.Chunks(10)(func(chunk []byte) bool {
			count++
			return false
		})
		assert.Equal(t, 1, count)
		assert.Equal(t, 10, r.Pos())
		assert.Equal(t, 90, r.Remaining())
	})

	t.Run("Chunks early exit mid-sequence", func(t *testing.T) {
		r := bin.NewReader(data)
		count := 0
		for range r.Chunks(10) {
			count++
			if count == 3 {
				break
			}
		}
		assert.Equal(t, 3, count)
		assert.Equal(t, 30, r.Pos())
	})

	t.Run("U32LESeq early exit after 0", func(t *testing.T) {
		r := bin.NewReader(data)
		count := 0
		r.U32LESeq()(func(v uint32) bool {
			count++
			return false
		})
		assert.Equal(t, 1, count)
		assert.Equal(t, 4, r.Pos())
	})

	t.Run("U32LESeq early exit mid-sequence", func(t *testing.T) {
		r := bin.NewReader(data)
		count := 0
		for range r.U32LESeq() {
			count++
			if count == 5 {
				break
			}
		}
		assert.Equal(t, 5, count)
		assert.Equal(t, 20, r.Pos())
	})

	t.Run("UnmarshalSeqBE early exit", func(t *testing.T) {
		rec := AdversarialRecord{ID: 1, Flag: 2, Value: 3, Tag: [4]byte{'T', 'E', 'S', 'T'}}
		recBytes, err := bin.MarshalBE(&rec, nil)
		require.NoError(t, err)

		multiRecs := make([]byte, 0, len(recBytes)*5)
		for range 5 {
			multiRecs = append(multiRecs, recBytes...)
		}

		count := 0
		for _, err := range bin.UnmarshalSeqBE[AdversarialRecord](multiRecs) {
			require.NoError(t, err)
			count++
			if count == 2 {
				break
			}
		}
		assert.Equal(t, 2, count)
	})
}

func TestReader_Adversarial_MalformedAndTruncated(t *testing.T) {
	t.Parallel()

	t.Run("Chunks with non-positive chunk size", func(t *testing.T) {
		r := bin.NewReader([]byte("test data"))
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

	t.Run("Chunks buffer smaller than chunk size", func(t *testing.T) {
		r := bin.NewReader([]byte("short"))
		count := 0
		for range r.Chunks(10) {
			count++
		}
		assert.Equal(t, 0, count)
		assert.Nil(t, r.Err())
		assert.Equal(t, 5, r.Remaining())
	})

	t.Run("Chunks buffer not multiple of chunk size", func(t *testing.T) {
		r := bin.NewReader([]byte("12345678901")) // 11 bytes
		var chunks [][]byte
		for c := range r.Chunks(4) {
			chunks = append(chunks, c)
		}
		assert.Equal(t, 2, len(chunks))
		assert.Equal(t, []byte("1234"), chunks[0])
		assert.Equal(t, []byte("5678"), chunks[1])
		assert.Equal(t, 3, r.Remaining())
		assert.Nil(t, r.Err())
	})

	t.Run("Integer sequences on truncated buffers", func(t *testing.T) {
		// 3 bytes truncated for U32
		r32 := bin.NewReader([]byte{1, 2, 3})
		count := 0
		for range r32.U32LESeq() {
			count++
		}
		assert.Equal(t, 0, count)
		assert.Nil(t, r32.Err())
		assert.Equal(t, 3, r32.Remaining())

		// 7 bytes truncated for U64
		r64 := bin.NewReader([]byte{1, 2, 3, 4, 5, 6, 7})
		count = 0
		for range r64.U64BESeq() {
			count++
		}
		assert.Equal(t, 0, count)
		assert.Nil(t, r64.Err())
		assert.Equal(t, 7, r64.Remaining())

		// 1 byte truncated for U16
		r16 := bin.NewReader([]byte{1})
		count = 0
		for range r16.U16LESeq() {
			count++
		}
		assert.Equal(t, 0, count)
		assert.Nil(t, r16.Err())
	})

	t.Run("Sticky error halts iteration", func(t *testing.T) {
		r := bin.NewReader([]byte{1, 2})
		// Force sticky error
		_ = r.U32LE()
		require.True(t, errors.Is(r.Err(), bin.ErrBufferTooShort))

		count := 0
		for range r.U8Seq() {
			count++
		}
		assert.Equal(t, 0, count)

		for range r.Chunks(1) {
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("UnmarshalSeq on non-struct types yields error", func(t *testing.T) {
		var errs []error
		for _, err := range bin.UnmarshalSeqBE[int]([]byte{1, 2, 3, 4}) {
			if err != nil {
				errs = append(errs, err)
			}
		}
		require.Equal(t, 1, len(errs))
		assert.Contains(t, errs[0].Error(), "target must be a pointer to struct")

		errs = nil
		for _, err := range bin.UnmarshalSeqLE[*AdversarialRecord]([]byte{1, 2, 3, 4}) {
			if err != nil {
				errs = append(errs, err)
			}
		}
		require.Equal(t, 1, len(errs))
		assert.Contains(t, errs[0].Error(), "target must be a pointer to struct")
	})

	t.Run("UnmarshalSeq on empty struct or unexported fields", func(t *testing.T) {
		type EmptyStruct struct{}
		count := 0
		for _, err := range bin.UnmarshalSeqBE[EmptyStruct]([]byte{1, 2, 3}) {
			require.NoError(t, err)
			count++
		}
		assert.Equal(t, 0, count)

		type UnexportedOnly struct {
			unexported int
		}
		count = 0
		for _, err := range bin.UnmarshalSeqLE[UnexportedOnly]([]byte{1, 2, 3}) {
			require.NoError(t, err)
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("UnmarshalSeq on truncated record buffer", func(t *testing.T) {
		rec := AdversarialRecord{ID: 10, Flag: 1, Value: 99, Tag: [4]byte{'A', 'B', 'C', 'D'}}
		full, err := bin.MarshalLE(&rec, nil)
		require.NoError(t, err)

		// Provide full record + truncated 5 bytes of second record
		truncated := append(full, full[:5]...)
		var records []AdversarialRecord
		for r, err := range bin.UnmarshalSeqLE[AdversarialRecord](truncated) {
			require.NoError(t, err)
			records = append(records, r)
		}
		require.Equal(t, 1, len(records))
		assert.Equal(t, uint32(10), records[0].ID)
	})
}

func TestReader_Adversarial_ExtremeInputs(t *testing.T) {
	t.Parallel()

	t.Run("nil buffer Reader", func(t *testing.T) {
		r := bin.NewReader(nil)
		assert.Equal(t, 0, r.Remaining())
		assert.Equal(t, 0, r.Pos())

		count := 0
		for range r.Chunks(10) {
			count++
		}
		assert.Equal(t, 0, count)

		for range r.U8Seq() {
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("massive buffer 100,000 U32 values", func(t *testing.T) {
		n := 100000
		buf := make([]byte, n*4)
		for i := range n {
			binary.LittleEndian.PutUint32(buf[i*4:(i+1)*4], uint32(i))
		}

		r := bin.NewReader(buf)
		count := 0
		expected := uint32(0)
		for v := range r.U32LESeq() {
			assert.Equal(t, expected, v)
			expected++
			count++
		}
		assert.Equal(t, n, count)
		assert.Equal(t, 0, r.Remaining())
	})

	t.Run("massive buffer Chunks 1MB in 64KB chunks", func(t *testing.T) {
		size := 1024 * 1024
		buf := make([]byte, size)
		r := bin.NewReader(buf)

		chunkSize := 64 * 1024
		chunks := 0
		for c := range r.Chunks(chunkSize) {
			assert.Equal(t, chunkSize, len(c))
			chunks++
		}
		assert.Equal(t, 16, chunks)
		assert.Equal(t, 0, r.Remaining())
	})
}

func TestReader_Adversarial_Concurrency(t *testing.T) {
	t.Parallel()

	rec := AdversarialRecord{ID: 777, Flag: 9, Value: 8888, Tag: [4]byte{'Z', 'Z', 'Z', 'Z'}}
	recBytes, err := bin.MarshalBE(&rec, nil)
	require.NoError(t, err)

	multiBytes := bytesRepeat(recBytes, 10)

	var wg sync.WaitGroup
	goroutines := 30

	for g := range goroutines {
		wg.Add(2)

		// Test concurrent UnmarshalSeqBE with shared layoutCache
		go func() {
			defer wg.Done()
			count := 0
			for val, err := range bin.UnmarshalSeqBE[AdversarialRecord](multiBytes) {
				assert.Nil(t, err)
				assert.Equal(t, uint32(777), val.ID)
				count++
			}
			assert.Equal(t, 10, count)
		}()

		// Test concurrent Reader sequential iterations
		go func(gid int) {
			defer wg.Done()
			localBuf := make([]byte, 40)
			for i := range localBuf {
				localBuf[i] = byte(gid)
			}
			r := bin.NewReader(localBuf)
			count := 0
			for range r.U32LESeq() {
				count++
			}
			assert.Equal(t, 10, count)
		}(g)
	}

	wg.Wait()
}

func bytesRepeat(b []byte, count int) []byte {
	res := make([]byte, 0, len(b)*count)
	for range count {
		res = append(res, b...)
	}
	return res
}

func TestReader_Adversarial_ZeroAlloc(t *testing.T) {
	buf := make([]byte, 1024)

	allocsChunks := testing.AllocsPerRun(100, func() {
		r := bin.NewReader(buf)
		for c := range r.Chunks(64) {
			if len(c) == 0 {
				t.Fatal("empty")
			}
		}
	})
	assert.Equal(t, float64(0), allocsChunks, "Reader.Chunks must be 0 allocs/op")

	allocsU32 := testing.AllocsPerRun(100, func() {
		r := bin.NewReader(buf)
		for v := range r.U32LESeq() {
			if v > 1000000 {
				t.Fatal("unexpected")
			}
		}
	})
	assert.Equal(t, float64(0), allocsU32, "Reader.U32LESeq must be 0 allocs/op")

	allocsU64 := testing.AllocsPerRun(100, func() {
		r := bin.NewReader(buf)
		for v := range r.U64BESeq() {
			if v > 1000000 {
				t.Fatal("unexpected")
			}
		}
	})
	assert.Equal(t, float64(0), allocsU64, "Reader.U64BESeq must be 0 allocs/op")
}
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
