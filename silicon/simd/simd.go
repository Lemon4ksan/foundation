// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd

import (
	"bytes"
	"math/bits"
	"runtime"
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

	_ = b[n-1] // BCE
	basePtr := unsafe.Pointer(unsafe.SliceData(b))
	targetWord := RepeatByte0x01 * uint64(target)
	i := 0

	// Unrolled 32-byte SWAR loop with coalesced zero-byte check
	for i+32 <= n {
		w0 := *(*uint64)(unsafe.Add(basePtr, i))
		w1 := *(*uint64)(unsafe.Add(basePtr, (i + 8)))
		w2 := *(*uint64)(unsafe.Add(basePtr, (i + 16)))
		w3 := *(*uint64)(unsafe.Add(basePtr, (i + 24)))

		x0 := w0 ^ targetWord
		x1 := w1 ^ targetWord
		x2 := w2 ^ targetWord
		x3 := w3 ^ targetWord

		z0 := (x0 - RepeatByte0x01) &^ x0 & RepeatByte0x80
		z1 := (x1 - RepeatByte0x01) &^ x1 & RepeatByte0x80
		z2 := (x2 - RepeatByte0x01) &^ x2 & RepeatByte0x80
		z3 := (x3 - RepeatByte0x01) &^ x3 & RepeatByte0x80

		if (z0 | z1 | z2 | z3) != 0 {
			if z0 != 0 {
				res := i + (bits.TrailingZeros64(z0) >> 3)
				runtime.KeepAlive(b)
				return res
			}
			if z1 != 0 {
				res := i + 8 + (bits.TrailingZeros64(z1) >> 3)
				runtime.KeepAlive(b)
				return res
			}
			if z2 != 0 {
				res := i + 16 + (bits.TrailingZeros64(z2) >> 3)
				runtime.KeepAlive(b)
				return res
			}
			res := i + 24 + (bits.TrailingZeros64(z3) >> 3)
			runtime.KeepAlive(b)
			return res
		}

		i += 32
	}

	// 8-byte uint64 remainder chunks
	for i+8 <= n {
		word := *(*uint64)(unsafe.Add(basePtr, i))
		xor := word ^ targetWord
		hasZero := (xor - RepeatByte0x01) &^ xor & RepeatByte0x80
		if hasZero != 0 {
			res := i + (bits.TrailingZeros64(hasZero) >> 3)
			runtime.KeepAlive(b)
			return res
		}
		i += 8
	}

	// Scalar tail processing
	for ; i < n; i++ {
		if *(*byte)(unsafe.Add(basePtr, i)) == target {
			runtime.KeepAlive(b)
			return i
		}
	}

	runtime.KeepAlive(b)
	return -1
}

// IndexByteTwoSWAR locates the first occurrence of either target1 or target2 in b using 64-bit SWAR.
// Returns -1 if neither byte is found.
func IndexByteTwoSWAR(b []byte, target1, target2 byte) int {
	n := len(b)
	if n == 0 {
		return -1
	}

	_ = b[n-1] // BCE
	basePtr := unsafe.Pointer(unsafe.SliceData(b))
	w1 := RepeatByte0x01 * uint64(target1)
	w2 := RepeatByte0x01 * uint64(target2)
	i := 0

	// 8-byte uint64 SWAR chunks
	for i+8 <= n {
		word := *(*uint64)(unsafe.Add(basePtr, i))
		xor1 := word ^ w1
		hasZero1 := (xor1 - RepeatByte0x01) &^ xor1 & RepeatByte0x80

		xor2 := word ^ w2
		hasZero2 := (xor2 - RepeatByte0x01) &^ xor2 & RepeatByte0x80

		mask := hasZero1 | hasZero2
		if mask != 0 {
			res := i + (bits.TrailingZeros64(mask) >> 3)
			runtime.KeepAlive(b)
			return res
		}

		i += 8
	}

	for ; i < n; i++ {
		c := *(*byte)(unsafe.Add(basePtr, i))
		if c == target1 || c == target2 {
			runtime.KeepAlive(b)
			return i
		}
	}

	runtime.KeepAlive(b)
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
// accelerated with 64-bit SWAR vector ASCII fast-path processing (processing 32 bytes/iter).
func ValidUTF8(b []byte) bool {
	n := len(b)
	if n == 0 {
		return true
	}
	_ = b[n-1] // BCE
	basePtr := unsafe.Pointer(unsafe.SliceData(b))
	i := 0

	// Fast 32-byte SWAR loop for ASCII runs
	for i+32 <= n {
		w0 := *(*uint64)(unsafe.Add(basePtr, i))
		w1 := *(*uint64)(unsafe.Add(basePtr, (i + 8)))
		w2 := *(*uint64)(unsafe.Add(basePtr, (i + 16)))
		w3 := *(*uint64)(unsafe.Add(basePtr, (i + 24)))
		if ((w0 | w1 | w2 | w3) & RepeatByte0x80) != 0 {
			break
		}
		i += 32
	}

	// 8-byte SWAR loop for remainder
	for i+8 <= n {
		w := *(*uint64)(unsafe.Add(basePtr, i))
		if (w & RepeatByte0x80) != 0 {
			break
		}
		i += 8
	}

	runtime.KeepAlive(b)

	if i == n {
		return true
	}

	// Tail and multibyte validation
	return utf8.Valid(b[i:])
}

// XORMask32 masks slice b using a 4-byte cyclic mask via SWAR 64-bit XOR.
func XORMask32(b []byte, mask uint32) {
	n := len(b)
	if n == 0 {
		return
	}
	_ = b[n-1] // BCE
	basePtr := unsafe.Pointer(unsafe.SliceData(b))
	mask64 := uint64(mask) | (uint64(mask) << 32)
	i := 0

	// 4x unrolled 32-byte SWAR XOR
	for i+32 <= n {
		*(*uint64)(unsafe.Add(basePtr, i)) ^= mask64
		*(*uint64)(unsafe.Add(basePtr, (i + 8))) ^= mask64
		*(*uint64)(unsafe.Add(basePtr, (i + 16))) ^= mask64
		*(*uint64)(unsafe.Add(basePtr, (i + 24))) ^= mask64
		i += 32
	}

	for i+8 <= n {
		*(*uint64)(unsafe.Add(basePtr, i)) ^= mask64
		i += 8
	}

	maskBytes := [4]byte{
		byte(mask),
		byte(mask >> 8),
		byte(mask >> 16),
		byte(mask >> 24),
	}

	for ; i < n; i++ {
		*(*byte)(unsafe.Add(basePtr, i)) ^= maskBytes[i&3]
	}
	runtime.KeepAlive(b)
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

const (
	spaceWordSWAR = RepeatByte0x01 * 0x20
	tabWordSWAR   = RepeatByte0x01 * 0x09
)

// TrimLeadingSpaceSWAR strips leading ASCII spaces (' ') and tabs ('\t') from b
// using 64-bit SWAR vector processing.
func TrimLeadingSpaceSWAR(b []byte) []byte {
	n := len(b)
	if n == 0 {
		return b
	}
	basePtr := unsafe.Pointer(unsafe.SliceData(b))
	i := 0

	// 32-byte unrolled SWAR loop
	for i+32 <= n {
		w0 := *(*uint64)(unsafe.Add(basePtr, i))
		w1 := *(*uint64)(unsafe.Add(basePtr, i+8))
		w2 := *(*uint64)(unsafe.Add(basePtr, i+16))
		w3 := *(*uint64)(unsafe.Add(basePtr, i+24))

		x0s := w0 ^ spaceWordSWAR
		z0s := (x0s - RepeatByte0x01) &^ x0s & RepeatByte0x80
		x0t := w0 ^ tabWordSWAR
		z0t := (x0t - RepeatByte0x01) &^ x0t & RepeatByte0x80
		not0 := (^(z0s | z0t)) & RepeatByte0x80

		x1s := w1 ^ spaceWordSWAR
		z1s := (x1s - RepeatByte0x01) &^ x1s & RepeatByte0x80
		x1t := w1 ^ tabWordSWAR
		z1t := (x1t - RepeatByte0x01) &^ x1t & RepeatByte0x80
		not1 := (^(z1s | z1t)) & RepeatByte0x80

		x2s := w2 ^ spaceWordSWAR
		z2s := (x2s - RepeatByte0x01) &^ x2s & RepeatByte0x80
		x2t := w2 ^ tabWordSWAR
		z2t := (x2t - RepeatByte0x01) &^ x2t & RepeatByte0x80
		not2 := (^(z2s | z2t)) & RepeatByte0x80

		x3s := w3 ^ spaceWordSWAR
		z3s := (x3s - RepeatByte0x01) &^ x3s & RepeatByte0x80
		x3t := w3 ^ tabWordSWAR
		z3t := (x3t - RepeatByte0x01) &^ x3t & RepeatByte0x80
		not3 := (^(z3s | z3t)) & RepeatByte0x80

		if (not0 | not1 | not2 | not3) != 0 {
			if not0 != 0 {
				idx := i + (bits.TrailingZeros64(not0) >> 3)
				runtime.KeepAlive(b)
				return b[idx:]
			}
			if not1 != 0 {
				idx := i + 8 + (bits.TrailingZeros64(not1) >> 3)
				runtime.KeepAlive(b)
				return b[idx:]
			}
			if not2 != 0 {
				idx := i + 16 + (bits.TrailingZeros64(not2) >> 3)
				runtime.KeepAlive(b)
				return b[idx:]
			}
			idx := i + 24 + (bits.TrailingZeros64(not3) >> 3)
			runtime.KeepAlive(b)
			return b[idx:]
		}
		i += 32
	}

	for i+8 <= n {
		w := *(*uint64)(unsafe.Add(basePtr, i))
		xs := w ^ spaceWordSWAR
		zs := (xs - RepeatByte0x01) &^ xs & RepeatByte0x80
		xt := w ^ tabWordSWAR
		zt := (xt - RepeatByte0x01) &^ xt & RepeatByte0x80
		not := (^(zs | zt)) & RepeatByte0x80
		if not != 0 {
			idx := i + (bits.TrailingZeros64(not) >> 3)
			runtime.KeepAlive(b)
			return b[idx:]
		}
		i += 8
	}

	for ; i < n; i++ {
		c := *(*byte)(unsafe.Add(basePtr, i))
		if c != ' ' && c != '\t' {
			runtime.KeepAlive(b)
			return b[i:]
		}
	}
	runtime.KeepAlive(b)
	return b[n:]
}

// TrimTrailingSpaceSWAR strips trailing ASCII spaces (' ') and tabs ('\t') from b
// using 64-bit SWAR vector processing.
func TrimTrailingSpaceSWAR(b []byte) []byte {
	n := len(b)
	if n == 0 {
		return b
	}
	basePtr := unsafe.Pointer(unsafe.SliceData(b))

	for n >= 8 {
		w := *(*uint64)(unsafe.Add(basePtr, n-8))
		xs := w ^ spaceWordSWAR
		zs := (xs - RepeatByte0x01) &^ xs & RepeatByte0x80
		xt := w ^ tabWordSWAR
		zt := (xt - RepeatByte0x01) &^ xt & RepeatByte0x80
		not := (^(zs | zt)) & RepeatByte0x80
		if not != 0 {
			lz := bits.LeadingZeros64(not)
			pos := 7 - (lz >> 3)
			runtime.KeepAlive(b)
			return b[:(n-8)+pos+1]
		}
		n -= 8
	}

	for n > 0 {
		c := *(*byte)(unsafe.Add(basePtr, n-1))
		if c != ' ' && c != '\t' {
			break
		}
		n--
	}
	runtime.KeepAlive(b)
	return b[:n]
}

// TrimSpaceSWAR strips both leading and trailing ASCII spaces and tabs from b.
//
//go:inline
func TrimSpaceSWAR(b []byte) []byte {
	return TrimTrailingSpaceSWAR(TrimLeadingSpaceSWAR(b))
}

// IndexColonSWAR locates the first ':' in b using 64-bit SWAR.
//
//go:inline
func IndexColonSWAR(b []byte) int {
	return IndexByteSWAR(b, ':')
}
