// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package huff0

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// TestAdversarialHuff0RandomBitstreams feeds arbitrary random bytes to decoders.
func TestAdversarialHuff0RandomBitstreams(t *testing.T) {
	lengths := []int{0, 1, 2, 3, 4, 7, 8, 15, 16, 31, 32, 63, 64, 128, 256, 512, 1024, 2048}
	s := &Scratch{MaxSymbolValue: 255}

	for _, sz := range lengths {
		for iter := 0; iter < 10; iter++ {
			b := make([]byte, sz)
			_, _ = rand.Read(b)

			// ReadTable must not panic on corrupt/random data
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("ReadTable panicked on random size %d: %v", sz, r)
					}
				}()
				_, _, _ = ReadTable(b, s)
			}()

			// Decompress1X must not panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("Decompress1X panicked on random size %d: %v", sz, r)
					}
				}()
				_, _ = s.Decompress1X(b)
			}()

			// Decompress4X must not panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("Decompress4X panicked on random size %d: %v", sz, r)
					}
				}()
				_, _ = s.Decompress4X(b, sz*4)
			}()
		}
	}
}

// TestAdversarialHuff0CorruptHeaders tests crafted malformed headers.
func TestAdversarialHuff0CorruptHeaders(t *testing.T) {
	headers := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"Truncated1", []byte{0x00}},
		{"Truncated2", []byte{0x00, 0x01}},
		{"Truncated3", []byte{0x00, 0x01, 0x02}},
		{"AllZeroes8", make([]byte, 8)},
		{"AllOnes8", bytes.Repeat([]byte{0xFF}, 8)},
		{"WeightZero", []byte{0x02, 0x00, 0x00}},
		{"LargeWeight", []byte{0x02, 0xFF, 0x11}},
	}

	s := &Scratch{MaxSymbolValue: 255}
	for _, tc := range headers {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("ReadTable panicked on %s: %v", tc.name, r)
				}
			}()
			_, _, _ = ReadTable(tc.data, s)
		})
	}
}

// TestAdversarialHuff0StreamTruncation compresses data and truncates at every byte boundary.
func TestAdversarialHuff0StreamTruncation(t *testing.T) {
	src := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. 1234567890"), 20)

	// 1X compression truncation test
	s1 := &Scratch{MaxSymbolValue: 255}
	c1, _, err := Compress1X(src, s1)
	if err != nil {
		t.Fatalf("Compress1X failed: %v", err)
	}

	for i := 0; i < len(c1); i++ {
		truncated := c1[:i]
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Decompress1X panicked on truncated byte %d / %d: %v", i, len(c1), r)
				}
			}()
			sDec := &Scratch{MaxSymbolValue: 255}
			_, _ = sDec.Decompress1X(truncated)
		}()
	}

	// 4X compression truncation test
	s4 := &Scratch{MaxSymbolValue: 255}
	c4, _, err := Compress4X(src, s4)
	if err != nil {
		t.Fatalf("Compress4X failed: %v", err)
	}

	for i := 0; i < len(c4); i++ {
		truncated := c4[:i]
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Decompress4X panicked on truncated byte %d / %d: %v", i, len(c4), r)
				}
			}()
			sDec := &Scratch{MaxSymbolValue: 255}
			_, _ = sDec.Decompress4X(truncated, len(src))
		}()
	}
}
