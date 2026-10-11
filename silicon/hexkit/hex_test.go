// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hexkit_test

import (
	"bytes"
	stdhex "encoding/hex"
	"math"
	"strconv"
	"strings"
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

func TestFormatHexUint(t *testing.T) {
	var buf [16]byte
	tests := []struct {
		val  int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{15, "f"},
		{16, "10"},
		{255, "ff"},
		{0x1A2B, "1a2b"},
		{0x1A2B3C, "1a2b3c"},
		{0x7FFFFFFFFFFFFFFF, "7fffffffffffffff"},
	}

	for _, tc := range tests {
		n := hexkit.FormatHexUint(&buf, tc.val)
		got := string(buf[:n])
		if got != tc.want {
			t.Errorf("FormatHexUint(%x) = %s; want %s", tc.val, got, tc.want)
		}
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on negative val")
		}
	}()
	hexkit.FormatHexUint(&buf, -1)
}

func TestParseHexUint(t *testing.T) {
	tests := []struct {
		input   string
		wantVal int
		wantN   int
		wantErr bool
	}{
		{"0", 0, 1, false},
		{"a", 10, 1, false},
		{"A", 10, 1, false},
		{"ff", 255, 2, false},
		{"1a2b", 0x1a2b, 4, false},
		{"1A2B", 0x1a2b, 4, false},
		{"1a2b;ext=foo", 0x1a2b, 4, false},
		{"00001a", 0x1a, 6, false},
		{"", 0, 0, true},
		{"xyz", 0, 0, true},
		{"7fffffffffffffff0", 0, 0, true}, // overflow
	}

	for _, tc := range tests {
		val, n, err := hexkit.ParseHexUint([]byte(tc.input))
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseHexUint(%q) err = %v, wantErr = %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr {
			if val != tc.wantVal || n != tc.wantN {
				t.Errorf("ParseHexUint(%q) = (%d, %d); want (%d, %d)", tc.input, val, n, tc.wantVal, tc.wantN)
			}
		}
	}
}

func BenchmarkFormatHexUint(b *testing.B) {
	var buf [16]byte
	b.ReportAllocs()
	for b.Loop() {
		_ = hexkit.FormatHexUint(&buf, 0x1A2B3C)
	}
}

func TestDecodeByte(t *testing.T) {
	tests := []struct {
		hi, lo byte
		wantB  byte
		wantOk bool
	}{
		{'0', '0', 0x00, true},
		{'0', '9', 0x09, true},
		{'a', 'f', 0xaf, true},
		{'A', 'F', 0xaf, true},
		{'7', 'e', 0x7e, true},
		{'1', 'A', 0x1a, true},
		{'F', 'F', 0xff, true},
		{'g', '0', 0x00, false},
		{'0', 'z', 0x00, false},
		{'/', '0', 0x00, false},
		{':', '0', 0x00, false},
		{'@', '0', 0x00, false},
		{'`', '0', 0x00, false},
		{0, 0, 0x00, false},
		{0xff, 0xff, 0x00, false},
		{'0', 0xff, 0x00, false},
		{0xff, '0', 0x00, false},
	}

	for _, tc := range tests {
		gotB, gotOk := hexkit.DecodeByte(tc.hi, tc.lo)
		if gotOk != tc.wantOk || gotB != tc.wantB {
			t.Errorf("DecodeByte(%q, %q) = (%x, %v); want (%x, %v)", tc.hi, tc.lo, gotB, gotOk, tc.wantB, tc.wantOk)
		}
	}

	// Exhaustive verification across all 256x256 byte pairs against standard library
	for hi := 0; hi < 256; hi++ {
		for lo := 0; lo < 256; lo++ {
			b, ok := hexkit.DecodeByte(byte(hi), byte(lo))
			var dec [1]byte
			_, err := stdhex.Decode(dec[:], []byte{byte(hi), byte(lo)})
			if ok != (err == nil) {
				t.Fatalf("DecodeByte(%02x, %02x) ok mismatch: got %v, want %v", hi, lo, ok, err == nil)
			}
			if ok && b != dec[0] {
				t.Fatalf("DecodeByte(%02x, %02x) val mismatch: got %02x, want %02x", hi, lo, b, dec[0])
			}
		}
	}

	allocs := testing.AllocsPerRun(100, func() {
		_, _ = hexkit.DecodeByte('a', 'b')
	})
	if allocs != 0 {
		t.Fatalf("DecodeByte allocated %f objects", allocs)
	}
}

func TestAppendHexUint(t *testing.T) {
	tests := []struct {
		val  int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{15, "f"},
		{16, "10"},
		{255, "ff"},
		{0x1A2B, "1a2b"},
		{0x1A2B3C, "1a2b3c"},
		{0x7FFFFFFFFFFFFFFF, "7fffffffffffffff"},
	}

	for _, tc := range tests {
		got := string(hexkit.AppendHexUint(nil, tc.val))
		if got != tc.want {
			t.Errorf("AppendHexUint(nil, %x) = %s; want %s", tc.val, got, tc.want)
		}
	}

	dst := []byte("prefix:")
	got := string(hexkit.AppendHexUint(dst, 0xABCD))
	if got != "prefix:abcd" {
		t.Errorf("AppendHexUint append mismatch: got %s; want %s", got, "prefix:abcd")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on negative val")
		}
	}()
	_ = hexkit.AppendHexUint(nil, -1)
}

func TestAppendHexUintAllocs(t *testing.T) {
	dst := make([]byte, 0, 32)
	allocs := testing.AllocsPerRun(100, func() {
		dst = hexkit.AppendHexUint(dst[:0], 0x1A2B3C)
	})
	if allocs != 0 {
		t.Fatalf("AppendHexUint allocated %f objects", allocs)
	}
}

func TestIsHex(t *testing.T) {
	for i := 0; i < 256; i++ {
		b := byte(i)
		expected := (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
		if got := hexkit.IsHex(b); got != expected {
			t.Fatalf("IsHex(%d / %q) = %v, want %v", b, b, got, expected)
		}
	}

	allocs := testing.AllocsPerRun(100, func() {
		_ = hexkit.IsHex('a')
	})
	if allocs != 0 {
		t.Fatalf("IsHex allocated %f objects", allocs)
	}
}

func TestUpperHexTableAndToHexUpper(t *testing.T) {
	if hexkit.UpperHexTable != "0123456789ABCDEF" {
		t.Fatalf("UpperHexTable mismatch: %q", hexkit.UpperHexTable)
	}

	for i := 0; i < 256; i++ {
		b := byte(i)
		hi, lo := hexkit.ToHexUpper(b)
		got := string([]byte{hi, lo})
		want := strings.ToUpper(stdhex.EncodeToString([]byte{b}))
		if got != want {
			t.Fatalf("ToHexUpper(%02x) = %s, want %s", b, got, want)
		}
	}

	allocs := testing.AllocsPerRun(100, func() {
		_, _ = hexkit.ToHexUpper(0xAB)
	})
	if allocs != 0 {
		t.Fatalf("ToHexUpper allocated %f objects", allocs)
	}
}

func TestAppendEncode(t *testing.T) {
	// Empty src
	dst := hexkit.AppendEncode([]byte("header:"), nil)
	if string(dst) != "header:" {
		t.Fatalf("AppendEncode empty src mismatch: %q", string(dst))
	}

	lengths := []int{0, 1, 2, 7, 8, 15, 16, 23, 32, 64, 128, 512, 1024}
	for _, l := range lengths {
		src := make([]byte, l)
		for i := range src {
			src[i] = byte((i*37 + 13) & 0xff)
		}
		got := hexkit.AppendEncode([]byte("init-"), src)
		want := "init-" + stdhex.EncodeToString(src)
		if string(got) != want {
			t.Fatalf("AppendEncode len %d mismatch", l)
		}
	}

	buf := make([]byte, 0, 1024)
	src := []byte("zero-allocation-check")
	allocs := testing.AllocsPerRun(100, func() {
		buf = hexkit.AppendEncode(buf[:0], src)
	})
	if allocs != 0 {
		t.Fatalf("AppendEncode allocated %f objects", allocs)
	}
}

func TestParseHexUint64(t *testing.T) {
	tests := []struct {
		input   string
		wantVal uint64
		wantN   int
		wantErr bool
	}{
		{"0", 0, 1, false},
		{"a", 10, 1, false},
		{"A", 10, 1, false},
		{"ff", 255, 2, false},
		{"1a2b", 0x1a2b, 4, false},
		{"1A2B", 0x1a2b, 4, false},
		{"1a2b;ext=foo", 0x1a2b, 4, false},
		{"00001a", 0x1a, 6, false},
		{"ffffffffffffffff", math.MaxUint64, 16, false},
		{"FFFFFFFFFFFFFFFF", math.MaxUint64, 16, false},
		{"7fffffffffffffff", math.MaxInt64, 16, false},
		{"8000000000000000", 0x8000000000000000, 16, false},
		{"1a2b3c4d5e6f7081", 0x1a2b3c4d5e6f7081, 16, false},
		{"00000000000000000000000000000001", 1, 32, false},
		{"0000000000000000ffffffffffffffff", math.MaxUint64, 32, false},
		{"", 0, 0, true},
		{"xyz", 0, 0, true},
		{";foo", 0, 0, true},
		{"10000000000000000", 0, 0, true},                 // 17 digits overflow
		{"fffffffffffffffff", 0, 0, true},                 // 17 F's overflow
		{"ffffffffffffffff0", 0, 0, true},                 // overflow
		{"000000000000000010000000000000000", 0, 0, true}, // overflow with leading zeros
	}

	for _, tc := range tests {
		val, n, err := hexkit.ParseHexUint64([]byte(tc.input))
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseHexUint64(%q) err = %v, wantErr = %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr {
			if val != tc.wantVal || n != tc.wantN {
				t.Errorf("ParseHexUint64(%q) = (%x, %d); want (%x, %d)", tc.input, val, n, tc.wantVal, tc.wantN)
			}
		}
	}

	// Differential checks with strconv.ParseUint
	checkVals := []uint64{0, 1, 15, 16, 255, 65535, 0x12345678, 0xDEADBEEFCAFE, math.MaxInt64, math.MaxUint64}
	for _, v := range checkVals {
		hexStr := strconv.FormatUint(v, 16)
		parsed, n, err := hexkit.ParseHexUint64([]byte(hexStr))
		if err != nil {
			t.Fatalf("ParseHexUint64(%s) failed: %v", hexStr, err)
		}
		if parsed != v || n != len(hexStr) {
			t.Fatalf("ParseHexUint64(%s) = (%x, %d); want (%x, %d)", hexStr, parsed, n, v, len(hexStr))
		}
	}

	allocs := testing.AllocsPerRun(100, func() {
		_, _, _ = hexkit.ParseHexUint64([]byte("1a2b3c4d5e6f7081"))
	})
	if allocs != 0 {
		t.Fatalf("ParseHexUint64 allocated %f objects", allocs)
	}
}

func BenchmarkDecodeByte(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = hexkit.DecodeByte('a', 'f')
	}
}

func BenchmarkAppendHexUint(b *testing.B) {
	dst := make([]byte, 0, 32)
	b.ReportAllocs()
	for b.Loop() {
		dst = hexkit.AppendHexUint(dst[:0], 0x1A2B3C)
	}
}

func BenchmarkIsHex(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = hexkit.IsHex('F')
	}
}

func BenchmarkToHexUpper(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = hexkit.ToHexUpper(0xAB)
	}
}

func BenchmarkAppendEncode(b *testing.B) {
	src := []byte("benchmarking_zero_alloc_hex_append")
	dst := make([]byte, 0, 128)
	b.ReportAllocs()
	for b.Loop() {
		dst = hexkit.AppendEncode(dst[:0], src)
	}
}

func BenchmarkParseHexUint64(b *testing.B) {
	src := []byte("1a2b3c4d5e6f7081")
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = hexkit.ParseHexUint64(src)
	}
}
