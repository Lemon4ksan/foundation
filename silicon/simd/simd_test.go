// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd_test

import (
	"bytes"
	"strings"
	"testing"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
	"github.com/lemon4ksan/foundation/silicon/simd"
	"github.com/lemon4ksan/foundation/testkit/assert"
)

func TestIndexByteSWAR(t *testing.T) {
	data := []byte("Host: api.example.com\r\nAccept: application/json\r\n")

	colonIdx := simd.IndexByteSWAR(data, ':')
	assert.Equal(t, 4, colonIdx)

	newlineIdx := simd.IndexByteSWAR(data, '\n')
	assert.Equal(t, 22, newlineIdx)

	missingIdx := simd.IndexByteSWAR(data, 'Z')
	assert.Equal(t, -1, missingIdx)
}

func TestIndexByteTwoSWAR(t *testing.T) {
	data := []byte("User-Agent: Mozilla/5.0; Chrome/120")

	idx := simd.IndexByteTwoSWAR(data, ';', ':')
	assert.Equal(t, 10, idx)
}

func TestIndexByteVector(t *testing.T) {
	data := []byte("Host: api.example.com\r\nAccept: application/json\r\nUser-Agent: foundation-client/1.0\r\n")

	colonIdx := simd.IndexByteVector(data, ':')
	assert.Equal(t, 4, colonIdx)

	newlineIdx := simd.IndexByteVector(data, '\n')
	assert.Equal(t, 22, newlineIdx)

	missingIdx := simd.IndexByteVector(data, 'Z')
	assert.Equal(t, -1, missingIdx)
}

func TestIndexTwoBytesVector(t *testing.T) {
	data := []byte("Host: api.example.com\r\nAccept: application/json\r\nUser-Agent: foundation-client/1.0\r\n")

	idx := simd.IndexTwoBytesVector(data, ':', '\n')
	assert.Equal(t, 4, idx)
}

func TestXORMask32(t *testing.T) {
	payload := []byte("Hello Foundation World! 123456789012345678901234567890")
	orig := make([]byte, len(payload))
	copy(orig, payload)

	mask := uint32(0x12345678)
	simd.XORMask32(payload, mask)
	assert.NotEqual(t, orig, payload)

	// Unmasking
	simd.XORMask32(payload, mask)
	assert.Equal(t, orig, payload)

	// Short payload (< 32 bytes)
	shortPayload := []byte("short payload")
	origShort := []byte("short payload")
	simd.XORMask32(shortPayload, mask)
	assert.NotEqual(t, origShort, shortPayload)
	simd.XORMask32(shortPayload, mask)
	assert.Equal(t, origShort, shortPayload)
}

func TestStreamCopy256(t *testing.T) {
	src := make([]byte, 128)
	for i := range src {
		src[i] = byte(i)
	}

	dst := make([]byte, 128)
	n := simd.StreamCopy256(dst, src)
	assert.Equal(t, 128, n)
	assert.Equal(t, src, dst)

	// Remainder unaligned copy
	srcOdd := make([]byte, 100)
	dstOdd := make([]byte, 100)
	nOdd := simd.StreamCopy256(dstOdd, srcOdd)
	assert.Equal(t, 100, nOdd)

	// Zero length
	assert.Equal(t, 0, simd.StreamCopy256(nil, nil))
}

func TestPrefetchAndBMI2(t *testing.T) {
	data := make([]byte, 64)
	simd.PrefetchL1(nil)
	simd.PrefetchL1(unsafe.Pointer(&data[0]))

	assert.Equal(t, 0, simd.CountLeadingZeros(0x8000000000000000))
	assert.Equal(t, 64, simd.CountLeadingZeros(0))
	assert.Equal(t, 0, simd.CountTrailingZeros(1))
	assert.Equal(t, 64, simd.CountTrailingZeros(0))

	val := uint64(0x123456789ABCDEF0)
	mask := uint64(0x0F0F0F0F0F0F0F0F)
	extracted := simd.ExtractBits(val, mask)
	deposited := simd.DepositBits(extracted, mask)
	assert.Equal(t, val&mask, deposited)
}

func TestWordMatch(t *testing.T) {
	p64 := simd.PackWord64("123456789")
	assert.NotZero(t, p64)
	p64Short := simd.PackWord64("123")
	assert.NotZero(t, p64Short)

	p32 := simd.PackWord32("12345")
	assert.NotZero(t, p32)
	p32Short := simd.PackWord32("12")
	assert.NotZero(t, p32Short)

	buf := []byte("HTTP/1.1 200 OK\r\n")
	assert.True(t, simd.MatchWord64(buf, simd.PackWord64("HTTP/1.1")))
	assert.True(t, simd.MatchWord64Str(buf, "HTTP/1.1"))
	assert.False(t, simd.MatchWord64(buf[:4], p64))
	assert.False(t, simd.MatchWord64Str(buf[:4], "HTTP/1.1"))

	assert.True(t, simd.MatchWord32(buf, simd.PackWord32("HTTP")))
	assert.True(t, simd.MatchWord32Str(buf, "HTTP"))
	assert.False(t, simd.MatchWord32(buf[:2], p32))
	assert.False(t, simd.MatchWord32Str(buf[:2], "HTTP"))
}

func BenchmarkIndexByte_Std(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = 'a'
	}

	data[1023] = 'z'

	b.ReportAllocs()

	for b.Loop() {
		_ = bytes.IndexByte(data, 'z')
	}
}

func BenchmarkIndexByte_SWAR(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = 'a'
	}

	data[1023] = 'z'

	b.ReportAllocs()

	for b.Loop() {
		_ = simd.IndexByteSWAR(data, 'z')
	}
}

func BenchmarkIndexByte_AVX2(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = 'a'
	}

	data[1023] = 'z'

	b.ReportAllocs()

	for b.Loop() {
		_ = simd.IndexByteVector(data, 'z')
	}
}

func BenchmarkIndexTwoBytes_SWAR(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = 'a'
	}

	data[1023] = '\n'

	b.ReportAllocs()

	for b.Loop() {
		_ = simd.IndexByteTwoSWAR(data, '\r', '\n')
	}
}

func BenchmarkIndexTwoBytes_AVX2(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = 'a'
	}

	data[1023] = '\n'

	b.ReportAllocs()

	for b.Loop() {
		_ = simd.IndexTwoBytesVector(data, '\r', '\n')
	}
}

func TestIndexCRLF(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\nBody")

	crlfIdx := simd.IndexCRLF(data)
	assert.Equal(t, 14, crlfIdx)

	doubleCRLFIdx := simd.IndexDoubleCRLF(data)
	assert.Equal(t, 33, doubleCRLFIdx)

	assert.Equal(t, -1, simd.IndexCRLF([]byte("no crlf here")))
	assert.Equal(t, -1, simd.IndexDoubleCRLF([]byte("single \r\n crlf")))
}

func BenchmarkIndexCRLF(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = 'x'
	}
	data[1022] = '\r'
	data[1023] = '\n'

	b.ReportAllocs()
	for b.Loop() {
		_ = simd.IndexCRLF(data)
	}
}

func BenchmarkIndexDoubleCRLF(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = 'x'
	}
	data[1020] = '\r'
	data[1021] = '\n'
	data[1022] = '\r'
	data[1023] = '\n'

	b.ReportAllocs()
	for b.Loop() {
		_ = simd.IndexDoubleCRLF(data)
	}
}

func TestValidUTF8(t *testing.T) {
	assert.True(t, simd.ValidUTF8([]byte("hello world 12345")))
	assert.True(t, simd.ValidUTF8([]byte("{\"message\":\"hello from json payload\",\"status\":200}")))
	assert.True(t, simd.ValidUTF8([]byte("Привет, мир! 🚀")))
	assert.False(t, simd.ValidUTF8([]byte{0xff, 0xfe, 0xfd}))
}

func BenchmarkValidUTF8_SWAR(b *testing.B) {
	data := []byte("{\"message\":\"hello from websocket text frame\",\"id\":12345678,\"active\":true}")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()

	for b.Loop() {
		_ = simd.ValidUTF8(data)
	}
}

func BenchmarkValidUTF8_Std(b *testing.B) {
	data := []byte("{\"message\":\"hello from websocket text frame\",\"id\":12345678,\"active\":true}")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()

	for b.Loop() {
		_ = bytes.Contains(data, []byte("\x00")) // standard benchmark baseline
	}
}

func TestEqualFoldVector(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"", "", true},
		{"Content-Type", "content-type", true},
		{
			"USER-AGENT: Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0",
			"user-agent: mozilla/5.0 (windows nt 10.0; win64; x64) chrome/120.0",
			true,
		},
		{
			"A very long header value that exceeds 32 bytes for AVX2 SIMD testing",
			"a very long header value that exceeds 32 bytes for avx2 simd testing",
			true,
		},
		{
			"A very long header value that exceeds 32 bytes with a single difference X",
			"a very long header value that exceeds 32 bytes with a single difference Y",
			false,
		},
		{"Short", "LongerString", false},
	}

	for _, tt := range tests {
		got := simd.EqualFoldVector([]byte(tt.a), []byte(tt.b))
		assert.Equal(t, tt.want, got)
	}
}

func BenchmarkEqualFold_AVX2(b *testing.B) {
	str1 := []byte(
		"User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	)
	str2 := []byte(
		"user-agent: mozilla/5.0 (windows nt 10.0; win64; x64) applewebkit/537.36 (khtml, like gecko) chrome/120.0.0.0 safari/537.36",
	)

	b.SetBytes(int64(len(str1)))
	b.ReportAllocs()

	for b.Loop() {
		_ = simd.EqualFoldVector(str1, str2)
	}
}

func BenchmarkEqualFold_PureGoTable(b *testing.B) {
	str1 := "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	str2 := "user-agent: mozilla/5.0 (windows nt 10.0; win64; x64) applewebkit/537.36 (khtml, like gecko) chrome/120.0.0.0 safari/537.36"

	b.SetBytes(int64(len(str1)))
	b.ReportAllocs()

	for b.Loop() {
		_ = bytesconv.EqualFoldASCII(str1, str2)
	}
}

func BenchmarkEqualFold_Stdlib(b *testing.B) {
	str1 := "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	str2 := "user-agent: mozilla/5.0 (windows nt 10.0; win64; x64) applewebkit/537.36 (khtml, like gecko) chrome/120.0.0.0 safari/537.36"

	b.SetBytes(int64(len(str1)))
	b.ReportAllocs()

	for b.Loop() {
		_ = strings.EqualFold(str1, str2)
	}
}

func TestScanByteVector(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 1234\r\n\r\n")
	idx := simd.ScanByteVector(data, ':')
	assert.Equal(t, 29, idx)

	idxNone := simd.ScanByteVector(data, 'Z')
	assert.Equal(t, -1, idxNone)
}

func TestIndexCRLFCRLFVector(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"status\":\"ok\"}")
	idx := simd.IndexCRLFCRLFVector(data)
	assert.Equal(t, 51, idx) // 51 is byte index right after \r\n\r\n

	noEnd := []byte("HTTP/1.1 200 OK\r\nIncomplete-Header: true\r\n")
	idxNoEnd := simd.IndexCRLFCRLFVector(noEnd)
	assert.Equal(t, -1, idxNoEnd)
}

func TestFindMatchLengthVector(t *testing.T) {
	a := []byte("The quick brown fox jumps over the lazy dog and runs away into the forest quickly")
	b := []byte("The quick brown fox jumps over the lazy cat and runs away into the forest quickly")

	matchLen := simd.FindMatchLengthVector(a, b, len(a))
	assert.Equal(t, 40, matchLen) // Matches "The quick brown fox jumps over the lazy " (length 40)
}

func TestHash64Vector(t *testing.T) {
	data := []byte("The quick brown fox jumps over the lazy dog")
	h1 := simd.Hash64Vector(data, 0)
	h2 := simd.Hash64Vector(data, 0)
	assert.Equal(t, h1, h2)
	assert.True(t, h1 != 0)

	differentData := []byte("The quick brown fox jumps over the lazy cat")
	h3 := simd.Hash64Vector(differentData, 0)
	assert.True(t, h1 != h3)
}

func BenchmarkScanByte_AVX2(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = 'A'
	}
	data[1023] = ':'

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()

	for b.Loop() {
		_ = simd.ScanByteVector(data, ':')
	}
}

func BenchmarkIndexCRLFCRLF_AVX2(b *testing.B) {
	data := make([]byte, 1024)
	copy(
		data,
		[]byte(
			"POST /api/v1/trade/submit HTTP/1.1\r\nHost: api.steampowered.com\r\nAuthorization: Bearer test\r\n\r\n",
		),
	)

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()

	for b.Loop() {
		_ = simd.IndexCRLFCRLFVector(data)
	}
}

func BenchmarkFindMatchLength_AVX2(b *testing.B) {
	a := make([]byte, 256)
	bSlice := make([]byte, 256)
	for i := range a {
		a[i] = byte(i)
		bSlice[i] = byte(i)
	}
	a[200] = 0xFF // mismatch at 200

	b.SetBytes(256)
	b.ReportAllocs()

	for b.Loop() {
		_ = simd.FindMatchLengthVector(a, bSlice, 256)
	}
}

func BenchmarkHash64_AVX2(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i)
	}

	b.SetBytes(1024)
	b.ReportAllocs()

	for b.Loop() {
		_ = simd.Hash64Vector(data, 0x12345678)
	}
}
