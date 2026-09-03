// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lzma

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
)

func TestLZMA1_Roundtrip(t *testing.T) {
	original := []byte("The quick brown fox jumps over the lazy dog. Repeat for higher compression ratio! " +
		"The quick brown fox jumps over the lazy dog. Repeat for higher compression ratio!")

	comp := NewCompressor(5)
	var compressed bytes.Buffer
	n, err := comp.Compress(bytes.NewReader(original), &compressed)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	if n != int64(len(original)) {
		t.Fatalf("Compress size mismatch: got %d, want %d", n, len(original))
	}

	decomp := NewDecompressor(0, uint64(len(original)))
	rc, err := decomp.Decompress(&compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	defer rc.Close()

	decoded, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(decoded, original) {
		t.Fatalf("LZMA1 roundtrip mismatch: got %d bytes, want %d bytes", len(decoded), len(original))
	}
}

func TestLZMA2_Roundtrip(t *testing.T) {
	original := []byte("Testing LZMA2 chunk-based streaming codec with dictionary property mapping. " +
		"Testing LZMA2 chunk-based streaming codec with dictionary property mapping.")

	comp := NewCompressor2(1 << 20)
	var compressed bytes.Buffer
	n, err := comp.Compress(bytes.NewReader(original), &compressed)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	if n != int64(len(original)) {
		t.Fatalf("Compress size mismatch: got %d, want %d", n, len(original))
	}

	decomp := NewDecompressor2(1 << 20)
	rc, err := decomp.Decompress(&compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	defer rc.Close()

	decoded, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(decoded, original) {
		t.Fatalf("LZMA2 roundtrip mismatch")
	}
}

func TestLZMA2_RandomData(t *testing.T) {
	r := rand.New(rand.NewSource(12345))
	original := make([]byte, 64*1024)
	r.Read(original)

	// Inject recurring blocks
	for i := 1000; i < 50000; i += 2000 {
		copy(original[i:i+500], original[0:500])
	}

	comp := NewCompressor2(1 << 21)
	var compressed bytes.Buffer
	n, err := comp.Compress(bytes.NewReader(original), &compressed)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	if n != int64(len(original)) {
		t.Fatalf("Compress size mismatch: got %d, want %d", n, len(original))
	}

	decomp := NewDecompressor2(1 << 21)
	rc, err := decomp.Decompress(&compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	defer rc.Close()

	decoded, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(decoded, original) {
		for i := 0; i < min(len(decoded), len(original)); i++ {
			if decoded[i] != original[i] {
				t.Fatalf(
					"LZMA2 random data mismatch at offset %d: decoded=%x, original=%x (len decoded=%d, len orig=%d)",
					i,
					decoded[i],
					original[i],
					len(decoded),
					len(original),
				)
			}
		}
		t.Fatalf("LZMA2 random data roundtrip mismatch: len decoded=%d, len orig=%d", len(decoded), len(original))
	}
}

func TestDictProps(t *testing.T) {
	for i := byte(0); i <= 40; i++ {
		size, err := DictSizeFromProp(i)
		if err != nil {
			t.Fatalf("prop %d: %v", i, err)
		}
		p := PropFromDictSize(size)
		if p != i {
			t.Fatalf("prop roundtrip: input=%d, size=%d, got=%d", i, size, p)
		}
	}
}
