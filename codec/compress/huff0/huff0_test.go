// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package huff0_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/huff0"
)

func TestHuff0Roundtrip(t *testing.T) {
	data := bytes.Repeat([]byte("ABRAKADABRA_ALAKAZAM_SIMSALABIM"), 30)

	var s huff0.Scratch
	compressed, _, err := huff0.Compress1X(data, &s)
	if err != nil {
		t.Fatalf("huff0.Compress1X failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Fatalf("compressed data is empty")
	}

	var decScratch huff0.Scratch
	s2, remain, err := huff0.ReadTable(compressed, &decScratch)
	if err != nil {
		t.Fatalf("huff0.ReadTable failed: %v", err)
	}

	s2.MaxDecodedSize = len(data)
	decompressed, err := s2.Decompress1X(remain)
	if err != nil {
		t.Fatalf("s2.Decompress1X failed: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Fatalf("Decompress mismatch: got %d bytes, want %d bytes", len(decompressed), len(data))
	}
}
