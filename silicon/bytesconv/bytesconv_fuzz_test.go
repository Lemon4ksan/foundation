// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytesconv_test

import (
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

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
