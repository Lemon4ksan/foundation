// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hexkit_test

import (
	"bytes"
	stdhex "encoding/hex"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/hexkit"
)

func FuzzHexEncodeDecode(f *testing.F) {
	f.Add([]byte("Hello, World!"))
	f.Add([]byte("0123456789abcdef"))
	f.Add([]byte(""))
	f.Add([]byte("\x00\xff\x12\x34\xfe\xdc"))

	f.Fuzz(func(t *testing.T, src []byte) {
		encodedStr := hexkit.EncodeToString(src)
		stdEncoded := stdhex.EncodeToString(src)

		if encodedStr != stdEncoded {
			t.Fatalf("EncodeToString mismatch: got %q, want %q", encodedStr, stdEncoded)
		}

		decoded, err := hexkit.DecodeString(encodedStr)
		if err != nil {
			t.Fatalf("DecodeString failed on valid encoded hex %q: %v", encodedStr, err)
		}

		if !bytes.Equal(decoded, src) {
			t.Fatalf("Roundtrip mismatch: got %x, want %x", decoded, src)
		}
	})
}

func FuzzHexDecodeArbitrary(f *testing.F) {
	f.Add("deadbeef")
	f.Add("DEADBEEF")
	f.Add("0123456789abcdef")
	f.Add("invalid_hex_string!")
	f.Add("123")
	f.Add("")

	f.Fuzz(func(t *testing.T, s string) {
		got, gotErr := hexkit.DecodeString(s)
		exp, expErr := stdhex.DecodeString(s)

		if (gotErr == nil) != (expErr == nil) {
			t.Fatalf("DecodeString error discrepancy on %q: gotErr=%v, expErr=%v", s, gotErr, expErr)
		}

		if gotErr == nil && !bytes.Equal(got, exp) {
			t.Fatalf("DecodeString mismatch on %q: got %x, want %x", s, got, exp)
		}
	})
}
