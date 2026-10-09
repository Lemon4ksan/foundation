// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/silicon/pool"
	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestRequestArena_AllocAndReset(t *testing.T) {
	arena := pool.AcquireRequestArena()
	defer pool.ReleaseRequestArena(arena)

	b1 := arena.Alloc(128)
	assert.Equal(t, 128, len(b1))

	b2 := arena.Alloc(256)
	assert.Equal(t, 256, len(b2))

	arena.Reset()
	b3 := arena.Alloc(64)
	assert.Equal(t, 64, len(b3))
}

func TestRequestArena_3IndexSliceCapacityBound(t *testing.T) {
	arena := pool.AcquireRequestArena()
	defer pool.ReleaseRequestArena(arena)

	testSizes := []int{1, 3, 7, 8, 9, 15, 16, 31, 32, 64, 127, 128, 512, 1024}
	for _, size := range testSizes {
		buf := arena.Alloc(size)
		assert.Equal(t, size, len(buf))
		assert.Equal(t, size, cap(buf))

		// Append to buf and ensure underlying memory reallocates without corrupting arena
		extended := append(buf, 0xFF)
		assert.Equal(t, size+1, len(extended))
	}
}
