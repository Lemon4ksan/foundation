//go:build !race

// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package casing_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/text/casing"
)

func TestAllocations_ToSnake(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_ = casing.ToSnake("PascalCaseExample")
	})
	if allocs > 0 {
		t.Logf("ToSnake allocated %v times", allocs)
	}
}
