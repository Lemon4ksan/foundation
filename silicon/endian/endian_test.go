// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package endian_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/silicon/endian"
	"github.com/lemon4ksan/foundation/testkit/assert"
)

func TestEndianLittleEndian(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 16)
	endian.PutUint64LE(buf, 0x0102030405060708)

	assert.Equal(t, uint64(0x0102030405060708), endian.Uint64LE(buf))
	assert.Equal(t, uint64(0x0102030405060708), endian.LoadLE64(buf, 0))
	assert.Equal(t, uint32(0x05060708), endian.LoadLE32(buf, 0))
	assert.Equal(t, uint16(0x0708), endian.LoadLE16(buf, 0))
	assert.Equal(t, byte(0x08), endian.Load8(buf, 0))

	endian.StoreLE64(buf, 8, 0x1122334455667788)
	assert.Equal(t, uint64(0x1122334455667788), endian.LoadLE64(buf, 8))

	endian.StoreLE32(buf, 0xDEADBEEF)
	assert.Equal(t, uint32(0xDEADBEEF), endian.LoadLE32(buf, 0))

	endian.StoreLE16(buf, 0xCAFE)
	assert.Equal(t, uint16(0xCAFE), endian.LoadLE16(buf, 0))
}

func TestEndianBigEndian(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 16)
	endian.PutUint64BE(buf, 0x0102030405060708)

	assert.Equal(t, uint64(0x0102030405060708), endian.Uint64BE(buf))
	assert.Equal(t, uint64(0x0102030405060708), endian.LoadBE64(buf, 0))
	assert.Equal(t, uint32(0x01020304), endian.LoadBE32(buf, 0))
	assert.Equal(t, uint32(0x010203), endian.LoadBE24(buf, 0))
	assert.Equal(t, uint16(0x0102), endian.LoadBE16(buf, 0))

	endian.StoreBE64(buf, 8, 0x1122334455667788)
	assert.Equal(t, uint64(0x1122334455667788), endian.LoadBE64(buf, 8))

	endian.StoreBE32(buf, 0xDEADBEEF)
	assert.Equal(t, uint32(0xDEADBEEF), endian.LoadBE32(buf, 0))

	endian.StoreBE24(buf, 0x00AABBCC)
	assert.Equal(t, uint32(0x00AABBCC), endian.LoadBE24(buf, 0))

	endian.StoreBE16(buf, 0xCAFE)
	assert.Equal(t, uint16(0xCAFE), endian.LoadBE16(buf, 0))
}
