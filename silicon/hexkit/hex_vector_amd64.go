// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package hexkit

import (
	"unsafe"

	"github.com/lemon4ksan/foundation/silicon/cpukit"
)

func encodeVector(dst, src []byte) int {
	if len(src) >= 16 && cpukit.HasAVX2() {
		chunks := len(src) &^ 15
		hex_encode_avx2(
			uint64(uintptr(unsafe.Pointer(&src[0]))),
			uint64(chunks),
			uint64(uintptr(unsafe.Pointer(&dst[0]))),
			0, 0, 0,
		)
		if chunks < len(src) {
			encodeScalar(dst[chunks*2:], src[chunks:])
		}
		return len(src) * 2
	}
	return encodeScalar(dst, src)
}

func decodeVector(dst, src []byte) (int, error) {
	if len(src) >= 32 && cpukit.HasAVX2() {
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
