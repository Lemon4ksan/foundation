// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race

package bcj_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/codec/filter/bcj"
)

func TestFilter_Allocs(t *testing.T) {
	data := []byte{0xE8, 0x05, 0x00, 0x00, 0x00}
	allocs := testing.AllocsPerRun(10, func() {
		bcj.Filter(bcj.X86, data, 0, true)
	})
	if allocs > 0 {
		t.Errorf("Filter allocated %v times, want 0", allocs)
	}
}
