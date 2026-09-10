// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool_test

import (
	"math"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/pool"
	"github.com/lemon4ksan/foundation/testkit/assert"
)

func TestRepro_RequestArena_IntegerOverflowGuard(t *testing.T) {
	t.Parallel()

	arena := pool.GetRequestArena()
	defer pool.ReleaseRequestArena(arena)

	// Allocating large value > 4096 must not panic and should not corrupt arena offset
	b := arena.Alloc(5000)
	assert.Equal(t, 5000, len(b))

	// Large n near MaxInt must not wrap around negatively inside arena and access a.buf
	defer func() {
		r := recover()
		t.Logf("caught panic: %v", r)
		assert.NotNil(t, r)
		assert.NotContains(
			t,
			r.(error).Error(),
			"4096",
			"integer overflow caused out-of-bounds slice indexing on a.buf!",
		)
	}()

	_ = arena.Alloc(math.MaxInt - 1)
}
