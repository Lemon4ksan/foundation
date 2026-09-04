// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shuffle_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/lemon4ksan/foundation/codec/filter/shuffle"
)

func TestShuffle_Roundtrip(t *testing.T) {
	widths := []int{1, 2, 3, 4, 8, 16}
	lengths := []int{0, 1, 2, 3, 4, 7, 8, 15, 16, 17, 100, 1024, 8192, 8195}

	r := rand.New(rand.NewSource(42))

	for _, w := range widths {
		for _, l := range lengths {
			orig := make([]byte, l)
			r.Read(orig)

			enc, err := shuffle.Encode(orig, nil, w)
			if err != nil {
				t.Fatalf("Encode failed width=%d len=%d: %v", w, l, err)
			}
			if len(enc) != len(orig) {
				t.Fatalf("length mismatch: got %d, want %d", len(enc), len(orig))
			}

			dec, err := shuffle.Decode(enc, nil, w)
			if err != nil {
				t.Fatalf("Decode failed width=%d len=%d: %v", w, l, err)
			}
			if !bytes.Equal(orig, dec) {
				t.Fatalf("roundtrip mismatch width=%d len=%d", w, l)
			}
		}
	}
}

func TestShuffle_FP32Clustering(t *testing.T) {
	// 4 float32 numbers: 1.0, 1.1, 1.2, 1.3
	// Their IEEE-754 binary encodings will share similar exponent bytes (0x3F)
	// Shuffling should place all high bytes contiguously.
	src := []byte{
		0x00, 0x00, 0x80, 0x3F, // 1.0f
		0xCD, 0xCC, 0x8C, 0x3F, // 1.1f
		0x9A, 0x99, 0x99, 0x3F, // 1.2f
		0x66, 0x66, 0xA6, 0x3F, // 1.3f
	}

	shuffled, err := shuffle.Encode(src, nil, 4)
	if err != nil {
		t.Fatal(err)
	}

	// Bytes 12..15 of shuffled should all be 0x3F (the exponent byte)
	expBytes := shuffled[12:16]
	for i, b := range expBytes {
		if b != 0x3F {
			t.Fatalf("expected exponent byte 0x3F at %d, got 0x%02X", i, b)
		}
	}

	unshuffled, err := shuffle.Decode(shuffled, nil, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src, unshuffled) {
		t.Fatalf("unshuffled data did not match original")
	}
}
