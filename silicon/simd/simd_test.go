// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
	"github.com/lemon4ksan/foundation/silicon/simd"
	"github.com/lemon4ksan/foundation/testing/assert"
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

	// Remainder unaligned copy (head + vector + tail)
	srcOdd := make([]byte, 100)
	for i := range srcOdd {
		srcOdd[i] = byte(i + 1)
	}
	dstOdd := make([]byte, 105)
	nOdd := simd.StreamCopy256(dstOdd[3:103], srcOdd)
	assert.Equal(t, 100, nOdd)
	assert.Equal(t, srcOdd, dstOdd[3:103])

	// Short copy (< 32 bytes)
	shortDst := make([]byte, 16)
	nShort := simd.StreamCopy256(shortDst, src[:16])
	assert.Equal(t, 16, nShort)
	assert.Equal(t, src[:16], shortDst)

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

	// Empty data and seeded
	assert.Equal(t, uint64(0), simd.Hash64Vector(nil, 0))
	assert.NotEqual(t, uint64(0), simd.Hash64Vector(data, 12345))
}

func TestSIMD_MaskCRLF_And_ToLowerSWAR(t *testing.T) {
	t.Parallel()

	// 1. MaskCRLF short
	shortData := []byte("hello\r\nworld\n!")
	simd.MaskCRLF(shortData)
	assert.Equal(t, []byte("hello  world !"), shortData)

	// Empty MaskCRLF
	simd.MaskCRLF(nil)

	// MaskCRLF long >= 32 bytes
	longData := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n")
	simd.MaskCRLF(longData)
	assert.Equal(t, 51, len(longData))
	assert.Equal(t, []byte("HTTP/1.1 200 OK  Content-Type: application/json    "), longData)

	// 2. ToLowerSWAR
	src := []byte("HELLO WORLD! 123 ABCDEFGHIJKLMNOPQRSTUVWXYZ-extra-tail")
	dst := make([]byte, len(src))
	simd.ToLowerSWAR(dst, src)
	assert.Equal(t, []byte(strings.ToLower(string(src))), dst)

	// 3. ParallelExtract64 & TrailingZeros32
	pext := simd.ParallelExtract64(0b10101010, 0b11110000)
	assert.True(t, pext > 0)
	assert.Equal(t, 3, simd.TrailingZeros32(0b1000))

	// 4. PrefetchL1
	simd.PrefetchL1(unsafe.Pointer(&src[0]))

	// 5. IndexCRLFCRLF
	rawHTTP := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\nBody")
	assert.Equal(t, 37, simd.IndexCRLFCRLF(rawHTTP))
	assert.Equal(t, -1, simd.IndexCRLFCRLF([]byte("no-boundary")))
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

// FuzzIndexByteSWAR asserts 100% equivalence between 64-bit SWAR vector scanning and stdlib bytes.IndexByte.
func FuzzIndexByteSWAR(f *testing.F) {
	f.Add([]byte("HTTP/1.1 200 OK\r\n"), byte('\r'))
	f.Add([]byte(""), byte('x'))
	f.Add([]byte("1234567890abcdef"), byte('f'))
	f.Add([]byte("aaaaaaaa"), byte('a'))
	f.Add([]byte("bbbbbbbb"), byte('z'))

	f.Fuzz(func(t *testing.T, data []byte, target byte) {
		got := simd.IndexByteSWAR(data, target)
		expected := bytes.IndexByte(data, target)

		if got != expected {
			t.Fatalf("IndexByteSWAR mismatch: got %d, expected %d on data %q, target %d", got, expected, data, target)
		}
	})
}

// FuzzIndexCRLF asserts 100% equivalence between SWAR CRLF scanning and stdlib bytes.Index.
func FuzzIndexCRLF(f *testing.F) {
	f.Add([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))
	f.Add([]byte("\r\n"))
	f.Add([]byte("\r"))
	f.Add([]byte("\n"))
	f.Add([]byte(""))
	f.Add([]byte("1234567\r\n890"))
	f.Add([]byte("12345678\r\n90"))
	f.Add([]byte("\r\r\r\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		got := simd.IndexCRLF(data)
		expected := bytes.Index(data, []byte("\r\n"))

		if got != expected {
			t.Fatalf("IndexCRLF mismatch: got %d, expected %d on data %q", got, expected, data)
		}
	})
}

// FuzzIndexDoubleCRLF verifies HTTP header terminator scanning against stdlib bytes.Index.
func FuzzIndexDoubleCRLF(f *testing.F) {
	f.Add([]byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"))
	f.Add([]byte("\r\n\r\n"))
	f.Add([]byte("\r\n\r\r\n\r\n"))
	f.Add([]byte(""))
	f.Add([]byte("header: val\r\n\r\nbody"))

	f.Fuzz(func(t *testing.T, data []byte) {
		got := simd.IndexDoubleCRLF(data)
		expected := bytes.Index(data, []byte("\r\n\r\n"))

		if got != expected {
			t.Fatalf("IndexDoubleCRLF mismatch: got %d, expected %d on data %q", got, expected, data)
		}
	})
}

func TestMaskCRLF_AllBytes(t *testing.T) {
	t.Parallel()

	// 256 bytes covering the entire byte range [0x00 .. 0xFF]
	data := make([]byte, 256)
	expected := make([]byte, 256)
	for i := range 256 {
		b := byte(i)
		data[i] = b
		if b == '\r' || b == '\n' {
			expected[i] = ' '
		} else {
			expected[i] = b
		}
	}

	simd.MaskCRLF(data)
	assert.Equal(t, expected, data)

	// Specifically verify that '\n' (0x0A) became 0x20 (' '), NOT 0x2A ('*')
	assert.Equal(t, byte(' '), data[0x0A])
	// And that '\r' (0x0D) became 0x20 (' '), NOT 0x2D ('-')
	assert.Equal(t, byte(' '), data[0x0D])
	// And that 0x2A ('*') was preserved
	assert.Equal(t, byte('*'), data[0x2A])
	// And that 0x2D ('-') was preserved
	assert.Equal(t, byte('-'), data[0x2D])
}

func TestMaskCRLF_LengthsAndBoundaries(t *testing.T) {
	t.Parallel()

	lengths := []int{0, 1, 2, 7, 8, 15, 16, 31, 32, 33, 47, 63, 64, 65, 96, 97, 128, 256, 1024}
	for _, l := range lengths {
		src := make([]byte, l)
		exp := make([]byte, l)
		for i := range l {
			b := byte('a' + (i % 26))
			src[i] = b
			exp[i] = b
		}
		if l > 0 {
			src[0] = '\r'
			exp[0] = ' '
		}
		if l > 15 {
			src[15] = '\n'
			exp[15] = ' '
		}
		if l > 31 {
			src[31] = '\r'
			exp[31] = ' '
		}
		if l > 32 {
			src[32] = '\n'
			exp[32] = ' '
		}
		if l > 63 {
			src[63] = '\r'
			exp[63] = ' '
		}
		if l > 64 {
			src[64] = '\n'
			exp[64] = ' '
		}
		if l > 1 {
			src[l-1] = '\n'
			exp[l-1] = ' '
		}

		simd.MaskCRLF(src)
		assert.Equal(t, exp, src)
	}
}

func TestIndexByteSWAR_UnalignedOffsets(t *testing.T) {
	t.Parallel()

	base := make([]byte, 256)
	for i := range base {
		base[i] = 'X'
	}

	for offset := range 8 {
		for length := 0; length <= 128; length++ {
			sub := make([]byte, length)
			copy(sub, base[offset:offset+length])

			// 1. Target not found
			got := simd.IndexByteSWAR(sub, 'A')
			assert.Equal(t, -1, got)

			if length > 0 {
				// Target at first position
				sub[0] = 'A'
				assert.Equal(t, 0, simd.IndexByteSWAR(sub, 'A'))
				sub[0] = 'X'

				// Target at last position
				sub[length-1] = 'A'
				assert.Equal(t, length-1, simd.IndexByteSWAR(sub, 'A'))
				sub[length-1] = 'X'

				if length > 8 {
					// Target in middle (SWAR chunk)
					sub[5] = 'A'
					assert.Equal(t, 5, simd.IndexByteSWAR(sub, 'A'))
					sub[5] = 'X'
				}
			}
		}
	}
}

func BenchmarkMaskCRLF_Short(b *testing.B) {
	data := make([]byte, 16)
	for i := range data {
		data[i] = 'A'
	}
	data[5] = '\r'
	data[10] = '\n'

	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for b.Loop() {
		simd.MaskCRLF(data)
	}
}

func BenchmarkMaskCRLF_Medium(b *testing.B) {
	data := make([]byte, 64)
	for i := range data {
		data[i] = 'A'
	}
	data[15] = '\r'
	data[16] = '\n'
	data[45] = '\r'
	data[46] = '\n'

	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for b.Loop() {
		simd.MaskCRLF(data)
	}
}

func BenchmarkMaskCRLF_Large(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = 'A'
	}
	for i := 0; i < 1024; i += 64 {
		data[i] = '\r'
		data[i+1] = '\n'
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for b.Loop() {
		simd.MaskCRLF(data)
	}
}

func BenchmarkIndexByteSWAR_Aligned(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = 'A'
	}
	data[1000] = 'Z'

	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for b.Loop() {
		_ = simd.IndexByteSWAR(data, 'Z')
	}
}

func BenchmarkIndexByteSWAR_Unaligned(b *testing.B) {
	raw := make([]byte, 1025)
	for i := range raw {
		raw[i] = 'A'
	}
	raw[1001] = 'Z'
	data := raw[1:] // unaligned by 1 byte

	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for b.Loop() {
		_ = simd.IndexByteSWAR(data, 'Z')
	}
}

func maskCRLFScalarOracle(b []byte) {
	for i := range b {
		if b[i] == '\r' || b[i] == '\n' {
			b[i] = ' '
		}
	}
}

func TestChallenge_MaskCRLF_DifferentialFuzzing_Concurrent(t *testing.T) {
	t.Parallel()

	// Phase 1: Exhaustive deterministic edge cases and canary overflow validation
	for length := 0; length <= 130; length++ {
		patterns := [][]byte{
			make([]byte, length),               // All zeros
			bytes.Repeat([]byte{0xFF}, length), // All 0xFF
			bytes.Repeat([]byte{'\r'}, length), // All CR
			bytes.Repeat([]byte{'\n'}, length), // All LF
			make([]byte, length),               // Alternating CRLF
			make([]byte, length),               // ASCII text
			make([]byte, length),               // Adjacent chars
		}

		for i := 0; i < length; i++ {
			if i%2 == 0 {
				patterns[4][i] = '\r'
			} else {
				patterns[4][i] = '\n'
			}
			patterns[5][i] = byte('A' + (i % 26))
			adjacentList := []byte{0x09, 0x0B, 0x0C, 0x0E, 0x20, 0x2A, 0x2D}
			patterns[6][i] = adjacentList[i%len(adjacentList)]
		}

		for pIdx, baseData := range patterns {
			const canaryPad = 64
			rawTest := make([]byte, length+canaryPad)
			rawRef := make([]byte, length+canaryPad)
			copy(rawTest, baseData)
			copy(rawRef, baseData)

			const canaryByte = byte(0xDE)
			for c := length; c < len(rawTest); c++ {
				rawTest[c] = canaryByte
				rawRef[c] = canaryByte
			}

			testSlice := rawTest[:length]
			refSlice := rawRef[:length]

			simd.MaskCRLF(testSlice)
			maskCRLFScalarOracle(refSlice)

			assert.Equal(t, refSlice, testSlice, fmt.Sprintf("pattern %d length %d mismatch", pIdx, length))

			// Verify canaries were untouched
			for c := length; c < len(rawTest); c++ {
				if rawTest[c] != canaryByte {
					t.Fatalf("buffer overrun! canary at index %d modified to 0x%02X (len=%d)", c, rawTest[c], length)
				}
			}
		}
	}

	// Phase 2: High concurrency differential fuzzing under -race
	const numGoroutines = 64
	const itersPerGoroutine = 500

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := range numGoroutines {
		go func(gID int) {
			defer wg.Done()

			r := rand.New(rand.NewPCG(uint64(gID+1), uint64(time.Now().UnixNano())))
			for it := range itersPerGoroutine {
				length := r.IntN(2049)

				probChoice := r.IntN(4)
				var crlfProb float64
				switch probChoice {
				case 0:
					crlfProb = 0.0
				case 1:
					crlfProb = 0.05
				case 2:
					crlfProb = 0.50
				case 3:
					crlfProb = 1.0
				}

				const canaryPad = 32
				rawTest := make([]byte, length+canaryPad)
				rawRef := make([]byte, length+canaryPad)

				for i := range length {
					var b byte
					if r.Float64() < crlfProb {
						if r.IntN(2) == 0 {
							b = '\r'
						} else {
							b = '\n'
						}
					} else {
						b = byte(r.IntN(256))
						if b == '\r' || b == '\n' {
							b = 'x'
						}
					}
					rawTest[i] = b
					rawRef[i] = b
				}

				const canaryByte = byte(0xFE)
				for c := length; c < len(rawTest); c++ {
					rawTest[c] = canaryByte
					rawRef[c] = canaryByte
				}

				testSlice := rawTest[:length]
				refSlice := rawRef[:length]

				simd.MaskCRLF(testSlice)
				maskCRLFScalarOracle(refSlice)

				if !bytes.Equal(refSlice, testSlice) {
					t.Errorf("g=%d it=%d len=%d mismatch: got %v, expected %v", gID, it, length, testSlice, refSlice)
					return
				}

				for c := length; c < len(rawTest); c++ {
					if rawTest[c] != canaryByte {
						t.Errorf("g=%d it=%d canary overwrite at %d: got 0x%02X", gID, it, c, rawTest[c])
						return
					}
				}
			}
		}(g)
	}

	wg.Wait()
}

func TestChallenge_Architecture_SiliconSimdIsolation(t *testing.T) {
	t.Parallel()

	// 1. Verify via 'go list -json github.com/lemon4ksan/foundation/silicon/simd/...'
	cmd := exec.Command("go", "list", "-json", "github.com/lemon4ksan/foundation/silicon/simd/...")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}

	type pkgMeta struct {
		ImportPath string   `json:"ImportPath"`
		Imports    []string `json:"Imports"`
		Deps       []string `json:"Deps"`
	}

	dec := json.NewDecoder(bytes.NewReader(out))
	const foundationPrefix = "github.com/lemon4ksan/foundation"

	for dec.More() {
		var p pkgMeta
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		for _, imp := range p.Imports {
			if strings.HasPrefix(imp, foundationPrefix) {
				t.Errorf(
					"[ADR-0002 VIOLATION] Package %s imports foundation package %s. Layer 0 silicon must import zero foundation packages.",
					p.ImportPath,
					imp,
				)
			}
		}

		for _, dep := range p.Deps {
			if strings.HasPrefix(dep, foundationPrefix) {
				rel := strings.TrimPrefix(dep, foundationPrefix+"/")
				parts := strings.Split(rel, "/")
				if parts[0] != "silicon" {
					t.Errorf(
						"[ADR-0002 VIOLATION] Package %s transitively depends on higher-layer package %s.",
						p.ImportPath,
						dep,
					)
				}
			}
		}
	}

	// 2. Direct AST parsing of all .go files in silicon/simd directory
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}

	fset := token.NewFileSet()
	for _, file := range files {
		node, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse file %s failed: %v", file, err)
		}

		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		for _, imp := range node.Imports {
			pathVal := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(pathVal, foundationPrefix) {
				t.Errorf("[ADR-0002 VIOLATION] Implementation file %s contains import %q.", file, pathVal)
			}
		}
	}

	// 3. Verify canonical doc.go existence
	docStat, err := os.Stat("doc.go")
	if err != nil || docStat.IsDir() {
		t.Errorf("[ADR-0008 VIOLATION] silicon/simd is missing canonical doc.go")
	}
}

func TestChallenge_IndexByteSWAR_UnalignedMemorySafety(t *testing.T) {
	t.Parallel()

	const bufferSize = 2048
	baseBuffer := make([]byte, bufferSize)
	for i := range baseBuffer {
		baseBuffer[i] = 0xAA
	}

	lengthsToTest := []int{
		0, 1, 2, 3, 4, 5, 6, 7,
		8, 9, 10, 11, 14, 15,
		16, 17, 23, 24, 25, 31,
		32, 33, 47, 48, 63, 64,
		65, 127, 128, 129, 255, 256, 512, 1024,
	}

	for offset := range 64 {
		for _, length := range lengthsToTest {
			if offset+length > bufferSize {
				continue
			}

			// Sub-slice strictly bounded with 3-index slicing: cap == len
			sub := baseBuffer[offset : offset+length : offset+length]

			// Case 1: Target absent
			const missingTarget = byte(0x55)
			got := simd.IndexByteSWAR(sub, missingTarget)
			assert.Equal(t, -1, got, fmt.Sprintf("offset=%d len=%d expected -1", offset, length))

			got2 := simd.IndexByteTwoSWAR(sub, 0x11, 0x22)
			assert.Equal(t, -1, got2, fmt.Sprintf("offset=%d len=%d expected -1 for two", offset, length))

			if length == 0 {
				continue
			}

			// Case 2: Target at position 0
			sub[0] = 0x55
			assert.Equal(t, 0, simd.IndexByteSWAR(sub, 0x55))
			sub[0] = 0xAA

			// Case 3: Target at position length - 1
			sub[length-1] = 0x55
			assert.Equal(t, length-1, simd.IndexByteSWAR(sub, 0x55))
			sub[length-1] = 0xAA

			// Case 4: Target at position length / 2
			mid := length / 2
			sub[mid] = 0x55
			assert.Equal(t, mid, simd.IndexByteSWAR(sub, 0x55))
			sub[mid] = 0xAA

			// Case 5: Target duplicated (first occurrence must be returned)
			if length >= 8 {
				sub[1] = 0x55
				sub[5] = 0x55
				assert.Equal(t, 1, simd.IndexByteSWAR(sub, 0x55), "should return earlier occurrence in chunk")
				sub[1] = 0xAA
				sub[5] = 0xAA

				if length >= 16 {
					sub[3] = 0x55
					sub[11] = 0x55
					assert.Equal(t, 3, simd.IndexByteSWAR(sub, 0x55), "should return earlier occurrence across chunks")
					sub[3] = 0xAA
					sub[11] = 0xAA
				}
			}

			// Case 6: IndexByteTwoSWAR with t1 and t2
			if length >= 2 {
				sub[0] = 0x11
				sub[length-1] = 0x22
				assert.Equal(t, 0, simd.IndexByteTwoSWAR(sub, 0x11, 0x22))
				assert.Equal(t, 0, simd.IndexByteTwoSWAR(sub, 0x22, 0x11))
				sub[0] = 0xAA
				sub[length-1] = 0xAA

				sub[0] = 0x22
				sub[length-1] = 0x11
				assert.Equal(t, 0, simd.IndexByteTwoSWAR(sub, 0x11, 0x22))
				sub[0] = 0xAA
				sub[length-1] = 0xAA
			}
		}
	}

	// Phase 2: Concurrent unaligned stress under -race
	const numGoroutines = 32
	const iters = 200
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := range numGoroutines {
		go func(gID int) {
			defer wg.Done()
			r := rand.New(rand.NewPCG(uint64(gID+100), uint64(time.Now().UnixNano())))

			buf := make([]byte, 2048)
			for it := range iters {
				for i := range buf {
					buf[i] = byte(r.IntN(256))
				}

				offset := r.IntN(64)
				length := r.IntN(1024)
				sub := buf[offset : offset+length : offset+length]

				target := byte(r.IntN(256))
				expected := bytes.IndexByte(sub, target)
				actual := simd.IndexByteSWAR(sub, target)

				if actual != expected {
					t.Errorf(
						"g=%d it=%d offset=%d len=%d target=0x%02X got %d expected %d",
						gID,
						it,
						offset,
						length,
						target,
						actual,
						expected,
					)
					return
				}
			}
		}(g)
	}

	wg.Wait()
}

func TestSWARSpaceTrimming(t *testing.T) {
	tests := []struct {
		input       string
		wantLeading string
		wantTrail   string
		wantBoth    string
	}{
		{"", "", "", ""},
		{"   ", "", "", ""},
		{"\t\t\t", "", "", ""},
		{" \t \t ", "", "", ""},
		{"hello", "hello", "hello", "hello"},
		{"   hello", "hello", "   hello", "hello"},
		{"hello   ", "hello   ", "hello", "hello"},
		{"   hello   ", "hello   ", "   hello", "hello"},
		{
			"\t\tContent-Type: text/plain\t ",
			"Content-Type: text/plain\t ",
			"\t\tContent-Type: text/plain",
			"Content-Type: text/plain",
		},
		{
			"                                a                                ",
			"a                                ",
			"                                a",
			"a",
		},
	}

	for _, tc := range tests {
		gotLead := string(simd.TrimLeadingSpaceSWAR([]byte(tc.input)))
		if gotLead != tc.wantLeading {
			t.Errorf("TrimLeadingSpaceSWAR(%q) = %q; want %q", tc.input, gotLead, tc.wantLeading)
		}

		gotTrail := string(simd.TrimTrailingSpaceSWAR([]byte(tc.input)))
		if gotTrail != tc.wantTrail {
			t.Errorf("TrimTrailingSpaceSWAR(%q) = %q; want %q", tc.input, gotTrail, tc.wantTrail)
		}

		gotBoth := string(simd.TrimSpaceSWAR([]byte(tc.input)))
		if gotBoth != tc.wantBoth {
			t.Errorf("TrimSpaceSWAR(%q) = %q; want %q", tc.input, gotBoth, tc.wantBoth)
		}
	}
}

func TestIndexColonSWAR(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", -1},
		{"Host: localhost", 4},
		{":authority", 0},
		{"Content-Length: 100", 14},
		{"No colon here", -1},
	}

	for _, tc := range tests {
		got := simd.IndexColonSWAR([]byte(tc.input))
		if got != tc.want {
			t.Errorf("IndexColonSWAR(%q) = %d; want %d", tc.input, got, tc.want)
		}
	}
}
