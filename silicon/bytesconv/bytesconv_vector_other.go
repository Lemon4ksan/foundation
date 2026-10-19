// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !amd64 || purego

package bytesconv

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
