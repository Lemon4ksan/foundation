// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hex_test

import (
	"bytes"
	stdhex "encoding/hex"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/hex"
)

func TestEncodeDecode(t *testing.T) {
	data := []byte("Hello, Silicon World! 1234567890")
	encoded := hex.EncodeToString(data)
	expected := stdhex.EncodeToString(data)

	if encoded != expected {
		t.Fatalf("expected %s, got %s", expected, encoded)
	}

	decoded, err := hex.DecodeString(encoded)
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
	hex.Encode16(&dst32, src16)

	expected32 := stdhex.EncodeToString(src16[:])
	if string(dst32[:]) != expected32 {
		t.Fatalf("expected %s, got %s", expected32, string(dst32[:]))
	}

	var decoded16 [16]byte
	if !hex.Decode32(&decoded16, expected32) {
		t.Fatalf("failed to decode 32-char hex string")
	}
	if decoded16 != src16 {
		t.Fatalf("decoded 16 mismatch")
	}

	src8 := [8]byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x11, 0x22}
	var dst16 [16]byte
	hex.Encode8(&dst16, src8)

	expected16 := stdhex.EncodeToString(src8[:])
	if string(dst16[:]) != expected16 {
		t.Fatalf("expected %s, got %s", expected16, string(dst16[:]))
	}

	var decoded8 [8]byte
	if !hex.Decode16(&decoded8, expected16) {
		t.Fatalf("failed to decode 16-char hex string")
	}
	if decoded8 != src8 {
		t.Fatalf("decoded 8 mismatch")
	}
}

func TestDecodeErrors(t *testing.T) {
	_, err := hex.DecodeString("abc")
	if err != hex.ErrInvalidLength {
		t.Fatalf("expected ErrInvalidLength, got %v", err)
	}

	_, err = hex.DecodeString("abcz")
	if err != hex.ErrInvalidByte {
		t.Fatalf("expected ErrInvalidByte, got %v", err)
	}
}

func BenchmarkSiliconHexEncode16(b *testing.B) {
	src := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	var dst [32]byte
	b.ReportAllocs()
	for b.Loop() {
		hex.Encode16(&dst, src)
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
	for i := 0; i < b.N; i++ {
		hex.Decode32(&out, s)
	}
}

func BenchmarkStdHexDecode32(b *testing.B) {
	s := "0102030405060708090a0b0c0d0e0f10"
	var out [16]byte
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		stdhex.Decode(out[:], []byte(s))
	}
}

func BenchmarkSiliconHexEncode1KB(b *testing.B) {
	data := make([]byte, 1024)
	out := make([]byte, 2048)
	b.SetBytes(1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hex.Encode(out, data)
	}
}

func BenchmarkStdHexEncode1KB(b *testing.B) {
	data := make([]byte, 1024)
	out := make([]byte, 2048)
	b.SetBytes(1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		stdhex.Encode(out, data)
	}
}
