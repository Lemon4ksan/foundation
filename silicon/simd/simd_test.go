// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd_test

import (
	"bytes"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"

	"github.com/lemon4ksan/foundation/silicon/simd"
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
	data := []byte("Host: api.example.com\r\nAccept: application/json\r\nUser-Agent: aoni-fast-client/1.0\r\n")

	colonIdx := simd.IndexByteVector(data, ':')
	assert.Equal(t, 4, colonIdx)

	newlineIdx := simd.IndexByteVector(data, '\n')
	assert.Equal(t, 22, newlineIdx)

	missingIdx := simd.IndexByteVector(data, 'Z')
	assert.Equal(t, -1, missingIdx)
}

func TestIndexTwoBytesVector(t *testing.T) {
	data := []byte("Host: api.example.com\r\nAccept: application/json\r\nUser-Agent: aoni-fast-client/1.0\r\n")

	idx := simd.IndexTwoBytesVector(data, ':', '\n')
	assert.Equal(t, 4, idx)
}

func TestApplyFastMaskVector(t *testing.T) {
	payload := []byte("Hello WebSocket World! 123456789012345678901234567890")
	orig := make([]byte, len(payload))
	copy(orig, payload)

	mask := uint32(0x12345678)
	simd.ApplyFastMaskVector(payload, mask)
	assert.NotEqual(t, orig, payload)

	// Unmasking
	simd.ApplyFastMaskVector(payload, mask)
	assert.Equal(t, orig, payload)

	// Short payload (< 32 bytes)
	shortPayload := []byte("short payload")
	origShort := []byte("short payload")
	simd.ApplyFastMaskVector(shortPayload, mask)
	assert.NotEqual(t, origShort, shortPayload)
	simd.ApplyFastMaskVector(shortPayload, mask)
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
