// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race

package shuffle_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/codec/filter/shuffle"
)

func TestShuffle_Allocs(t *testing.T) {
	src := make([]byte, 1024)
	dst := make([]byte, 1024)

	allocs := testing.AllocsPerRun(10, func() {
		shuffle.Encode(src, dst, shuffle.WidthFP32)
		shuffle.Decode(dst, src, shuffle.WidthFP32)
	})
	if allocs > 0 {
		t.Errorf("Encode/Decode allocated %v times, want 0", allocs)
	}
}
