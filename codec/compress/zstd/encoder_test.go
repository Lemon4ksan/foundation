// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zstd

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
)

func TestZstd_Encoder_Roundtrip(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"SmallString", []byte("Hello, Zstandard RFC 8878 native compression engine in Seal!")},
		{"SingleByte", []byte{'A'}},
		{"RLE_1KB", bytes.Repeat([]byte{'X'}, 1024)},
		{"RLE_128KB", bytes.Repeat([]byte{0x55}, 128*1024)},
		{"MultipleBlocks_300KB", func() []byte {
			buf := make([]byte, 300*1024)
			_, _ = rand.Read(buf)
			return buf
		}()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var compressed bytes.Buffer
			w, err := NewWriter(&compressed, WithEncoderChecksum(true))
			if err != nil {
				t.Fatalf("NewWriter failed: %v", err)
			}

			n, err := w.Write(tc.data)
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}
			if n != len(tc.data) {
				t.Fatalf("Write count mismatch: got %d, want %d", n, len(tc.data))
			}

			if err := w.Close(); err != nil {
				t.Fatalf("Close failed: %v", err)
			}

			// Decompress using our native zstd.Decoder
			r, err := NewReader(&compressed)
			if err != nil {
				t.Fatalf("NewReader failed: %v", err)
			}
			defer r.Close()

			decompressed, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("ReadAll failed: %v", err)
			}

			if !bytes.Equal(decompressed, tc.data) {
				t.Fatalf("Payload mismatch for %s: got %d bytes, want %d bytes", tc.name, len(decompressed), len(tc.data))
			}
		})
	}
}
