// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fse_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/fse"
)

func TestFSERoundtrip(t *testing.T) {
	data := bytes.Repeat([]byte("ABRAKADABRA_ALAKAZAM_SIMSALABIM"), 32)

	var s fse.Scratch
	compressed, err := fse.Compress(data, &s)
	if err != nil {
		t.Fatalf("fse.Compress failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Fatalf("compressed data is empty")
	}

	var decScratch fse.Scratch
	decScratch.DecompressLimit = len(data)
	decompressed, err := fse.Decompress(compressed, &decScratch)
	if err != nil {
		t.Fatalf("fse.Decompress failed: %v", err)
	}

	if len(decompressed) < len(data) || !bytes.Equal(decompressed[:len(data)], data) {
		t.Fatalf("Decompress prefix mismatch: got %d bytes, want %d bytes", len(decompressed), len(data))
	}
}
