// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/simd"
)

// FuzzIndexByteSWAR asserts 100% equivalence between 64-bit SWAR vector scanning and stdlib bytes.IndexByte.
func FuzzIndexByteSWAR(f *testing.F) {
	f.Add([]byte("HTTP/1.1 200 OK\r\n"), byte('\r'))
	f.Add([]byte(""), byte('x'))
	f.Add([]byte("1234567890abcdef"), byte('f'))
	f.Add([]byte("aaaaaaaa"), byte('a'))
	f.Add([]byte("bbbbbbbb"), byte('z'))

	f.Fuzz(func(t *testing.T, data []byte, target byte) {
		got := simd.IndexByteSWAR(data, target)
		expected := bytes.IndexByte(data, target)

		if got != expected {
			t.Fatalf("IndexByteSWAR mismatch: got %d, expected %d on data %q, target %d", got, expected, data, target)
		}
	})
}

// FuzzIndexCRLF asserts 100% equivalence between SWAR CRLF scanning and stdlib bytes.Index.
func FuzzIndexCRLF(f *testing.F) {
	f.Add([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))
	f.Add([]byte("\r\n"))
	f.Add([]byte("\r"))
	f.Add([]byte("\n"))
	f.Add([]byte(""))
	f.Add([]byte("1234567\r\n890"))
	f.Add([]byte("12345678\r\n90"))
	f.Add([]byte("\r\r\r\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		got := simd.IndexCRLF(data)
		expected := bytes.Index(data, []byte("\r\n"))

		if got != expected {
			t.Fatalf("IndexCRLF mismatch: got %d, expected %d on data %q", got, expected, data)
		}
	})
}

// FuzzIndexDoubleCRLF verifies HTTP header terminator scanning against stdlib bytes.Index.
func FuzzIndexDoubleCRLF(f *testing.F) {
	f.Add([]byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"))
	f.Add([]byte("\r\n\r\n"))
	f.Add([]byte("\r\n\r\r\n\r\n"))
	f.Add([]byte(""))
	f.Add([]byte("header: val\r\n\r\nbody"))

	f.Fuzz(func(t *testing.T, data []byte) {
		got := simd.IndexDoubleCRLF(data)
		expected := bytes.Index(data, []byte("\r\n\r\n"))

		if got != expected {
			t.Fatalf("IndexDoubleCRLF mismatch: got %d, expected %d on data %q", got, expected, data)
		}
	})
}
