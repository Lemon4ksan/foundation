// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package binkit

import "encoding/binary"

// Writer provides fluent zero-allocation binary serialization into a preallocated byte slice.
type Writer struct {
	buf []byte
}

// NewWriter creates a new [Writer] writing into buf (or appending if buf is non-empty).
//
//go:inline
func NewWriter(buf []byte, reserveCap ...int) Writer {
	if len(reserveCap) > 0 && cap(buf)-len(buf) < reserveCap[0] {
		newBuf := make([]byte, len(buf), len(buf)+reserveCap[0])
		copy(newBuf, buf)
		buf = newBuf
	}
	return Writer{buf: buf}
}

// Bytes returns the serialized byte slice.
//
//go:inline
func (w *Writer) Bytes() []byte {
	return w.buf
}

// Len returns the current length of the written buffer.
//
//go:inline
func (w *Writer) Len() int {
	return len(w.buf)
}

// Cap returns the capacity of the written buffer.
//
//go:inline
func (w *Writer) Cap() int {
	return cap(w.buf)
}

// U8 appends a single byte.
//
//go:inline
func (w *Writer) U8(v uint8) *Writer {
	w.buf = append(w.buf, v)
	return w
}

// U16LE appends a little-endian uint16.
//
//go:inline
func (w *Writer) U16LE(v uint16) *Writer {
	w.buf = binary.LittleEndian.AppendUint16(w.buf, v)
	return w
}

// U32LE appends a little-endian uint32.
//
//go:inline
func (w *Writer) U32LE(v uint32) *Writer {
	w.buf = binary.LittleEndian.AppendUint32(w.buf, v)
	return w
}

// U64LE appends a little-endian uint64.
//
//go:inline
func (w *Writer) U64LE(v uint64) *Writer {
	w.buf = binary.LittleEndian.AppendUint64(w.buf, v)
	return w
}

// U16BE appends a big-endian uint16.
//
//go:inline
func (w *Writer) U16BE(v uint16) *Writer {
	w.buf = binary.BigEndian.AppendUint16(w.buf, v)
	return w
}

// U32BE appends a big-endian uint32.
//
//go:inline
func (w *Writer) U32BE(v uint32) *Writer {
	w.buf = binary.BigEndian.AppendUint32(w.buf, v)
	return w
}

// U64BE appends a big-endian uint64.
//
//go:inline
func (w *Writer) U64BE(v uint64) *Writer {
	w.buf = binary.BigEndian.AppendUint64(w.buf, v)
	return w
}

// BytesSlice appends a raw byte slice.
//
//go:inline
func (w *Writer) BytesSlice(b []byte) *Writer {
	w.buf = append(w.buf, b...)
	return w
}

// Raw appends a raw byte slice (alias for BytesSlice).
//
//go:inline
func (w *Writer) Raw(b []byte) *Writer {
	w.buf = append(w.buf, b...)
	return w
}

// String appends a string.
//
//go:inline
func (w *Writer) String(s string) *Writer {
	w.buf = append(w.buf, s...)
	return w
}

// RawString appends a string (alias for String).
//
//go:inline
func (w *Writer) RawString(s string) *Writer {
	w.buf = append(w.buf, s...)
	return w
}

// Pad appends n bytes of value b.
//
//go:inline
func (w *Writer) Pad(n int, b byte) *Writer {
	for i := 0; i < n; i++ {
		w.buf = append(w.buf, b)
	}
	return w
}
