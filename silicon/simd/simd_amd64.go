// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package simd

import (
	"math/bits"
	"unsafe"

	"golang.org/x/sys/cpu"
)

var hasAVX2 = cpu.X86.HasAVX2

var hasBMI2 = cpu.X86.HasBMI2

//go:noescape
func applyFastMaskAVX2(b []byte, mask uint32)

//go:noescape
func pext64(val, mask uint64) uint64

//go:noescape
func pdep64(val, mask uint64) uint64

//go:noescape
func prefetchL1(ptr unsafe.Pointer)

//go:noescape
func streamCopy256(dst, src []byte)

// MaskCRLF modifies slice in-place to neutralize CR and LF characters to space.
func MaskCRLF(b []byte) {
	n := len(b)
	if n == 0 {
		return
	}

	tailStart := 0
	if n >= 32 && hasAVX2 {
		applyFastMaskAVX2(b, 0x20202020)
		tailStart = n &^ 31
	}

	for i := tailStart; i < n; i++ {
		if b[i] == '\r' || b[i] == '\n' {
			b[i] = ' '
		}
	}
}

// ParallelExtract64 performs parallel bit extraction using hardware PEXT instruction when BMI2 is available.
func ParallelExtract64(val, mask uint64) uint64 {
	if hasBMI2 {
		return pext64(val, mask)
	}

	return extractBitsSWAR(val, mask)
}

// ParallelDeposit64 performs parallel bit deposit using hardware PDEP instruction when BMI2 is available.
func ParallelDeposit64(val, mask uint64) uint64 {
	if hasBMI2 {
		return pdep64(val, mask)
	}

	return depositBitsSWAR(val, mask)
}

// PrefetchL1 issues an L1 cache prefetch instruction for the memory address.
func PrefetchL1(p unsafe.Pointer) {
	if p != nil {
		prefetchL1(p)
	}
}

// StreamCopy256 performs a 256-bit streaming copy when AVX2 is available, falling back to copy for unaligned head/tail bytes.
func StreamCopy256(dst, src []byte) int {
	n := min(len(dst), len(src))
	if n == 0 {
		return 0
	}

	if n >= 32 && hasAVX2 {
		addr := uintptr(unsafe.Pointer(&dst[0]))
		head := int((32 - (addr & 31)) & 31)
		if head > 0 {
			copy(dst[:head], src[:head])
		}
		rem := n - head
		vecLen := rem &^ 31
		if vecLen >= 32 {
			streamCopy256(dst[head:head+vecLen], src[head:head+vecLen])
		}
		tailPos := head + vecLen
		if tailPos < n {
			copy(dst[tailPos:n], src[tailPos:n])
		}
		return n
	}

	return copy(dst[:n], src[:n])
}

// TrailingZeros32 returns the number of trailing zero bits in x.
//
//go:inline
func TrailingZeros32(x uint32) int {
	return bits.TrailingZeros32(x)
}
