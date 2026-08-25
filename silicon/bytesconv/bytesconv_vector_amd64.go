// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package bytesconv

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/cpu"
)

var ErrInvalidBase64 = errors.New("bytesconv: invalid base64 data")

var hasAVX2 = cpu.X86.HasAVX2

func toLowerVector(dst, src []byte) {
	n := len(src)
	if n >= 32 && hasAVX2 {
		to_lower_ascii_avx2(
			uint64(uintptr(unsafe.Pointer(&src[0]))),
			uint64(n),
			uint64(uintptr(unsafe.Pointer(&dst[0]))),
			0, 0, 0,
		)
		return
	}
	toLowerScalar(dst, src)
}

func toUpperVector(dst, src []byte) {
	n := len(src)
	if n >= 32 && hasAVX2 {
		to_upper_ascii_avx2(
			uint64(uintptr(unsafe.Pointer(&src[0]))),
			uint64(n),
			uint64(uintptr(unsafe.Pointer(&dst[0]))),
			0, 0, 0,
		)
		return
	}
	toUpperScalar(dst, src)
}

func base64EncodeVector(dst, src []byte) int {
	if len(src) == 0 {
		return 0
	}
	res := base64_encode_avx2(
		uint64(uintptr(unsafe.Pointer(&src[0]))),
		uint64(len(src)),
		uint64(uintptr(unsafe.Pointer(&dst[0]))),
		0, 0, 0,
	)
	return int(res)
}

func base64DecodeVector(dst, src []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	res := base64_decode_avx2(
		uint64(uintptr(unsafe.Pointer(&src[0]))),
		uint64(len(src)),
		uint64(uintptr(unsafe.Pointer(&dst[0]))),
		0, 0, 0,
	)
	n := int(int64(res))
	if n < 0 {
		return 0, ErrInvalidBase64
	}
	return n, nil
}
