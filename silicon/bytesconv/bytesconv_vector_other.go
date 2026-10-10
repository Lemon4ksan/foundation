// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !amd64 || purego

package bytesconv

import (
	"encoding/base64"
	"errors"
)

var ErrInvalidBase64 = errors.New("bytesconv: invalid base64 data")

func toLowerVector(dst, src []byte) {
	n := min(len(dst), len(src))
	if n == 0 {
		return
	}
	_ = dst[n-1]
	_ = src[n-1]
	_ = toLowerTable[255]
	for i := 0; i < n; i++ {
		dst[i] = toLowerTable[src[i]]
	}
}

func toUpperVector(dst, src []byte) {
	n := min(len(dst), len(src))
	if n == 0 {
		return
	}
	_ = dst[n-1]
	_ = src[n-1]
	_ = toUpperTable[255]
	for i := 0; i < n; i++ {
		dst[i] = toUpperTable[src[i]]
	}
}

func base64EncodeVector(dst, src []byte) int {
	base64.StdEncoding.Encode(dst, src)
	return base64.StdEncoding.EncodedLen(len(src))
}

func base64DecodeVector(dst, src []byte) (int, error) {
	return base64.StdEncoding.Decode(dst, src)
}
