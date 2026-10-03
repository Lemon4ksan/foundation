// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package deque

import (
	"fmt"
	"math/rand"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lemon4ksan/foundation/testing/require"
)

// -----------------------------------------------------------------------------
// Independent Reference Slice Oracle Model
// -----------------------------------------------------------------------------

type sliceOracle[T comparable] struct {
	data []T
}

func newSliceOracle[T comparable]() *sliceOracle[T] {
	return &sliceOracle[T]{data: make([]T, 0)}
}

func (o *sliceOracle[T]) pushFront(val T) {
	o.data = append([]T{val}, o.data...)
}

func (o *sliceOracle[T]) pushBack(val T) {
	o.data = append(o.data, val)
}

func (o *sliceOracle[T]) popFront() (T, bool) {
	if len(o.data) == 0 {
		var zero T
		return zero, false
	}
	val := o.data[0]
	o.data = o.data[1:]
	return val, true
}

func (o *sliceOracle[T]) popBack() (T, bool) {
	if len(o.data) == 0 {
		var zero T
		return zero, false
	}
	lastIdx := len(o.data) - 1
	val := o.data[lastIdx]
	o.data = o.data[:lastIdx]
	return val, true
}

func (o *sliceOracle[T]) front() (T, bool) {
	if len(o.data) == 0 {
		var zero T
		return zero, false
	}
	return o.data[0], true
}

func (o *sliceOracle[T]) back() (T, bool) {
	if len(o.data) == 0 {
		var zero T
		return zero, false
	}
	return o.data[len(o.data)-1], true
}

func (o *sliceOracle[T]) at(i int) T {
	if i < 0 {
		i += len(o.data)
	}
	return o.data[i]
}

func (o *sliceOracle[T]) set(i int, val T) {
	if i < 0 {
		i += len(o.data)
	}
	o.data[i] = val
}

func (o *sliceOracle[T]) clear() {
	o.data = o.data[:0]
}

func (o *sliceOracle[T]) len() int {
	return len(o.data)
}

func (o *sliceOracle[T]) toSlice() []T {
	cp := make([]T, len(o.data))
	copy(cp, o.data)
	return cp
}

func (o *sliceOracle[T]) backwardSlice() []T {
	cp := make([]T, len(o.data))
	for i := 0; i < len(o.data); i++ {
		cp[i] = o.data[len(o.data)-1-i]
	}
	return cp
}

// -----------------------------------------------------------------------------
// 1. Independent Slice Oracle Challenge across 100,000+ Randomized Operations
// -----------------------------------------------------------------------------

func TestOracle_100kRandomizedOps(t *testing.T) {
	const totalOps = 100_000
	rng := rand.New(rand.NewSource(998244353))

	d := New[int]()
	oracle := newSliceOracle[int]()

	for op := 0; op < totalOps; op++ {
		action := rng.Intn(100)
		val := rng.Intn(1_000_000)

		switch {
		case action < 25: // PushBack (25%)
			d.PushBack(val)
			oracle.pushBack(val)

		case action < 50: // PushFront (25%)
			d.PushFront(val)
			oracle.pushFront(val)

		case action < 68: // PopFront (18%)
			dVal, dOk := d.PopFront()
			oVal, oOk := oracle.popFront()
			require.Equal(t, oOk, dOk, fmt.Sprintf("op %d: PopFront ok mismatch", op))
			require.Equal(t, oVal, dVal, fmt.Sprintf("op %d: PopFront val mismatch", op))

		case action < 86: // PopBack (18%)
			dVal, dOk := d.PopBack()
			oVal, oOk := oracle.popBack()
			require.Equal(t, oOk, dOk, fmt.Sprintf("op %d: PopBack ok mismatch", op))
			require.Equal(t, oVal, dVal, fmt.Sprintf("op %d: PopBack val mismatch", op))

		case action < 92: // Front & Back Peek (6%)
			dFront, dFrontOk := d.Front()
			oFront, oFrontOk := oracle.front()
			require.Equal(t, oFrontOk, dFrontOk, fmt.Sprintf("op %d: Front ok mismatch", op))
			require.Equal(t, oFront, dFront, fmt.Sprintf("op %d: Front val mismatch", op))

			dBack, dBackOk := d.Back()
			oBack, oBackOk := oracle.back()
			require.Equal(t, oBackOk, dBackOk, fmt.Sprintf("op %d: Back ok mismatch", op))
			require.Equal(t, oBack, dBack, fmt.Sprintf("op %d: Back val mismatch", op))

		case action < 96: // Random access At & Set (4%)
			if oracle.len() > 0 {
				idx := rng.Intn(oracle.len())
				// Positive indexing
				require.Equal(t, oracle.at(idx), d.At(idx), fmt.Sprintf("op %d: At(%d) mismatch", op, idx))
				// Negative indexing
				negIdx := -1 - idx
				require.Equal(t, oracle.at(negIdx), d.At(negIdx), fmt.Sprintf("op %d: At(%d) mismatch", op, negIdx))

				// Set positive
				newVal := rng.Intn(1_000_000)
				d.Set(idx, newVal)
				oracle.set(idx, newVal)
				require.Equal(t, newVal, d.At(idx))

				// Set negative
				newVal2 := rng.Intn(1_000_000)
				d.Set(negIdx, newVal2)
				oracle.set(negIdx, newVal2)
				require.Equal(t, newVal2, d.At(negIdx))
			}

		default: // Clear (4%)
			d.Clear()
			oracle.clear()
		}

		// State invariants verification after EVERY operation
		require.Equal(t, oracle.len(), d.Len(), fmt.Sprintf("op %d: Len mismatch", op))
		require.Equal(t, oracle.len() == 0, d.Empty(), fmt.Sprintf("op %d: Empty mismatch", op))

		if oracle.len() > 0 {
			f, ok := d.Front()
			require.True(t, ok)
			require.Equal(t, oracle.at(0), f)
			require.Equal(t, oracle.at(0), d.At(0))

			b, ok := d.Back()
			require.True(t, ok)
			require.Equal(t, oracle.at(-1), b)
			require.Equal(t, oracle.at(-1), d.At(-1))
		}

		// Boundary panic invariant check on At and Set
		require.PanicsWithError(t, "deque: index out of range", func() { d.At(d.Len()) })
		require.PanicsWithError(t, "deque: index out of range", func() { d.At(-d.Len() - 1) })
		require.PanicsWithError(t, "deque: index out of range", func() { d.Set(d.Len(), 99) })
		require.PanicsWithError(t, "deque: index out of range", func() { d.Set(-d.Len()-1, 99) })

		// Periodic full iterator verification
		if op%1_000 == 0 {
			// Values iterator vs oracle slice
			collectedValues := slices.Collect(d.Values())
			expectedSlice := oracle.toSlice()
			if len(expectedSlice) == 0 {
				require.Equal(t, 0, len(collectedValues))
			} else {
				require.Equal(t, expectedSlice, collectedValues)
			}

			// Backward iterator vs reversed oracle slice
			collectedBackward := slices.Collect(d.Backward())
			expectedBackward := oracle.backwardSlice()
			if len(expectedBackward) == 0 {
				require.Equal(t, 0, len(collectedBackward))
			} else {
				require.Equal(t, expectedBackward, collectedBackward)
			}

			// All iterator vs indexed elements
			idxCount := 0
			for i, val := range d.All() {
				require.Equal(t, idxCount, i)
				require.Equal(t, expectedSlice[i], val)
				idxCount++
			}
			require.Equal(t, oracle.len(), idxCount)
		}
	}
}

// -----------------------------------------------------------------------------
// 2. Strict FIFO Queue Invariant Verification (Streaming & Steady-State)
// -----------------------------------------------------------------------------

func TestChallenge_FIFOInvariants_Streaming(t *testing.T) {
	// Test streaming across various fixed window capacities
	windowSizes := []int{1, 2, 3, 5, 8, 16, 31, 64, 128, 512, 1024}

	for _, win := range windowSizes {
		d := NewWithCapacity[int](win)
		const streamLength = 50_000

		// Fill window
		for i := 0; i < win; i++ {
			d.PushBack(i)
		}
		require.Equal(t, win, d.Len())

		// Slide window: for each new item pushed to back, pop front and verify strict FIFO monotonicity
		for i := win; i < streamLength; i++ {
			d.PushBack(i)
			expectedOut := i - win
			val, ok := d.PopFront()
			require.True(t, ok)
			require.Equal(t, expectedOut, val, fmt.Sprintf("window %d, step %d: FIFO order violation", win, i))
			require.Equal(t, win, d.Len())
		}

		// Drain the remainder: must be strictly ascending
		for i := streamLength - win; i < streamLength; i++ {
			val, ok := d.PopFront()
			require.True(t, ok)
			require.Equal(t, i, val)
		}
		require.True(t, d.Empty())
	}
}

func TestChallenge_ReverseFIFOInvariants_Streaming(t *testing.T) {
	// Reverse FIFO: PushFront and PopBack
	windowSizes := []int{1, 2, 4, 7, 16, 63, 100, 256}

	for _, win := range windowSizes {
		d := NewWithCapacity[int](win)
		const streamLength = 30_000

		// Fill window by pushing front
		for i := 0; i < win; i++ {
			d.PushFront(i)
		}
		require.Equal(t, win, d.Len())

		// Slide window: push front, pop back
		for i := win; i < streamLength; i++ {
			d.PushFront(i)
			expectedOut := i - win
			val, ok := d.PopBack()
			require.True(t, ok)
			require.Equal(t, expectedOut, val, fmt.Sprintf("rev window %d, step %d: FIFO order violation", win, i))
			require.Equal(t, win, d.Len())
		}

		// Drain remainder from back
		for i := streamLength - win; i < streamLength; i++ {
			val, ok := d.PopBack()
			require.True(t, ok)
			require.Equal(t, i, val)
		}
		require.True(t, d.Empty())
	}
}

// -----------------------------------------------------------------------------
// 3. Double-Ended Queue Properties & Dual LIFO Stacks
// -----------------------------------------------------------------------------

func TestChallenge_DoubleEnded_DualLIFOStacks(t *testing.T) {
	// Front stack: PushFront and PopFront must behave as pure LIFO
	frontStack := New[int]()
	const n = 10_000
	for i := 0; i < n; i++ {
		frontStack.PushFront(i)
		f, ok := frontStack.Front()
		require.True(t, ok)
		require.Equal(t, i, f)
	}
	for i := n - 1; i >= 0; i-- {
		val, ok := frontStack.PopFront()
		require.True(t, ok)
		require.Equal(t, i, val)
	}
	require.True(t, frontStack.Empty())

	// Back stack: PushBack and PopBack must behave as pure LIFO
	backStack := New[int]()
	for i := 0; i < n; i++ {
		backStack.PushBack(i)
		b, ok := backStack.Back()
		require.True(t, ok)
		require.Equal(t, i, b)
	}
	for i := n - 1; i >= 0; i-- {
		val, ok := backStack.PopBack()
		require.True(t, ok)
		require.Equal(t, i, val)
	}
	require.True(t, backStack.Empty())
}

func TestChallenge_DoubleEnded_SymmetricalPalindrome(t *testing.T) {
	d := New[int]()
	const pairs = 5_000

	// Push i to back, then i to front -> creates a palindrome:
	// front: (pairs-1), ..., 1, 0, 0, 1, ..., (pairs-1) :back
	for i := 0; i < pairs; i++ {
		d.PushBack(i)
		d.PushFront(i)
	}
	require.Equal(t, pairs*2, d.Len())

	// Symmetrical index invariance: At(i) == At(-1-i) == At(len - 1 - i)
	for i := 0; i < d.Len(); i++ {
		valFront := d.At(i)
		valBack := d.At(-1 - i)
		valBackPos := d.At(d.Len() - 1 - i)
		require.Equal(t, valFront, valBack)
		require.Equal(t, valFront, valBackPos)
	}

	// Symmetrical simultaneous pop
	for i := pairs - 1; i >= 0; i-- {
		fVal, fOk := d.PopFront()
		bVal, bOk := d.PopBack()
		require.True(t, fOk)
		require.True(t, bOk)
		require.Equal(t, i, fVal)
		require.Equal(t, i, bVal)
	}
	require.True(t, d.Empty())
}

// -----------------------------------------------------------------------------
// 4. Iterator Consistency & Early Exit Invariants
// -----------------------------------------------------------------------------

func TestChallenge_IteratorConsistency_ExhaustiveEarlyExit(t *testing.T) {
	const size = 128
	d := New[int]()
	for i := 0; i < size; i++ {
		d.PushBack(i * 10)
	}

	// Values() early exit at every cutoff point from 0 to size
	for cutoff := 0; cutoff <= size; cutoff++ {
		var collected []int
		for val := range d.Values() {
			if len(collected) == cutoff {
				break
			}
			collected = append(collected, val)
		}
		require.Equal(t, cutoff, len(collected))
		for i, v := range collected {
			require.Equal(t, i*10, v)
		}
		require.Equal(t, size, d.Len(), "early break must not alter deque length")
	}

	// Backward() early exit at every cutoff point from 0 to size
	for cutoff := 0; cutoff <= size; cutoff++ {
		var collected []int
		for val := range d.Backward() {
			if len(collected) == cutoff {
				break
			}
			collected = append(collected, val)
		}
		require.Equal(t, cutoff, len(collected))
		for i, v := range collected {
			expected := (size - 1 - i) * 10
			require.Equal(t, expected, v)
		}
		require.Equal(t, size, d.Len(), "early break must not alter deque length")
	}

	// All() early exit at every cutoff point from 0 to size
	for cutoff := 0; cutoff <= size; cutoff++ {
		count := 0
		for i, val := range d.All() {
			if count == cutoff {
				break
			}
			require.Equal(t, count, i)
			require.Equal(t, count*10, val)
			count++
		}
		require.Equal(t, cutoff, count)
	}

	// Drain() early exit: test partial consumption
	dDrain := New[int]()
	for i := 0; i < 50; i++ {
		dDrain.PushBack(i)
	}

	drainedCount := 0
	for val := range dDrain.Drain() {
		require.Equal(t, drainedCount, val)
		drainedCount++
		if drainedCount == 20 {
			break
		}
	}
	require.Equal(t, 20, drainedCount)
	require.Equal(t, 30, dDrain.Len())
	require.Equal(t, 20, dDrain.At(0))
	require.Equal(t, 49, dDrain.At(-1))
}

// -----------------------------------------------------------------------------
// 5. Ring Buffer Wraparound Boundary Stress
// -----------------------------------------------------------------------------

func TestChallenge_ContinuousHeadRotations(t *testing.T) {
	// Rotate the head continuously through thousands of full 360-degree cycles of the circular buffer
	const capSize = 16
	d := NewWithCapacity[int](capSize)

	// Keep size at 10 while shifting head 1,000,000 steps
	for i := 0; i < 10; i++ {
		d.PushBack(i)
	}

	const steps = 1_000_000
	for step := 0; step < steps; step++ {
		// Pop front
		oldVal, ok := d.PopFront()
		require.True(t, ok)
		require.Equal(t, step, oldVal)

		// Push back next val
		newVal := step + 10
		d.PushBack(newVal)

		// Invariant checks
		require.Equal(t, 10, d.Len())
		require.Equal(t, capSize, d.Cap()) // no reallocations!
	}
}

// -----------------------------------------------------------------------------
// 6. Generic Types Variety (Structs, Strings, Arrays, Pointers)
// -----------------------------------------------------------------------------

func TestChallenge_GenericTypesVariety(t *testing.T) {
	// Test strings
	dStr := New[string]()
	dStr.PushBack("alpha")
	dStr.PushFront("beta")
	dStr.PushBack("gamma")
	require.Equal(t, []string{"beta", "alpha", "gamma"}, slices.Collect(dStr.Values()))
	require.Equal(t, "beta", dStr.At(0))
	require.Equal(t, "gamma", dStr.At(-1))

	// Test complex structs
	type record struct {
		id   int
		name string
		data [16]byte
	}
	dRec := New[record]()
	rec1 := record{id: 1, name: "one", data: [16]byte{1, 2, 3}}
	rec2 := record{id: 2, name: "two", data: [16]byte{4, 5, 6}}
	dRec.PushBack(rec1)
	dRec.PushFront(rec2)
	require.Equal(t, rec2, dRec.At(0))
	require.Equal(t, rec1, dRec.At(1))
	poppedFront, _ := dRec.PopFront()
	require.Equal(t, rec2, poppedFront)

	// Test zero-sized types struct{}
	dEmpty := New[struct{}]()
	for i := 0; i < 100; i++ {
		dEmpty.PushBack(struct{}{})
	}
	require.Equal(t, 100, dEmpty.Len())
	for i := 0; i < 100; i++ {
		_, ok := dEmpty.PopFront()
		require.True(t, ok)
	}
	require.True(t, dEmpty.Empty())
}

// -----------------------------------------------------------------------------
// 7. Multi-Threaded Concurrent Torture & Race Validation
// -----------------------------------------------------------------------------

func TestChallenge_ConcurrentTortureWithRace(t *testing.T) {
	d := NewWithCapacity[int64](128)
	var mu sync.RWMutex
	var wg sync.WaitGroup

	const readers = 12
	const writers = 6
	const opsPerWorker = 1_000

	var totalPushed atomic.Int64
	var totalPopped atomic.Int64

	// Concurrent synchronized writers (mix of PushFront, PushBack, PopFront, PopBack)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				mu.Lock()
				val := int64(wid*opsPerWorker + i)
				if wid%2 == 0 {
					d.PushBack(val)
				} else {
					d.PushFront(val)
				}
				totalPushed.Add(1)

				if d.Len() > 64 {
					if wid%2 == 0 {
						_, ok := d.PopFront()
						if ok {
							totalPopped.Add(1)
						}
					} else {
						_, ok := d.PopBack()
						if ok {
							totalPopped.Add(1)
						}
					}
				}
				mu.Unlock()
			}
		}(w)
	}

	// Concurrent readers
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				mu.RLock()
				_ = d.Len()
				_ = d.Empty()
				_ = d.Cap()
				if d.Len() > 0 {
					_, _ = d.Front()
					_, _ = d.Back()
					_ = d.At(0)
					_ = d.At(-1)
				}
				mu.RUnlock()
			}
		}()
	}

	wg.Wait()

	// Drain remaining
	for {
		mu.Lock()
		_, ok := d.PopFront()
		mu.Unlock()
		if !ok {
			break
		}
		totalPopped.Add(1)
	}

	require.Equal(t, totalPushed.Load(), totalPopped.Load(), "every pushed item must be accounted for")
	require.True(t, d.Empty())
}
