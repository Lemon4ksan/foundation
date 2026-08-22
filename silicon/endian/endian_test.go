// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package endian_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lemon4ksan/foundation/silicon/endian"
)

func TestEndianLoadsAndStores(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 16)
	endian.PutUint64(buf, 0x0102030405060708)

	assert.Equal(t, uint64(0x0102030405060708), endian.Uint64(buf))
	assert.Equal(t, uint64(0x0102030405060708), endian.Load64(buf, 0))
	assert.Equal(t, uint32(0x05060708), endian.Load32(buf, 0))
	assert.Equal(t, uint16(0x0708), endian.Load16(buf, 0))
	assert.Equal(t, byte(0x08), endian.Load8(buf, 0))

	endian.Store64(buf, 8, 0x1122334455667788)
	assert.Equal(t, uint64(0x1122334455667788), endian.Load64(buf, 8))

	endian.Store32(buf, 0xDEADBEEF)
	assert.Equal(t, uint32(0xDEADBEEF), endian.Load32(buf, 0))

	endian.Store16(buf, 0xCAFE)
	assert.Equal(t, uint16(0xCAFE), endian.Load16(buf, 0))
}
