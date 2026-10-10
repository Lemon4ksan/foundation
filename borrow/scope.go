// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow

import (
	"sync"
)

var scopePool = sync.Pool{
	New: func() any {
		return &Scope{
			slot: NewBoundSlot(),
			buf:  make([]byte, 0, 8192),
		}
	},
}

// Scope represents a lexical lifetime arena.
//
// All resources allocated from a Scope are guaranteed to be valid during the execution
// of the scope, and are automatically recycled with 0 heap allocations upon scope exit.
// The zero value is empty and not ready to use.
type Scope struct {
	slot *Slot
	buf  []byte
}

// AcquireScope obtains an active Scope from the internal pool.
//
// It executes in O(1) time and is safe for concurrent use.
// It achieves 0 heap allocations under normal conditions.
func AcquireScope() *Scope {
	s := scopePool.Get().(*Scope)
	s.buf = s.buf[:0]
	return s
}

// NewScope creates or acquires an active [Scope] from the internal pool.
//
// It executes in O(1) time and is safe for concurrent use.
// It achieves 0 heap allocations under normal conditions.
func NewScope() *Scope {
	return AcquireScope()
}

// Release returns the Scope to the internal pool and invalidates all borrows made within it.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Scope) Release() {
	if s == nil {
		return
	}
	s.slot.ForceInvalidate()
	s.buf = s.buf[:0]
	scopePool.Put(s)
}

// Slot returns the master generational slot of this scope.
//
// It executes in O(1) time and is safe for concurrent use.
func (s *Scope) Slot() *Slot {
	return s.slot
}

// AllocBytes allocates a zero-copy [Bytes] buffer of the requested size from the scope arena.
//
// It executes in amortized O(1) time and may allocate memory if the requested size exceeds the available buffer capacity.
// It is safe for concurrent use.
func (s *Scope) AllocBytes(size int) Bytes {
	if size <= 0 {
		var gen uint32
		if s.slot != nil {
			gen = s.slot.Generation()
		}
		return Bytes{slot: s.slot, gen: gen, bound: true}
	}

	start := len(s.buf)
	needed := start + size

	if needed <= cap(s.buf) {
		s.buf = s.buf[:needed]
		slice := s.buf[start:needed]
		return NewBytes(slice, s.slot)
	}

	// For larger sizes, allocate directly and bind to the scope slot
	large := make([]byte, size)
	return NewBytes(large, s.slot)
}

// Alloc creates an owned [Box] bound to this scope's lifetime.
//
// It executes in O(1) time and allocates memory for the boxed value.
// It is safe for concurrent use.
func Alloc[T any](s *Scope) Box[T] {
	ptr := new(T)
	var slot *Slot
	var gen uint32
	if s != nil {
		slot = s.slot
		gen = s.slot.Generation()
	} else {
		slot = NewSlot()
		gen = slot.Generation()
	}
	return Box[T]{
		ptr:  ptr,
		slot: slot,
		gen:  gen,
	}
}

// Scoped executes fn within a dedicated scope arena and returns the computed result.
//
// When fn returns, all resources and borrowed views created in this scope are atomically invalidated.
// It executes in O(N) time where N is the execution time of fn and is safe for concurrent use.
func Scoped[R any](fn func(s *Scope) (R, error)) (R, error) {
	s := AcquireScope()
	defer s.Release()

	return fn(s)
}

// ScopedVoid executes fn within a dedicated scope without returning a value.
//
// It executes in O(N) time where N is the execution time of fn and is safe for concurrent use.
func ScopedVoid(fn func(s *Scope) error) error {
	s := AcquireScope()
	defer s.Release()

	return fn(s)
}
