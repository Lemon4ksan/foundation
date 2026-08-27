// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow_test

import (
	"testing"
	"unsafe"

	"github.com/lemon4ksan/foundation/borrow"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
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

	// Mutable borrow
	mut := box.BorrowMut()
	mut.Get().Name = "Aoni-Engine"
	require.Equal(t, "Aoni-Engine", ref.Get().Name)
	assert.Equal(t, uint32(1), mut.Generation())
	assert.True(t, mut.IsValid())

	// Write
	mut.Write(SampleData{ID: 99, Name: "Written"})
	assert.Equal(t, 99, box.Get().ID)

	// Freeze
	frozen := mut.Freeze()
	assert.Equal(t, 99, frozen.Get().ID)

	// Release invalidates all handles
	box.Release()
	require.False(t, box.IsValid())
	require.False(t, ref.IsValid())
	require.False(t, mut.IsValid())

	// Accessing invalid ref must panic deterministically
	assert.Panics(t, func() {
		_ = ref.Get()
	})
	assert.Panics(t, func() {
		_ = mut.Get()
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
	cell.Slot().ReleaseShared()
	cell.Slot().ReleaseShared()

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

	cell.Slot().ReleaseExclusive()

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
