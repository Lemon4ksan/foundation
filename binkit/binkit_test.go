// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package binkit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/binkit"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
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

	w := binkit.NewWriter(nil, 64)
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

	r := binkit.NewReader(buf)
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
	assert.Equal(t, binkit.ErrBufferTooShort, r.Err())
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

	wire, err := binkit.MarshalLE(&orig, nil)
	require.NoError(t, err)
	assert.Equal(t, 20, len(wire)) // 4 + 2 + 2 + 8 + 4 = 20

	var decoded TestHeader
	err = binkit.UnmarshalLE(wire, &decoded)
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

	wire, err := binkit.MarshalBE(&orig, nil)
	require.NoError(t, err)
	assert.Equal(t, 20, len(wire))

	var decoded TestHeader
	err = binkit.UnmarshalBE(wire, &decoded)
	require.NoError(t, err)

	assert.Equal(t, orig.Magic, decoded.Magic)
	assert.Equal(t, orig.Version, decoded.Version)
	assert.Equal(t, orig.Flags, decoded.Flags)
	assert.Equal(t, orig.Size, decoded.Size)
	assert.Equal(t, orig.Tag, decoded.Tag)
}
