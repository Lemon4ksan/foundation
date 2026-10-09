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

func TestAllocations_ContainsFoldASCII(t *testing.T) {
	src := []byte("Content-Type: text/html; charset=UTF-8")
	target := "charset=utf-8"

	allocs := testing.AllocsPerRun(100, func() {
		_ = bytesconv.ContainsFoldASCII(src, target)
	})

	if allocs > 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}
}

func TestAllocations_FastHash64(t *testing.T) {
	data := []byte("zero-allocation hashing over unaligned payload")

	allocs := testing.AllocsPerRun(100, func() {
		_ = bytesconv.FastHash64(data[1:])
	})

	if allocs > 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}
}

func TestAllocations_PatternSlicer_SliceInto(t *testing.T) {
	slicer := bytesconv.NewPatternSlicer([]byte(":"), 1)
	data := []byte("Host: localhost")
	dst := make([][]byte, 2)

	allocs := testing.AllocsPerRun(100, func() {
		_, _ = slicer.SliceInto(data, dst)
	})

	if allocs > 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}
}
