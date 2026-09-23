// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xxhash_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/zstd/xxhash"
)

// TestStandardVectors verifies standard 64-bit test vectors (seed = 0).
func TestStandardVectors(t *testing.T) {
	const s63 = "Call me Ishmael. Some years ago--never mind how long precisely-"
	testCases := []struct {
		name  string
		input string
		want  uint64
	}{
		{"Empty", "", 0xef46db3751d8e999},
		{"SingleA", "a", 0xd24ec4f1a98c6e5b},
		{"TwoChars", "as", 0x1c330fb2d66be179},
		{"ThreeChars", "asd", 0x631c37ce72a97393},
		{"FourChars", "asdf", 0x415872f599cea71e},
		{"Text63", s63, 0x02a2e85470d6fd96},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// One-shot Sum64
			gotSum := xxhash.Sum64([]byte(tc.input))
			if gotSum != tc.want {
				t.Fatalf("Sum64(%q) = 0x%016x; want 0x%016x", tc.input, gotSum, tc.want)
			}

			// Sum64String
			gotStr := xxhash.Sum64String(tc.input)
			if gotStr != tc.want {
				t.Fatalf("Sum64String(%q) = 0x%016x; want 0x%016x", tc.input, gotStr, tc.want)
			}

			// Streaming Digest
			d := xxhash.New()
			n, err := d.Write([]byte(tc.input))
			if err != nil || n != len(tc.input) {
				t.Fatalf("Digest.Write failed: (%d, %v)", n, err)
			}
			if d.Sum64() != tc.want {
				t.Fatalf("Digest.Sum64() = 0x%016x; want 0x%016x", d.Sum64(), tc.want)
			}

			// WriteString
			d2 := xxhash.New()
			n2, err2 := d2.WriteString(tc.input)
			if err2 != nil || n2 != len(tc.input) {
				t.Fatalf("Digest.WriteString failed: (%d, %v)", n2, err2)
			}
			if d2.Sum64() != tc.want {
				t.Fatalf("Digest.WriteString Sum64 = 0x%016x; want 0x%016x", d2.Sum64(), tc.want)
			}
		})
	}
}

// TestVariedTailLengths verifies tail handling across all modulo-32 byte lengths.
func TestVariedTailLengths(t *testing.T) {
	buf := make([]byte, 100)
	for i := range buf {
		buf[i] = byte(i*31 + 7)
	}

	for length := 0; length <= 96; length++ {
		sub := buf[:length]
		oneShot := xxhash.Sum64(sub)

		d := xxhash.New()
		d.Write(sub)
		streamed := d.Sum64()

		if streamed != oneShot {
			t.Fatalf("length %d mismatch: streamed 0x%x != one-shot 0x%x", length, streamed, oneShot)
		}
	}
}

// TestChunkingInvariance verifies that writing arbitrary chunk sizes yields identical output.
func TestChunkingInvariance(t *testing.T) {
	data := []byte("The quick brown fox jumps over the lazy dog. 1234567890!@#$%^&*()_+~`")
	want := xxhash.Sum64(data)

	for chunkSize := 1; chunkSize <= len(data); chunkSize++ {
		t.Run(fmt.Sprintf("chunk_%d", chunkSize), func(t *testing.T) {
			d := xxhash.New()
			for i := 0; i < len(data); i += chunkSize {
				end := i + chunkSize
				if end > len(data) {
					end = len(data)
				}
				d.Write(data[i:end])
			}
			if d.Sum64() != want {
				t.Fatalf("chunkSize %d produced 0x%x; want 0x%x", chunkSize, d.Sum64(), want)
			}
		})
	}
}

// TestHash64Interface verifies hash.Hash64 compliance (Size, BlockSize, Sum, Reset).
func TestHash64Interface(t *testing.T) {
	var _ hash.Hash64 = xxhash.New()

	d := xxhash.New()
	if d.Size() != 8 {
		t.Fatalf("Size() = %d; want 8", d.Size())
	}
	if d.BlockSize() != 32 {
		t.Fatalf("BlockSize() = %d; want 32", d.BlockSize())
	}

	data := []byte("testing hash interface")
	d.Write(data)
	expected64 := d.Sum64()

	sumBytes := d.Sum(nil)
	if len(sumBytes) != 8 {
		t.Fatalf("Sum(nil) length = %d; want 8", len(sumBytes))
	}
	if binary.BigEndian.Uint64(sumBytes) != expected64 {
		t.Fatalf("Sum(nil) bytes mismatch with Sum64()")
	}

	prefix := []byte("prefix-")
	prefixedSum := d.Sum(prefix)
	if !bytes.HasPrefix(prefixedSum, prefix) || len(prefixedSum) != len(prefix)+8 {
		t.Fatalf("Sum with prefix failed")
	}

	// Reset test
	d.Reset()
	if d.Sum64() != 0xef46db3751d8e999 {
		t.Fatalf("Sum64() after Reset() = 0x%x; want empty hash", d.Sum64())
	}
}

// TestMarshalUnmarshalBinary verifies binary state preservation and error branches.
func TestMarshalUnmarshalBinary(t *testing.T) {
	d := xxhash.New()
	d.Write([]byte("first segment of payload data"))

	marshaled, err := d.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	d2 := xxhash.New()
	if err := d2.UnmarshalBinary(marshaled); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	// Continue writing to both
	suffix := []byte(" and second segment")
	d.Write(suffix)
	d2.Write(suffix)

	if d.Sum64() != d2.Sum64() {
		t.Fatalf("Sum64 mismatch after UnmarshalBinary: 0x%x vs 0x%x", d.Sum64(), d2.Sum64())
	}

	// Error cases
	if err := d2.UnmarshalBinary([]byte{1, 2, 3}); err == nil {
		t.Fatalf("expected error for short unmarshal buffer")
	}
	badMagic := make([]byte, len(marshaled))
	copy(badMagic, marshaled)
	badMagic[0] = 0xFF
	if err := d2.UnmarshalBinary(badMagic); err == nil {
		t.Fatalf("expected error for bad magic")
	}
	badSize := make([]byte, len(marshaled)+5)
	copy(badSize, marshaled)
	if err := d2.UnmarshalBinary(badSize); err == nil {
		t.Fatalf("expected error for bad state size")
	}
}

// TestLargeInputExceedingAsmBlock exercises the chunking loop for inputs >= 128KB.
func TestLargeInputExceedingAsmBlock(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789ABCDEF0123456789abcdef"), 8192) // 256 KB
	h1 := xxhash.Sum64(data)

	d := xxhash.New()
	d.Write(data)
	if d.Sum64() != h1 {
		t.Fatalf("streaming large sum mismatch")
	}
}
