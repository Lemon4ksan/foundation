// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package deque

import "testing"

func BenchmarkDeque_PushPopFront_WithinCap(b *testing.B) {
	d := NewWithCapacity[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	var v int
	for b.Loop() {
		d.PushFront(v)
		d.PopFront()
		v++
	}
}

func BenchmarkDeque_PushPopBack_WithinCap(b *testing.B) {
	d := NewWithCapacity[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	var v int
	for b.Loop() {
		d.PushBack(v)
		d.PopBack()
		v++
	}
}

func BenchmarkDeque_FIFO_SteadyState(b *testing.B) {
	d := NewWithCapacity[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	var v int
	for b.Loop() {
		d.PushBack(v)
		d.PopFront()
		v++
	}
}

func BenchmarkDeque_ReverseFIFO_SteadyState(b *testing.B) {
	d := NewWithCapacity[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	var v int
	for b.Loop() {
		d.PushFront(v)
		d.PopBack()
		v++
	}
}

func BenchmarkDeque_Front(b *testing.B) {
	d := NewWithCapacity[int](1024)
	d.PushBack(42)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = d.Front()
	}
}

func BenchmarkDeque_Back(b *testing.B) {
	d := NewWithCapacity[int](1024)
	d.PushBack(42)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = d.Back()
	}
}

func BenchmarkDeque_At_Positive(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 512; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	idx := 256
	for b.Loop() {
		_ = d.At(idx)
	}
}

func BenchmarkDeque_At_Negative(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 512; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.At(-1)
	}
}

func BenchmarkDeque_Set(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 512; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var v int
	for b.Loop() {
		d.Set(256, v)
		v++
	}
}

func BenchmarkDeque_Values_Iterate_1024(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 1024; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var sum int
	for b.Loop() {
		for val := range d.Values() {
			sum += val
		}
	}
	_ = sum
}

func BenchmarkDeque_All_Iterate_1024(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 1024; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var sum int
	for b.Loop() {
		for i, val := range d.All() {
			sum += i + val
		}
	}
	_ = sum
}

func BenchmarkDeque_Backward_Iterate_1024(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 1024; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var sum int
	for b.Loop() {
		for val := range d.Backward() {
			sum += val
		}
	}
	_ = sum
}

func BenchmarkDeque_Drain_64(b *testing.B) {
	d := NewWithCapacity[int](1024)
	for i := 0; i < 64; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for v := range d.Drain() {
			_ = v
		}
		for i := 0; i < 64; i++ {
			d.PushBack(i)
		}
	}
}

func BenchmarkDeque_Clear_Reuse(b *testing.B) {
	d := NewWithCapacity[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for i := 0; i < 100; i++ {
			d.PushBack(i)
		}
		d.Clear()
	}
}

func BenchmarkDeque_Wrapped_FIFO(b *testing.B) {
	d := NewWithCapacity[int](1024)
	// Pre-fill and pop half to shift head into the middle
	for i := 0; i < 500; i++ {
		d.PushBack(i)
		d.PopFront()
	}
	for i := 0; i < 500; i++ {
		d.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var v int
	for b.Loop() {
		d.PushBack(v)
		d.PopFront()
		v++
	}
}
