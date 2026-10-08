// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow_test

import (
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/borrow"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

// TestRegression_Bug1_GhostBorrowChecker_ExclusiveVsSharedConflict verifies that Box
// properly enforces Aliasing XOR Mutability at runtime via Slot borrow counters.
func TestRegression_Bug1_GhostBorrowChecker_ExclusiveVsSharedConflict(t *testing.T) {
	t.Parallel()

	box := borrow.NewBox(100)
	require.True(t, box.IsValid())

	// 1. Multiple shared borrows can coexist
	r1 := box.Borrow()
	r2 := box.Borrow()
	assert.Equal(t, 100, *r1.Get())
	assert.Equal(t, 100, *r2.Get())
	assert.True(t, r1.IsValid())
	assert.True(t, r2.IsValid())

	// 2. Mutable borrow must fail while shared borrows are active
	_, err := box.TryBorrowMut()
	assert.ErrorIs(t, err, borrow.ErrAlreadyBorrowedShared)
	assert.Panics(t, func() {
		_ = box.BorrowMut()
	})

	// 3. Release first reader, second is still active -> mutable borrow still fails
	r1.Release()
	assert.False(t, r1.IsValid())
	// Double release is idempotent
	r1.Release()

	_, err = box.TryBorrowMut()
	assert.ErrorIs(t, err, borrow.ErrAlreadyBorrowedShared)

	// 4. Release second reader -> mutable borrow now succeeds
	r2.Release()
	assert.False(t, r2.IsValid())

	mut := box.BorrowMut()
	assert.True(t, mut.IsValid())
	mut.Write(200)

	// 5. Shared and mutable borrows must fail while mutable borrow is active
	_, err = box.TryBorrow()
	assert.ErrorIs(t, err, borrow.ErrAlreadyBorrowedMut)
	assert.Panics(t, func() {
		_ = box.Borrow()
	})

	_, err = box.TryBorrowMut()
	assert.ErrorIs(t, err, borrow.ErrAlreadyBorrowedMut)
	assert.Panics(t, func() {
		_ = box.BorrowMut()
	})

	// 6. Release exclusive borrow -> shared borrow succeeds again
	mut.Release()
	assert.False(t, mut.IsValid())
	// Double release is idempotent
	mut.Release()

	r3 := box.Borrow()
	assert.Equal(t, 200, *r3.Get())
	r3.Release()

	// 7. Accessing released handles panics
	assert.Panics(t, func() {
		_ = r1.Get()
	})
	assert.Panics(t, func() {
		_ = r2.Get()
	})
	assert.Panics(t, func() {
		_ = mut.Get()
	})

	box.Release()
}

// TestRegression_Bug1_GhostBorrowChecker_FreezeLifecycle verifies that Mut.Freeze
// atomically transitions an exclusive mutable borrow into a shared immutable borrow.
func TestRegression_Bug1_GhostBorrowChecker_FreezeLifecycle(t *testing.T) {
	t.Parallel()

	box := borrow.NewBox("initial")
	mut := box.BorrowMut()
	mut.Write("mutated")

	// Freeze mut into ref
	frozen := mut.Freeze()
	assert.False(t, mut.IsValid(), "mut handle must be invalid after freeze")
	assert.True(t, frozen.IsValid(), "frozen handle must be valid")
	assert.Equal(t, "mutated", *frozen.Get())

	// Another shared reader can now join
	r2 := box.Borrow()
	assert.Equal(t, "mutated", *r2.Get())

	// Exclusive borrow must still fail because readers are active
	_, err := box.TryBorrowMut()
	assert.ErrorIs(t, err, borrow.ErrAlreadyBorrowedShared)

	// Release both readers
	frozen.Release()
	r2.Release()

	// Exclusive borrow can now be acquired
	mut2 := box.BorrowMut()
	mut2.Write("final")
	mut2.Release()

	assert.Equal(t, "final", *box.Get())
	box.Release()
}

// TestRegression_Bug2_ScopeAllocBytes_ReleaseDoesNotInvalidateScope verifies that
// calling Release() on a Scope-allocated Bytes does not invalidate other resources or the Scope.
func TestRegression_Bug2_ScopeAllocBytes_ReleaseDoesNotInvalidateScope(t *testing.T) {
	t.Parallel()

	s := borrow.NewScope()

	b1 := s.AllocBytes(64)
	b2 := s.AllocBytes(64)
	boxVal := borrow.Alloc[int](s)
	*boxVal.Get() = 42

	s1 := b1.AsSlice()
	s2 := b2.AsSlice()
	require.NotNil(t, s1)
	require.NotNil(t, s2)
	s1[0] = 'A'
	s2[0] = 'B'

	assert.True(t, b1.IsBound())
	assert.True(t, b2.IsBound())

	// Release b1 explicitly
	b1.Release()
	assert.False(t, b1.IsValid(), "b1 must be invalid after Release()")
	assert.Nil(t, b1.AsSlice(), "b1.AsSlice() must be nil after Release()")
	// Double release is idempotent
	b1.Release()

	// b2 and boxVal MUST STILL BE VALID and completely functional!
	assert.True(t, b2.IsValid(), "b2 must remain valid after b1.Release()")
	assert.Equal(t, byte('B'), b2.AsSlice()[0])
	assert.True(t, boxVal.IsValid(), "boxVal must remain valid after b1.Release()")
	assert.Equal(t, 42, *boxVal.Get())

	// Allocating new resources from the same scope must continue to succeed
	b3 := s.AllocBytes(32)
	assert.True(t, b3.IsValid(), "subsequent allocation from scope must succeed")
	b3.AsSlice()[0] = 'C'
	assert.Equal(t, byte('C'), b3.AsSlice()[0])

	// Sub-slice release must also not invalidate the parent or scope
	sub := b2.Slice(0, 16)
	assert.True(t, sub.IsValid())
	sub.Release()
	assert.False(t, sub.IsValid())
	assert.True(t, b2.IsValid(), "parent b2 must remain valid after sub-slice Release()")
	assert.Equal(t, byte('B'), b2.AsSlice()[0])

	// Finally, releasing the Scope itself invalidates all remaining resources
	s.Release()
	assert.False(t, b2.IsValid(), "b2 must be invalidated on scope.Release()")
	assert.False(t, b3.IsValid(), "b3 must be invalidated on scope.Release()")
	assert.False(t, boxVal.IsValid(), "boxVal must be invalidated on scope.Release()")

	assert.Panics(t, func() {
		_ = b2.AsSlice()
	})
	assert.Panics(t, func() {
		_ = b3.AsSlice()
	})
	assert.Panics(t, func() {
		_ = boxVal.Get()
	})
}

// TestRegression_Bug3_BoxMove_InvalidatesPreMoveBorrows verifies that Box.Move()
// advances the slot generation so that any pre-move borrows deterministically fail.
func TestRegression_Bug3_BoxMove_InvalidatesPreMoveBorrows(t *testing.T) {
	t.Parallel()

	box1 := borrow.NewBox("original-value")
	require.True(t, box1.IsValid())

	// Acquire pre-move shared borrow
	refPreMove := box1.Borrow()
	assert.Equal(t, "original-value", *refPreMove.Get())
	assert.True(t, refPreMove.IsValid())

	// Move ownership to box2
	box2 := box1.Move()

	// box1 is invalidated
	assert.False(t, box1.IsValid(), "box1 must be invalid after move")
	assert.Panics(t, func() {
		_ = box1.Get()
	})

	// Pre-move ref MUST BE INVALIDATED
	assert.False(t, refPreMove.IsValid(), "pre-move Ref must be invalid after Box.Move()")
	assert.Panics(t, func() {
		_ = refPreMove.Get()
	}, "pre-move Ref.Get() must panic due to generation mismatch")

	// box2 is valid and has the transferred value
	assert.True(t, box2.IsValid(), "box2 must be valid owner")
	assert.Equal(t, "original-value", *box2.Get())

	// New borrows on box2 succeed
	refPostMove := box2.Borrow()
	assert.True(t, refPostMove.IsValid())
	assert.Equal(t, "original-value", *refPostMove.Get())
	refPostMove.Release()

	// Mutable borrow on box2 succeeds
	mutPostMove := box2.BorrowMut()
	mutPostMove.Write("updated-value")
	mutPostMove.Release()

	assert.Equal(t, "updated-value", *box2.Get())
	box2.Release()
}

// TestRegression_Bug3_BoxMove_InvalidatesPreMoveMutableBorrow verifies that Box.Move()
// also invalidates any outstanding pre-move mutable borrows.
func TestRegression_Bug3_BoxMove_InvalidatesPreMoveMutableBorrow(t *testing.T) {
	t.Parallel()

	box1 := borrow.NewBox(999)
	mutPreMove := box1.BorrowMut()
	assert.True(t, mutPreMove.IsValid())

	box2 := box1.Move()

	assert.False(t, box1.IsValid())
	assert.False(t, mutPreMove.IsValid(), "pre-move Mut must be invalid after Box.Move()")
	assert.Panics(t, func() {
		_ = mutPreMove.Get()
	})
	assert.Panics(t, func() {
		mutPreMove.Write(888)
	})

	assert.True(t, box2.IsValid())
	assert.Equal(t, 999, *box2.Get())
	box2.Release()
}

// TestRegression_Concurrency_BoxSharedBorrows verifies race-free concurrent reads.
func TestRegression_Concurrency_BoxSharedBorrows(t *testing.T) {
	t.Parallel()

	box := borrow.NewBox(42)

	const iterations = 20
	var wg sync.WaitGroup
	wg.Add(iterations)

	for range iterations {
		go func() {
			defer wg.Done()
			ref := box.Borrow()
			assert.Equal(t, 42, *ref.Get())
			ref.Release()
		}()
	}

	wg.Wait()
	box.Release()
}

// TestRegression_Concurrency_ScopeParallelPool verifies race-free scope recycling.
func TestRegression_Concurrency_ScopeParallelPool(t *testing.T) {
	t.Parallel()

	const iterations = 50
	var wg sync.WaitGroup
	wg.Add(iterations)

	for i := range iterations {
		go func(val int) {
			defer wg.Done()
			s := borrow.AcquireScope()
			b := s.AllocBytes(128)
			slice := b.AsSlice()
			slice[0] = byte(val)
			assert.Equal(t, byte(val), slice[0])
			b.Release()
			s.Release()
		}(i)
	}

	wg.Wait()
}
