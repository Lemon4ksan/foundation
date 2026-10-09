// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow_test

import (
	"sync"
	"testing"
	"unsafe"

	"github.com/lemon4ksan/foundation/borrow"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

type SampleData struct {
	ID   int
	Name string
}

func TestBox_Ownership_And_Release(t *testing.T) {
	t.Parallel()

	box := borrow.NewBox(SampleData{ID: 42, Name: "Aoni"})
	require.True(t, box.IsValid())

	// Shared borrow
	ref := box.Borrow()
	require.Equal(t, 42, ref.Get().ID)
	require.Equal(t, "Aoni", ref.Val().Name)
	assert.Equal(t, uint32(1), ref.Generation())

	// Mutable borrow while shared borrow is active must fail/panic
	assert.Panics(t, func() {
		_ = box.BorrowMut()
	})
	_, err := box.TryBorrowMut()
	assert.ErrorIs(t, err, borrow.ErrAlreadyBorrowedShared)

	// Release shared borrow
	ref.Release()
	assert.False(t, ref.IsValid())

	// Mutable borrow now succeeds
	mut := box.BorrowMut()
	mut.Get().Name = "Aoni-Engine"
	assert.Equal(t, uint32(1), mut.Generation())
	assert.True(t, mut.IsValid())

	// Shared borrow while mutable borrow is active must fail/panic
	assert.Panics(t, func() {
		_ = box.Borrow()
	})
	_, err = box.TryBorrow()
	assert.ErrorIs(t, err, borrow.ErrAlreadyBorrowedMut)

	// Write
	mut.Write(SampleData{ID: 99, Name: "Written"})
	assert.Equal(t, 99, box.Get().ID)

	// Freeze mut into ref
	frozen := mut.Freeze()
	assert.Equal(t, 99, frozen.Get().ID)
	assert.False(t, mut.IsValid())

	// Release frozen ref
	frozen.Release()
	assert.False(t, frozen.IsValid())

	// Release invalidates all handles
	box.Release()
	require.False(t, box.IsValid())
	require.False(t, ref.IsValid())
	require.False(t, mut.IsValid())
	require.False(t, frozen.IsValid())

	// Accessing invalid ref must panic deterministically
	assert.Panics(t, func() {
		_ = ref.Get()
	})
	assert.Panics(t, func() {
		_ = mut.Get()
	})
	assert.Panics(t, func() {
		_ = frozen.Get()
	})
	assert.Panics(t, func() {
		_ = box.Get()
	})
}

func TestBox_Move_And_NewBoxPtr(t *testing.T) {
	t.Parallel()

	box1 := borrow.NewBox(SampleData{ID: 100, Name: "Source"})
	require.True(t, box1.IsValid())

	box2 := box1.Move()
	require.False(t, box1.IsValid(), "box1 must be invalid after move")
	require.True(t, box2.IsValid(), "box2 must be valid owner")
	require.Equal(t, 100, box2.Get().ID)

	box2.Release()
	require.False(t, box2.IsValid())

	// Move on released box panics
	assert.Panics(t, func() {
		_ = box2.Move()
	})

	// NewBoxPtr with nil slot
	data := SampleData{ID: 200, Name: "Ptr"}
	boxPtr := borrow.NewBoxPtr(&data, nil)
	assert.True(t, boxPtr.IsValid())
	assert.Equal(t, 200, boxPtr.Get().ID)
}

func TestCell_InteriorMutability_And_BorrowChecking(t *testing.T) {
	t.Parallel()

	cell := borrow.NewCell("initial")
	assert.NotNil(t, cell.Slot())

	// 1. Get and Set
	assert.Equal(t, "initial", cell.Get())
	cell.Set("updated")
	assert.Equal(t, "updated", cell.Get())

	// 2. Shared borrows
	r1, err := cell.Borrow()
	require.NoError(t, err)
	r2 := cell.MustBorrow()
	assert.Equal(t, "updated", *r1.Get())
	assert.Equal(t, "updated", *r2.Get())

	// 3. Mutable borrow fails while shared borrows are active
	_, err = cell.BorrowMut()
	assert.ErrorIs(t, err, borrow.ErrAlreadyBorrowedShared)
	assert.Panics(t, func() {
		_ = cell.MustBorrowMut()
	})

	// Release shared
	r1.Release()
	r2.Release()

	// 4. Mutable borrow exclusive
	m1, err := cell.BorrowMut()
	require.NoError(t, err)
	*m1.Get() = "mutated"

	// Second borrow fails while exclusive
	_, err = cell.Borrow()
	assert.ErrorIs(t, err, borrow.ErrAlreadyBorrowedMut)
	assert.Panics(t, func() {
		_ = cell.MustBorrow()
	})
	_, err = cell.BorrowMut()
	assert.ErrorIs(t, err, borrow.ErrAlreadyBorrowedMut)

	m1.Release()

	// Nil cell Slot()
	var nilCell *borrow.Cell[int]
	assert.Nil(t, nilCell.Slot())
}

func TestRef_And_Mut_NilSlot(t *testing.T) {
	t.Parallel()

	val := 42
	// Ref with nil slot
	r := borrow.NewRef(&val, nil)
	assert.Equal(t, 42, *r.Get())
	assert.Equal(t, 42, r.Val())
	assert.True(t, r.IsValid())
	assert.Equal(t, uint32(0), r.Generation())

	var nilPtrRef borrow.Ref[int]
	assert.Equal(t, 0, nilPtrRef.Val())

	// Mut with nil slot
	m := borrow.NewMut(&val, nil)
	assert.Equal(t, 42, *m.Get())
	m.Write(100)
	assert.Equal(t, 100, val)
	assert.True(t, m.IsValid())
	assert.Equal(t, uint32(0), m.Generation())
	f := m.Freeze()
	assert.Equal(t, 100, *f.Get())
}

func TestSlot_GenerationalValidation_And_NilReceiver(t *testing.T) {
	t.Parallel()

	slot := borrow.NewSlot()
	assert.Equal(t, uint32(1), slot.Generation())
	assert.True(t, slot.IsValid(1))
	assert.False(t, slot.IsValid(2))

	slot.CheckValid(1)
	assert.Panics(t, func() {
		slot.CheckValid(999)
	})

	slot.Invalidate()
	assert.Equal(t, uint32(2), slot.Generation())
	assert.False(t, slot.IsValid(1))

	// Nil slot methods
	var nilSlot *borrow.Slot
	assert.Equal(t, uint32(0), nilSlot.Generation())
	assert.False(t, nilSlot.IsValid(1))
	nilSlot.Invalidate()
	nilSlot.ReleaseShared()
	nilSlot.ReleaseExclusive()
	assert.ErrorIs(t, nilSlot.TryAcquireShared(1), borrow.ErrExpiredGeneration)
	assert.ErrorIs(t, nilSlot.TryAcquireExclusive(1), borrow.ErrExpiredGeneration)
	assert.Panics(t, func() {
		nilSlot.CheckValid(1)
	})
}

func TestBytes_ZeroCopy_And_UAF_Panic(t *testing.T) {
	t.Parallel()

	slot := borrow.NewSlot()
	raw := []byte("Hello, Silicon Line Speed!")
	b := borrow.NewBytes(raw, slot)

	require.True(t, b.IsValid())
	require.Equal(t, raw, b.AsSlice())
	require.Equal(t, raw, b.Bytes())
	require.Equal(t, len(raw), b.Len())
	require.Equal(t, cap(raw), b.Cap())

	// Sub-slice
	sub := b.Slice(0, 5)
	assert.Equal(t, []byte("Hello"), sub.AsSlice())

	// Slice panics on bad bounds
	assert.Panics(t, func() {
		_ = b.Slice(-1, 5)
	})

	// Clone creates independent GC copy
	clone := b.Clone()
	require.Equal(t, raw, clone)

	// Invalidate slot (simulate buffer recycled into pool)
	slot.Invalidate()
	require.False(t, b.IsValid())

	// Accessing recycled buffer must panic
	assert.Panics(t, func() {
		_ = b.AsSlice()
	})

	// Cloned copy remains fully accessible and untouched
	require.Equal(t, "Hello, Silicon Line Speed!", string(clone))

	// Empty bytes
	emptyBytes := borrow.NewBytes(nil, slot)
	assert.Nil(t, emptyBytes.AsSlice())
	assert.Nil(t, emptyBytes.Clone())

	// NewBytesRaw
	rawBytes := borrow.NewBytesRaw(unsafe.Pointer(&raw[0]), len(raw), cap(raw), nil)
	assert.Equal(t, raw, rawBytes.AsSlice())
	rawBytes.Release()
	assert.Nil(t, rawBytes.AsSlice())
}

func TestScope_Allocation_And_LexicalLifetime(t *testing.T) {
	t.Parallel()

	// 1. Manual scope allocation
	s := borrow.NewScope()
	assert.NotNil(t, s.Slot())

	bSmall := s.AllocBytes(64)
	assert.Equal(t, 64, bSmall.Len())

	bZero := s.AllocBytes(0)
	assert.Equal(t, 0, bZero.Len())

	bLarge := s.AllocBytes(16384) // exceeds 8192
	assert.Equal(t, 16384, bLarge.Len())

	boxFromScope := borrow.Alloc[int](s)
	*boxFromScope.Get() = 55
	assert.Equal(t, 55, *boxFromScope.Get())

	boxFromNilScope := borrow.Alloc[int](nil)
	*boxFromNilScope.Get() = 77
	assert.Equal(t, 77, *boxFromNilScope.Get())

	s.Release()
	assert.False(t, bSmall.IsValid())

	var nilScope *borrow.Scope
	nilScope.Release()

	// 2. Scoped execution
	var leakedBytes borrow.Bytes
	var leakedBox borrow.Box[int]

	res, err := borrow.Scoped(func(s *borrow.Scope) (string, error) {
		leakedBytes = s.AllocBytes(32)
		leakedBox = borrow.Alloc[int](s)
		*leakedBox.Get() = 42

		assert.True(t, leakedBytes.IsValid())
		assert.True(t, leakedBox.IsValid())
		assert.Equal(t, 42, *leakedBox.Get())

		return "success", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "success", res)

	// After Scope exits, all tracked resources are strictly invalidated
	assert.False(t, leakedBytes.IsValid())
	assert.False(t, leakedBox.IsValid())
	assert.Panics(t, func() {
		_ = leakedBytes.AsSlice()
	})
	assert.Panics(t, func() {
		_ = leakedBox.Get()
	})

	// 3. ScopedVoid
	err = borrow.ScopedVoid(func(s *borrow.Scope) error {
		b := s.AllocBytes(16)
		assert.True(t, b.IsValid())
		return nil
	})
	require.NoError(t, err)
}

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

func BenchmarkSlot_SharedAcquireRelease(b *testing.B) {
	slot := borrow.NewSlot()
	gen := slot.Generation()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if err := slot.TryAcquireShared(gen); err != nil {
			b.Fatal(err)
		}
		slot.ReleaseShared()
	}
}

func BenchmarkSlot_SharedAcquireRelease_Parallel(b *testing.B) {
	slot := borrow.NewSlot()
	gen := slot.Generation()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := slot.TryAcquireShared(gen); err != nil {
				b.Fatal(err)
			}
			slot.ReleaseShared()
		}
	})
}

func BenchmarkSlot_ExclusiveAcquireRelease(b *testing.B) {
	slot := borrow.NewSlot()
	gen := slot.Generation()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if err := slot.TryAcquireExclusive(gen); err != nil {
			b.Fatal(err)
		}
		slot.ReleaseExclusive()
	}
}

func BenchmarkSlot_GenerationCheck(b *testing.B) {
	slot := borrow.NewSlot()
	gen := slot.Generation()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if !slot.IsValid(gen) {
			b.Fatal("slot unexpectedly invalid")
		}
	}
}

func BenchmarkBytes_AsSlice(b *testing.B) {
	slot := borrow.NewSlot()
	raw := make([]byte, 256)
	bHandle := borrow.NewBytes(raw, slot)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		s := bHandle.AsSlice()
		if len(s) != 256 {
			b.Fatal("unexpected length")
		}
	}
}

func BenchmarkBytes_Slice(b *testing.B) {
	slot := borrow.NewSlot()
	raw := make([]byte, 256)
	bHandle := borrow.NewBytes(raw, slot)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		sub := bHandle.Slice(16, 128)
		if sub.Len() != 112 {
			b.Fatal("unexpected length")
		}
	}
}

func BenchmarkScope_AllocBytes_Recycle(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		s := borrow.AcquireScope()
		_ = s.AllocBytes(128)
		_ = s.AllocBytes(256)
		s.Release()
	}
}

func BenchmarkCell_Borrow_Release(b *testing.B) {
	cell := borrow.NewCell(12345)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ref, err := cell.Borrow()
		if err != nil {
			b.Fatal(err)
		}
		_ = ref.Val()
		cell.Slot().ReleaseShared()
	}
}

func BenchmarkRef_Get(b *testing.B) {
	slot := borrow.NewSlot()
	val := 42
	ref := borrow.NewRef(&val, slot)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = ref.Get()
	}
}

func BenchmarkCell_GetSet(b *testing.B) {
	cell := borrow.NewCell(100)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		cell.Set(42)
		_ = cell.Get()
	}
}
