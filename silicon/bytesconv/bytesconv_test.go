// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytesconv

import (
	"bytes"
	"maps"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/testing/assert"
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
		name   string
		src    string
		target string
		want   bool
	}{
		{"substring at start", "gzip, deflate", "gzip", true},
		{"mixed case", "Bz gZip", "gzip", true},
		{"all uppercase", "GZIP", "gzip", true},
		{"mixed case exact", "gZip", "gzip", true},
		{"short uppercase", "BR", "br", true},
		{"mixed case middle", "zStD, gzip", "zstd", true},
		{"mismatch", "gzip", "br", false},
		{"empty src", "", "gzip", false},
		{"empty target", "gzip", "", true},
		{"both empty", "", "", true},
		{"target longer than src", "abc", "abcdef", false},
		{"single char match lower", "hello", "h", true},
		{"single char match upper", "hello", "H", true},
		{"single char match non-letter", "user:123", ":", true},
		{"single char mismatch", "hello", "z", false},
		{"single char mismatch non-letter", "user:123", ";", false},
		{"equal length match", "ABCdef", "abcDEF", true},
		{"equal length mismatch", "ABCdef", "abcDEG", false},
		{"substring at end", "hello WORLD", "world", true},
		{"substring with punctuation", "content-type: application/json", "CONTENT-TYPE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ContainsFoldASCII([]byte(tt.src), tt.target))
		})
	}

	t.Run("adversarial_repeated_prefix", func(t *testing.T) {
		t.Parallel()

		// 50,000 'a's followed by 'b'
		src := append(bytes.Repeat([]byte("a"), 50000), 'b')
		// target with 1,000 'a's ending in 'c' (should not match)
		mismatchTarget := strings.Repeat("A", 1000) + "C"
		assert.False(t, ContainsFoldASCII(src, mismatchTarget))

		// target with 1,000 'a's ending in 'b' (should match at the end)
		matchTarget := strings.Repeat("A", 1000) + "B"
		assert.True(t, ContainsFoldASCII(src, matchTarget))
	})
}

func TestParseUintFast(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in      string
		wantVal uint64
		wantOK  bool
	}{
		{"200", 200, true},
		{"0", 0, true},
		{"1234567890", 1234567890, true},
		{"9223372036854775808", 9223372036854775808, true},
		{"18446744073709551615", 18446744073709551615, true},
		{"18446744073709551616", 0, false},
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

func TestFastHash64_Unaligned(t *testing.T) {
	t.Parallel()

	// Create a backing buffer with distinct byte patterns
	data := make([]byte, 128)
	for i := range data {
		data[i] = byte(i*31 + 7)
	}

	// Verify that slices at unaligned byte offsets (1 through 7) do not panic
	// and produce hash values identical to freshly allocated, aligned copies.
	for offset := range 8 {
		unaligned := data[offset : offset+64]
		copied := append([]byte(nil), unaligned...)

		hUnaligned := FastHash64(unaligned)
		hCopied := FastHash64(copied)

		assert.Equal(t, hCopied, hUnaligned)
		assert.NotZero(t, hUnaligned)
	}

	// Verify short, unaligned slices of various lengths
	for offset := 1; offset < 8; offset++ {
		for length := range 32 {
			slice := data[offset : offset+length]
			h := FastHash64(slice)
			copied := append([]byte(nil), slice...)
			assert.Equal(t, FastHash64(copied), h)
		}
	}
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

	// Boundary slices: index 0 and index len(data)
	sBoundaryZero := NewPatternSlicer([]byte("key"), 0)
	parts0, ok0 := sBoundaryZero.Slice([]byte("key:val"))
	assert.True(t, ok0)
	assert.Len(t, parts0, 2)
	assert.Empty(t, parts0[0])
	assert.Equal(t, "key:val", string(parts0[1]))

	sBoundaryEnd := NewPatternSlicer([]byte("val"), 3)
	partsEnd, okEnd := sBoundaryEnd.Slice([]byte("key:val"))
	assert.True(t, okEnd)
	assert.Len(t, partsEnd, 2)
	assert.Equal(t, "key:val", string(partsEnd[0]))
	assert.Empty(t, partsEnd[1])

	// Out-of-bounds offset checks
	sNegative := NewPatternSlicer([]byte("key"), -5)
	_, okNegative := sNegative.Slice([]byte("key:val"))
	assert.False(t, okNegative)

	sOverflow := NewPatternSlicer([]byte("val"), 50)
	_, okOverflow := sOverflow.Slice([]byte("key:val"))
	assert.False(t, okOverflow)

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

	pairs := maps.Collect(ScanPairs("a=1; b=2; c=3", ';', '='))
	assert.Equal(t, map[string]string{"a": "1", "b": "2", "c": "3"}, pairs)

	pairBytes := make(map[string]string)
	for k, v := range ScanPairsBytes([]byte("k1=v1&k2=v2"), '&', '=') {
		pairBytes[string(k)] = string(v)
	}
	assert.Equal(t, map[string]string{"k1": "v1", "k2": "v2"}, pairBytes)
}

func BenchmarkAppendToLower1KB(b *testing.B) {
	src := bytes.Repeat([]byte("The Quick Brown Fox Jumps Over The Lazy Dog. "), 23) // ~1KB
	dst := make([]byte, 0, len(src))
	b.SetBytes(int64(len(src)))

	b.ReportAllocs()

	for b.Loop() {
		_ = AppendToLower(dst[:0], src)
	}
}

func BenchmarkContainsFoldASCII_Adversarial(b *testing.B) {
	// Adversarial payload: 64KB of repeated 'A's, target with 1024 'a's ending in 'B'.
	// Under a naive O(N*M) nested loop, this incurs 64K * 1K = ~65M operations per check.
	// With Rabin-Karp O(N), this executes in linear time with zero allocations.
	src := bytes.Repeat([]byte("A"), 65536)
	target := strings.Repeat("a", 1024) + "B"
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()

	for b.Loop() {
		_ = ContainsFoldASCII(src, target)
	}
}

func BenchmarkFastHash64_Unaligned(b *testing.B) {
	buf := make([]byte, 1025)
	for i := range buf {
		buf[i] = byte(i)
	}
	unaligned := buf[1:] // unaligned 1024 bytes
	b.SetBytes(1024)
	b.ReportAllocs()

	for b.Loop() {
		_ = FastHash64(unaligned)
	}
}

func TestParseUintFast_ValidUint64(t *testing.T) {
	input := []byte("9223372036854775808")
	val, ok := ParseUintFast(input)
	if !ok {
		t.Fatalf("expected to parse successfully, got ok=false")
	}
	if val != 9223372036854775808 {
		t.Fatalf("expected 9223372036854775808, got %d", val)
	}
}

func TestAppendToUpper_And_FormatNumber(t *testing.T) {
	t.Parallel()

	assert.Empty(t, AppendToUpper(nil, nil))
	up := AppendToUpper([]byte("Key: "), []byte("x-custom-header-with-long-name-1234567890"))
	assert.Equal(t, "Key: X-CUSTOM-HEADER-WITH-LONG-NAME-1234567890", string(up))

	// Non-ASCII bytes in AppendToLower and AppendToUpper (>32 bytes and 8..31 bytes)
	nonASCII := append(bytes.Repeat([]byte("AbCdEfGh"), 5), 0x80, 'Z', 'a')
	lowNon := AppendToLower(nil, nonASCII)
	assert.Equal(t, len(nonASCII), len(lowNon))
	upNon := AppendToUpper(nil, nonASCII)
	assert.Equal(t, len(nonASCII), len(upNon))

	assert.Equal(t, "42", FormatNumber(int(42)))
	assert.Equal(t, "-8", FormatNumber(int8(-8)))
	assert.Equal(t, "-16", FormatNumber(int16(-16)))
	assert.Equal(t, "-32", FormatNumber(int32(-32)))
	assert.Equal(t, "-64", FormatNumber(int64(-64)))
	assert.Equal(t, "42", FormatNumber(uint(42)))
	assert.Equal(t, "8", FormatNumber(uint8(8)))
	assert.Equal(t, "16", FormatNumber(uint16(16)))
	assert.Equal(t, "32", FormatNumber(uint32(32)))
	assert.Equal(t, "64", FormatNumber(uint64(64)))
	assert.Equal(t, "100", FormatNumber(uintptr(100)))
	assert.Equal(t, "3.5", FormatNumber(float32(3.5)))
	assert.Equal(t, "2.25", FormatNumber(float64(2.25)))
}

func TestByteBuffer_And_Copy_And_StreamReader(t *testing.T) {
	t.Parallel()

	bb := AcquireByteBuffer()
	defer ReleaseByteBuffer(bb)
	ReleaseByteBuffer(nil)

	_, _ = bb.Write([]byte("hello"))
	_ = bb.WriteByte(' ')
	_, _ = bb.WriteString("world")
	assert.Equal(t, 11, bb.Len())
	assert.Equal(t, "hello world", bb.String())
	assert.Equal(t, []byte("hello world"), bb.Bytes())

	var out bytes.Buffer
	bb.ResetWriter(&out)
	_, _ = bb.WriteString("flushed")
	assert.NoError(t, bb.Flush())
	assert.Equal(t, "flushed", out.String())
	assert.NoError(t, bb.Flush()) // empty flush

	bb.Set([]byte("set-bytes"))
	assert.Equal(t, "set-bytes", bb.String())
	bb.SetString("set-string")
	assert.Equal(t, "set-string", bb.String())

	bb.Reset()
	nFrom, errFrom := bb.ReadFrom(strings.NewReader(strings.Repeat("x", 200)))
	assert.NoError(t, errFrom)
	assert.Equal(t, int64(200), nFrom)

	out.Reset()
	nTo, errTo := bb.WriteTo(&out)
	assert.NoError(t, errTo)
	assert.Equal(t, int64(200), nTo)

	// CopyZeroAlloc and CopyZeroAllocWithLimit
	var dst bytes.Buffer
	nCopy, errCopy := CopyZeroAlloc(&dst, strings.NewReader("copy payload"))
	assert.NoError(t, errCopy)
	assert.Equal(t, int64(12), nCopy)

	dst.Reset()
	_, errLim := CopyZeroAllocWithLimit(&dst, strings.NewReader("too long payload"), 4)
	assert.ErrorIs(t, errLim, ErrBodyTooLarge)

	dst.Reset()
	nLimOK, errLimOK := CopyZeroAllocWithLimit(&dst, strings.NewReader("ok"), 10)
	assert.NoError(t, errLimOK)
	assert.Equal(t, int64(2), nLimOK)

	dst.Reset()
	nNoLim, errNoLim := CopyZeroAllocWithLimit(&dst, strings.NewReader("nolim"), 0)
	assert.NoError(t, errNoLim)
	assert.Equal(t, int64(5), nNoLim)

	// StreamReader methods
	sr := NewStreamReader(strings.NewReader("line1\nline2\nrest"), 128)
	assert.Equal(t, 0, sr.Buffered())
	_, errNegPeek := sr.Peek(-1)
	assert.Error(t, errNegPeek)
	_, errNegDisc := sr.Discard(-1)
	assert.Error(t, errNegDisc)
	nZeroDisc, errZeroDisc := sr.Discard(0)
	assert.NoError(t, errZeroDisc)
	assert.Equal(t, 0, nZeroDisc)

	line1, errSlice := sr.ReadSlice('\n')
	assert.NoError(t, errSlice)
	assert.Equal(t, "line1\n", string(line1))

	b1, errB := sr.ReadByte()
	assert.NoError(t, errB)
	assert.Equal(t, byte('l'), b1)

	p := make([]byte, 4)
	nRead, errRead := sr.Read(p)
	assert.NoError(t, errRead)
	assert.Equal(t, "ine2", string(p[:nRead]))

	nEmptyRead, errEmptyRead := sr.Read(nil)
	assert.NoError(t, errEmptyRead)
	assert.Equal(t, 0, nEmptyRead)

	sr.Reset(strings.NewReader("short"))
	_, errPeekEOF := sr.Peek(100)
	assert.Error(t, errPeekEOF)
}
