// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xxhash

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"hash"
	"testing"
)

// TestAdversarialXXHashContract validates the complete hash.Hash64 contract.
func TestAdversarialXXHashContract(t *testing.T) {
	var h hash.Hash64 = New()

	if h.Size() != 8 {
		t.Fatalf("Size() got %d, want 8", h.Size())
	}
	if h.BlockSize() != 32 {
		t.Fatalf("BlockSize() got %d, want 32", h.BlockSize())
	}

	// Empty digest verification
	emptySum := h.Sum(nil)
	if len(emptySum) != 8 {
		t.Fatalf("Sum(nil) length got %d, want 8", len(emptySum))
	}
	expectedEmpty64 := h.Sum64()
	if binary.BigEndian.Uint64(emptySum) != expectedEmpty64 {
		t.Fatalf("Sum(nil) %x != Sum64() %x", emptySum, expectedEmpty64)
	}

	// Multiple Sum calls must be idempotent
	prefix := []byte{0x01, 0x02, 0x03}
	appended := h.Sum(prefix)
	if !bytes.Equal(appended[:3], prefix) {
		t.Fatalf("Sum did not preserve prefix")
	}
	if !bytes.Equal(appended[3:], emptySum) {
		t.Fatalf("Sum append mismatch")
	}

	// Write and Reset verification
	sample := []byte("The quick brown fox jumps over the lazy dog")
	n, err := h.Write(sample)
	if err != nil || n != len(sample) {
		t.Fatalf("Write failed: n=%d, err=%v", n, err)
	}
	sampleSum := h.Sum64()
	if sampleSum == expectedEmpty64 {
		t.Fatalf("Hash did not change after Write")
	}

	h.Reset()
	if h.Sum64() != expectedEmpty64 {
		t.Fatalf("Reset() did not restore initial empty state: got %x, want %x", h.Sum64(), expectedEmpty64)
	}
}

// TestAdversarialXXHashStreamingChunkSizes validates chunk sizes from 1 to 64 bytes.
func TestAdversarialXXHashStreamingChunkSizes(t *testing.T) {
	data := make([]byte, 8192)
	_, _ = rand.Read(data)
	expected64 := Sum64(data)

	for chunkSize := 1; chunkSize <= 64; chunkSize++ {
		d := New()
		for pos := 0; pos < len(data); pos += chunkSize {
			end := pos + chunkSize
			if end > len(data) {
				end = len(data)
			}
			n, err := d.Write(data[pos:end])
			if err != nil || n != (end-pos) {
				t.Fatalf("chunkSize %d at pos %d: Write failed: %v", chunkSize, pos, err)
			}
		}

		got64 := d.Sum64()
		if got64 != expected64 {
			t.Fatalf("chunkSize %d mismatch: got %x, want %x", chunkSize, got64, expected64)
		}
	}
}

// TestAdversarialXXHashLargeInputs256KBPlus validates 256KB, 512KB, and 1MB inputs.
func TestAdversarialXXHashLargeInputs256KBPlus(t *testing.T) {
	sizes := []int{
		256 * 1024,
		256*1024 + 1,
		256*1024 + 31,
		256*1024 + 33,
		512 * 1024,
		1024 * 1024,
	}

	for _, sz := range sizes {
		data := make([]byte, sz)
		// Deterministic pseudo-random fill
		for i := range data {
			data[i] = byte((i * 109) ^ (i >> 3))
		}

		expectedOneshot := Sum64(data)
		expectedString := Sum64String(string(data))
		if expectedOneshot != expectedString {
			t.Fatalf("size %d: Sum64 %x != Sum64String %x", sz, expectedOneshot, expectedString)
		}

		// Streaming write in one single Write
		d1 := New()
		_, err := d1.Write(data)
		if err != nil {
			t.Fatalf("size %d: Write failed: %v", sz, err)
		}
		if d1.Sum64() != expectedOneshot {
			t.Fatalf("size %d: single Write Sum64 %x != expected %x", sz, d1.Sum64(), expectedOneshot)
		}

		// Streaming write in chunks crossing maxAsmSize (128KB)
		d2 := New()
		chunk := 65536
		for pos := 0; pos < len(data); pos += chunk {
			end := min(pos+chunk, len(data))
			_, _ = d2.Write(data[pos:end])
		}
		if d2.Sum64() != expectedOneshot {
			t.Fatalf("size %d: chunked Write Sum64 %x != expected %x", sz, d2.Sum64(), expectedOneshot)
		}
	}
}

// TestAdversarialXXHashBinaryMarshalRoundtrip tests state serialization across chunk boundaries.
func TestAdversarialXXHashBinaryMarshalRoundtrip(t *testing.T) {
	data := []byte("The quick brown fox jumps over the lazy dog and runs across the field")

	for split := 0; split <= len(data); split++ {
		d1 := New()
		_, _ = d1.Write(data[:split])

		marshaled, err := d1.MarshalBinary()
		if err != nil {
			t.Fatalf("split %d: MarshalBinary failed: %v", split, err)
		}

		d2 := New()
		err = d2.UnmarshalBinary(marshaled)
		if err != nil {
			t.Fatalf("split %d: UnmarshalBinary failed: %v", split, err)
		}

		// Finish writing the remainder to both
		_, _ = d1.Write(data[split:])
		_, _ = d2.Write(data[split:])

		if d1.Sum64() != d2.Sum64() {
			t.Fatalf("split %d: post-unmarshal sum %x != original sum %x", split, d2.Sum64(), d1.Sum64())
		}
	}

	// Corrupt binary state handling
	d := New()
	if err := d.UnmarshalBinary([]byte("short")); err == nil {
		t.Fatalf("expected error on truncated binary state")
	}
	corruptMagic := make([]byte, marshaledSize)
	copy(corruptMagic, []byte("bad\x00"))
	if err := d.UnmarshalBinary(corruptMagic); err == nil {
		t.Fatalf("expected error on invalid magic")
	}
}
