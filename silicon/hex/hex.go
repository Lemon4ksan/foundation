// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package hex provides ultra-high-performance, zero-allocation hexadecimal encoding and decoding primitives
// optimized with mechanical sympathy for CPU L1 cache, branchless arithmetic, and 16-bit LUT stores.
package hex

import (
	"errors"
	"unsafe"
)

// Common hex errors.
var (
	ErrInvalidLength  = errors.New("hex: odd length hex string")
	ErrInvalidByte    = errors.New("hex: invalid byte in hex string")
	ErrBufferTooSmall = errors.New("hex: destination buffer too small")
)

const hextable = "0123456789abcdef"

// hexLUT16 maps every uint8 value (0..255) to its 2-byte lowercase ASCII representation in Little-Endian.
var hexLUT16 [256]uint16

// decodeLUT maps ASCII bytes to their 4-bit nibble value (0..15), or 0xFF if invalid.
var decodeLUT [256]uint8

func init() {
	for i := 0; i < 256; i++ {
		hi := hextable[i>>4]
		lo := hextable[i&0x0f]
		// Little-endian uint16 representation of [hi, lo]
		hexLUT16[i] = uint16(hi) | (uint16(lo) << 8)
	}

	for i := 0; i < 256; i++ {
		decodeLUT[i] = 0xff
	}
	for i := byte('0'); i <= byte('9'); i++ {
		decodeLUT[i] = i - '0'
	}
	for i := byte('a'); i <= byte('f'); i++ {
		decodeLUT[i] = i - 'a' + 10
	}
	for i := byte('A'); i <= byte('F'); i++ {
		decodeLUT[i] = i - 'A' + 10
	}
}

// EncodedLen returns the length of an encoding of n source bytes.
//
//go:inline
func EncodedLen(n int) int {
	return n * 2
}

// DecodedLen returns the length of a decoding of x source bytes.
//
//go:inline
func DecodedLen(x int) int {
	return x / 2
}

// FromHexChar converts a single ASCII hex character to its 4-bit nibble.
// Returns (val, true) on success, or (0, false) if invalid.
//
//go:inline
func FromHexChar(c byte) (byte, bool) {
	val := decodeLUT[c]
	if val > 0x0f {
		return 0, false
	}
	return val, true
}

// Encode encodes src into dst using hardware SIMD vectorization or 16-bit LUT stores.
// dst must have length of at least EncodedLen(len(src)).
//
//go:inline
func Encode(dst, src []byte) int {
	needed := len(src) * 2
	if len(dst) < needed {
		return 0
	}
	return encodeVector(dst, src)
}

func encodeScalar(dst, src []byte) int {
	for i, v := range src {
		*(*uint16)(unsafe.Pointer(&dst[i*2])) = hexLUT16[v]
	}
	return len(src) * 2
}

// Encode16 encodes a fixed 16-byte buffer (e.g. UUID / TraceID) into a 32-byte hex slice.
//
//go:inline
func Encode16[T ~[16]byte](dst *[32]byte, src T) {
	s := (*[16]byte)(unsafe.Pointer(&src))
	*(*uint16)(unsafe.Pointer(&dst[0])) = hexLUT16[s[0]]
	*(*uint16)(unsafe.Pointer(&dst[2])) = hexLUT16[s[1]]
	*(*uint16)(unsafe.Pointer(&dst[4])) = hexLUT16[s[2]]
	*(*uint16)(unsafe.Pointer(&dst[6])) = hexLUT16[s[3]]
	*(*uint16)(unsafe.Pointer(&dst[8])) = hexLUT16[s[4]]
	*(*uint16)(unsafe.Pointer(&dst[10])) = hexLUT16[s[5]]
	*(*uint16)(unsafe.Pointer(&dst[12])) = hexLUT16[s[6]]
	*(*uint16)(unsafe.Pointer(&dst[14])) = hexLUT16[s[7]]
	*(*uint16)(unsafe.Pointer(&dst[16])) = hexLUT16[s[8]]
	*(*uint16)(unsafe.Pointer(&dst[18])) = hexLUT16[s[9]]
	*(*uint16)(unsafe.Pointer(&dst[20])) = hexLUT16[s[10]]
	*(*uint16)(unsafe.Pointer(&dst[22])) = hexLUT16[s[11]]
	*(*uint16)(unsafe.Pointer(&dst[24])) = hexLUT16[s[12]]
	*(*uint16)(unsafe.Pointer(&dst[26])) = hexLUT16[s[13]]
	*(*uint16)(unsafe.Pointer(&dst[28])) = hexLUT16[s[14]]
	*(*uint16)(unsafe.Pointer(&dst[30])) = hexLUT16[s[15]]
}

// Encode8 encodes a fixed 8-byte buffer (e.g. SpanID / uint64) into a 16-byte hex slice.
//
//go:inline
func Encode8[T ~[8]byte](dst *[16]byte, src T) {
	s := (*[8]byte)(unsafe.Pointer(&src))
	*(*uint16)(unsafe.Pointer(&dst[0])) = hexLUT16[s[0]]
	*(*uint16)(unsafe.Pointer(&dst[2])) = hexLUT16[s[1]]
	*(*uint16)(unsafe.Pointer(&dst[4])) = hexLUT16[s[2]]
	*(*uint16)(unsafe.Pointer(&dst[6])) = hexLUT16[s[3]]
	*(*uint16)(unsafe.Pointer(&dst[8])) = hexLUT16[s[4]]
	*(*uint16)(unsafe.Pointer(&dst[10])) = hexLUT16[s[5]]
	*(*uint16)(unsafe.Pointer(&dst[12])) = hexLUT16[s[6]]
	*(*uint16)(unsafe.Pointer(&dst[14])) = hexLUT16[s[7]]
}

// EncodeToString returns the hexadecimal encoding of src as a string.
func EncodeToString(src []byte) string {
	if len(src) == 0 {
		return ""
	}
	buf := make([]byte, len(src)*2)
	Encode(buf, src)
	return unsafe.String(unsafe.SliceData(buf), len(buf))
}

// Decode decodes src into dst using hardware vectorization or branchless nibble lookup.
// Returns the number of bytes written to dst.
func Decode(dst, src []byte) (int, error) {
	if len(src)%2 != 0 {
		return 0, ErrInvalidLength
	}
	needed := len(src) / 2
	if len(dst) < needed {
		return 0, ErrBufferTooSmall
	}
	return decodeVector(dst, src)
}

func decodeScalar(dst, src []byte) (int, error) {
	needed := len(src) / 2
	for i := 0; i < needed; i++ {
		hi := decodeLUT[src[i*2]]
		lo := decodeLUT[src[i*2+1]]
		if (hi|lo)&0xf0 != 0 {
			return i, ErrInvalidByte
		}
		dst[i] = (hi << 4) | lo
	}
	return needed, nil
}

// DecodeString decodes a hex string into a newly allocated byte slice.
func DecodeString(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, ErrInvalidLength
	}
	dst := make([]byte, len(s)/2)
	src := unsafe.Slice(unsafe.StringData(s), len(s))
	_, err := Decode(dst, src)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

// Decode32 decodes a 32-character hex string into a 16-byte array without allocations.
// Returns true on success, false if invalid.
//
//go:inline
func Decode32[T ~[16]byte](dst *T, s string) bool {
	if len(s) != 32 || dst == nil {
		return false
	}
	src := unsafe.Slice(unsafe.StringData(s), len(s))
	d := (*[16]byte)(unsafe.Pointer(dst))
	for i := 0; i < 16; i++ {
		hi := decodeLUT[src[i*2]]
		lo := decodeLUT[src[i*2+1]]
		if (hi|lo)&0xf0 != 0 {
			return false
		}
		d[i] = (hi << 4) | lo
	}
	return true
}

// Decode16 decodes a 16-character hex string into an 8-byte array without allocations.
// Returns true on success, false if invalid.
//
//go:inline
func Decode16[T ~[8]byte](dst *T, s string) bool {
	if len(s) != 16 || dst == nil {
		return false
	}
	src := unsafe.Slice(unsafe.StringData(s), len(s))
	d := (*[8]byte)(unsafe.Pointer(dst))
	for i := 0; i < 8; i++ {
		hi := decodeLUT[src[i*2]]
		lo := decodeLUT[src[i*2+1]]
		if (hi|lo)&0xf0 != 0 {
			return false
		}
		d[i] = (hi << 4) | lo
	}
	return true
}
