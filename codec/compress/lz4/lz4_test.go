// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Copyright (c) 2015 Pierre Curto All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lz4_test

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/lz4"
)

func TestLZ4_BlockRoundtrip(t *testing.T) {
	sizes := []int{0, 1, 10, 128, 1024, 65536, 131072}
	for _, sz := range sizes {
		t.Run(fmt.Sprintf("size_%d", sz), func(t *testing.T) {
			raw := bytes.Repeat([]byte("Hello, LZ4 high-speed compression in Foundation! "), (sz/45)+1)
			raw = raw[:sz]

			maxBound := lz4.CompressBlockBound(len(raw))
			dst := make([]byte, maxBound)

			n, err := lz4.CompressBlock(raw, dst)
			if err != nil {
				t.Fatalf("CompressBlock failed: %v", err)
			}

			decomp := make([]byte, len(raw))
			m, err := lz4.UncompressBlock(dst[:n], decomp)
			if err != nil {
				t.Fatalf("UncompressBlock failed: %v", err)
			}
			if m != len(raw) {
				t.Fatalf("uncompressed size mismatch: got %d, want %d", m, len(raw))
			}
			if !bytes.Equal(raw, decomp) {
				t.Fatalf("decompressed data mismatch")
			}
		})
	}
}

func TestLZ4_FrameStreamingRoundtrip(t *testing.T) {
	r := rand.New(rand.NewSource(12345))
	payload := make([]byte, 256*1024)
	r.Read(payload)

	var buf bytes.Buffer
	w := lz4.NewWriter(&buf)
	if _, err := w.Write(payload); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	reader := lz4.NewReader(&buf)
	decomp, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(payload, decomp) {
		t.Fatalf("streamed roundtrip mismatch")
	}
}

func TestLZ4_CompressionLevels(t *testing.T) {
	text := bytes.Repeat([]byte("EON Container Format with Prolly Tree and LZ4 fast mounting\n"), 100)
	levels := []lz4.CompressionLevel{lz4.Fast, lz4.Level1, lz4.Level5, lz4.Level9}

	for _, lvl := range levels {
		t.Run(fmt.Sprintf("level_%v", lvl), func(t *testing.T) {
			var buf bytes.Buffer
			w := lz4.NewWriter(&buf)
			_ = w.Apply(lz4.CompressionLevelOption(lvl))
			if _, err := w.Write(text); err != nil {
				t.Fatalf("Write level %v failed: %v", lvl, err)
			}
			if err := w.Close(); err != nil {
				t.Fatalf("Close level %v failed: %v", lvl, err)
			}

			r := lz4.NewReader(&buf)
			res, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("Read level %v failed: %v", lvl, err)
			}
			if !bytes.Equal(text, res) {
				t.Fatalf("level %v mismatch", lvl)
			}
		})
	}
}
