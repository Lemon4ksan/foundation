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

// MaskCRLF modifies slice in-place to neutralize CR and LF characters to space.
func MaskCRLF(b []byte) {
	n := len(b)
	if n == 0 {
		return
	}

	if n >= 32 && hasAVX2 {
		applyFastMaskAVX2(b, 0x20202020)
		return
	}

	for i := range b {
		if b[i] == '\r' || b[i] == '\n' {
			b[i] = ' '
		}
	}
}

// ToLowerSWAR converts ASCII uppercase characters to lowercase using 64-bit SWAR.
func ToLowerSWAR(dst, src []byte) {
	n := min(len(dst), len(src))
	i := 0

	for i+8 <= n {
		v := *(*uint64)(unsafe.Pointer(&src[i]))
		// SWAR ASCII lowercase
		mask := v + 0x7f7f7f7f7f7f7f7f
		mask = (mask ^ v) & 0x8080808080808080
		mask = ((v + 0x3f3f3f3f3f3f3f3f) ^ v) & mask
		mask = (mask >> 2) & 0x2020202020202020
		*(*uint64)(unsafe.Pointer(&dst[i])) = v | mask
		i += 8
	}

	for ; i < n; i++ {
		c := src[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		dst[i] = c
	}
}

// ParallelExtract64 performs parallel bit extraction using hardware PEXT instruction when BMI2 is available.
func ParallelExtract64(val, mask uint64) uint64 {
	if hasBMI2 {
		return pext64(val, mask)
	}

	return extractBitsSWAR(val, mask)
}

// TrailingZeros32 returns the number of trailing zero bits in x.
//
//go:inline
func TrailingZeros32(x uint32) int {
	return bits.TrailingZeros32(x)
}

func extractBitsHW(val, mask uint64) uint64 {
	return ParallelExtract64(val, mask)
}

func depositBitsHW(val, mask uint64) uint64 {
	return depositBitsSWAR(val, mask)
}
