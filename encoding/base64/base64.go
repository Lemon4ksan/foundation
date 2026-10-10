// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base64

import (
	"encoding/base64"
	"math"
)

// Base64URLEncodedLen returns the unpadded URL-safe Base64 encoded length for n source bytes.
func Base64URLEncodedLen(n int) int {
	if n <= 0 {
		return 0
	}
	if n > (math.MaxInt-5)/8 {
		panic("base64: encoded length overflows int")
	}

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

// URLEncodedLen returns the unpadded URL-safe Base64 encoded length for n source bytes.
// It provides an idiomatic, non-stuttering entry point to [Base64URLEncodedLen].
func URLEncodedLen(n int) int {
	return Base64URLEncodedLen(n)
}

// EncodeURL writes the unpadded URL-safe Base64 representation of src into dst.
// Returns the number of bytes written to dst.
// It provides an idiomatic, non-stuttering entry point to [Base64EncodeURL].
//
// It panics if len(dst) < URLEncodedLen(len(src)).
func EncodeURL(src, dst []byte) int {
	return Base64EncodeURL(src, dst)
}
