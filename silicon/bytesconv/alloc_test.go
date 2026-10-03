// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race

package bytesconv_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

func TestAllocations_ScanTokensBytes(t *testing.T) {
	header := []byte("application/json; charset=utf-8; boundary=something")

	allocs := testing.AllocsPerRun(100, func() {
		for token := range bytesconv.ScanTokensBytes(header, ';') {
			_ = bytesconv.B2S(token)
		}
	})

	if allocs > 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}
}
