// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/simd"
	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestMatchCRLF(t *testing.T) {
	assert.True(t, simd.MatchCRLF([]byte("\r\nContent-Type: text/plain")))
	assert.False(t, simd.MatchCRLF([]byte("\n\rContent-Type: text/plain")))
	assert.False(t, simd.MatchCRLF([]byte("H")))
}

func TestMatchCRLFCRLF(t *testing.T) {
	assert.True(t, simd.MatchCRLFCRLF([]byte("\r\n\r\nHello")))
	assert.False(t, simd.MatchCRLFCRLF([]byte("\r\n\n\rHello")))
	assert.False(t, simd.MatchCRLFCRLF([]byte("\r\n")))
}

func TestIndexCRLFCRLF(t *testing.T) {
	// 1. CRLF CRLF standard HTTP boundary
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nHELLO")
	idx := simd.IndexCRLFCRLF(raw)
	assert.Equal(t, len(raw)-5, idx)
	assert.Equal(t, "HELLO", string(raw[idx:]))

	// 2. LF LF legacy HTTP boundary
	rawLF := []byte("HTTP/1.1 200 OK\nContent-Length: 5\n\nWORLD")
	idxLF := simd.IndexCRLFCRLF(rawLF)
	assert.Equal(t, len(rawLF)-5, idxLF)
	assert.Equal(t, "WORLD", string(rawLF[idxLF:]))

	// 3. Incomplete header streams
	incomplete := []byte("HTTP/1.1 200 OK\r\nServer: aoni\r\nContent-Length: 10\r\n")
	assert.Equal(t, -1, simd.IndexCRLFCRLF(incomplete))
	assert.Equal(t, -1, simd.IndexCRLFCRLF([]byte("no-boundary-at-all")))
	assert.Equal(t, -1, simd.IndexCRLFCRLF(nil))
	assert.Equal(t, -1, simd.IndexCRLFCRLF([]byte("\r\n\r")))

	// 4. IndexCRLFCRLFVector consistency
	assert.Equal(t, idx, simd.IndexCRLFCRLFVector(raw))
	assert.Equal(t, -1, simd.IndexCRLFCRLFVector(incomplete))
}

func TestIndexCRLFCRLFVector_Equivalence(t *testing.T) {
	// 1. Edge cases
	edgeCases := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"lone_cr", []byte("\r")},
		{"lone_lf", []byte("\n")},
		{"crlf", []byte("\r\n")},
		{"lfcr", []byte("\n\r")},
		{"crlfcr", []byte("\r\n\r")},
		{"lflf", []byte("\n\n")},
		{"crlfcrlf", []byte("\r\n\r\n")},
		{"lflf_with_prefix", []byte("prefix\n\nbody")},
		{"crlfcrlf_with_prefix", []byte("prefix\r\n\r\nbody")},
		{"lflf_at_end", []byte("header\n\n")},
		{"crlfcrlf_at_end", []byte("header\r\n\r\n")},
		{"mixed_lf_before_crlfcrlf", []byte("part1\n\npart2\r\n\r\nbody")},
		{"mixed_crlfcrlf_before_lf", []byte("part1\r\n\r\npart2\n\nbody")},
		{"repeated_cr", []byte("\r\r\r\r")},
		{"repeated_lf", []byte("\n\n\n\n")},
		{"crlf_repeated", []byte("\r\n\r\n\r\n\r\n")},
		{"no_boundary_short", []byte("HTTP/1.1 200 OK")},
		{"no_boundary_long", bytes.Repeat([]byte("A"), 128)},
	}

	for _, tc := range edgeCases {
		expected := simd.IndexCRLFCRLF(tc.data)
		got := simd.IndexCRLFCRLFVector(tc.data)
		assert.Equal(t, expected, got, fmt.Sprintf("edge case mismatch on %s (%q)", tc.name, tc.data))
	}

	// 2. Length sweeps from 0 to 1024
	for l := 0; l <= 1024; l++ {
		// 2a. Buffer with no delimiters
		buf := bytes.Repeat([]byte("A"), l)
		expected := simd.IndexCRLFCRLF(buf)
		got := simd.IndexCRLFCRLFVector(buf)
		if expected != got {
			t.Fatalf("mismatch at length %d (no delimiter): expected %d, got %d", l, expected, got)
		}

		// 2b. Buffer with \r\n\r\n placed at start, middle, and end
		if l >= 4 {
			positions := []int{0, (l - 4) / 2, l - 4}
			for _, pos := range positions {
				b := make([]byte, l)
				copy(b, buf)
				copy(b[pos:], "\r\n\r\n")
				exp := simd.IndexCRLFCRLF(b)
				actual := simd.IndexCRLFCRLFVector(b)
				if exp != actual {
					t.Fatalf("mismatch at length %d with CRLFCRLF at %d: expected %d, got %d", l, pos, exp, actual)
				}
			}
		}

		// 2c. Buffer with \n\n placed at start, middle, and end
		if l >= 2 {
			positions := []int{0, (l - 2) / 2, l - 2}
			for _, pos := range positions {
				b := make([]byte, l)
				copy(b, buf)
				copy(b[pos:], "\n\n")
				exp := simd.IndexCRLFCRLF(b)
				actual := simd.IndexCRLFCRLFVector(b)
				if exp != actual {
					t.Fatalf("mismatch at length %d with LFLF at %d: expected %d, got %d", l, pos, exp, actual)
				}
			}
		}
	}

	// 3. Boundary straddling around 32-byte AVX2 chunk boundaries
	for offset := 28; offset <= 36; offset++ {
		buf := make([]byte, 80)
		for i := range buf {
			buf[i] = 'X'
		}
		// Test \r\n\r\n straddling boundary
		copy(buf[offset:], "\r\n\r\n")
		exp := simd.IndexCRLFCRLF(buf)
		actual := simd.IndexCRLFCRLFVector(buf)
		assert.Equal(t, exp, actual, fmt.Sprintf("boundary straddle CRLFCRLF at offset %d", offset))

		// Test \n\n straddling boundary
		buf2 := make([]byte, 80)
		for i := range buf2 {
			buf2[i] = 'X'
		}
		copy(buf2[offset:], "\n\n")
		expLF := simd.IndexCRLFCRLF(buf2)
		actualLF := simd.IndexCRLFCRLFVector(buf2)
		assert.Equal(t, expLF, actualLF, fmt.Sprintf("boundary straddle LFLF at offset %d", offset))
	}
}
