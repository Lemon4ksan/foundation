// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package url

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/cpu"
)

var ErrInvalidEscape = errors.New("url: invalid URL escape sequence")

var hasAVX2 = cpu.X86.HasAVX2

func unescapeVector(dst, src []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	if hasAVX2 {
		res := url_unescape_avx2(
			uint64(uintptr(unsafe.Pointer(&src[0]))),
			uint64(len(src)),
			uint64(uintptr(unsafe.Pointer(&dst[0]))),
			0, 0, 0,
		)
		n := int(int64(res))
		if n < 0 {
			return 0, ErrInvalidEscape
		}
		return n, nil
	}
	return unescapeScalar(dst, src)
}
