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

// TestChallenge_GCLeakVerification proves that popped elements or cleared elements
// do not remain referenced in the backing slice, preventing memory leaks.
func TestChallenge_GCLeakVerification(t *testing.T) {
	type trackedObj struct {
		id int
	}

	testCases := []struct {
		name string
		op   func(r *Buffer[*trackedObj])
	}{
		{
			name: "PopFront clears reference",
			op: func(r *Buffer[*trackedObj]) {
				_ = r.PopFront()
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
			runtime.SetFinalizer(obj, func(o *trackedObj) {
				finalized.Add(1)
			})

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

		for step := 0; step < numSteps; step++ {
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
					require.Equal(t, expected, r.PeekFront())
					actual := r.PopFront()
					require.Equal(t, expected, actual)
				}

			case op == 8: // 10% Verify All() and Backward()
				require.Equal(t, len(oracle), r.Len())
				require.Equal(t, len(oracle) == 0, r.Empty())

				allItems := make([]int, 0, r.Len())
				for v := range r.All() {
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
				_ = r.PopFront()
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
				require.Equal(t, 100+i, r.PopFront())
			}
			require.Equal(t, 999, r.PopFront())
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
	require.Equal(t, "alpha", r.PeekFront())
	require.Equal(t, "alpha", r.PopFront())
	require.True(t, r.Empty())
}
