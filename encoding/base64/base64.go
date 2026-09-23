// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package base64 provides high-performance and SIMD-accelerated Base64 encoding.
package base64

import (
	"encoding/base64"
)

// Base64URLEncodedLen returns the unpadded URL-safe Base64 encoded length for n source bytes.
func Base64URLEncodedLen(n int) int {
	return (n*8 + 5) / 6
}

// Base64EncodeURL writes the unpadded URL-safe Base64 representation of src into dst.
// Returns the number of bytes written to dst.
//
// It panics if len(dst) < Base64URLEncodedLen(len(src)) with "base64: dst buffer too small for Base64EncodeURL".
func Base64EncodeURL(src, dst []byte) int {
	reqLen := Base64URLEncodedLen(len(src))
	if len(dst) < reqLen {
		panic("base64: dst buffer too small for Base64EncodeURL")
	}

	if hasVectorBase64 {
		return vectorBase64EncodeURL(src, dst)
	}

	enc := base64.RawURLEncoding
	enc.Encode(dst, src)

	return reqLen
}
