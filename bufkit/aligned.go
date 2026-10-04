// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit

import (
	"math/bits"
	"unsafe"
)

// CacheLineSize represents standard CPU cacheline size (64 bytes on x86/ARM64).
const CacheLineSize = 64

// PageSize represents standard OS memory page size (4096 bytes).
const PageSize = 4096

// AlignedBytes allocates a byte slice whose first element is aligned to the given alignment boundary.
// Alignment must be a power of 2 (typically [CacheLineSize] or [PageSize]).
func AlignedBytes(size, alignment int) []byte {
	if size <= 0 {
		return nil
	}
	if alignment < 1 {
		alignment = 1
	}

	// Fast branchless power-of-2 rounding using math/bits
	if (alignment & (alignment - 1)) != 0 {
		alignment = 1 << bits.Len(uint(alignment-1))
	}

	totalSize := size + alignment
	raw := make([]byte, totalSize)

	ptr := uintptr(unsafe.Pointer(unsafe.SliceData(raw)))
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
	ptr := uintptr(unsafe.Pointer(unsafe.SliceData(b)))
	return (ptr & uintptr(alignment-1)) == 0
}
