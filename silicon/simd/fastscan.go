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

// IsCompleteFast performs an early boundary check for HTTP/1.x headers.
//
// Ported from hyper's is_complete_fast:
// When reading chunked / fragmented packets from a TCP stream, this function
// avoids scanning the entire buffer from index 0 on each read iteration.
// Instead, it starts scanning from max(0, prevLen - 3) to find the "\r\n\r\n" or "\n\n" terminator.
func IsCompleteFast(buf []byte, prevLen int) bool {
	start := prevLen - 3
	if start < 0 {
		start = 0
	}
	if start >= len(buf) {
		return false
	}

	tail := buf[start:]
	n := len(tail)
	if n < 2 {
		return false
	}

	_ = tail[n-1]

	for i := 0; i < n; i++ {
		b := tail[i]
		if b == '\r' {
			if i+3 < n && *(*uint32)(unsafe.Pointer(&tail[i])) == CRLFCRLFUint32 {
				return true
			}
		} else if b == '\n' {
			if i+1 < n && tail[i+1] == '\n' {
				return true
			}
			if i+2 < n && tail[i+1] == '\r' && tail[i+2] == '\n' {
				return true
			}
		}
	}

	return false
}

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
