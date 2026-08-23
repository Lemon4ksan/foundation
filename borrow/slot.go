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

// Slot manages the generational lifecycle and borrow counts for an owned or borrowed resource.
type Slot struct {
	gen     atomic.Uint32
	readers atomic.Int32
	writer  atomic.Bool
}

// NewSlot creates and initializes a new generational slot with generation 1.
func NewSlot() *Slot {
	s := &Slot{}
	s.gen.Store(1)
	return s
}

// Generation returns the current generation counter of the slot.
func (s *Slot) Generation() uint32 {
	if s == nil {
		return 0
	}
	return s.gen.Load()
}

// IsValid checks if the provided gen matches the current slot generation.
func (s *Slot) IsValid(gen uint32) bool {
	if s == nil {
		return false
	}
	return s.gen.Load() == gen
}

// Invalidate increments the generation counter, immediately revoking all active
// and outstanding borrows associated with previous generations.
func (s *Slot) Invalidate() {
	if s == nil {
		return
	}
	s.writer.Store(false)
	s.readers.Store(0)
	s.gen.Add(1)
}

// TryAcquireShared attempts to acquire an immutable shared borrow.
func (s *Slot) TryAcquireShared(gen uint32) error {
	if s == nil || s.gen.Load() != gen {
		return ErrExpiredGeneration
	}
	if s.writer.Load() {
		return ErrAlreadyBorrowedMut
	}
	s.readers.Add(1)
	// Double check writer didn't acquire in between
	if s.writer.Load() {
		s.readers.Add(-1)
		return ErrAlreadyBorrowedMut
	}
	return nil
}

// ReleaseShared releases one active immutable shared borrow.
func (s *Slot) ReleaseShared() {
	if s == nil {
		return
	}
	s.readers.Add(-1)
}

// TryAcquireExclusive attempts to acquire an exclusive mutable borrow.
func (s *Slot) TryAcquireExclusive(gen uint32) error {
	if s == nil || s.gen.Load() != gen {
		return ErrExpiredGeneration
	}
	if !s.writer.CompareAndSwap(false, true) {
		return ErrAlreadyBorrowedMut
	}
	if s.readers.Load() > 0 {
		s.writer.Store(false)
		return ErrAlreadyBorrowedShared
	}
	return nil
}

// ReleaseExclusive releases an exclusive mutable borrow.
func (s *Slot) ReleaseExclusive() {
	if s == nil {
		return
	}
	s.writer.Store(false)
}

// CheckValid verifies the generation and panics with a descriptive message if invalid.
func (s *Slot) CheckValid(expectedGen uint32) {
	if s == nil {
		panic("borrow: nil slot reference")
	}
	curr := s.gen.Load()
	if curr != expectedGen {
		panic(fmt.Sprintf("borrow: use-after-free violation (current generation: %d, expected: %d)", curr, expectedGen))
	}
}
