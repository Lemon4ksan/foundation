// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package simd

import "unsafe"

// EqualFoldVector compares byte slices a and b case-insensitively using AVX2 SIMD instructions.
func EqualFoldVector(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	if len(a) >= 32 && hasAVX2 {
		return equal_fold_ascii_avx2(
			uint64(uintptr(unsafe.Pointer(&a[0]))),
			uint64(uintptr(unsafe.Pointer(&b[0]))),
			uint64(len(a)),
			0, 0, 0,
		) == 1
	}

	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
