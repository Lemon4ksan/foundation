// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package minheap

import (
	"testing"

	"github.com/lemon4ksan/foundation/testing/require"
)

func TestHeap(t *testing.T) {
	var h Heap[int, string]
	require.True(t, h.Empty())
	require.Panics(t, func() { h.Peek() })
	require.Panics(t, func() { h.Pop() })

	h.Push(4, "four")
	h.Push(1, "one")
	h.Push(3, "three")
	h.Push(2, "two")
	require.Equal(t, 4, h.Len())
	key, value := h.Peek()
	require.Equal(t, 1, key)
	require.Equal(t, "one", value)

	for _, expected := range []struct {
		key   int
		value string
	}{{1, "one"}, {2, "two"}, {3, "three"}, {4, "four"}} {
		key, value = h.Pop()
		require.Equal(t, expected.key, key)
		require.Equal(t, expected.value, value)
	}

	require.True(t, h.Empty())

	h.Push(1, "one")
	h.Push(2, "two")
	h.Clear()
	require.True(t, h.Empty())
}

func TestHeap_Iterators(t *testing.T) {
	var h Heap[int, string]

	// Empty heap iterators
	count := 0
	for range h.All() {
		count++
	}
	require.Equal(t, 0, count)

	for range h.Values() {
		count++
	}
	require.Equal(t, 0, count)

	for range h.Drain() {
		count++
	}
	require.Equal(t, 0, count)

	for range h.DrainValues() {
		count++
	}
	require.Equal(t, 0, count)

	// Populate
	h.Push(30, "thirty")
	h.Push(10, "ten")
	h.Push(20, "twenty")
	require.Equal(t, 3, h.Len())

	// All() non-destructive with early exit
	var allKeys []int
	for k := range h.All() {
		allKeys = append(allKeys, k)
		if len(allKeys) == 2 {
			break
		}
	}
	require.Equal(t, 2, len(allKeys))
	require.Equal(t, 3, h.Len())

	// Values() non-destructive
	var values []string
	for v := range h.Values() {
		values = append(values, v)
	}
	require.Equal(t, 3, len(values))
	require.Equal(t, 3, h.Len())

	// Drain() priority order consuming
	type pair struct {
		k int
		v string
	}
	var drained []pair
	for k, v := range h.Drain() {
		drained = append(drained, pair{k, v})
	}
	require.Equal(t, []pair{
		{10, "ten"},
		{20, "twenty"},
		{30, "thirty"},
	}, drained)
	require.True(t, h.Empty())

	// DrainValues()
	h.Push(2, "two")
	h.Push(1, "one")
	h.Push(3, "three")
	var drainedVals []string
	for v := range h.DrainValues() {
		drainedVals = append(drainedVals, v)
		if len(drainedVals) == 1 {
			break // test early exit
		}
	}
	require.Equal(t, []string{"one"}, drainedVals)
	require.Equal(t, 2, h.Len())
}

func BenchmarkHeap_PushPop(b *testing.B) {
	h := make(Heap[int, int], 0, 1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		h.Push(42, 100)
		h.Pop()
	}
}

func BenchmarkHeap_Drain(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		var h Heap[int, int]
		for i := 100; i > 0; i-- {
			h.Push(i, i)
		}
		b.StartTimer()

		for range h.Drain() {
		}
	}
}

func TestHeap_CapacityAndShrinkToFit(t *testing.T) {
	h := NewWithCapacity[int, string](100)
	require.Equal(t, 0, h.Len())
	require.Equal(t, 100, h.Cap())

	for i := range 50 {
		h.Push(i, "val")
	}
	require.Equal(t, 50, h.Len())
	require.Equal(t, 100, h.Cap())

	h.ShrinkToFit()
	require.Equal(t, 50, h.Len())
	require.Equal(t, 50, h.Cap())

	h.Clear()
	require.Equal(t, 0, h.Len())
	require.Equal(t, 50, h.Cap())

	h.ShrinkToFit()
	require.Equal(t, 0, h.Len())
	require.Equal(t, 0, h.Cap())
}

func TestHeap_Reserve(t *testing.T) {
	var h Heap[int, int]
	h.Reserve(64)
	require.True(t, h.Cap() >= 64)

	allocs := testing.AllocsPerRun(10, func() {
		for i := range 64 {
			h.Push(i, i)
		}
		for range 64 {
			h.Pop()
		}
	})
	require.Equal(t, float64(0), allocs)
}

func TestHeap_BoundaryEdgeCases(t *testing.T) {
	// Zero / negative capacity
	emptyH := NewWithCapacity[int, string](0)
	require.Nil(t, emptyH)

	negH := NewWithCapacity[int, string](-5)
	require.Nil(t, negH)

	// Zero / negative reserve
	var h Heap[int, string]
	h.Reserve(0)
	h.Reserve(-1)
	require.Equal(t, 0, h.Cap())

	// Early break in Values()
	h.Push(1, "one")
	h.Push(2, "two")
	for v := range h.Values() {
		if v == "one" {
			break
		}
	}

	// Early break in Drain()
	for k := range h.Drain() {
		if k == 1 {
			break
		}
	}
	require.Equal(t, 1, h.Len())
}
