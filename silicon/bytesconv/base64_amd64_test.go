//go:build !race

package bytesconv_test

// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"strings"
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

func FuzzEqualFoldASCII(f *testing.F) {
	f.Add("Content-Type", "content-type")
	f.Add("Authorization", "AUTHORIZATION")
	f.Add("HTTP/2.0", "http/2.0")
	f.Add("hello", "world")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, a, b string) {
		got := bytesconv.EqualFoldASCII(a, b)
		expected := strings.EqualFold(a, b)

		// strings.EqualFold handles Unicode, EqualFoldASCII is strictly ASCII
		isASCII := true
		for i := 0; i < len(a); i++ {
			if a[i] >= 128 {
				isASCII = false
				break
			}
		}
		for i := 0; i < len(b); i++ {
			if b[i] >= 128 {
				isASCII = false
				break
			}
		}

		if isASCII && got != expected {
			t.Fatalf("EqualFoldASCII mismatch on %q vs %q: got %v, want %v", a, b, got, expected)
		}
	})
}

func FuzzAppendToLower(f *testing.F) {
	f.Add([]byte("Hello, World 123!"))
	f.Add([]byte("ALREADY LOWER"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, src []byte) {
		dst := bytesconv.AppendToLower(nil, src)
		if len(dst) != len(src) {
			t.Fatalf("length mismatch: got %d, want %d", len(dst), len(src))
		}
	})
}
