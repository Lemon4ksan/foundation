// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow

// Box represents a linear, uniquely owned resource of type T.
//
// In accordance with affine ownership semantics, a Box must either be:
//  1. Explicitly released via [Box.Release] when no longer needed, or
//  2. Transferred to another function via [Box.Move].
//
// Once released or moved, any outstanding [Ref] or [Mut] handles derived from it
// are immediately invalidated via generational increment.
type Box[T any] struct {
	ptr  *T
	slot *Slot
	gen  uint32
}

// NewBox creates a new owned Box wrapping a heap-allocated instance of T.
func NewBox[T any](val T) Box[T] {
	ptr := new(T)
	*ptr = val
	slot := NewSlot()
	return Box[T]{
		ptr:  ptr,
		slot: slot,
		gen:  slot.Generation(),
	}
}

// NewBoxPtr creates a new owned Box wrapping an existing pointer and slot.
func NewBoxPtr[T any](ptr *T, slot *Slot) Box[T] {
	if slot == nil {
		slot = NewSlot()
	}
	return Box[T]{
		ptr:  ptr,
		slot: slot,
		gen:  slot.Generation(),
	}
}

// Borrow borrows the boxed value as an immutable reference.
func (b Box[T]) Borrow() Ref[T] {
	if b.slot != nil {
		b.slot.CheckValid(b.gen)
	}
	return Ref[T](b)
}

// BorrowMut borrows the boxed value as an exclusive mutable reference.
func (b *Box[T]) BorrowMut() Mut[T] {
	if b.slot != nil {
		b.slot.CheckValid(b.gen)
	}
	return Mut[T](*b)
}

// Get returns the underlying pointer after verifying generation validity.
func (b Box[T]) Get() *T {
	if b.slot != nil {
		b.slot.CheckValid(b.gen)
	}
	return b.ptr
}

// Move transfers ownership of this Box to a new Box instance,
// invalidating the current instance.
func (b *Box[T]) Move() Box[T] {
	if b.slot != nil {
		b.slot.CheckValid(b.gen)
	}
	res := Box[T]{
		ptr:  b.ptr,
		slot: b.slot,
		gen:  b.gen,
	}
	// Zero out original to prevent double usage
	b.ptr = nil
	b.gen = 0
	return res
}

// Release consumes and destroys this Box, incrementing the generational counter
// and invalidating all borrowed references derived from it.
func (b *Box[T]) Release() {
	if b.slot != nil && b.gen != 0 {
		b.slot.Invalidate()
	}
	b.ptr = nil
	b.gen = 0
}

// IsValid checks whether this Box is still valid and not yet released or moved.
func (b Box[T]) IsValid() bool {
	if b.slot == nil || b.ptr == nil {
		return false
	}
	return b.slot.IsValid(b.gen)
}
