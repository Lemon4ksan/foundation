// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package uuid

import (
	"unsafe"

	"golang.org/x/sys/cpu"
)

var hasAVX2 = cpu.X86.HasAVX2

func formatVector(u *UUID, dst *[36]byte) {
	if hasAVX2 {
		uuid_format_avx2(
			uint64(uintptr(unsafe.Pointer(u))),
			uint64(uintptr(unsafe.Pointer(dst))),
			0, 0, 0, 0,
		)
		return
	}
	formatScalar(u, dst)
}

func parseVector(s string) (UUID, bool) {
	var u UUID
	if len(s) == 36 && hasAVX2 {
		strPtr := unsafe.StringData(s)
		ok := uuid_parse_avx2(
			uint64(uintptr(unsafe.Pointer(strPtr))),
			uint64(uintptr(unsafe.Pointer(&u[0]))),
			0, 0, 0, 0,
		)
		if ok == 1 {
			return u, true
		}
		return u, false
	}
	return parseScalar(s)
}
