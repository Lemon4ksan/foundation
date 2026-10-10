// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit

import (
	"math/bits"
	"unsafe"
)

// CacheLineSize is the standard CPU cache line size (64 bytes on x86/ARM64),
// used as an alignment boundary to prevent false sharing across cores.
const CacheLineSize = 64

// PageSize is the standard OS memory page size (4096 bytes).
// Aligning bulk allocations to this boundary minimizes TLB misses
// and enables hardware-level memory mapping optimizations.
const PageSize = 4096

// AlignedBytes allocates a byte slice of length size whose first element is aligned
// to the given boundary (typically [CacheLineSize] or [PageSize]) for DMA or SIMD operations.
// Non-power-of-two alignments are rounded up to the next power of two; returns nil if size <= 0.
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

// IsAligned reports whether the first element of b is aligned to the specified byte boundary.
// Empty slices and alignments <= 1 always return true.
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
