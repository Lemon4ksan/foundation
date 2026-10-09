// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simd

import (
	"bytes"
	"unsafe"
)

const (
	// CRLFUint16 is "\r\n" packed into a Little-Endian 16-bit word.
	CRLFUint16 uint16 = 0x0A0D

	// CRLFCRLFUint32 is "\r\n\r\n" packed into a Little-Endian 32-bit word.
	CRLFCRLFUint32 uint32 = 0x0A0D0A0D

	// LFLFUint16 is "\n\n" packed into a Little-Endian 16-bit word.
	LFLFUint16 uint16 = 0x0A0A
)

// MatchCRLF returns true if buf starts with "\r\n" in a single 16-bit CPU load.
//
//go:inline
func MatchCRLF(buf []byte) bool {
	if len(buf) < 2 {
		return false
	}
	return *(*uint16)(unsafe.Pointer(&buf[0])) == CRLFUint16
}

// MatchCRLFCRLF returns true if buf starts with "\r\n\r\n" in a single 32-bit CPU load.
//
//go:inline
func MatchCRLFCRLF(buf []byte) bool {
	if len(buf) < 4 {
		return false
	}
	return *(*uint32)(unsafe.Pointer(&buf[0])) == CRLFCRLFUint32
}

var (
	crlfCrlfBytes = []byte("\r\n\r\n")
	lflfBytes     = []byte("\n\n")
)

// IndexCRLFCRLF searches for the position immediately after "\r\n\r\n" in buf.
// Returns the index of the first body byte, or -1 if headers are incomplete.
func IndexCRLFCRLF(buf []byte) int {
	idx := bytes.Index(buf, crlfCrlfBytes)
	if idx >= 0 {
		return idx + 4
	}
	idx = bytes.Index(buf, lflfBytes)
	if idx >= 0 {
		return idx + 2
	}
	return -1
}

// IndexByteVector locates the first occurrence of byte target in b using SIMD/vector routines.
func IndexByteVector(b []byte, target byte) int {
	return IndexByteSWAR(b, target)
}

// IndexTwoBytesVector locates the first occurrence of either b1 or b2 in b using SIMD/vector routines.
func IndexTwoBytesVector(b []byte, b1, b2 byte) int {
	return IndexByteTwoSWAR(b, b1, b2)
}
