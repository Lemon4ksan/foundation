// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enctest

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/text/encoding"
	"github.com/lemon4ksan/foundation/text/transform"
)

// Encoder or Decoder
type Transcoder interface {
	transform.Transformer
	Bytes([]byte) ([]byte, error)
	String(string) (string, error)
}

func TestEncoding(t *testing.T, e encoding.Encoding, encoded, utf8, prefix, suffix string) {
	for _, direction := range []string{"Decode", "Encode"} {
		t.Run(fmt.Sprintf("%v/%s", e, direction), func(t *testing.T) {

			var coder Transcoder
			var want, src, wPrefix, sPrefix, wSuffix, sSuffix string
			if direction == "Decode" {
				coder, want, src = e.NewDecoder(), utf8, encoded
				wPrefix, sPrefix, wSuffix, sSuffix = "", prefix, "", suffix
			} else {
				coder, want, src = e.NewEncoder(), encoded, utf8
				wPrefix, sPrefix, wSuffix, sSuffix = prefix, "", suffix, ""
			}

			dst := make([]byte, len(wPrefix)+len(want)+len(wSuffix))
			nDst, nSrc, err := coder.Transform(dst, []byte(sPrefix+src+sSuffix), true)
			if err != nil {
				t.Fatal(err)
			}
			if nDst != len(wPrefix)+len(want)+len(wSuffix) {
				t.Fatalf("nDst got %d, want %d",
					nDst, len(wPrefix)+len(want)+len(wSuffix))
			}
			if nSrc != len(sPrefix)+len(src)+len(sSuffix) {
				t.Fatalf("nSrc got %d, want %d",
					nSrc, len(sPrefix)+len(src)+len(sSuffix))
			}
			if got := string(dst); got != wPrefix+want+wSuffix {
				t.Fatalf("\ngot  %q\nwant %q", got, wPrefix+want+wSuffix)
			}

			for _, n := range []int{0, 1, 2, 10, 123, 4567} {
				input := sPrefix + strings.Repeat(src, n) + sSuffix
				g, err := coder.String(input)
				if err != nil {
					t.Fatalf("Bytes: n=%d: %v", n, err)
				}
				if len(g) == 0 && len(input) == 0 {
					// If the input is empty then the output can be empty,
					// regardless of whatever wPrefix is.
					continue
				}
				got1, want1 := g, wPrefix+strings.Repeat(want, n)+wSuffix
				if got1 != want1 {
					t.Fatalf("ReadAll: n=%d\ngot  %q\nwant %q",
						n, trim(got1), trim(want1))
				}
			}
		})
	}
}

func TestFile(t *testing.T, e encoding.Encoding) {
	sample := "Hello World! The quick brown fox jumps over the lazy dog. 1234567890."
	enc := e.NewEncoder()
	encoded, err := enc.String(sample)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	dec := e.NewDecoder()
	decoded, err := dec.String(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded != sample {
		t.Fatalf("roundtrip mismatch: got %q, want %q", decoded, sample)
	}
}

func Benchmark(b *testing.B, enc encoding.Encoding) {
	src := []byte("Hello World! The quick brown fox jumps over the lazy dog. 1234567890.")
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := transform.NewReader(bytes.NewReader(src), enc.NewDecoder())
		_, _ = io.Copy(io.Discard, r)
	}
}

func trim(s string) string {
	if len(s) < 120 {
		return s
	}
	return s[:50] + "..." + s[len(s)-50:]
}
