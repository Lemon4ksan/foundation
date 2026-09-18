// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base64_test

import (
	"crypto/sha256"
	stdbase64 "encoding/base64"
	"testing"

	"github.com/lemon4ksan/foundation/testing/assert"

	"github.com/lemon4ksan/foundation/encoding/base64"
)

func TestBase64EncodeURL(t *testing.T) {
	t.Parallel()

	testData := [][]byte{
		[]byte(""),
		[]byte("f"),
		[]byte("fo"),
		[]byte("foo"),
		[]byte("foob"),
		[]byte("fooba"),
		[]byte("foobar"),
		[]byte("https://example.com/api/v1/auth?code=1234567890"),
	}

	for _, data := range testData {
		expected := stdbase64.RawURLEncoding.EncodeToString(data)
		dst := make([]byte, base64.Base64URLEncodedLen(len(data)))
		n := base64.Base64EncodeURL(data, dst)
		assert.Equal(t, len(expected), n)
		assert.Equal(t, expected, string(dst[:n]))
	}
}

func BenchmarkBase64EncodeURL_SHA256(b *testing.B) {
	sum := sha256.Sum256([]byte("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"))
	dst := make([]byte, 64)

	b.ReportAllocs()

	for b.Loop() {
		_ = base64.Base64EncodeURL(sum[:], dst)
	}
}
