// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow

import (
	"errors"
	"fmt"
	"sync/atomic"
)

// Standard error indicators for borrow violations.
var (
	// ErrAlreadyBorrowedMut is returned when attempting to borrow a resource that is already exclusively borrowed.
	ErrAlreadyBorrowedMut = errors.New("borrow: resource is already mutably borrowed")

	// ErrAlreadyBorrowedShared is returned when attempting to mutably borrow a resource that is actively shared.
	ErrAlreadyBorrowedShared = errors.New("borrow: resource is already immutably borrowed")

	// ErrExpiredGeneration is returned or panicked when accessing memory from an expired or recycled slot.
	ErrExpiredGeneration = errors.New("borrow: use-after-free detected, generation mismatch")
)

// SlotKind defines whether a slot uniquely owns its lifetime or is bound to an enclosing scope.
//
// The zero value is empty and ready to use.
type SlotKind uint8

const (
	// SlotOwning indicates that the slot owns its resource lifetime and can be invalidated.
	SlotOwning SlotKind = iota
	// SlotBound indicates that the slot is bound to an enclosing scope arena.
	SlotBound
)

// Slot manages the generational lifecycle and borrow counts for an owned or borrowed resource.
//
// The zero value is empty and not ready to use.
type Slot struct {
	gen     atomic.Uint32
	readers atomic.Int32
	writer  atomic.Bool
	kind    SlotKind
}

// NewSlot creates and initializes a new owning generational slot with generation 1.
//
// It executes in O(1) time and allocates memory.
func NewSlot() *Slot {
	s := &Slot{kind: SlotOwning}
	s.gen.Store(1)
	return s
}

// NewBoundSlot creates and initializes a new bound generational slot with generation 1.
//
// Bound slots are tied to an enclosing scope and cannot be invalidated by sub-allocated resources.
// It executes in O(1) time and allocates memory.
func NewBoundSlot() *Slot {
	s := &Slot{kind: SlotBound}
	s.gen.Store(1)
	return s
}

// Kind returns whether this slot is owning or bound.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) Kind() SlotKind {
	if s == nil {
		return SlotOwning
	}
	return s.kind
}

// IsBound reports whether this slot is bound to an enclosing scope.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) IsBound() bool {
	if s == nil {
		return false
	}
	return s.kind == SlotBound
}

// Generation returns the current generation counter of the slot.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) Generation() uint32 {
	if s == nil {
		return 0
	}
	return s.gen.Load()
}

// IsValid checks if the provided gen matches the current slot generation.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) IsValid(gen uint32) bool {
	if s == nil {
		return false
	}
	return s.gen.Load() == gen
}

// Invalidate increments the generation counter if the slot is an owning slot, immediately revoking all active and outstanding borrows.
//
// If the slot is bound to an enclosing scope, Invalidate is a safe no-op.
// Use [Slot.ForceInvalidate] or scope lifecycle methods to invalidate a bound slot.
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) Invalidate() {
	if s == nil || s.kind == SlotBound {
		return
	}
	s.ForceInvalidate()
}

// ForceInvalidate unconditionally increments the generation counter and revokes all active borrows regardless of whether the slot is owning or bound.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) ForceInvalidate() {
	if s == nil {
		return
	}
	s.gen.Add(1)
	s.writer.Store(false)
	s.readers.Store(0)
}

// AdvanceGeneration increments the generation counter and resets active borrows, returning the new generation.
//
// This revokes prior borrows while keeping the slot active for a moved owner.
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) AdvanceGeneration() uint32 {
	if s == nil {
		return 0
	}
	newGen := s.gen.Add(1)
	s.writer.Store(false)
	s.readers.Store(0)
	return newGen
}

// TryAcquireShared attempts to acquire an immutable shared borrow.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) TryAcquireShared(gen uint32) error {
	if s == nil || s.gen.Load() != gen {
		return ErrExpiredGeneration
	}
	if s.writer.Load() {
		return ErrAlreadyBorrowedMut
	}
	s.readers.Add(1)
	// Double check writer didn't acquire in between and generation is still valid.
	if s.writer.Load() || s.gen.Load() != gen {
		s.ReleaseShared()
		if s.gen.Load() != gen {
			return ErrExpiredGeneration
		}
		return ErrAlreadyBorrowedMut
	}
	return nil
}

// ReleaseShared releases one active immutable shared borrow.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) ReleaseShared() {
	if s == nil {
		return
	}
	for {
		r := s.readers.Load()
		if r <= 0 {
			return
		}
		if s.readers.CompareAndSwap(r, r-1) {
			return
		}
	}
}

// TryAcquireExclusive attempts to acquire an exclusive mutable borrow.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) TryAcquireExclusive(gen uint32) error {
	if s == nil || s.gen.Load() != gen {
		return ErrExpiredGeneration
	}
	if !s.writer.CompareAndSwap(false, true) {
		return ErrAlreadyBorrowedMut
	}
	if s.gen.Load() != gen {
		s.writer.Store(false)
		return ErrExpiredGeneration
	}
	if s.readers.Load() > 0 {
		s.writer.Store(false)
		return ErrAlreadyBorrowedShared
	}
	return nil
}

// ReleaseExclusive releases an exclusive mutable borrow.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) ReleaseExclusive() {
	if s == nil {
		return
	}
	s.writer.Store(false)
}

// Freeze transitions an active exclusive mutable borrow to a single shared immutable borrow.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) Freeze() {
	if s == nil {
		return
	}
	s.writer.Store(false)
	s.readers.Add(1)
}

// CheckValid verifies the generation and panics with a descriptive message if invalid.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Slot) CheckValid(expectedGen uint32) {
	if s == nil {
		panic("borrow: nil slot reference")
	}
	curr := s.gen.Load()
	if curr != expectedGen {
		panic(fmt.Sprintf("borrow: use-after-free violation (current generation: %d, expected: %d)", curr, expectedGen))
	}
}
