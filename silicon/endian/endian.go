// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package endian provides high-throughput zero-copy Little and Big Endian memory primitives
// using unsafe pointer indexing and CPU hardware BSWAP/REV byte swap intrinsics.
package endian

import (
	"encoding/binary"
	"math/bits"
	"unsafe"
)

// Indexer represents any integer type suitable for indexing byte slices.
type Indexer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// -----------------------------------------------------------------------------
// Little-Endian Primitives (LE)
// -----------------------------------------------------------------------------

// Load8 loads a single byte from b at offset i.
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

// LoadLE16 loads a little-endian uint16 from b at offset i.
//
//go:inline
func LoadLE16[I Indexer](b []byte, i I) uint16 {
	return Load16(b, i)
}

// LoadLE32 loads a little-endian uint32 from b at offset i.
//
//go:inline
func LoadLE32[I Indexer](b []byte, i I) uint32 {
	return Load32(b, i)
}

// LoadLE64 loads a little-endian uint64 from b at offset i.
//
//go:inline
func LoadLE64[I Indexer](b []byte, i I) uint64 {
	return Load64(b, i)
}

// StoreLE16 stores a little-endian uint16 into b at offset 0.
//
//go:inline
func StoreLE16(b []byte, v uint16) {
	Store16(b, v)
}

// StoreLE32 stores a little-endian uint32 into b at offset 0.
//
//go:inline
func StoreLE32(b []byte, v uint32) {
	Store32(b, v)
}

// StoreLE64 stores a little-endian uint64 into b at offset i.
//
//go:inline
func StoreLE64[I Indexer](b []byte, i I, v uint64) {
	Store64(b, i, v)
}

// Uint16 returns a little-endian uint16 from b.
//
//go:inline
func Uint16(b []byte) uint16 {
	return binary.LittleEndian.Uint16(b)
}

// Uint16LE returns a little-endian uint16 from b.
//
//go:inline
func Uint16LE(b []byte) uint16 {
	return binary.LittleEndian.Uint16(b)
}

// Uint32 returns a little-endian uint32 from b.
//
//go:inline
func Uint32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

// Uint32LE returns a little-endian uint32 from b.
//
//go:inline
func Uint32LE(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

// Uint64 returns a little-endian uint64 from b.
//
//go:inline
func Uint64(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}

// Uint64LE returns a little-endian uint64 from b.
//
//go:inline
func Uint64LE(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}

// PutUint16 encodes a little-endian uint16 into b.
//
//go:inline
func PutUint16(b []byte, v uint16) {
	binary.LittleEndian.PutUint16(b, v)
}

// PutUint16LE encodes a little-endian uint16 into b.
//
//go:inline
func PutUint16LE(b []byte, v uint16) {
	binary.LittleEndian.PutUint16(b, v)
}

// PutUint32 encodes a little-endian uint32 into b.
//
//go:inline
func PutUint32(b []byte, v uint32) {
	binary.LittleEndian.PutUint32(b, v)
}

// PutUint32LE encodes a little-endian uint32 into b.
//
//go:inline
func PutUint32LE(b []byte, v uint32) {
	binary.LittleEndian.PutUint32(b, v)
}

// PutUint64 encodes a little-endian uint64 into b.
//
//go:inline
func PutUint64(b []byte, v uint64) {
	binary.LittleEndian.PutUint64(b, v)
}

// PutUint64LE encodes a little-endian uint64 into b.
//
//go:inline
func PutUint64LE(b []byte, v uint64) {
	binary.LittleEndian.PutUint64(b, v)
}

// -----------------------------------------------------------------------------
// Big-Endian Primitives (BE)
// -----------------------------------------------------------------------------

// LoadBE16 loads a big-endian uint16 from b at offset i.
//
//go:inline
func LoadBE16[I Indexer](b []byte, i I) uint16 {
	return bits.ReverseBytes16(Load16(b, i))
}

// LoadBE24 loads a 24-bit big-endian integer from b at offset i (common in HTTP/2 and TLS).
//
//go:inline
func LoadBE24[I Indexer](b []byte, i I) uint32 {
	ptr := unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), i)
	b0 := uint32(*(*byte)(ptr))
	b1 := uint32(*(*byte)(unsafe.Add(ptr, 1)))
	b2 := uint32(*(*byte)(unsafe.Add(ptr, 2)))

	return (b0 << 16) | (b1 << 8) | b2
}

// LoadBE32 loads a big-endian uint32 from b at offset i.
//
//go:inline
func LoadBE32[I Indexer](b []byte, i I) uint32 {
	return bits.ReverseBytes32(Load32(b, i))
}

// LoadBE64 loads a big-endian uint64 from b at offset i.
//
//go:inline
func LoadBE64[I Indexer](b []byte, i I) uint64 {
	return bits.ReverseBytes64(Load64(b, i))
}

// StoreBE16 stores a big-endian uint16 into b at offset 0.
//
//go:inline
func StoreBE16(b []byte, v uint16) {
	Store16(b, bits.ReverseBytes16(v))
}

// StoreBE24 stores a 24-bit big-endian integer into b at offset 0 (e.g. HTTP/2 frame length).
//
//go:inline
func StoreBE24(b []byte, v uint32) {
	ptr := unsafe.Pointer(unsafe.SliceData(b))
	*(*byte)(ptr) = byte(v >> 16)
	*(*byte)(unsafe.Add(ptr, 1)) = byte(v >> 8)
	*(*byte)(unsafe.Add(ptr, 2)) = byte(v)
}

// StoreBE32 stores a big-endian uint32 into b at offset 0.
//
//go:inline
func StoreBE32(b []byte, v uint32) {
	Store32(b, bits.ReverseBytes32(v))
}

// StoreBE64 stores a big-endian uint64 into b at offset i.
//
//go:inline
func StoreBE64[I Indexer](b []byte, i I, v uint64) {
	Store64(b, i, bits.ReverseBytes64(v))
}

// Uint16BE returns a big-endian uint16 from b.
//
//go:inline
func Uint16BE(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}

// Uint24BE returns a 24-bit big-endian integer from b.
//
//go:inline
func Uint24BE(b []byte) uint32 {
	return (uint32(b[0]) << 16) | (uint32(b[1]) << 8) | uint32(b[2])
}

// Uint32BE returns a big-endian uint32 from b.
//
//go:inline
func Uint32BE(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

// Uint64BE returns a big-endian uint64 from b.
//
//go:inline
func Uint64BE(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// PutUint16BE encodes a big-endian uint16 into b.
//
//go:inline
func PutUint16BE(b []byte, v uint16) {
	binary.BigEndian.PutUint16(b, v)
}

// PutUint24BE encodes a 24-bit big-endian integer into b.
//
//go:inline
func PutUint24BE(b []byte, v uint32) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}

// PutUint32BE encodes a big-endian uint32 into b.
//
//go:inline
func PutUint32BE(b []byte, v uint32) {
	binary.BigEndian.PutUint32(b, v)
}

// PutUint64BE encodes a big-endian uint64 into b.
//
//go:inline
func PutUint64BE(b []byte, v uint64) {
	binary.BigEndian.PutUint64(b, v)
}
