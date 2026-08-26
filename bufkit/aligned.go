// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit

import (
	"unsafe"
)

// CacheLineSize represents standard CPU cacheline size (64 bytes on x86/ARM64).
const CacheLineSize = 64

// PageSize represents standard OS memory page size (4096 bytes).
const PageSize = 4096

// AlignedBytes allocates a byte slice whose first element is aligned to the given alignment boundary.
// Alignment must be a power of 2 (typically [CacheLineSize] or [PageSize]).
func AlignedBytes(size, alignment int) []byte {
	if alignment < 1 {
		alignment = 1
	}

	// Round alignment to power of 2
	if (alignment & (alignment - 1)) != 0 {
		pow2 := 1
		for pow2 < alignment {
			pow2 <<= 1
		}
		alignment = pow2
	}

	totalSize := size + alignment
	raw := make([]byte, totalSize)

	ptr := uintptr(unsafe.Pointer(&raw[0]))
	offset := int((uintptr(alignment) - (ptr & uintptr(alignment-1))) & uintptr(alignment-1))

	return raw[offset : offset+size : offset+size]
}

// IsAligned checks whether the given byte slice is aligned to the specified byte boundary.
func IsAligned(b []byte, alignment int) bool {
	if len(b) == 0 {
		return true
	}
	if alignment <= 1 {
		return true
	}
	ptr := uintptr(unsafe.Pointer(&b[0]))
	return (ptr & uintptr(alignment-1)) == 0
}
