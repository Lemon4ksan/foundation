// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego && !goexperiment.simd

package simd

import (
	"bytes"
	"unsafe"
)

// ScanByteVector scans data for target byte using 256-bit AVX2 SIMD instructions.
func ScanByteVector(data []byte, target byte) int {
	n := len(data)
	if n == 0 {
		return -1
	}

	if n >= 32 && hasAVX2 {
		idx := int(scan_byte_avx2(
			uint64(uintptr(unsafe.Pointer(&data[0]))),
			uint64(n),
			uint64(target),
			0, 0, 0,
		))
		if idx >= 0 && idx < n {
			return idx
		}
		return -1
	}

	return bytes.IndexByte(data, target)
}

// IndexCRLFCRLFVector searches for "\r\n\r\n" in data and returns the index of the first byte after the sequence.
func IndexCRLFCRLFVector(data []byte) int {
	n := len(data)
	if n < 4 {
		return -1
	}

	if n >= 32 && hasAVX2 {
		idx := int(scan_crlfcrlf_avx2(
			uint64(uintptr(unsafe.Pointer(&data[0]))),
			uint64(n),
			0, 0, 0, 0,
		))
		if idx >= 0 && idx <= n {
			return idx
		}
		return -1
	}

	return IndexDoubleCRLF(data)
}

