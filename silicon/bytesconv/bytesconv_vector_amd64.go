// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package bytesconv

import (
	"encoding/base64"
	"errors"
	"runtime"
	"unsafe"
)

var ErrInvalidBase64 = errors.New("bytesconv: invalid base64 data")

//go:inline
func toLowerSWAR64(v uint64) uint64 {
	m1 := v + 0x3f3f3f3f3f3f3f3f
	m2 := ^(v + 0x2525252525252525)
	mask := ((m1 & m2 & 0x8080808080808080) >> 2) & 0x2020202020202020
	return v | mask
}

//go:inline
func toUpperSWAR64(v uint64) uint64 {
	m1 := v + 0x1f1f1f1f1f1f1f1f
	m2 := ^(v + 0x0505050505050505)
	mask := ((m1 & m2 & 0x8080808080808080) >> 2) & 0x2020202020202020
	return v ^ mask
}

func toLowerVector(dst, src []byte) {
	n := min(len(dst), len(src))
	if n == 0 {
		return
	}
	_ = dst[n-1]
	_ = src[n-1]

	i := 0
	for i+32 <= n {
		v0 := *(*uint64)(unsafe.Pointer(&src[i]))
		v1 := *(*uint64)(unsafe.Pointer(&src[i+8]))
		v2 := *(*uint64)(unsafe.Pointer(&src[i+16]))
		v3 := *(*uint64)(unsafe.Pointer(&src[i+24]))

		if ((v0 | v1 | v2 | v3) & 0x8080808080808080) == 0 {
			*(*uint64)(unsafe.Pointer(&dst[i])) = toLowerSWAR64(v0)
			*(*uint64)(unsafe.Pointer(&dst[i+8])) = toLowerSWAR64(v1)
			*(*uint64)(unsafe.Pointer(&dst[i+16])) = toLowerSWAR64(v2)
			*(*uint64)(unsafe.Pointer(&dst[i+24])) = toLowerSWAR64(v3)
			i += 32
			continue
		}
		for j := 0; j < 32; j++ {
			dst[i+j] = toLowerTable[src[i+j]]
		}
		i += 32
	}

	for i+8 <= n {
		v := *(*uint64)(unsafe.Pointer(&src[i]))
		if (v & 0x8080808080808080) == 0 {
			*(*uint64)(unsafe.Pointer(&dst[i])) = toLowerSWAR64(v)
		} else {
			for j := 0; j < 8; j++ {
				dst[i+j] = toLowerTable[src[i+j]]
			}
		}
		i += 8
	}

	for ; i < n; i++ {
		dst[i] = toLowerTable[src[i]]
	}
	runtime.KeepAlive(src)
	runtime.KeepAlive(dst)
}

func toUpperVector(dst, src []byte) {
	n := min(len(dst), len(src))
	if n == 0 {
		return
	}
	_ = dst[n-1]
	_ = src[n-1]

	i := 0
	for i+32 <= n {
		v0 := *(*uint64)(unsafe.Pointer(&src[i]))
		v1 := *(*uint64)(unsafe.Pointer(&src[i+8]))
		v2 := *(*uint64)(unsafe.Pointer(&src[i+16]))
		v3 := *(*uint64)(unsafe.Pointer(&src[i+24]))

		if ((v0 | v1 | v2 | v3) & 0x8080808080808080) == 0 {
			*(*uint64)(unsafe.Pointer(&dst[i])) = toUpperSWAR64(v0)
			*(*uint64)(unsafe.Pointer(&dst[i+8])) = toUpperSWAR64(v1)
			*(*uint64)(unsafe.Pointer(&dst[i+16])) = toUpperSWAR64(v2)
			*(*uint64)(unsafe.Pointer(&dst[i+24])) = toUpperSWAR64(v3)
			i += 32
			continue
		}
		for j := 0; j < 32; j++ {
			dst[i+j] = toUpperTable[src[i+j]]
		}
		i += 32
	}

	for i+8 <= n {
		v := *(*uint64)(unsafe.Pointer(&src[i]))
		if (v & 0x8080808080808080) == 0 {
			*(*uint64)(unsafe.Pointer(&dst[i])) = toUpperSWAR64(v)
		} else {
			for j := 0; j < 8; j++ {
				dst[i+j] = toUpperTable[src[i+j]]
			}
		}
		i += 8
	}

	for ; i < n; i++ {
		dst[i] = toUpperTable[src[i]]
	}
	runtime.KeepAlive(src)
	runtime.KeepAlive(dst)
}

func base64EncodeVector(dst, src []byte) int {
	if len(src) == 0 {
		return 0
	}
	base64.StdEncoding.Encode(dst, src)
	return base64.StdEncoding.EncodedLen(len(src))
}

func base64DecodeVector(dst, src []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	n, err := base64.StdEncoding.Decode(dst, src)
	if err != nil {
		return 0, ErrInvalidBase64
	}
	return n, nil
}
