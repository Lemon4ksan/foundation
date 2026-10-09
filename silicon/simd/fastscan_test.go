// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd_test

import (
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
