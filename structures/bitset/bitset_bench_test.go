// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitset

import (
	"testing"
)

func BenchmarkBitSet_Test(b *testing.B) {
	bs := New(1024)
	bs.Set(512)
	b.ReportAllocs()
	b.ResetTimer()

	idx := 512
	for b.Loop() {
		_ = bs.Test(idx)
	}
}

func BenchmarkBitSet_Set_WithinCap(b *testing.B) {
	bs := New(1024)
	b.ReportAllocs()
	b.ResetTimer()

	var i int
	for b.Loop() {
		bs.Set(i & 1023)
		i++
	}
}

func BenchmarkBitSet_Clear(b *testing.B) {
	bs := New(1024).SetAll()
	b.ReportAllocs()
	b.ResetTimer()

	var i int
	for b.Loop() {
		bs.Clear(i & 1023)
		i++
	}
}

func BenchmarkBitSet_Toggle(b *testing.B) {
	bs := New(1024)
	b.ReportAllocs()
	b.ResetTimer()

	var i int
	for b.Loop() {
		bs.Toggle(i & 1023)
		i++
	}
}

func BenchmarkBitSet_Count_64(b *testing.B) {
	bs := New(64)
	bs.Set(1).Set(10).Set(30).Set(60)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = bs.Count()
	}
}

func BenchmarkBitSet_Count_1024(b *testing.B) {
	bs := New(1024)
	for i := 0; i < 1024; i += 3 {
		bs.Set(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = bs.Count()
	}
}

func BenchmarkBitSet_Count_65536(b *testing.B) {
	bs := New(65536)
	for i := 0; i < 65536; i += 5 {
		bs.Set(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = bs.Count()
	}
}

func BenchmarkBitSet_NextSet_Dense(b *testing.B) {
	bs := New(1024).SetAll()
	b.ReportAllocs()
	b.ResetTimer()

	var start int
	for b.Loop() {
		idx, _ := bs.NextSet(start)
		start = (idx + 1) & 1023
	}
}

func BenchmarkBitSet_NextSet_Sparse(b *testing.B) {
	bs := New(65536)
	bs.Set(1000).Set(20000).Set(50000)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = bs.NextSet(1001)
	}
}

func BenchmarkBitSet_NextClear(b *testing.B) {
	bs := New(1024).SetAll()
	bs.Clear(500)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = bs.NextClear(0)
	}
}

func BenchmarkBitSet_Bits_Iterate(b *testing.B) {
	bs := New(1024)
	for i := 0; i < 1024; i += 16 {
		bs.Set(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var sum int
	for b.Loop() {
		for idx := range bs.Bits() {
			sum += idx
		}
	}
	_ = sum
}

func BenchmarkBitSet_And_InPlace(b *testing.B) {
	b1 := New(1024).SetAll()
	b2 := New(1024)
	for i := 0; i < 1024; i += 2 {
		b2.Set(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		b1.And(b2)
	}
}

func BenchmarkBitSet_Or_InPlace(b *testing.B) {
	b1 := New(1024)
	b2 := New(1024)
	for i := 0; i < 1024; i += 2 {
		b2.Set(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		b1.Or(b2)
	}
}

func BenchmarkBitSet_Xor_InPlace(b *testing.B) {
	b1 := New(1024)
	b2 := New(1024)
	for i := 0; i < 1024; i += 2 {
		b2.Set(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		b1.Xor(b2)
	}
}

func BenchmarkBitSet_Not_InPlace(b *testing.B) {
	b1 := New(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		b1.Not()
	}
}

func BenchmarkBitSet_Equal(b *testing.B) {
	b1 := New(1024).Set(10).Set(500).Set(1000)
	b2 := New(1024).Set(10).Set(500).Set(1000)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = b1.Equal(b2)
	}
}

func BenchmarkBitSet_SetAll_ClearAll(b *testing.B) {
	b1 := New(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		b1.SetAll()
		b1.ClearAll()
	}
}
