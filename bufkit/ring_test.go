// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lemon4ksan/foundation/bufkit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestRing_WrapAround(t *testing.T) {
	t.Parallel()

	r := bufkit.NewRing[int](4)
	require.Equal(t, 4, r.Cap())

	// Push and pop multiple times to force head and tail to wrap past capacity
	for cycle := range 10 {
		for i := range 3 {
			val := cycle*10 + i
			require.True(t, r.Push(val))
		}
		require.Equal(t, 3, r.Len())

		for i := range 3 {
			expected := cycle*10 + i
			val, ok := r.Pop()
			require.True(t, ok)
			require.Equal(t, expected, val)
		}
		require.Equal(t, 0, r.Len())
	}
}

func TestRing_ZeroOrNegativeCapacity(t *testing.T) {
	t.Parallel()

	r0 := bufkit.NewRing[int](0)
	assert.Equal(t, 2, r0.Cap())

	rNeg := bufkit.NewRing[int](-5)
	assert.Equal(t, 2, rNeg.Cap())
}

func TestRing_Iterators(t *testing.T) {
	t.Parallel()

	r := bufkit.NewRing[int](8)

	// Empty ring
	count := 0
	for range r.All() {
		count++
	}
	assert.Equal(t, 0, count)

	for range r.Drain() {
		count++
	}
	assert.Equal(t, 0, count)

	// Add items
	for i := 1; i <= 5; i++ {
		require.True(t, r.Push(i*10))
	}
	require.Equal(t, 5, r.Len())

	// All() with early break
	var items []int
	for v := range r.All() {
		items = append(items, v)
		if len(items) == 2 {
			break
		}
	}
	assert.Equal(t, []int{10, 20}, items)
	assert.Equal(t, 5, r.Len())

	// Values() full traversal
	items = nil
	for v := range r.Values() {
		items = append(items, v)
	}
	assert.Equal(t, []int{10, 20, 30, 40, 50}, items)
	assert.Equal(t, 5, r.Len())

	// Drain() consuming with early exit
	items = nil
	for v := range r.Drain() {
		items = append(items, v)
		if len(items) == 2 {
			break
		}
	}
	assert.Equal(t, []int{10, 20}, items)
	assert.Equal(t, 3, r.Len())

	// Drain() remaining
	items = nil
	for v := range r.Drain() {
		items = append(items, v)
	}
	assert.Equal(t, []int{30, 40, 50}, items)
	assert.Equal(t, 0, r.Len())
}

func TestRing_ByteSliceIterators(t *testing.T) {
	t.Parallel()

	r := bufkit.NewRing[[]byte](4)
	r.Push([]byte("chunk1"))
	r.Push([]byte("chunk2"))

	var collected []string
	for b := range r.All() {
		collected = append(collected, string(b))
	}
	assert.Equal(t, []string{"chunk1", "chunk2"}, collected)

	var drained []string
	for b := range r.Drain() {
		drained = append(drained, string(b))
	}
	assert.Equal(t, []string{"chunk1", "chunk2"}, drained)
	assert.Equal(t, 0, r.Len())
}

func BenchmarkRing_PushPop_Sequential(b *testing.B) {
	ring := bufkit.NewRing[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ring.Push(42)
		ring.Pop()
	}
}

func BenchmarkRing_SPSC_Pipeline(b *testing.B) {
	ring := bufkit.NewRing[int](4096)
	var wg sync.WaitGroup

	b.ReportAllocs()
	b.ResetTimer()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			for !ring.Push(i) {
			}
		}
	}()

	for i := 0; i < b.N; i++ {
		for {
			if _, ok := ring.Pop(); ok {
				break
			}
		}
	}

	wg.Wait()
}

func BenchmarkRing_SPSC_Push_Parallel(b *testing.B) {
	ring := bufkit.NewRing[int](4096)
	var active atomic.Bool
	active.Store(true)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for active.Load() {
			ring.Pop()
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for !ring.Push(1) {
		}
	}

	b.StopTimer()
	active.Store(false)
	wg.Wait()
}

func BenchmarkRing_SPSC_Pop_Parallel(b *testing.B) {
	ring := bufkit.NewRing[int](4096)
	var active atomic.Bool
	active.Store(true)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for active.Load() {
			ring.Push(1)
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for {
			if _, ok := ring.Pop(); ok {
				break
			}
		}
	}

	b.StopTimer()
	active.Store(false)
	wg.Wait()
}

func BenchmarkRing_Reset(b *testing.B) {
	ring := bufkit.NewRing[int](1024)
	for i := range 1024 {
		ring.Push(i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ring.Reset()
	}
}
