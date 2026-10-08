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
		if err := slot.TryAcquireShared(gen); err != nil {
			panic(err)
		}
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
	if r.ptr == nil {
		return false
	}
	if r.slot == nil {
		return true
	}
	return r.slot.IsValid(r.gen)
}

// Generation returns the generation at which this reference was created.
func (r Ref[T]) Generation() uint32 {
	return r.gen
}

// Release safely releases the shared immutable borrow on the slot.
// Calling Release multiple times is safe and idempotent.
func (r *Ref[T]) Release() {
	if r == nil || r.ptr == nil {
		return
	}
	if r.slot != nil && r.gen != 0 {
		r.slot.ReleaseShared()
	}
	r.ptr = nil
	r.gen = 0
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
		if err := slot.TryAcquireExclusive(gen); err != nil {
			panic(err)
		}
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
func (m *Mut[T]) Freeze() Ref[T] {
	if m == nil || m.ptr == nil {
		return Ref[T]{}
	}
	if m.slot != nil {
		m.slot.CheckValid(m.gen)
		m.slot.Freeze()
	}
	ref := Ref[T]{
		ptr:  m.ptr,
		slot: m.slot,
		gen:  m.gen,
	}
	m.ptr = nil
	m.gen = 0
	return ref
}

// Release safely releases the exclusive mutable borrow on the slot.
// Calling Release multiple times is safe and idempotent.
func (m *Mut[T]) Release() {
	if m == nil || m.ptr == nil {
		return
	}
	if m.slot != nil && m.gen != 0 {
		m.slot.ReleaseExclusive()
	}
	m.ptr = nil
	m.gen = 0
}

// IsValid checks whether this mutable reference is still active and valid.
func (m Mut[T]) IsValid() bool {
	if m.ptr == nil {
		return false
	}
	if m.slot == nil {
		return true
	}
	return m.slot.IsValid(m.gen)
}

// Generation returns the generation at which this reference was created.
func (m Mut[T]) Generation() uint32 {
	return m.gen
}
