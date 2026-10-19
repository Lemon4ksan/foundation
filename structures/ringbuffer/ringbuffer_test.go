// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ringbuffer

import (
	"math/rand/v2"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/require"
)

func TestPushPeekPop(t *testing.T) {
	r := New[int]()
	require.Equal(t, 0, len(r.ring))
	v, ok := r.PopFront()
	require.False(t, ok)
	require.Equal(t, 0, v)
	v, ok = r.PeekFront()
	require.False(t, ok)
	require.Equal(t, 0, v)
	require.Panics(t, func() { r.MustPopFront() })
	require.Panics(t, func() { r.MustPeekFront() })
	r.PushBack(1)
	r.PushBack(2)
	r.PushBack(3)
	require.Equal(t, 1, r.MustPeekFront())
	v, ok = r.PeekFront()
	require.True(t, ok)
	require.Equal(t, 1, v)
	require.Equal(t, 1, r.MustPopFront())
	require.Equal(t, 2, r.MustPeekFront())
	v, ok = r.PopFront()
	require.True(t, ok)
	require.Equal(t, 2, v)
	r.PushBack(4)
	r.PushBack(5)
	require.Equal(t, 3, r.Len())
	r.PushBack(6)
	require.Equal(t, 4, r.Len())
	r.PushBack(7) // grow with the buffer wrapped around
	require.Equal(t, 5, r.Len())
	require.Equal(t, 3, r.MustPopFront())
	require.Equal(t, 4, r.MustPopFront())
	require.Equal(t, 5, r.MustPopFront())
	require.Equal(t, 6, r.MustPopFront())
	require.Equal(t, 7, r.MustPopFront())
}

func TestPanicOnEmptyBuffer(t *testing.T) {
	r := Buffer[string]{}
	require.True(t, r.Empty())
	require.Zero(t, r.Len())

	func() {
		defer func() {
			val := recover()
			require.Equal(t, "ringbuffer: peek from an empty queue", val)
		}()
		r.MustPeekFront()
		t.Fatal("expected peek to panic")
	}()

	func() {
		defer func() {
			val := recover()
			require.Equal(t, "ringbuffer: pop from an empty queue", val)
		}()
		r.MustPopFront()
		t.Fatal("expected pop to panic")
	}()
}

func TestRingBuffer_Iterators(t *testing.T) {
	r := NewWithCapacity[int](4)

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
	require.Equal(t, 10, r.MustPopFront())
	r.PushBack(30)
	r.PushBack(40)
	r.PushBack(50) // Now elements: 20, 30, 40, 50 (wrapped)

	// All() non-destructive
	var items []int
	var indices []int
	for i, v := range r.All() {
		indices = append(indices, i)
		items = append(items, v)
		if len(items) == 2 {
			break // test early exit
		}
	}
	require.Equal(t, []int{0, 1}, indices)
	require.Equal(t, []int{20, 30}, items)

	// Values() non-destructive early exit
	items = nil
	for v := range r.Values() {
		items = append(items, v)
		if len(items) == 2 {
			break
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
	r := Buffer[int]{}
	r.Init(2)
	r.PushBack(1)
	r.PushBack(2)
	require.Equal(t, 2, r.Len())
	r.Clear()
	require.True(t, r.Empty())
	r.PushBack(3)
	require.Equal(t, 3, r.MustPopFront())
}

func BenchmarkRingBuffer(b *testing.B) {
	r := Buffer[int]{}

	var val int
	for b.Loop() {
		r.PushBack(val)
		_, _ = r.PopFront()

		val++
	}
}

func BenchmarkRingBufferWrapped(b *testing.B) {
	var r Buffer[int]
	r.Init(32)

	for i := range 16 {
		r.PushBack(i)
	}

	var val int
	for b.Loop() {
		r.PushBack(val)
		val, _ = r.PopFront()
	}
}

// TestChallenge_GCLeakVerification proves that popped elements or cleared elements
// do not remain referenced in the backing slice, preventing memory leaks.
func TestChallenge_GCLeakVerification(t *testing.T) {
	type trackedObj struct {
		payload [64]byte
		id      int
	}

	testCases := []struct {
		name string
		op   func(r *Buffer[*trackedObj])
	}{
		{
			name: "PopFront clears reference",
			op: func(r *Buffer[*trackedObj]) {
				_, _ = r.PopFront()
			},
		},
		{
			name: "Clear clears references",
			op: func(r *Buffer[*trackedObj]) {
				r.Clear()
			},
		},
		{
			name: "Drain clears references",
			op: func(r *Buffer[*trackedObj]) {
				for range r.Drain() {
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var finalized atomic.Int32

			r := new(Buffer[*trackedObj])
			r.Init(8)

			obj := &trackedObj{id: 42}
			runtime.AddCleanup(obj, func(counter *atomic.Int32) {
				counter.Add(1)
			}, &finalized)

			r.PushBack(obj)
			obj = nil // Drop local reference

			// Perform popping / clearing operation
			tc.op(r)

			// Force multiple GC cycles to verify object was collected
			for range 5 {
				runtime.GC()
				if finalized.Load() > 0 {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}

			require.Equal(t, int32(1), finalized.Load(), "object reference was leaked in ring buffer backing array")
		})
	}
}

// TestChallenge_DifferentialFuzz stresses Buffer against an unambiguous slice-based FIFO queue.
func TestChallenge_DifferentialFuzz(t *testing.T) {
	rng := rand.New(rand.NewPCG(12345, 67890))

	const numSteps = 50_000

	initialCapacities := []int{0, 1, 2, 3, 5, 8, 17, 64}

	for _, initCap := range initialCapacities {
		var r Buffer[int]
		if initCap > 0 {
			r.Init(initCap)
		}

		oracle := make([]int, 0)
		counter := 0

		for range numSteps {
			op := rng.IntN(10)

			switch {
			case op < 5: // 50% PushBack
				val := counter
				counter++
				r.PushBack(val)
				oracle = append(oracle, val)

			case op < 8: // 30% PopFront
				if len(oracle) == 0 {
					require.True(t, r.Empty())
					require.Zero(t, r.Len())
				} else {
					expected := oracle[0]
					oracle = oracle[1:]

					require.False(t, r.Empty())
					peekVal, ok := r.PeekFront()
					require.True(t, ok)
					require.Equal(t, expected, peekVal)
					actual, ok := r.PopFront()
					require.True(t, ok)
					require.Equal(t, expected, actual)
				}

			case op == 8: // 10% Verify All() and Backward()
				require.Equal(t, len(oracle), r.Len())
				require.Equal(t, len(oracle) == 0, r.Empty())

				allItems := make([]int, 0, r.Len())
				for idx, v := range r.All() {
					require.Equal(t, len(allItems), idx)
					allItems = append(allItems, v)
				}
				require.Equal(t, oracle, allItems)

				backItems := make([]int, 0, r.Len())
				for v := range r.Backward() {
					backItems = append(backItems, v)
				}
				expectedBack := make([]int, len(oracle))
				for i, v := range oracle {
					expectedBack[len(oracle)-1-i] = v
				}
				require.Equal(t, expectedBack, backItems)

			case op == 9: // 10% Partial drain or clear
				if rng.IntN(2) == 0 {
					r.Clear()
					oracle = oracle[:0]
					require.True(t, r.Empty())
					require.Zero(t, r.Len())
				} else {
					// Drain completely
					drained := make([]int, 0, r.Len())
					for v := range r.Drain() {
						drained = append(drained, v)
					}
					require.Equal(t, oracle, drained)
					oracle = oracle[:0]
					require.True(t, r.Empty())
					require.Zero(t, r.Len())
				}
			}

			require.Equal(t, len(oracle), r.Len())
			require.Equal(t, len(oracle) == 0, r.Empty())
		}
	}
}

// TestChallenge_WrapAroundGrowTransitions tests exact growth triggering at various head positions.
func TestChallenge_WrapAroundGrowTransitions(t *testing.T) {
	for initCap := 2; initCap <= 16; initCap++ {
		for headOffset := 0; headOffset < initCap; headOffset++ {
			var r Buffer[int]
			r.Init(initCap)

			// Advance head position by pushing and popping headOffset times
			for i := 0; i < headOffset; i++ {
				r.PushBack(i)
				_, _ = r.PopFront()
			}

			// Now fill the buffer to capacity
			for i := 0; i < initCap; i++ {
				r.PushBack(100 + i)
			}
			require.Equal(t, initCap, r.Len())

			// Next push forces grow() while headPos == headOffset
			r.PushBack(999)
			require.Equal(t, initCap+1, r.Len())

			// Verify all elements are returned in strict FIFO order
			for i := 0; i < initCap; i++ {
				require.Equal(t, 100+i, r.MustPopFront())
			}
			require.Equal(t, 999, r.MustPopFront())
			require.True(t, r.Empty())
		}
	}
}

// TestChallenge_ZeroValueUsability ensures that a zero-value Buffer without Init works cleanly.
func TestChallenge_ZeroValueUsability(t *testing.T) {
	var r Buffer[string]
	require.True(t, r.Empty())
	require.Zero(t, r.Len())

	r.PushBack("alpha")
	require.False(t, r.Empty())
	require.Equal(t, 1, r.Len())
	require.Equal(t, "alpha", r.MustPeekFront())
	require.Equal(t, "alpha", r.MustPopFront())
	require.True(t, r.Empty())
}

func TestBuffer_Drain_EarlyBreak(t *testing.T) {
	var r Buffer[int]
	r.PushBack(1)
	r.PushBack(2)
	r.PushBack(3)
	for v := range r.Drain() {
		if v == 1 {
			break
		}
	}
	require.Equal(t, 2, r.Len())
}
