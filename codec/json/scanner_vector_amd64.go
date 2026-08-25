// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package json

import (
	"unsafe"

	"golang.org/x/sys/cpu"
)

var hasAVX2 = cpu.X86.HasAVX2

func skipWhitespaceVector(data []byte, cursor int) int {
	n := len(data)
	if n-cursor >= 32 && hasAVX2 {
		ptr := uintptr(unsafe.Pointer(&data[0]))
		res := json_skip_whitespace_avx2(
			uint64(ptr),
			uint64(n),
			uint64(cursor),
			0, 0, 0,
		)
		return int(res)
	}
	return skipWhitespaceScalar(data, cursor)
}

func scanStringSpecialVector(data []byte, cursor int) int {
	n := len(data)
	if n-cursor >= 32 && hasAVX2 {
		ptr := uintptr(unsafe.Pointer(&data[0]))
		res := json_scan_string_avx2(
			uint64(ptr),
			uint64(n),
			uint64(cursor),
			0, 0, 0,
		)
		return int(int64(res))
	}
	return scanStringSpecialScalar(data, cursor)
}
