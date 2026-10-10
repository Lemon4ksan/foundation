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
	ptr   unsafe.Pointer
	len   int
	cap   int
	slot  *Slot
	gen   uint32
	bound bool
}

// NewBytes creates a new Bytes handle wrapping a Go byte slice.
func NewBytes(b []byte, slot *Slot) Bytes {
	if len(b) == 0 {
		var gen uint32
		bound := false
		if slot != nil {
			gen = slot.Generation()
			bound = slot.IsBound()
		}
		return Bytes{slot: slot, gen: gen, bound: bound}
	}
	var gen uint32
	bound := false
	if slot != nil {
		gen = slot.Generation()
		bound = slot.IsBound()
	}
	return Bytes{
		ptr:   unsafe.Pointer(unsafe.SliceData(b)),
		len:   len(b),
		cap:   cap(b),
		slot:  slot,
		gen:   gen,
		bound: bound,
	}
}

// NewBytesRaw creates a new Bytes handle wrapping a raw memory pointer and length.
func NewBytesRaw(ptr unsafe.Pointer, length, capacity int, slot *Slot) Bytes {
	var gen uint32
	bound := false
	if slot != nil {
		gen = slot.Generation()
		bound = slot.IsBound()
	}
	return Bytes{
		ptr:   ptr,
		len:   length,
		cap:   capacity,
		slot:  slot,
		gen:   gen,
		bound: bound,
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

// Bytes returns a []byte view of the underlying memory (alias for [AsSlice]).
func (b Bytes) Bytes() []byte {
	return b.AsSlice()
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
	if b.ptr == nil && b.len != 0 {
		return false
	}
	if b.slot == nil {
		return b.ptr != nil
	}
	if b.ptr == nil && b.gen == 0 {
		return false
	}
	return b.slot.IsValid(b.gen)
}

// IsBound reports whether this buffer is bound to an enclosing scope arena.
func (b Bytes) IsBound() bool {
	return b.bound || (b.slot != nil && b.slot.IsBound())
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
		return Bytes{slot: b.slot, gen: b.gen, bound: true}
	}

	subPtr := unsafe.Add(b.ptr, low)

	return Bytes{
		ptr:   subPtr,
		len:   subLen,
		cap:   subCap,
		slot:  b.slot,
		gen:   b.gen,
		bound: true,
	}
}

// Release invalidates this Bytes handle. If the underlying slot is an owning slot,
// it also invalidates the slot. If the slot is bound to an enclosing scope arena,
// the slot is preserved for other scope allocations.
// Calling Release multiple times is safe and idempotent.
func (b *Bytes) Release() {
	if b == nil || (b.ptr == nil && b.len == 0) {
		return
	}
	if b.slot != nil && b.gen != 0 {
		if !b.slot.IsBound() && !b.bound {
			b.slot.Invalidate()
		}
	}

	b.ptr = nil
	b.len = 0
	b.cap = 0
	b.gen = 0
	b.bound = false
}
