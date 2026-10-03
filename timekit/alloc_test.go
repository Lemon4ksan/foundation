//go:build !race

// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timekit_test

import (
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/timekit"
)

func TestAllocations_AppendHTTPDate(t *testing.T) {
	var buf [64]byte
	now := time.Now()
	allocs := testing.AllocsPerRun(100, func() {
		_ = timekit.AppendHTTPDate(buf[:0], now)
	})
	if allocs > 0 {
		t.Errorf("AppendHTTPDate allocated %v times, expected 0", allocs)
	}
}
