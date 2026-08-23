// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow

// Ref represents a shared, immutable borrowed reference to a value of type T.
// Multiple Ref instances to the same resource can coexist safely across goroutines.
type Ref[T any] struct {
	ptr  *T
	slot *Slot
	gen  uint32
}

// NewRef constructs a new immutable reference bound to a generational slot.
func NewRef[T any](ptr *T, slot *Slot) Ref[T] {
	var gen uint32
	if slot != nil {
		gen = slot.Generation()
	}
	return Ref[T]{
		ptr:  ptr,
		slot: slot,
		gen:  gen,
	}
}

// Get returns the underlying pointer to T after verifying that the reference
// has not expired. If the resource has been recycled, Get panics immediately.
func (r Ref[T]) Get() *T {
	if r.slot != nil {
		r.slot.CheckValid(r.gen)
	}
	return r.ptr
}

// Val returns a shallow copy of the underlying value of T.
func (r Ref[T]) Val() T {
	p := r.Get()
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// IsValid checks whether this reference is still active and valid without panicking.
func (r Ref[T]) IsValid() bool {
	if r.slot == nil {
		return r.ptr != nil
	}
	return r.slot.IsValid(r.gen)
}

// Generation returns the generation at which this reference was created.
func (r Ref[T]) Generation() uint32 {
	return r.gen
}

// Mut represents an exclusive, mutable borrowed reference to a value of type T.
// Only one Mut can exist per resource at any given moment.
type Mut[T any] struct {
	ptr  *T
	slot *Slot
	gen  uint32
}

// NewMut constructs a new exclusive mutable reference bound to a generational slot.
func NewMut[T any](ptr *T, slot *Slot) Mut[T] {
	var gen uint32
	if slot != nil {
		gen = slot.Generation()
	}
	return Mut[T]{
		ptr:  ptr,
		slot: slot,
		gen:  gen,
	}
}

// Get returns the mutable pointer to T after verifying generation validity.
// If the underlying memory has been recycled or invalidated, Get panics immediately.
func (m Mut[T]) Get() *T {
	if m.slot != nil {
		m.slot.CheckValid(m.gen)
	}
	return m.ptr
}

// Write mutates the underlying value in-place after generation validation.
func (m Mut[T]) Write(val T) {
	p := m.Get()
	if p != nil {
		*p = val
	}
}

// Freeze downgrades this exclusive mutable borrow into a shared immutable reference.
func (m Mut[T]) Freeze() Ref[T] {
	if m.slot != nil {
		m.slot.CheckValid(m.gen)
	}
	return Ref[T](m)
}

// IsValid checks whether this mutable reference is still active and valid.
func (m Mut[T]) IsValid() bool {
	if m.slot == nil {
		return m.ptr != nil
	}
	return m.slot.IsValid(m.gen)
}

// Generation returns the generation at which this reference was created.
func (m Mut[T]) Generation() uint32 {
	return m.gen
}
