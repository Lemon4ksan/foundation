// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flate

import (
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
)

func TestRepro_DictDecoder_WildcopyBufferOverflow_8Bytes(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 64+16)
	canary := []byte("CANARY_SAFE_BYTE")
	copy(buf[64:], canary)

	var dd dictDecoder
	dd.hist = buf[:64]
	dd.wrPos = 60 // 4 bytes remaining until len(dd.hist) == 64
	copy(dd.hist[40:60], []byte("0123456789abcdefghij"))

	// Copy 4 bytes from distance 10 (srcPos = 50, endPos = 64)
	copied := dd.tryWriteCopy(10, 4)
	assert.Equal(t, 4, copied)

	// Invariant: Memory beyond dd.hist (the canary) must never be modified
	assert.Equal(t, string(canary), string(buf[64:]), "canary bytes after buffer boundary were overwritten by wildcopy!")
}

func TestRepro_DictDecoder_WildcopyBufferOverflow_16Bytes(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 64+16)
	canary := []byte("CANARY_SAFE_BYTE")
	copy(buf[64:], canary)

	var dd dictDecoder
	dd.hist = buf[:64]
	dd.wrPos = 52 // 12 bytes remaining
	copy(dd.hist[20:52], []byte("0123456789abcdefghijklmnopqrstuv"))

	// Copy 10 bytes from distance 20 (srcPos = 32, endPos = 62 <= 64)
	// Triggers length <= 16 && dist >= 16 branch
	copied := dd.tryWriteCopy(20, 10)
	assert.Equal(t, 10, copied)

	// Invariant: Memory beyond dd.hist (the canary) must never be modified
	assert.Equal(t, string(canary), string(buf[64:]), "canary bytes after buffer boundary were overwritten by 16-byte wildcopy!")
}
