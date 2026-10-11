// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import "errors"

// WireType represents a Protocol Buffers wire type identifier.
type WireType uint8

const (
	// WireVarint represents variable-length integers (int32, int64, uint32, uint64, sint32, sint64, bool, enum).
	WireVarint WireType = 0

	// WireFixed64 represents 64-bit fixed numbers (fixed64, sfixed64, double).
	WireFixed64 WireType = 1

	// WireBytes represents length-delimited data (string, bytes, embedded messages, packed repeated fields).
	WireBytes WireType = 2

	// WireStartGroup represents group start (deprecated).
	WireStartGroup WireType = 3

	// WireEndGroup represents group end (deprecated).
	WireEndGroup WireType = 4

	// WireFixed32 represents 32-bit fixed numbers (fixed32, sfixed32, float).
	WireFixed32 WireType = 5
)

// MaxFieldNumber is the maximum permitted Protocol Buffers field number (2^29 - 1).
const MaxFieldNumber uint32 = (1 << 29) - 1

// Common wire decoding errors.
var (
	ErrTruncated       = errors.New("proto: unexpected end of buffer")
	ErrInvalidWireType = errors.New("proto: invalid wire type")
	ErrVarintOverflow  = errors.New("proto: varint overflow")
	ErrInvalidTag      = errors.New("proto: zero or invalid field number")
)

// Tag computes the encoded tag integer for a given field number and wire type.
//
//go:inline
func Tag(fieldNum uint32, wireType WireType) uint32 {
	return (fieldNum << 3) | uint32(wireType&0x7)
}

// SplitTag decodes an encoded tag integer into its field number and wire type.
//
//go:inline
func SplitTag(tag uint32) (uint32, WireType) {
	return tag >> 3, WireType(tag & 0x7)
}

// VarintLen returns the number of bytes required to encode x as a Protocol Buffers varint.
//
//go:inline
func VarintLen(x uint64) int {
	return SizeVarint(x)
}

// TagSize returns the encoded byte size of a field tag.
//
//go:inline
func TagSize(fieldNum uint32, wireType WireType) int {
	return SizeVarint(uint64(Tag(fieldNum, wireType)))
}
