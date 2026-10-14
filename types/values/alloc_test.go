// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race

package values_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/types/values"
)

func TestValues_Allocations(t *testing.T) {
	data := []byte(`"12345"`)
	var v values.Uint64String
	allocs := testing.AllocsPerRun(100, func() {
		_ = v.UnmarshalJSON(data)
	})
	if allocs > 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}

	var bi values.BoolInt = true
	allocsBool := testing.AllocsPerRun(100, func() {
		_, _ = bi.MarshalJSON()
	})
	if allocsBool > 0 {
		t.Errorf("expected 0 allocations for BoolInt.MarshalJSON, got %v", allocsBool)
	}

	var i values.Int64String
	allocsInt := testing.AllocsPerRun(100, func() {
		_ = i.UnmarshalJSON(data)
	})
	if allocsInt > 0 {
		t.Errorf("expected 0 allocations for Int64String.UnmarshalJSON, got %v", allocsInt)
	}
}
