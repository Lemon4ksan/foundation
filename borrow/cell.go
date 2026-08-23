// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow

// Cell provides interior mutability with runtime borrow checking,
// analogous to Rust's RefCell<T>.
//
// It allows mutation of data even when referenced through a shared handle,
// while dynamically enforcing the Aliasing XOR Mutability invariant at runtime.
type Cell[T any] struct {
	val  T
	slot *Slot
}

// NewCell constructs a new Cell wrapping the initial value.
func NewCell[T any](val T) *Cell[T] {
	return &Cell[T]{
		val:  val,
		slot: NewSlot(),
	}
}

// Borrow attempts to borrow the value immutably.
// Returns an error if the Cell is currently exclusively mutably borrowed.
func (c *Cell[T]) Borrow() (Ref[T], error) {
	if err := c.slot.TryAcquireShared(c.slot.Generation()); err != nil {
		return Ref[T]{}, err
	}
	return Ref[T]{
		ptr:  &c.val,
		slot: c.slot,
		gen:  c.slot.Generation(),
	}, nil
}

// MustBorrow borrows the value immutably, panicking if already mutably borrowed.
func (c *Cell[T]) MustBorrow() Ref[T] {
	r, err := c.Borrow()
	if err != nil {
		panic(err)
	}
	return r
}

// BorrowMut attempts to borrow the value exclusively for mutation.
// Returns an error if the Cell is actively borrowed (shared or mutable).
func (c *Cell[T]) BorrowMut() (Mut[T], error) {
	if err := c.slot.TryAcquireExclusive(c.slot.Generation()); err != nil {
		return Mut[T]{}, err
	}
	return Mut[T]{
		ptr:  &c.val,
		slot: c.slot,
		gen:  c.slot.Generation(),
	}, nil
}

// MustBorrowMut borrows the value exclusively, panicking on conflict.
func (c *Cell[T]) MustBorrowMut() Mut[T] {
	m, err := c.BorrowMut()
	if err != nil {
		panic(err)
	}
	return m
}

// Get returns a shallow copy of the current inner value.
func (c *Cell[T]) Get() T {
	r := c.MustBorrow()
	defer c.slot.ReleaseShared()
	return *r.Get()
}

// Set replaces the inner value with a new one.
func (c *Cell[T]) Set(val T) {
	m := c.MustBorrowMut()
	defer c.slot.ReleaseExclusive()
	*m.Get() = val
}

// Slot returns the underlying generational slot of this Cell.
func (c *Cell[T]) Slot() *Slot {
	if c == nil {
		return nil
	}
	return c.slot
}
