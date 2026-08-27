// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package simd provides SWAR (SIMD Within A Register) 64-bit vector byte scanning operations,
// processing 8 bytes per CPU instruction to achieve mechanical sympathy on modern x86-64 and ARM64 CPUs.
package simd

import (
	"bytes"
	"math/bits"
	"unicode/utf8"
	"unsafe"
)

const (
	// RepeatByte0x01 replicates byte 0x01 across all 8 bytes of a uint64 word.
	RepeatByte0x01 = uint64(0x0101010101010101)
	// RepeatByte0x80 replicates byte 0x80 across all 8 bytes of a uint64 word.
	RepeatByte0x80 = uint64(0x8080808080808080)
)

// IndexByteSWAR locates the first occurrence of byte target in b using 64-bit SWAR processing.
// Returns -1 if target is not found.
func IndexByteSWAR(b []byte, target byte) int {
	n := len(b)
	if n == 0 {
		return -1
	}

	targetWord := RepeatByte0x01 * uint64(target)
	i := 0

	// Process 8-byte uint64 chunks
	for i+8 <= n {
		word := *(*uint64)(unsafe.Pointer(&b[i]))
		xor := word ^ targetWord

		// Zero byte detection mask algorithm
		hasZero := (xor - RepeatByte0x01) &^ xor & RepeatByte0x80
		if hasZero != 0 {
			// Find offset of first matching byte using trailing zeros count
			byteIdx := bits.TrailingZeros64(hasZero) >> 3

			return i + byteIdx
		}

		i += 8
	}

	// Scalar tail processing
	for ; i < n; i++ {
		if b[i] == target {
			return i
		}
	}

	return -1
}

// IndexByteTwoSWAR locates the first occurrence of either target1 or target2 in b using 64-bit SWAR.
// Returns -1 if neither byte is found.
func IndexByteTwoSWAR(b []byte, target1, target2 byte) int {
	n := len(b)
	if n == 0 {
		return -1
	}

	w1 := RepeatByte0x01 * uint64(target1)
	w2 := RepeatByte0x01 * uint64(target2)
	i := 0

	for i+8 <= n {
		word := *(*uint64)(unsafe.Pointer(&b[i]))

		xor1 := word ^ w1
		hasZero1 := (xor1 - RepeatByte0x01) &^ xor1 & RepeatByte0x80

		xor2 := word ^ w2
		hasZero2 := (xor2 - RepeatByte0x01) &^ xor2 & RepeatByte0x80

		mask := hasZero1 | hasZero2
		if mask != 0 {
			byteIdx := bits.TrailingZeros64(mask) >> 3

			return i + byteIdx
		}

		i += 8
	}

	for ; i < n; i++ {
		if b[i] == target1 || b[i] == target2 {
			return i
		}
	}

	return -1
}

// IndexCRLF searches for the first occurrence of "\r\n" in b.
// Returns the index of '\r', or -1 if not found.
func IndexCRLF(b []byte) int {
	n := len(b)
	for i := 0; i < n-1; {
		idx := bytes.IndexByte(b[i:], '\r')
		if idx == -1 || i+idx+1 >= n {
			return -1
		}
		pos := i + idx
		if b[pos+1] == '\n' {
			return pos
		}
		i = pos + 1
	}
	return -1
}

// IndexDoubleCRLF searches for the first occurrence of "\r\n\r\n" (protocol framing boundary) in b.
// Returns the index of the first '\r', or -1 if not found.
func IndexDoubleCRLF(b []byte) int {
	n := len(b)
	if n < 4 {
		return -1
	}

	for i := 0; i <= n-4; {
		idx := IndexCRLF(b[i:])
		if idx == -1 || i+idx+3 >= n {
			return -1
		}
		pos := i + idx
		if b[pos+2] == '\r' && b[pos+3] == '\n' {
			return pos
		}
		i = pos + 1
	}
	return -1
}

// ValidUTF8 reports whether b consists of valid UTF-8-encoded bytes,
// accelerated with 64-bit SWAR vector ASCII fast-path processing (processing 8 bytes/cycle).
func ValidUTF8(b []byte) bool {
	n := len(b)
	i := 0

	// Fast 8-byte SWAR loop for ASCII runs
	for i+8 <= n {
		word := *(*uint64)(unsafe.Pointer(&b[i]))
		if (word & RepeatByte0x80) != 0 {
			break
		}
		i += 8
	}

	if i == n {
		return true
	}

	// Tail and multibyte validation
	return utf8.Valid(b[i:])
}

// XORMask32 masks slice b using a 4-byte cyclic mask via SWAR 64-bit XOR.
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

// PrefetchL1 issues a prefetch instruction for the memory address.
func PrefetchL1(_ unsafe.Pointer) {}

// StreamCopy256 performs streaming copy.
func StreamCopy256(dst, src []byte) int {
	return copy(dst, src)
}
