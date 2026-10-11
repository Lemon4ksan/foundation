// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

// EncodeZigZag32 encodes a signed 32-bit integer into an unsigned ZigZag integer.
//
//go:inline
func EncodeZigZag32(v int32) uint32 {
	return uint32((v << 1) ^ (v >> 31))
}

// DecodeZigZag32 decodes an unsigned ZigZag integer into a signed 32-bit integer.
//
//go:inline
func DecodeZigZag32(v uint32) int32 {
	return int32((v >> 1) ^ -(v & 1))
}

// EncodeZigZag64 encodes a signed 64-bit integer into an unsigned ZigZag integer.
//
//go:inline
func EncodeZigZag64(v int64) uint64 {
	return uint64((v << 1) ^ (v >> 63))
}

// DecodeZigZag64 decodes an unsigned ZigZag integer into a signed 64-bit integer.
//
//go:inline
func DecodeZigZag64(v uint64) int64 {
	return int64((v >> 1) ^ -(v & 1))
}
