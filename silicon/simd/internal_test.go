// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd

import (
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
)

func TestSWAR_Direct(t *testing.T) {
	val := uint64(0b10101010)
	mask := uint64(0b11110000)

	ext := extractBitsSWAR(val, mask)
	assert.Equal(t, uint64(0b1010), ext)

	dep := depositBitsSWAR(ext, mask)
	assert.Equal(t, uint64(0b10100000), dep)

	clz := CountLeadingZeros(0x00F0000000000000)
	assert.Equal(t, 8, clz)
}
