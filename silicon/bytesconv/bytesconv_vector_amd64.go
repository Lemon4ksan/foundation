// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package bytesconv

import (
	"encoding/base64"
	"errors"
)

var ErrInvalidBase64 = errors.New("bytesconv: invalid base64 data")

func toLowerVector(dst, src []byte) {
	toLowerScalar(dst, src)
}

func toUpperVector(dst, src []byte) {
	toUpperScalar(dst, src)
}

func base64EncodeVector(dst, src []byte) int {
	if len(src) == 0 {
		return 0
	}
	base64.StdEncoding.Encode(dst, src)
	return base64.StdEncoding.EncodedLen(len(src))
}

func base64DecodeVector(dst, src []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	n, err := base64.StdEncoding.Decode(dst, src)
	if err != nil {
		return 0, ErrInvalidBase64
	}
	return n, nil
}
