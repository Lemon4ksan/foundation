// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fse

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// TestAdversarialFSERandomBitstreams tests that feeding random bitstreams never panics.
func TestAdversarialFSERandomBitstreams(t *testing.T) {
	lengths := []int{0, 1, 2, 3, 4, 7, 8, 15, 16, 31, 32, 63, 64, 127, 128, 255, 256, 512, 1024, 4096}
	scratch := &Scratch{DecompressLimit: 65536}

	for _, sz := range lengths {
		for iter := 0; iter < 10; iter++ {
			b := make([]byte, sz)
			_, _ = rand.Read(b)

			// Decompress must either return an error or return decoded bytes without panicking
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("Decompress panicked on random input size %d: %v", sz, r)
					}
				}()
				_, _ = Decompress(b, scratch)
			}()
		}
	}
}

// TestAdversarialFSECorruptHeaders tests crafted corrupt table headers.
func TestAdversarialFSECorruptHeaders(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"Truncated1", []byte{0x00}},
		{"Truncated2", []byte{0x00, 0x01}},
		{"Truncated3", []byte{0x00, 0x01, 0x02}},
		{"TableLogTooLarge", []byte{0x0F, 0xFF, 0xFF, 0xFF}}, // 0x0F + minTablelog(5) = 20 > 15
		{"ZeroTableLogBits", []byte{0x00, 0x00, 0x00, 0x00}},
		{"AllOnes4B", []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{"AllZeros16B", make([]byte, 16)},
		{"AllOnes16B", bytes.Repeat([]byte{0xFF}, 16)},
	}

	scratch := &Scratch{DecompressLimit: 65536}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Decompress panicked on %s: %v", tc.name, r)
				}
			}()
			_, _ = Decompress(tc.data, scratch)
		})
	}
}

// TestAdversarialFSEStreamTruncation compresses valid data and truncates at every byte boundary.
func TestAdversarialFSEStreamTruncation(t *testing.T) {
	// Sample data with non-trivial symbol distribution
	src := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. 1234567890"), 10)
	compressed, err := Compress(src, nil)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	scratch := &Scratch{DecompressLimit: len(src) * 2}
	for i := 0; i < len(compressed); i++ {
		truncated := compressed[:i]
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Decompress panicked on truncated stream at byte %d / %d: %v", i, len(compressed), r)
				}
			}()
			_, _ = Decompress(truncated, scratch)
		}()
	}
}
