// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/borrow"
)

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
