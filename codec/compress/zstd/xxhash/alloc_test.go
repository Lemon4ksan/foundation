// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race

package xxhash_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/zstd/xxhash"
)

func TestAllocsPerRun_Sum64(t *testing.T) {
	sample := make([]byte, 1024)
	allocs := testing.AllocsPerRun(100, func() {
		_ = xxhash.Sum64(sample)
	})

	if allocs > 0 {
		t.Errorf("Sum64 allocated %f times per run, want 0", allocs)
	}
}
