// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ringbuffer

import (
	"testing"

	"github.com/lemon4ksan/foundation/testing/require"
)

func TestPushPeekPop(t *testing.T) {
	r := RingBuffer[int]{}
	require.Equal(t, 0, len(r.ring))
	require.Panics(t, func() { r.PopFront() })
	r.PushBack(1)
	r.PushBack(2)
	r.PushBack(3)
	require.Equal(t, 1, r.PeekFront())
	require.Equal(t, 1, r.PeekFront())
	require.Equal(t, 1, r.PopFront())
	require.Equal(t, 2, r.PeekFront())
	require.Equal(t, 2, r.PopFront())
	r.PushBack(4)
	r.PushBack(5)
	require.Equal(t, 3, r.Len())
	r.PushBack(6)
	require.Equal(t, 4, r.Len())
	r.PushBack(7) // grow with the buffer wrapped around
	require.Equal(t, 5, r.Len())
	require.Equal(t, 3, r.PopFront())
	require.Equal(t, 4, r.PopFront())
	require.Equal(t, 5, r.PopFront())
	require.Equal(t, 6, r.PopFront())
	require.Equal(t, 7, r.PopFront())
}

func TestPanicOnEmptyBuffer(t *testing.T) {
	r := RingBuffer[string]{}
	require.True(t, r.Empty())
	require.Zero(t, r.Len())

	func() {
		defer func() {
			val := recover()
			require.Equal(t, "ringbuffer: peek from an empty queue", val)
		}()
		r.PeekFront()
		t.Fatal("expected peek to panic")
	}()

	func() {
		defer func() {
			val := recover()
			require.Equal(t, "ringbuffer: pop from an empty queue", val)
		}()
		r.PopFront()
		t.Fatal("expected pop to panic")
	}()
}

func TestRingBuffer_Iterators(t *testing.T) {
	var r RingBuffer[int]
	r.Init(4)

	// Empty iterators
	count := 0
	for range r.All() {
		count++
	}
	require.Equal(t, 0, count)

	for range r.Values() {
		count++
	}
	require.Equal(t, 0, count)

	for range r.Drain() {
		count++
	}
	require.Equal(t, 0, count)

	// Push items with wrap-around
	r.PushBack(10)
	r.PushBack(20)
	require.Equal(t, 10, r.PopFront())
	r.PushBack(30)
	r.PushBack(40)
	r.PushBack(50) // Now elements: 20, 30, 40, 50 (wrapped)

	// All() non-destructive
	var items []int
	for v := range r.All() {
		items = append(items, v)
		if len(items) == 2 {
			break // test early exit
		}
	}
	require.Equal(t, []int{20, 30}, items)

	// Values() non-destructive full
	items = nil
	for v := range r.Values() {
		items = append(items, v)
	}
	require.Equal(t, []int{20, 30, 40, 50}, items)
	require.Equal(t, 4, r.Len())

	// Backward() non-destructive reverse
	items = nil
	for v := range r.Backward() {
		items = append(items, v)
		if len(items) == 2 {
			break // test early exit
		}
	}
	require.Equal(t, []int{50, 40}, items)

	items = nil
	for v := range r.Backward() {
		items = append(items, v)
	}
	require.Equal(t, []int{50, 40, 30, 20}, items)
	require.Equal(t, 4, r.Len())

	// Drain() consuming
	items = nil
	for v := range r.Drain() {
		items = append(items, v)
	}
	require.Equal(t, []int{20, 30, 40, 50}, items)
	require.True(t, r.Empty())
}

func TestClear(t *testing.T) {
	r := RingBuffer[int]{}
	r.Init(2)
	r.PushBack(1)
	r.PushBack(2)
	require.Equal(t, 2, r.Len())
	r.Clear()
	require.True(t, r.Empty())
	r.PushBack(3)
	require.Equal(t, 3, r.PopFront())
}

func BenchmarkRingBuffer(b *testing.B) {
	r := RingBuffer[int]{}

	var val int
	for b.Loop() {
		r.PushBack(val)
		r.PopFront()

		val++
	}
}

func BenchmarkRingBufferWrapped(b *testing.B) {
	var r RingBuffer[int]
	r.Init(32)

	for i := range 16 {
		r.PushBack(i)
	}

	var val int
	for b.Loop() {
		r.PushBack(val)
		val = r.PopFront()
	}
}
