// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race

package delta_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/codec/filter/delta"
)

func TestFilter_Allocs(t *testing.T) {
	filter, _ := delta.NewFilter(2)
	data := make([]byte, 1024)

	allocs := testing.AllocsPerRun(10, func() {
		filter.Encode(data)
		filter.Decode(data)
	})
	if allocs > 0 {
		t.Errorf("Encode/Decode allocated %v times, want 0", allocs)
	}
}
