// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package simd

import "unsafe"

// Hash64Vector computes a rapid 64-bit hash across data buffer using AVX2 SIMD acceleration.
func Hash64Vector(data []byte, seed uint64) uint64 {
	if len(data) == 0 {
		return seed
	}

	if len(data) >= 32 && hasAVX2 {
		return hash64_avx2(
			uint64(uintptr(unsafe.Pointer(&data[0]))),
			uint64(len(data)),
			seed,
			0, 0, 0,
		)
	}

	h := seed + 0x27D4EB2F165667C5 + uint64(len(data))
	for _, b := range data {
		h ^= uint64(b) * 0x27D4EB2F165667C5
		h = (h << 11) | (h >> (64 - 11))
		h *= 0x9E3779B185EBCA87
	}
	return h
}
