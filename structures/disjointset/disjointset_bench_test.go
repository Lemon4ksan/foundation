// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package disjointset

import (
	"math/rand/v2"
	"testing"
)

func BenchmarkDisjointSet_Find_DirectRoot(b *testing.B) {
	d := New(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.Find(512)
	}
}

func BenchmarkDisjointSet_Find_FlatTree(b *testing.B) {
	d := New(1024)
	for i := 1; i < 1024; i++ {
		d.Union(0, i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var idx int
	for b.Loop() {
		_ = d.Find(idx & 1023)
		idx++
	}
}

func BenchmarkDisjointSet_Find_Deep(b *testing.B) {
	n := 1024
	d := New(n)
	// Build chain without flattening
	for i := 0; i < n-1; i++ {
		d.parent[i+1] = i
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.Find(n - 1)
	}
}

func BenchmarkDisjointSet_Union_Disjoint(b *testing.B) {
	d := New(1_000_000)
	b.ReportAllocs()
	b.ResetTimer()

	var i int
	for b.Loop() {
		u := (i * 2) % 1_000_000
		v := (u + 1) % 1_000_000
		d.Union(u, v)
		i++
	}
}

func BenchmarkDisjointSet_Union_AlreadyConnected(b *testing.B) {
	d := New(1024)
	d.Union(10, 20)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.Union(10, 20)
	}
}

func BenchmarkDisjointSet_Union_Self(b *testing.B) {
	d := New(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.Union(42, 42)
	}
}

func BenchmarkDisjointSet_Connected(b *testing.B) {
	d := New(1024)
	for i := 0; i < 512; i++ {
		d.Union(i, i+512)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var i int
	for b.Loop() {
		_ = d.Connected(i&511, (i&511)+512)
		i++
	}
}

func BenchmarkDisjointSet_Size(b *testing.B) {
	d := New(1024)
	for i := 0; i < 512; i++ {
		d.Union(i, i+512)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var i int
	for b.Loop() {
		_ = d.Size(i & 1023)
		i++
	}
}

func BenchmarkDisjointSet_Count(b *testing.B) {
	d := New(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.Count()
	}
}

func BenchmarkDisjointSet_Len(b *testing.B) {
	d := New(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.Len()
	}
}

func BenchmarkDisjointSet_Reset_1024(b *testing.B) {
	d := New(1024)
	for i := 0; i < 500; i++ {
		d.Union(i, i+1)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		d.Reset()
	}
}

func BenchmarkDisjointSet_Reset_65536(b *testing.B) {
	d := New(65536)
	for i := 0; i < 30000; i++ {
		d.Union(i, i+1)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		d.Reset()
	}
}

func BenchmarkDisjointSet_Kruskal_Workload(b *testing.B) {
	type edge struct {
		u, v int
	}

	n := 1000
	numEdges := 5000
	rng := rand.New(rand.NewPCG(123, 456))

	edges := make([]edge, numEdges)
	for i := 0; i < numEdges; i++ {
		edges[i] = edge{
			u: rng.IntN(n),
			v: rng.IntN(n),
		}
	}

	d := New(n)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		d.Reset()
		for _, e := range edges {
			if !d.Connected(e.u, e.v) {
				d.Union(e.u, e.v)
			}
		}
	}
}
