// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package simd

import "unsafe"

// ScanByteVector scans data for target byte using AVX2 SIMD instructions (32 bytes per cycle).
func ScanByteVector(data []byte, target byte) int {
	if len(data) == 0 {
		return -1
	}
	if len(data) >= 32 && hasAVX2 {
		idx := int64(scan_byte_avx2(
			uint64(uintptr(unsafe.Pointer(&data[0]))),
			uint64(len(data)),
			uint64(target),
			0, 0, 0,
		))
		return int(idx)
	}

	for i, b := range data {
		if b == target {
			return i
		}
	}
	return -1
}

// IndexCRLFCRLFVector searches for "\r\n\r\n" in data and returns the index after the sequence.
func IndexCRLFCRLFVector(data []byte) int {
	if len(data) < 4 {
		return -1
	}
	if len(data) >= 32 && hasAVX2 {
		idx := int64(scan_crlfcrlf_avx2(
			uint64(uintptr(unsafe.Pointer(&data[0]))),
			uint64(len(data)),
			0, 0, 0, 0,
		))
		return int(idx)
	}

	return IndexDoubleCRLF(data)
}
