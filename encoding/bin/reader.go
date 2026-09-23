// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bin

import (
	"encoding/binary"
	"errors"
	"iter"
)

// ErrBufferTooShort is returned when attempting to read beyond available buffer length.
var ErrBufferTooShort = errors.New("binkit: buffer too short")

// Reader provides high-throughput, sequential binary decoding from byte buffers with sticky error tracking.
type Reader struct {
	data []byte
	pos  int
	err  error
}

// NewReader constructs a new [Reader] bound to data.
//
//go:inline
func NewReader(data []byte) Reader {
	return Reader{data: data}
}

// Err returns the first error encountered during decoding, or nil.
//
//go:inline
func (r *Reader) Err() error {
	return r.err
}

// Pos returns the current byte offset in the buffer.
//
//go:inline
func (r *Reader) Pos() int {
	return r.pos
}

// Remaining returns the number of unread bytes left in the buffer.
//
//go:inline
func (r *Reader) Remaining() int {
	if r.pos >= len(r.data) {
		return 0
	}
	return len(r.data) - r.pos
}

// U8 reads a single byte.
//
//go:inline
func (r *Reader) U8() uint8 {
	if r.err != nil {
		return 0
	}
	if r.pos >= len(r.data) {
		r.err = ErrBufferTooShort
		return 0
	}
	v := r.data[r.pos]
	r.pos++
	return v
}

// U16LE reads a little-endian uint16.
//
//go:inline
func (r *Reader) U16LE() uint16 {
	if r.err != nil {
		return 0
	}
	if r.pos+2 > len(r.data) {
		r.err = ErrBufferTooShort
		return 0
	}
	v := binary.LittleEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return v
}

// U32LE reads a little-endian uint32.
//
//go:inline
func (r *Reader) U32LE() uint32 {
	if r.err != nil {
		return 0
	}
	if r.pos+4 > len(r.data) {
		r.err = ErrBufferTooShort
		return 0
	}
	v := binary.LittleEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return v
}

// U64LE reads a little-endian uint64.
//
//go:inline
func (r *Reader) U64LE() uint64 {
	if r.err != nil {
		return 0
	}
	if r.pos+8 > len(r.data) {
		r.err = ErrBufferTooShort
		return 0
	}
	v := binary.LittleEndian.Uint64(r.data[r.pos:])
	r.pos += 8
	return v
}

// U16BE reads a big-endian uint16.
//
//go:inline
func (r *Reader) U16BE() uint16 {
	if r.err != nil {
		return 0
	}
	if r.pos+2 > len(r.data) {
		r.err = ErrBufferTooShort
		return 0
	}
	v := binary.BigEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return v
}

// U32BE reads a big-endian uint32.
//
//go:inline
func (r *Reader) U32BE() uint32 {
	if r.err != nil {
		return 0
	}
	if r.pos+4 > len(r.data) {
		r.err = ErrBufferTooShort
		return 0
	}
	v := binary.BigEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return v
}

// U64BE reads a big-endian uint64.
//
//go:inline
func (r *Reader) U64BE() uint64 {
	if r.err != nil {
		return 0
	}
	if r.pos+8 > len(r.data) {
		r.err = ErrBufferTooShort
		return 0
	}
	v := binary.BigEndian.Uint64(r.data[r.pos:])
	r.pos += 8
	return v
}

// Bytes returns a slice of n bytes without heap allocation.
//
//go:inline
func (r *Reader) Bytes(n int) []byte {
	if r.err != nil || n <= 0 {
		return nil
	}
	if r.pos+n > len(r.data) {
		r.err = ErrBufferTooShort
		return nil
	}
	res := r.data[r.pos : r.pos+n]
	r.pos += n
	return res
}

// String returns a string of length n.
//
//go:inline
func (r *Reader) String(n int) string {
	b := r.Bytes(n)
	if len(b) == 0 {
		return ""
	}
	return string(b)
}

// Skip advances the reader offset by n bytes.
//
//go:inline
func (r *Reader) Skip(n int) {
	if r.err != nil || n <= 0 {
		return
	}
	if r.pos+n > len(r.data) {
		r.err = ErrBufferTooShort
		return
	}
	r.pos += n
}

// Chunks returns a push iterator yielding consecutive fixed-size byte slices of chunkSize from the unread buffer.
// Iteration stops when remaining unread bytes are fewer than chunkSize or when yield returns false.
// Zero heap allocations.
func (r *Reader) Chunks(chunkSize int) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		if chunkSize <= 0 {
			return
		}
		for r.Remaining() >= chunkSize && r.err == nil {
			chunk := r.Bytes(chunkSize)
			if !yield(chunk) {
				return
			}
		}
	}
}

// U8Seq returns a push iterator yielding all remaining bytes sequentially.
func (r *Reader) U8Seq() iter.Seq[uint8] {
	return func(yield func(uint8) bool) {
		for r.Remaining() > 0 && r.err == nil {
			v := r.U8()
			if !yield(v) {
				return
			}
		}
	}
}

// U16LESeq returns a push iterator yielding remaining little-endian uint16 values.
func (r *Reader) U16LESeq() iter.Seq[uint16] {
	return func(yield func(uint16) bool) {
		for r.Remaining() >= 2 && r.err == nil {
			v := r.U16LE()
			if !yield(v) {
				return
			}
		}
	}
}

// U32LESeq returns a push iterator yielding remaining little-endian uint32 values.
func (r *Reader) U32LESeq() iter.Seq[uint32] {
	return func(yield func(uint32) bool) {
		for r.Remaining() >= 4 && r.err == nil {
			v := r.U32LE()
			if !yield(v) {
				return
			}
		}
	}
}

// U64LESeq returns a push iterator yielding remaining little-endian uint64 values.
func (r *Reader) U64LESeq() iter.Seq[uint64] {
	return func(yield func(uint64) bool) {
		for r.Remaining() >= 8 && r.err == nil {
			v := r.U64LE()
			if !yield(v) {
				return
			}
		}
	}
}

// U16BESeq returns a push iterator yielding remaining big-endian uint16 values.
func (r *Reader) U16BESeq() iter.Seq[uint16] {
	return func(yield func(uint16) bool) {
		for r.Remaining() >= 2 && r.err == nil {
			v := r.U16BE()
			if !yield(v) {
				return
			}
		}
	}
}

// U32BESeq returns a push iterator yielding remaining big-endian uint32 values.
func (r *Reader) U32BESeq() iter.Seq[uint32] {
	return func(yield func(uint32) bool) {
		for r.Remaining() >= 4 && r.err == nil {
			v := r.U32BE()
			if !yield(v) {
				return
			}
		}
	}
}

// U64BESeq returns a push iterator yielding remaining big-endian uint64 values.
func (r *Reader) U64BESeq() iter.Seq[uint64] {
	return func(yield func(uint64) bool) {
		for r.Remaining() >= 8 && r.err == nil {
			v := r.U64BE()
			if !yield(v) {
				return
			}
		}
	}
}
