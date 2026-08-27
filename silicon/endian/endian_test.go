// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package endian_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/silicon/endian"
	"github.com/lemon4ksan/foundation/testkit/assert"
)

func TestEndian_LittleEndian_All(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 16)

	// PutUint64, PutUint64LE, Uint64, Uint64LE, Load64, LoadLE64, Store64, StoreLE64
	endian.PutUint64(buf, 0x0102030405060708)
	assert.Equal(t, uint64(0x0102030405060708), endian.Uint64(buf))
	assert.Equal(t, uint64(0x0102030405060708), endian.Uint64LE(buf))
	assert.Equal(t, uint64(0x0102030405060708), endian.Load64(buf, 0))
	assert.Equal(t, uint64(0x0102030405060708), endian.LoadLE64(buf, 0))

	endian.PutUint64LE(buf, 0x1122334455667788)
	assert.Equal(t, uint64(0x1122334455667788), endian.Uint64LE(buf))

	endian.Store64(buf, 8, 0xAABBCCDDEEFF0011)
	assert.Equal(t, uint64(0xAABBCCDDEEFF0011), endian.Load64(buf, 8))
	endian.StoreLE64(buf, 8, 0x1122334455667788)
	assert.Equal(t, uint64(0x1122334455667788), endian.LoadLE64(buf, 8))

	// PutUint32, PutUint32LE, Uint32, Uint32LE, Load32, LoadLE32, Store32, StoreLE32
	endian.PutUint32(buf, 0x12345678)
	assert.Equal(t, uint32(0x12345678), endian.Uint32(buf))
	assert.Equal(t, uint32(0x12345678), endian.Uint32LE(buf))
	assert.Equal(t, uint32(0x12345678), endian.Load32(buf, 0))
	assert.Equal(t, uint32(0x12345678), endian.LoadLE32(buf, 0))

	endian.PutUint32LE(buf, 0xDEADBEEF)
	assert.Equal(t, uint32(0xDEADBEEF), endian.Uint32LE(buf))

	endian.Store32(buf, 0xCAFEBABE)
	assert.Equal(t, uint32(0xCAFEBABE), endian.Load32(buf, 0))
	endian.StoreLE32(buf, 0xDEADBEEF)
	assert.Equal(t, uint32(0xDEADBEEF), endian.LoadLE32(buf, 0))

	// PutUint16, PutUint16LE, Uint16, Uint16LE, Load16, LoadLE16, Store16, StoreLE16, Load8
	endian.PutUint16(buf, 0x1234)
	assert.Equal(t, uint16(0x1234), endian.Uint16(buf))
	assert.Equal(t, uint16(0x1234), endian.Uint16LE(buf))
	assert.Equal(t, uint16(0x1234), endian.Load16(buf, 0))
	assert.Equal(t, uint16(0x1234), endian.LoadLE16(buf, 0))

	endian.PutUint16LE(buf, 0x5678)
	assert.Equal(t, uint16(0x5678), endian.Uint16LE(buf))

	endian.Store16(buf, 0xABCD)
	assert.Equal(t, uint16(0xABCD), endian.Load16(buf, 0))
	endian.StoreLE16(buf, 0xCAFE)
	assert.Equal(t, uint16(0xCAFE), endian.LoadLE16(buf, 0))

	assert.Equal(t, byte(0xFE), endian.Load8(buf, 0))
}

func TestEndian_BigEndian_All(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 16)

	// PutUint64BE, Uint64BE, LoadBE64, StoreBE64
	endian.PutUint64BE(buf, 0x0102030405060708)
	assert.Equal(t, uint64(0x0102030405060708), endian.Uint64BE(buf))
	assert.Equal(t, uint64(0x0102030405060708), endian.LoadBE64(buf, 0))

	endian.StoreBE64(buf, 8, 0x1122334455667788)
	assert.Equal(t, uint64(0x1122334455667788), endian.LoadBE64(buf, 8))

	// PutUint32BE, Uint32BE, LoadBE32, StoreBE32
	endian.PutUint32BE(buf, 0x12345678)
	assert.Equal(t, uint32(0x12345678), endian.Uint32BE(buf))
	assert.Equal(t, uint32(0x12345678), endian.LoadBE32(buf, 0))

	endian.StoreBE32(buf, 0xDEADBEEF)
	assert.Equal(t, uint32(0xDEADBEEF), endian.LoadBE32(buf, 0))

	// PutUint24BE, Uint24BE, LoadBE24, StoreBE24
	endian.PutUint24BE(buf, 0x123456)
	assert.Equal(t, uint32(0x123456), endian.Uint24BE(buf))
	assert.Equal(t, uint32(0x123456), endian.LoadBE24(buf, 0))

	endian.StoreBE24(buf, 0x00AABBCC)
	assert.Equal(t, uint32(0x00AABBCC), endian.LoadBE24(buf, 0))
	assert.Equal(t, uint32(0x00AABBCC), endian.Uint24BE(buf))

	// PutUint16BE, Uint16BE, LoadBE16, StoreBE16
	endian.PutUint16BE(buf, 0x1234)
	assert.Equal(t, uint16(0x1234), endian.Uint16BE(buf))
	assert.Equal(t, uint16(0x1234), endian.LoadBE16(buf, 0))

	endian.StoreBE16(buf, 0xCAFE)
	assert.Equal(t, uint16(0xCAFE), endian.LoadBE16(buf, 0))
}
