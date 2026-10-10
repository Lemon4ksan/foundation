// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hexkit_test

import (
	"bytes"
	stdhex "encoding/hex"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/hexkit"
)

func TestEncodeDecode(t *testing.T) {
	data := []byte("Hello, Silicon World! 1234567890")
	encoded := hexkit.EncodeToString(data)
	expected := stdhex.EncodeToString(data)

	if encoded != expected {
		t.Fatalf("expected %s, got %s", expected, encoded)
	}

	decoded, err := hexkit.DecodeString(encoded)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if !bytes.Equal(decoded, data) {
		t.Fatalf("expected %s, got %s", data, decoded)
	}
}

func TestFixed16And8(t *testing.T) {
	src16 := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	var dst32 [32]byte
	hexkit.Encode16(&dst32, src16)

	expected32 := stdhex.EncodeToString(src16[:])
	if string(dst32[:]) != expected32 {
		t.Fatalf("expected %s, got %s", expected32, string(dst32[:]))
	}

	var decoded16 [16]byte
	if !hexkit.Decode32(&decoded16, expected32) {
		t.Fatalf("failed to decode 32-char hex string")
	}
	if decoded16 != src16 {
		t.Fatalf("decoded 16 mismatch")
	}

	src8 := [8]byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x11, 0x22}
	var dst16 [16]byte
	hexkit.Encode8(&dst16, src8)

	expected16 := stdhex.EncodeToString(src8[:])
	if string(dst16[:]) != expected16 {
		t.Fatalf("expected %s, got %s", expected16, string(dst16[:]))
	}

	var decoded8 [8]byte
	if !hexkit.Decode16(&decoded8, expected16) {
		t.Fatalf("failed to decode 16-char hex string")
	}
	if decoded8 != src8 {
		t.Fatalf("decoded 8 mismatch")
	}
}

func TestDecodeErrors(t *testing.T) {
	_, err := hexkit.DecodeString("abc")
	if err != hexkit.ErrInvalidLength {
		t.Fatalf("expected ErrInvalidLength, got %v", err)
	}

	_, err = hexkit.DecodeString("abcz")
	if err != hexkit.ErrInvalidByte {
		t.Fatalf("expected ErrInvalidByte, got %v", err)
	}
}

func TestEncodeDecodeVariousLengths(t *testing.T) {
	lengths := []int{
		0, 1, 2, 7, 8, 15, 16, 17, 23, 24, 31, 32, 33,
		47, 48, 63, 64, 65, 127, 128, 255, 256, 511, 512, 1024, 4096,
	}
	for _, l := range lengths {
		src := make([]byte, l)
		for i := range src {
			src[i] = byte((i*31 + 7) & 0xff)
		}

		dst := make([]byte, hexkit.EncodedLen(l))
		n := hexkit.Encode(dst, src)
		if n != hexkit.EncodedLen(l) {
			t.Fatalf("len %d: expected %d bytes encoded, got %d", l, hexkit.EncodedLen(l), n)
		}

		expected := stdhex.EncodeToString(src)
		if string(dst) != expected {
			t.Fatalf("len %d: encoded mismatch: got %s, want %s", l, string(dst), expected)
		}

		decoded := make([]byte, l)
		dn, err := hexkit.Decode(decoded, dst)
		if err != nil {
			t.Fatalf("len %d: unexpected decode error: %v", l, err)
		}
		if dn != l {
			t.Fatalf("len %d: expected %d bytes decoded, got %d", l, l, dn)
		}
		if !bytes.Equal(decoded, src) {
			t.Fatalf("len %d: roundtrip mismatch", l)
		}
	}
}

func TestUnalignedDestination(t *testing.T) {
	// Allocate a buffer and write into an unaligned offset (+1) to verify no alignment faults / SIGBUS.
	src := []byte("The quick brown fox jumps over the lazy dog. 1234567890!@#$%^&*()")
	needed := hexkit.EncodedLen(len(src))

	rawDst := make([]byte, needed+3)
	unalignedDst := rawDst[1 : 1+needed]

	n := hexkit.Encode(unalignedDst, src)
	if n != needed {
		t.Fatalf("expected %d, got %d", needed, n)
	}

	expected := stdhex.EncodeToString(src)
	if string(unalignedDst) != expected {
		t.Fatalf("unaligned encode mismatch")
	}

	rawDec := make([]byte, len(src)+3)
	unalignedDec := rawDec[1 : 1+len(src)]

	dn, err := hexkit.Decode(unalignedDec, unalignedDst)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if dn != len(src) {
		t.Fatalf("expected %d, got %d", len(src), dn)
	}
	if !bytes.Equal(unalignedDec, src) {
		t.Fatalf("unaligned decode mismatch")
	}
}

func BenchmarkSiliconHexEncode16(b *testing.B) {
	src := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	var dst [32]byte
	b.ReportAllocs()
	for b.Loop() {
		hexkit.Encode16(&dst, src)
	}
}

func BenchmarkStdHexEncode16(b *testing.B) {
	src := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	var dst [32]byte
	b.ReportAllocs()
	for b.Loop() {
		stdhex.Encode(dst[:], src[:])
	}
}

func BenchmarkSiliconHexDecode32(b *testing.B) {
	s := "0102030405060708090a0b0c0d0e0f10"
	var out [16]byte
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		hexkit.Decode32(&out, s)
	}
}

func BenchmarkStdHexDecode32(b *testing.B) {
	s := "0102030405060708090a0b0c0d0e0f10"
	var out [16]byte
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		stdhex.Decode(out[:], []byte(s))
	}
}

func BenchmarkSiliconHexEncode1KB(b *testing.B) {
	data := make([]byte, 1024)
	out := make([]byte, 2048)
	b.SetBytes(1024)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		hexkit.Encode(out, data)
	}
}

func BenchmarkStdHexEncode1KB(b *testing.B) {
	data := make([]byte, 1024)
	out := make([]byte, 2048)
	b.SetBytes(1024)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		stdhex.Encode(out, data)
	}
}
