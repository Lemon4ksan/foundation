// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package simd

import "unsafe"

// FindMatchLengthVector returns how many consecutive bytes match between a and b (up to maxLen).
// Used as the core LZ77 sliding window match finder for Brotli and Deflate compression.
func FindMatchLengthVector(a, b []byte, maxLen int) int {
	if maxLen <= 0 || len(a) == 0 || len(b) == 0 {
		return 0
	}
	if maxLen > len(a) {
		maxLen = len(a)
	}
	if maxLen > len(b) {
		maxLen = len(b)
	}

	if maxLen >= 32 && hasAVX2 {
		return int(find_match_length_avx2(
			uint64(uintptr(unsafe.Pointer(&a[0]))),
			uint64(uintptr(unsafe.Pointer(&b[0]))),
			uint64(maxLen),
			0, 0, 0,
		))
	}

	for i := 0; i < maxLen; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return maxLen
}
