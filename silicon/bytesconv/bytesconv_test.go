// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytesconv

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestB2SAndS2B(t *testing.T) {
	t.Parallel()

	assert.Empty(t, B2S(nil))
	assert.Nil(t, S2B(""))

	str := "Hello, Foundation!"

	b := S2B(str)
	assert.True(t, bytes.Equal(b, []byte(str)))

	s := B2S(b)
	assert.Equal(t, str, s)
}

func TestLowercaseByte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   byte
		want byte
	}{
		{'A', 'a'},
		{'Z', 'z'},
		{'a', 'a'},
		{'z', 'z'},
		{'0', '0'},
		{'-', '-'},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, LowercaseByte(tt.in))
	}
}

func TestEqualFoldASCII(t *testing.T) {
	t.Parallel()

	tests := []struct {
		a, b string
		want bool
	}{
		{"", "", true},
		{"Content-Type", "content-type", true},
		{"USER-AGENT", "user-agent", true},
		{"Hello", "World", false},
		{"Short", "LongerString", false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, EqualFoldASCII(tt.a, tt.b))
	}
}

func TestAppendToLower(t *testing.T) {
	t.Parallel()

	assert.Empty(t, AppendToLower(nil, nil))

	dst := []byte("Header: ")
	src := []byte("X-Foundation-Test")
	got := AppendToLower(dst, src)

	assert.Equal(t, "Header: x-foundation-test", string(got))
}

func TestTrimQuotes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{`"hello"`, "hello"},
		{`"a"`, "a"},
		{`""`, ""},
		{`noquotes`, "noquotes"},
		{`"unmatched`, `"unmatched`},
	}

	for _, tt := range tests {
		got := TrimQuotes([]byte(tt.in))
		assert.Equal(t, tt.want, string(got))
	}
}

func TestContainsFoldASCII(t *testing.T) {
	t.Parallel()

	tests := []struct {
		src    string
		target string
		want   bool
	}{
		{"gzip, deflate", "gzip", true},
		{"Bz gZip", "gzip", true},
		{"GZIP", "gzip", true},
		{"gZip", "gzip", true},
		{"BR", "br", true},
		{"zStD, gzip", "zstd", true},
		{"gzip", "br", false},
		{"", "gzip", false},
		{"gzip", "", true},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, ContainsFoldASCII([]byte(tt.src), tt.target))
	}
}

func TestParseUintFast(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in      string
		wantVal int64
		wantOK  bool
	}{
		{"200", 200, true},
		{"0", 0, true},
		{"1234567890", 1234567890, true},
		{"", 0, false},
		{"abc", 0, false},
		{"12a34", 0, false},
	}

	for _, tt := range tests {
		val, ok := ParseUintFast([]byte(tt.in))
		assert.Equal(t, tt.wantOK, ok)

		if ok {
			assert.Equal(t, tt.wantVal, val)
		}
	}
}

func TestFastHash64(t *testing.T) {
	t.Parallel()

	h1 := FastHash64([]byte("hello world"))
	h2 := FastHash64([]byte("hello world"))
	h3 := FastHash64([]byte("different data"))

	assert.Equal(t, h1, h2)
	assert.NotEqual(t, h1, h3)
	assert.NotZero(t, FastHash64(nil))
}

func TestPatternSlicer(t *testing.T) {
	t.Parallel()

	// 1-byte pattern
	s1 := NewPatternSlicer([]byte(":"), 1)
	parts, ok := s1.Slice([]byte("key:value"))
	assert.True(t, ok)
	assert.Len(t, parts, 2)
	assert.Equal(t, "key:", string(parts[0]))
	assert.Equal(t, "value", string(parts[1]))

	// 2-byte pattern
	s2 := NewPatternSlicer([]byte("\r\n"), 2)
	parts2, ok2 := s2.Slice([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain"))
	assert.True(t, ok2)
	assert.Len(t, parts2, 2)
	assert.Equal(t, "HTTP/1.1 200 OK\r\n", string(parts2[0]))

	// Multi-byte pattern
	s3 := NewPatternSlicer([]byte("DELIM"), 5)
	all := s3.SliceAll([]byte("AADELIMBBDELIMCC"))
	assert.Len(t, all, 3)

	// Slice edge cases: empty or no match
	assert.Len(t, s1.SliceAll(nil), 1)
	emptySlicer := NewPatternSlicer(nil, 0)
	assert.Len(t, emptySlicer.SliceAll([]byte("data")), 1)
	_, okNoMatch := s1.Slice([]byte("no-match"))
	assert.False(t, okNoMatch)
}

func TestScannerRoutines(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "abc", TrimSpaceASCII("  abc  \t\r\n"))
	assert.Equal(t, "abc", string(TrimSpaceASCIIBytes([]byte(" \r\n abc \t"))))
	assert.Equal(t, "hello", TrimQuotesStr(`"hello"`))
	assert.Equal(t, `hello"`, TrimQuotesStr(`hello"`))

	b1, b2, ok := CutByte("a:b", ':')
	assert.True(t, ok)
	assert.Equal(t, "a", b1)
	assert.Equal(t, "b", b2)

	bb1, bb2, okBytes := CutByteBytes([]byte("foo=bar"), '=')
	assert.True(t, okBytes)
	assert.Equal(t, "foo", string(bb1))
	assert.Equal(t, "bar", string(bb2))

	var tokens []string
	for token := range ScanTokens("gzip, deflate, br", ',') {
		tokens = append(tokens, token)
	}
	assert.Equal(t, []string{"gzip", "deflate", "br"}, tokens)

	var tokenBytes [][]byte
	for tb := range ScanTokensBytes([]byte("a, b, c"), ',') {
		tokenBytes = append(tokenBytes, tb)
	}
	assert.Len(t, tokenBytes, 3)

	pairs := make(map[string]string)
	for k, v := range ScanPairs("a=1; b=2; c=3", ';', '=') {
		pairs[k] = v
	}
	assert.Equal(t, map[string]string{"a": "1", "b": "2", "c": "3"}, pairs)

	pairBytes := make(map[string]string)
	for k, v := range ScanPairsBytes([]byte("k1=v1&k2=v2"), '&', '=') {
		pairBytes[string(k)] = string(v)
	}
	assert.Equal(t, map[string]string{"k1": "v1", "k2": "v2"}, pairBytes)
}

func TestBase64EncodeDecode(t *testing.T) {
	raw := []byte("Hello, High-Performance Zero-Allocation Silicon World 12345!")
	dst := make([]byte, 128)
	n := Base64Encode(dst, raw)
	encoded := dst[:n]

	decDst := make([]byte, 128)
	decN, err := Base64Decode(decDst, encoded)
	assert.NoError(t, err)
	assert.Equal(t, raw, decDst[:decN])
}

func BenchmarkAppendToLower1KB(b *testing.B) {
	src := bytes.Repeat([]byte("The Quick Brown Fox Jumps Over The Lazy Dog. "), 23) // ~1KB
	dst := make([]byte, 0, len(src))
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = AppendToLower(dst[:0], src)
	}
}

func BenchmarkBase64Encode1KB(b *testing.B) {
	src := make([]byte, 1024)
	for i := range src {
		src[i] = byte(i)
	}
	dst := make([]byte, 2048)
	b.SetBytes(1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Base64Encode(dst, src)
	}
}
