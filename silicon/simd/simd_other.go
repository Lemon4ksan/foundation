// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (!amd64 && !arm64) || purego

package simd

import "unsafe"

// IndexByteVector falls back to SWAR on non-amd64 architectures.
func IndexByteVector(b []byte, c byte) int {
	return IndexByteSWAR(b, c)
}

// IndexTwoBytesVector falls back to SWAR on non-amd64 architectures.
func IndexTwoBytesVector(b []byte, c1, c2 byte) int {
	return IndexByteTwoSWAR(b, c1, c2)
}

// XORMask32 masks slice b using a 4-byte cyclic mask via SWAR 64-bit XOR on non-amd64 architectures.
func XORMask32(b []byte, mask uint32) {
	if len(b) == 0 {
		return
	}

	mask64 := uint64(mask) | (uint64(mask) << 32)
	i := 0

	for i+8 <= len(b) {
		*(*uint64)(unsafe.Pointer(&b[i])) ^= mask64
		i += 8
	}

	maskBytes := [4]byte{
		byte(mask),
		byte(mask >> 8),
		byte(mask >> 16),
		byte(mask >> 24),
	}

	for ; i < len(b); i++ {
		b[i] ^= maskBytes[i&3]
	}
}

func extractBitsHW(val, mask uint64) uint64 {
	return extractBitsSWAR(val, mask)
}

func depositBitsHW(val, mask uint64) uint64 {
	return depositBitsSWAR(val, mask)
}

// PrefetchL1 is a no-op fallback for non-amd64 architectures.
func PrefetchL1(_ unsafe.Pointer) {}

// StreamCopy256 falls back to standard copy on non-amd64 architectures.
func StreamCopy256(dst, src []byte) int {
	return copy(dst, src)
}

// EqualFoldVector compares byte slices a and b case-insensitively.
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

// ScanByteVector falls back to linear search on non-amd64 architectures.
func ScanByteVector(data []byte, target byte) int {
	for i, b := range data {
		if b == target {
			return i
		}
	}
	return -1
}

// IndexCRLFCRLFVector searches for "\r\n\r\n" in data.
func IndexCRLFCRLFVector(data []byte) int {
	return IndexCRLFCRLF(data)
}

// FindMatchLengthVector falls back to byte comparison on non-amd64 architectures.
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

// Hash64Vector computes a 64-bit hash fallback on non-amd64 architectures.
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
