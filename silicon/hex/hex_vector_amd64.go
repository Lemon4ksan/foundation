// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package hex

import (
	"unsafe"

	"golang.org/x/sys/cpu"
)

var hasAVX2 = cpu.X86.HasAVX2

func encodeVector(dst, src []byte) int {
	if len(src) >= 16 && hasAVX2 {
		hex_encode_avx2(
			uint64(uintptr(unsafe.Pointer(&src[0]))),
			uint64(len(src)),
			uint64(uintptr(unsafe.Pointer(&dst[0]))),
			0, 0, 0,
		)
		return len(src) * 2
	}
	return encodeScalar(dst, src)
}

func decodeVector(dst, src []byte) (int, error) {
	if len(src) >= 32 && hasAVX2 {
		ok := hex_decode_avx2(
			uint64(uintptr(unsafe.Pointer(&src[0]))),
			uint64(len(src)),
			uint64(uintptr(unsafe.Pointer(&dst[0]))),
			0, 0, 0,
		)
		if ok == 1 {
			return len(src) / 2, nil
		}
	}
	return decodeScalar(dst, src)
}
