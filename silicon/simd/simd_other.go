// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !amd64 || purego

package simd

import (
	"math/bits"
	"unsafe"
)

// ParallelExtract64 falls back to SWAR bit extraction on non-amd64 architectures.
func ParallelExtract64(val, mask uint64) uint64 {
	return extractBitsSWAR(val, mask)
}

// ParallelDeposit64 falls back to SWAR bit deposit on non-amd64 architectures.
func ParallelDeposit64(val, mask uint64) uint64 {
	return depositBitsSWAR(val, mask)
}

// PrefetchL1 issues a prefetch instruction for the memory address.
func PrefetchL1(_ unsafe.Pointer) {}

// StreamCopy256 performs streaming copy.
func StreamCopy256(dst, src []byte) int {
	return copy(dst, src)
}

// EqualFoldVector compares byte slices a and b case-insensitively on non-amd64 architectures.
func EqualFoldVector(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// FindMatchLengthVector returns the matching prefix length on non-amd64 architectures.
func FindMatchLengthVector(a, b []byte, maxLen int) int {
	if maxLen <= 0 || len(a) == 0 || len(b) == 0 {
		return 0
	}
	if maxLen > len(a) {
		maxLen = len(a)
	}
	if maxLen > len(b) {
		maxLen = len(b)
	}
	for i := 0; i < maxLen; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return maxLen
}

// Hash64Vector computes a fast 64-bit hash on non-amd64 architectures.
func Hash64Vector(data []byte, seed uint64) uint64 {
	if len(data) == 0 {
		return seed
	}
	h := seed + 0x27D4EB2F165667C5 + uint64(len(data))
	for _, b := range data {
		h ^= uint64(b) * 0x27D4EB2F165667C5
		h = (h << 11) | (h >> (64 - 11))
		h *= 0x9E3779B185EBCA87
	}
	return h
}

// ScanByteVector falls back to SWAR search on non-amd64 architectures.
func ScanByteVector(data []byte, target byte) int {
	return IndexByteSWAR(data, target)
}

// IndexCRLFCRLFVector searches for "\r\n\r\n" in data on non-amd64 architectures.
func IndexCRLFCRLFVector(data []byte) int {
	return IndexCRLFCRLF(data)
}

// MaskCRLF modifies slice in-place to neutralize CR and LF characters to space on non-amd64.
func MaskCRLF(b []byte) {
	for i := range b {
		if b[i] == '\r' || b[i] == '\n' {
			b[i] = ' '
		}
	}
}

// TrailingZeros32 returns the number of trailing zero bits in x on non-amd64.
func TrailingZeros32(x uint32) int {
	return bits.TrailingZeros32(x)
}
