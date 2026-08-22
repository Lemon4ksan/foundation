// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package endian provides high-throughput zero-copy Little and Big Endian memory primitives.
package endian

import (
	"encoding/binary"
	"unsafe"
)

// Indexer represents any integer type suitable for indexing byte slices.
type Indexer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Load8 loads a single byte from b at offset i without bounds check overhead.
//
//go:inline
func Load8[I Indexer](b []byte, i I) byte {
	return *(*byte)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), i))
}

// Load16 loads a little-endian uint16 from b at offset i.
//
//go:inline
func Load16[I Indexer](b []byte, i I) uint16 {
	return *(*uint16)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), i))
}

// Load32 loads a little-endian uint32 from b at offset i.
//
//go:inline
func Load32[I Indexer](b []byte, i I) uint32 {
	return *(*uint32)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), i))
}

// Load64 loads a little-endian uint64 from b at offset i.
//
//go:inline
func Load64[I Indexer](b []byte, i I) uint64 {
	return *(*uint64)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), i))
}

// Store16 stores a little-endian uint16 into b at offset 0.
//
//go:inline
func Store16(b []byte, v uint16) {
	*(*uint16)(unsafe.Pointer(unsafe.SliceData(b))) = v
}

// Store32 stores a little-endian uint32 into b at offset 0.
//
//go:inline
func Store32(b []byte, v uint32) {
	*(*uint32)(unsafe.Pointer(unsafe.SliceData(b))) = v
}

// Store64 stores a little-endian uint64 into b at offset i.
//
//go:inline
func Store64[I Indexer](b []byte, i I, v uint64) {
	*(*uint64)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), i)) = v
}

// Uint16 returns a little-endian uint16 from b.
//
//go:inline
func Uint16(b []byte) uint16 {
	return binary.LittleEndian.Uint16(b)
}

// Uint32 returns a little-endian uint32 from b.
//
//go:inline
func Uint32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

// Uint64 returns a little-endian uint64 from b.
//
//go:inline
func Uint64(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}

// PutUint16 encodes a little-endian uint16 into b.
//
//go:inline
func PutUint16(b []byte, v uint16) {
	binary.LittleEndian.PutUint16(b, v)
}

// PutUint32 encodes a little-endian uint32 into b.
//
//go:inline
func PutUint32(b []byte, v uint32) {
	binary.LittleEndian.PutUint32(b, v)
}

// PutUint64 encodes a little-endian uint64 into b.
//
//go:inline
func PutUint64(b []byte, v uint64) {
	binary.LittleEndian.PutUint64(b, v)
}
