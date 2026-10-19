// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hexkit

import (
	"errors"
	"math"
	"math/bits"
	"slices"
	"unsafe"
)

// Common hex errors.
var (
	ErrInvalidLength  = errors.New("hexkit: odd length hex string")
	ErrInvalidByte    = errors.New("hexkit: invalid byte in hex string")
	ErrBufferTooSmall = errors.New("hexkit: destination buffer too small")
	ErrEmptyHexNum    = errors.New("hexkit: empty hex number")
	ErrHexOverflow    = errors.New("hexkit: integer overflow")
)

const (
	hextable      = "0123456789abcdef"
	UpperHexTable = "0123456789ABCDEF"
)

// hexLUT16 maps every uint8 value (0..255) to its 2-byte lowercase ASCII representation in Little-Endian.
var hexLUT16 [256]uint16

// decodeLUT maps ASCII bytes to their 4-bit nibble value (0..15), or 0xFF if invalid.
var decodeLUT [256]uint8

func init() {
	for i := range 256 {
		hi := hextable[i>>4]
		lo := hextable[i&0x0f]
		// Little-endian uint16 representation of [hi, lo]
		hexLUT16[i] = uint16(hi) | (uint16(lo) << 8)
	}

	for i := range 256 {
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

// IsHex reports whether c is an ASCII hexadecimal digit ('0'-'9', 'a'-'f', 'A'-'F').
//
//go:inline
func IsHex(c byte) bool {
	return decodeLUT[c] <= 0x0F
}

// DecodeByte decodes two ASCII hex characters into a single byte using a fast branchless check.
// Returns (val, true) on success, or (0, false) if either character is invalid hex.
//
//go:inline
func DecodeByte(hi, lo byte) (byte, bool) {
	h := decodeLUT[hi]
	l := decodeLUT[lo]
	if (h | l) > 0x0F {
		return 0, false
	}
	return (h << 4) | l, true
}

// ToHexUpper returns uppercase ASCII hex bytes for b (e.g. for percent-encoding).
//
//go:inline
func ToHexUpper(b byte) (hi, lo byte) {
	return UpperHexTable[b>>4], UpperHexTable[b&0x0F]
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

// AppendEncode encodes src into lowercase hex and appends directly to dst,
// pre-growing the slice to avoid multiple reallocations.
//
//go:inline
func AppendEncode(dst, src []byte) []byte {
	needed := len(src) * 2
	if needed == 0 {
		return dst
	}
	start := len(dst)
	dst = slices.Grow(dst, needed)[:start+needed]
	Encode(dst[start:], src)
	return dst
}

func encodeScalar(dst, src []byte) int {
	n := len(src)
	if n == 0 {
		return 0
	}
	_ = dst[n*2-1]
	_ = src[n-1]
	_ = hexLUT16[255]

	i := 0
	for i+4 <= n {
		s0 := uint64(hexLUT16[src[i]])
		s1 := uint64(hexLUT16[src[i+1]])
		s2 := uint64(hexLUT16[src[i+2]])
		s3 := uint64(hexLUT16[src[i+3]])
		*(*uint64)(unsafe.Pointer(&dst[i*2])) = s0 | (s1 << 16) | (s2 << 32) | (s3 << 48)
		i += 4
	}
	for ; i < n; i++ {
		*(*uint16)(unsafe.Pointer(&dst[i*2])) = hexLUT16[src[i]]
	}
	return n * 2
}

// Encode16 encodes a fixed 16-byte buffer (e.g. UUID / TraceID) into a 32-byte hex slice.
//
//go:inline
func Encode16[T ~[16]byte](dst *[32]byte, src T) {
	s := (*[16]byte)(unsafe.Pointer(&src))
	_ = dst[31]
	_ = hexLUT16[255]
	w0 := uint64(
		hexLUT16[s[0]],
	) | (uint64(hexLUT16[s[1]]) << 16) | (uint64(hexLUT16[s[2]]) << 32) | (uint64(hexLUT16[s[3]]) << 48)
	w1 := uint64(
		hexLUT16[s[4]],
	) | (uint64(hexLUT16[s[5]]) << 16) | (uint64(hexLUT16[s[6]]) << 32) | (uint64(hexLUT16[s[7]]) << 48)
	w2 := uint64(
		hexLUT16[s[8]],
	) | (uint64(hexLUT16[s[9]]) << 16) | (uint64(hexLUT16[s[10]]) << 32) | (uint64(hexLUT16[s[11]]) << 48)
	w3 := uint64(
		hexLUT16[s[12]],
	) | (uint64(hexLUT16[s[13]]) << 16) | (uint64(hexLUT16[s[14]]) << 32) | (uint64(hexLUT16[s[15]]) << 48)
	*(*uint64)(unsafe.Pointer(&dst[0])) = w0
	*(*uint64)(unsafe.Pointer(&dst[8])) = w1
	*(*uint64)(unsafe.Pointer(&dst[16])) = w2
	*(*uint64)(unsafe.Pointer(&dst[24])) = w3
}

// Encode8 encodes a fixed 8-byte buffer (e.g. SpanID / uint64) into a 16-byte hex slice.
//
//go:inline
func Encode8[T ~[8]byte](dst *[16]byte, src T) {
	s := (*[8]byte)(unsafe.Pointer(&src))
	_ = dst[15]
	_ = hexLUT16[255]
	w0 := uint64(
		hexLUT16[s[0]],
	) | (uint64(hexLUT16[s[1]]) << 16) | (uint64(hexLUT16[s[2]]) << 32) | (uint64(hexLUT16[s[3]]) << 48)
	w1 := uint64(
		hexLUT16[s[4]],
	) | (uint64(hexLUT16[s[5]]) << 16) | (uint64(hexLUT16[s[6]]) << 32) | (uint64(hexLUT16[s[7]]) << 48)
	*(*uint64)(unsafe.Pointer(&dst[0])) = w0
	*(*uint64)(unsafe.Pointer(&dst[8])) = w1
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
	if needed == 0 {
		return 0, nil
	}
	_ = dst[needed-1]
	_ = src[len(src)-1]
	_ = decodeLUT[255]
	for i := range needed {
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
	src := unsafe.Slice(unsafe.StringData(s), 32)
	d := (*[16]byte)(unsafe.Pointer(dst))
	_ = src[31]
	_ = d[15]
	_ = decodeLUT[255]
	for i := range 16 {
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
	src := unsafe.Slice(unsafe.StringData(s), 16)
	d := (*[8]byte)(unsafe.Pointer(dst))
	_ = src[15]
	_ = d[7]
	_ = decodeLUT[255]
	for i := range 8 {
		hi := decodeLUT[src[i*2]]
		lo := decodeLUT[src[i*2+1]]
		if (hi|lo)&0xf0 != 0 {
			return false
		}
		d[i] = (hi << 4) | lo
	}
	return true
}

// FormatHexUint writes the lowercase hexadecimal representation of val into buf
// without heap allocations, returning the number of bytes written.
// FormatHexUint panics if val is negative.
//
//go:inline
func FormatHexUint(buf *[16]byte, val int) int {
	if val < 0 {
		panic("FormatHexUint: int must be positive")
	}
	if val == 0 {
		buf[0] = '0'
		return 1
	}

	u := uint64(val)
	count := (67 - bits.LeadingZeros64(u)) >> 2

	i := count - 1
	for u >= 0x100 {
		pair := hexLUT16[byte(u)]
		buf[i] = byte(pair >> 8)
		buf[i-1] = byte(pair)
		u >>= 8
		i -= 2
	}
	for u >= 16 {
		buf[i] = hextable[u&0x0F]
		u >>= 4
		i--
	}
	buf[i] = hextable[u]

	return count
}

// AppendHexUint formats val as lowercase hexadecimal and appends directly to dst.
// AppendHexUint panics if val is negative.
//
//go:inline
func AppendHexUint(dst []byte, val int) []byte {
	var buf [16]byte
	n := FormatHexUint(&buf, val)
	return append(dst, buf[:n]...)
}

// ParseHexUint parses a hex-encoded unsigned integer from src.
// Prevents parsing errors due to integer overflow.
// Returns the parsed integer, number of bytes consumed, or an error.
//
//go:inline
func ParseHexUint(src []byte) (int, int, error) {
	n := len(src)
	if n == 0 {
		return 0, 0, ErrEmptyHexNum
	}

	c0 := src[0]
	d0 := decodeLUT[c0]
	if d0 > 0x0F {
		return 0, 0, ErrEmptyHexNum
	}

	val := int(d0)
	i := 1

	// Fast-path: up to 15 hex digits (60 bits) cannot overflow math.MaxInt (63 bits)
	maxSafe := min(n, 15)
	for i < maxSafe {
		c := src[i]
		d := decodeLUT[c]
		if d > 0x0F {
			return val, i, nil
		}
		val = (val << 4) | int(d)
		i++
	}

	// Remainder checks for overflow
	for i < n {
		c := src[i]
		d := decodeLUT[c]
		if d > 0x0F {
			return val, i, nil
		}
		if val > (math.MaxInt-int(d))>>4 {
			return 0, 0, ErrHexOverflow
		}
		val = (val << 4) | int(d)
		i++
	}

	return val, i, nil
}

// ParseHexUint64 parses a hex-encoded uint64 from src.
// Prevents parsing errors due to integer overflow against math.MaxUint64.
// Returns the parsed integer, number of bytes consumed, or an error.
//
//go:inline
func ParseHexUint64(src []byte) (uint64, int, error) {
	n := len(src)
	if n == 0 {
		return 0, 0, ErrEmptyHexNum
	}

	c0 := src[0]
	d0 := decodeLUT[c0]
	if d0 > 0x0F {
		return 0, 0, ErrEmptyHexNum
	}

	val := uint64(d0)
	i := 1

	// Fast-path: up to 16 hex digits (64 bits) cannot overflow uint64
	maxSafe := min(n, 16)
	for i < maxSafe {
		c := src[i]
		d := decodeLUT[c]
		if d > 0x0F {
			return val, i, nil
		}
		val = (val << 4) | uint64(d)
		i++
	}

	// Remainder checks for overflow
	for i < n {
		c := src[i]
		d := decodeLUT[c]
		if d > 0x0F {
			return val, i, nil
		}
		if val > (math.MaxUint64-uint64(d))>>4 {
			return 0, 0, ErrHexOverflow
		}
		val = (val << 4) | uint64(d)
		i++
	}

	return val, i, nil
}
