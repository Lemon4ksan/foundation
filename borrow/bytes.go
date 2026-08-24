// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow

import (
	"bytes"
	"unsafe"
)

// Bytes represents a zero-copy byte slice handle bound to a generational lifetime.
// It allows sub-nanosecond access to network buffers, decompression ring windows,
// and parser outputs while preventing Use-After-Free in asynchronous contexts.
type Bytes struct {
	ptr  unsafe.Pointer
	len  int
	cap  int
	slot *Slot
	gen  uint32
}

// NewBytes creates a new Bytes handle wrapping a Go byte slice.
func NewBytes(b []byte, slot *Slot) Bytes {
	if len(b) == 0 {
		return Bytes{slot: slot}
	}
	var gen uint32
	if slot != nil {
		gen = slot.Generation()
	}
	return Bytes{
		ptr:  unsafe.Pointer(unsafe.SliceData(b)),
		len:  len(b),
		cap:  cap(b),
		slot: slot,
		gen:  gen,
	}
}

// NewBytesRaw creates a new Bytes handle wrapping a raw memory pointer and length.
func NewBytesRaw(ptr unsafe.Pointer, length, capacity int, slot *Slot) Bytes {
	var gen uint32
	if slot != nil {
		gen = slot.Generation()
	}
	return Bytes{
		ptr:  ptr,
		len:  length,
		cap:  capacity,
		slot: slot,
		gen:  gen,
	}
}

// AsSlice returns a []byte view of the underlying memory after verifying
// that the buffer has not been recycled. If the generation is expired, AsSlice panics.
func (b Bytes) AsSlice() []byte {
	if b.len == 0 || b.ptr == nil {
		return nil
	}
	if b.slot != nil {
		b.slot.CheckValid(b.gen)
	}
	return unsafe.Slice((*byte)(b.ptr), b.len)
}

// Clone creates an independent, GC-heap copy of the data that safely outlives
// the current borrow scope.
func (b Bytes) Clone() []byte {
	s := b.AsSlice()
	if len(s) == 0 {
		return nil
	}
	return bytes.Clone(s)
}

// Len returns the byte length of the buffer.
func (b Bytes) Len() int {
	return b.len
}

// Cap returns the byte capacity of the buffer.
func (b Bytes) Cap() int {
	return b.cap
}

// IsValid checks whether this buffer handle is still valid without panicking.
func (b Bytes) IsValid() bool {
	if b.slot == nil {
		return b.ptr != nil
	}
	return b.slot.IsValid(b.gen)
}

// Slice returns a sub-slice of Bytes within [low:high] bound to the same generational slot.
func (b Bytes) Slice(low, high int) Bytes {
	if b.slot != nil {
		b.slot.CheckValid(b.gen)
	}

	if low < 0 || high > b.len || low > high {
		panic("borrow: slice index out of bounds")
	}

	subLen := high - low
	subCap := b.cap - low

	if b.ptr == nil || subLen == 0 {
		return Bytes{slot: b.slot, gen: b.gen}
	}

	subPtr := unsafe.Pointer(uintptr(b.ptr) + uintptr(low))

	return Bytes{
		ptr:  subPtr,
		len:  subLen,
		cap:  subCap,
		slot: b.slot,
		gen:  b.gen,
	}
}

// Release invalidates this Bytes handle and increments the slot's generation.
func (b *Bytes) Release() {
	if b.slot != nil && b.gen != 0 {
		b.slot.Invalidate()
	}

	b.ptr = nil
	b.len = 0
	b.cap = 0
	b.gen = 0
}
