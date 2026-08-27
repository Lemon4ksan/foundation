// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow_test

import (
	"sync"
	"testing"

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

	// Mutable borrow
	mut := box.BorrowMut()
	mut.Get().Name = "Aoni-Engine"
	require.Equal(t, "Aoni-Engine", ref.Get().Name)

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
}

func TestBox_Move(t *testing.T) {
	t.Parallel()

	box1 := borrow.NewBox(SampleData{ID: 100, Name: "Source"})
	require.True(t, box1.IsValid())

	box2 := box1.Move()
	require.False(t, box1.IsValid(), "box1 must be invalid after move")
	require.True(t, box2.IsValid(), "box2 must be valid owner")
	require.Equal(t, 100, box2.Get().ID)

	box2.Release()
	require.False(t, box2.IsValid())
}

func TestBytes_ZeroCopy_And_UAF_Panic(t *testing.T) {
	t.Parallel()

	slot := borrow.NewSlot()
	raw := []byte("Hello, Silicon Line Speed!")
	b := borrow.NewBytes(raw, slot)

	require.True(t, b.IsValid())
	require.Equal(t, raw, b.AsSlice())
	require.Equal(t, len(raw), b.Len())

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
}

func TestScoped_LexicalLifetime(t *testing.T) {
	t.Parallel()

	var leakedBytes borrow.Bytes

	res, err := borrow.Scoped(func(s *borrow.Scope) (string, error) {
		b := s.AllocBytes(64)
		copy(b.AsSlice(), []byte("Scoped Data"))

		leakedBytes = b
		require.True(t, leakedBytes.IsValid())

		return string(b.AsSlice()[:11]), nil
	})

	require.NoError(t, err)
	require.Equal(t, "Scoped Data", res)

	// After Scoped return, leakedBytes must be invalidated
	require.False(t, leakedBytes.IsValid())
	assert.Panics(t, func() {
		_ = leakedBytes.AsSlice()
	})
}

func TestCell_Aliasing_XOR_Mutability(t *testing.T) {
	t.Parallel()

	cell := borrow.NewCell(42)

	// Shared borrow
	r1, err := cell.Borrow()
	require.NoError(t, err)
	require.Equal(t, 42, *r1.Get())

	// Multiple shared borrows are allowed
	r2, err := cell.Borrow()
	require.NoError(t, err)
	require.Equal(t, 42, *r2.Get())

	// Mutable borrow must be rejected while shared borrows exist
	_, err = cell.BorrowMut()
	require.ErrorIs(t, err, borrow.ErrAlreadyBorrowedShared)

	// Release shared borrows
	cell.Slot().ReleaseShared()
	cell.Slot().ReleaseShared()

	// Now mutable borrow succeeds
	m, err := cell.BorrowMut()
	require.NoError(t, err)
	m.Write(100)

	// Shared borrow is rejected while mutable borrow is active
	_, err = cell.Borrow()
	require.ErrorIs(t, err, borrow.ErrAlreadyBorrowedMut)

	// Release mutable borrow
	cell.Slot().ReleaseExclusive()

	// Direct get & set
	require.Equal(t, 100, cell.Get())
	cell.Set(200)
	require.Equal(t, 200, cell.Get())
}

func TestConcurrent_Readers(t *testing.T) {
	t.Parallel()

	box := borrow.NewBox(12345)
	ref := box.Borrow()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				assert.Equal(t, 12345, ref.Val())
			}
		}()
	}
	wg.Wait()

	box.Release()
}

func BenchmarkScoped_AllocBytes(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = borrow.ScopedVoid(func(s *borrow.Scope) error {
			buf := s.AllocBytes(256)
			_ = buf.AsSlice()
			return nil
		})
	}
}

func BenchmarkBox_Lifecycle(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		box := borrow.NewBox(SampleData{ID: i, Name: "Benchmark"})
		ref := box.Borrow()
		_ = ref.Get().ID
		box.Release()
	}
}
