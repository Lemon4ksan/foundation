// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/silicon/simd"
	"github.com/lemon4ksan/foundation/testkit/assert"
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

func TestIsCompleteFast(t *testing.T) {
	incomplete := []byte("HTTP/1.1 200 OK\r\nServer: aoni\r\nContent-Length: 10\r\n")
	assert.False(t, simd.IsCompleteFast(incomplete, len(incomplete)))

	complete := []byte("HTTP/1.1 200 OK\r\nServer: aoni\r\nContent-Length: 10\r\n\r\nHello")
	assert.True(t, simd.IsCompleteFast(complete, len(incomplete)))

	completeLF := []byte("HTTP/1.1 200 OK\nServer: aoni\nContent-Length: 10\n\nHello")
	assert.True(t, simd.IsCompleteFast(completeLF, len(completeLF)-5))
}

func TestIndexCRLFCRLF(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nHELLO")
	idx := simd.IndexCRLFCRLF(raw)
	assert.Equal(t, len(raw)-5, idx)
	assert.Equal(t, "HELLO", string(raw[idx:]))
}
