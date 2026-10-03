// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitset

import (
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/testing/require"
)

// Reference oracle using a simple bool slice to verify BitSet behavior bit-by-bit.
type oracleBitSet struct {
	bits []bool
}

func newOracle(n int) *oracleBitSet {
	return &oracleBitSet{bits: make([]bool, n)}
}

func (o *oracleBitSet) set(i int) {
	if i >= len(o.bits) {
		newBits := make([]bool, i+1)
		copy(newBits, o.bits)
		o.bits = newBits
	}
	o.bits[i] = true
}

func (o *oracleBitSet) clear(i int) {
	if i < len(o.bits) {
		o.bits[i] = false
	}
}

func (o *oracleBitSet) test(i int) bool {
	if i < 0 || i >= len(o.bits) {
		return false
	}
	return o.bits[i]
}

func (o *oracleBitSet) count() int {
	c := 0
	for _, b := range o.bits {
		if b {
			c++
		}
	}
	return c
}

// 1. Empirical Challenge: Word Boundary Transitions
func TestWordBoundaryTransitionsExhaustive(t *testing.T) {
	boundaries := []int{0, 1, 62, 63, 64, 65, 126, 127, 128, 129, 1022, 1023, 1024, 1025, 2047, 2048, 2049}

	for _, bIdx := range boundaries {
		// Test BitSet of size bIdx + 1
		b := New(bIdx + 1)
		require.Equal(t, bIdx+1, b.Len())
		require.False(t, b.Test(bIdx))

		// Set boundary bit
		b.Set(bIdx)
		require.True(t, b.Test(bIdx))
		require.Equal(t, 1, b.Count())

		// Clear boundary bit
		b.Clear(bIdx)
		require.False(t, b.Test(bIdx))
		require.Equal(t, 0, b.Count())

		// Toggle boundary bit
		b.Toggle(bIdx)
		require.True(t, b.Test(bIdx))
		require.Equal(t, 1, b.Count())
		b.Toggle(bIdx)
		require.False(t, b.Test(bIdx))
		require.Equal(t, 0, b.Count())

		// Auto-expansion across boundary: BitSet initialized to bIdx, set at bIdx
		bAuto := New(bIdx)
		require.Equal(t, bIdx, bAuto.Len())
		bAuto.Set(bIdx)
		require.Equal(t, bIdx+1, bAuto.Len())
		require.True(t, bAuto.Test(bIdx))

		// NextSet around boundary
		bSearch := New(bIdx + 10)
		bSearch.Set(bIdx)
		// NextSet from 0 up to bIdx should find bIdx
		idx, ok := bSearch.NextSet(bIdx)
		require.True(t, ok)
		require.Equal(t, bIdx, idx)

		if bIdx > 0 {
			idx, ok = bSearch.NextSet(bIdx - 1)
			require.True(t, ok)
			require.Equal(t, bIdx, idx)
		}
		// NextSet past bIdx should not find bIdx
		idx, ok = bSearch.NextSet(bIdx + 1)
		require.False(t, ok)
		require.Equal(t, -1, idx)

		// NextClear around boundary
		bAll := New(bIdx + 2).SetAll()
		bAll.Clear(bIdx)
		idx, ok = bAll.NextClear(bIdx)
		require.True(t, ok)
		require.Equal(t, bIdx, idx)
		if bIdx > 0 {
			idx, ok = bAll.NextClear(bIdx - 1)
			require.True(t, ok)
			require.Equal(t, bIdx, idx)
		}
		idx, ok = bAll.NextClear(bIdx + 1)
		require.False(t, ok)
		require.Equal(t, -1, idx)

		// AllSet boundary precision
		bExact := New(bIdx + 1).SetAll()
		require.True(t, bExact.AllSet())
		bExact.Clear(bIdx)
		require.False(t, bExact.AllSet())
		bExact.Set(bIdx)
		require.True(t, bExact.AllSet())
	}
}

// 2. Empirical Challenge: Unequal Length Bitwise Operations
func TestUnequalLengthOperationsExhaustive(t *testing.T) {
	pairs := [][2]int{
		{100, 200},
		{200, 100},
		{63, 64},
		{64, 63},
		{64, 128},
		{128, 64},
		{65, 127},
		{1, 1000},
		{1000, 1},
		{0, 100},
		{100, 0},
		{64, 64},
		{128, 128},
	}

	for _, p := range pairs {
		len1, len2 := p[0], p[1]

		// Construct bitsets and corresponding reference oracles
		b1 := New(len1)
		b2 := New(len2)
		o1 := newOracle(len1)
		o2 := newOracle(len2)

		// Populate pseudorandom pattern
		for i := 0; i < len1; i++ {
			if (i*7+3)%5 == 0 {
				b1.Set(i)
				o1.set(i)
			}
		}
		for i := 0; i < len2; i++ {
			if (i*11+5)%3 == 0 {
				b2.Set(i)
				o2.set(i)
			}
		}

		// --- Test AND ---
		{
			res := b1.Clone().And(b2)
			// AND length must remain b1.Len()
			require.Equal(t, len1, res.Len())
			for i := 0; i < len1; i++ {
				expected := o1.test(i) && o2.test(i)
				require.Equal(t, expected, res.Test(i))
			}
			// Past len1 must be false
			require.False(t, res.Test(len1))
		}

		// --- Test OR ---
		{
			res := b1.Clone().Or(b2)
			expectedLen := max(len1, len2)
			require.Equal(t, expectedLen, res.Len())
			for i := 0; i < expectedLen; i++ {
				expected := o1.test(i) || o2.test(i)
				require.Equal(t, expected, res.Test(i))
			}
			require.False(t, res.Test(expectedLen))
		}

		// --- Test XOR ---
		{
			res := b1.Clone().Xor(b2)
			expectedLen := max(len1, len2)
			require.Equal(t, expectedLen, res.Len())
			for i := 0; i < expectedLen; i++ {
				expected := o1.test(i) != o2.test(i)
				require.Equal(t, expected, res.Test(i))
			}
			require.False(t, res.Test(expectedLen))
		}

		// --- Test AND NOT ---
		{
			res := b1.Clone().AndNot(b2)
			// AND NOT length must remain b1.Len()
			require.Equal(t, len1, res.Len())
			for i := 0; i < len1; i++ {
				expected := o1.test(i) && !o2.test(i)
				require.Equal(t, expected, res.Test(i))
			}
			require.False(t, res.Test(len1))
		}
	}
}

// 3. Empirical Challenge: Rapid Resize, Grow, ShrinkToFit Transitions
func TestRapidResizeGrowShrinkStress(t *testing.T) {
	b := New(0)
	oracle := newOracle(0)

	rnd := rand.New(rand.NewPCG(42, 99))

	for step := 0; step < 1000; step++ {
		action := rnd.IntN(5)
		switch action {
		case 0:
			// Resize to random size (0 to 1500)
			newSize := rnd.IntN(1500)
			b.Resize(newSize)
			if newSize < len(oracle.bits) {
				oracle.bits = oracle.bits[:newSize]
			} else if newSize > len(oracle.bits) {
				ext := make([]bool, newSize-len(oracle.bits))
				oracle.bits = append(oracle.bits, ext...)
			}
			require.Equal(t, newSize, b.Len())

		case 1:
			// Set / clear random bits
			if b.Len() > 0 {
				idx := rnd.IntN(b.Len())
				val := rnd.IntN(2) == 1
				b.SetTo(idx, val)
				if val {
					oracle.set(idx)
				} else {
					oracle.clear(idx)
				}
			}

		case 2:
			// Auto-expand via Set
			idx := rnd.IntN(2000)
			b.Set(idx)
			oracle.set(idx)
			require.GreaterOrEqual(t, b.Len(), idx+1)

		case 3:
			// Grow capacity
			capTarget := rnd.IntN(3000)
			b.Grow(capTarget)
			require.GreaterOrEqual(t, b.Cap(), capTarget)
			require.Equal(t, len(oracle.bits), b.Len())

		case 4:
			// ShrinkToFit
			b.ShrinkToFit()
			expectedCapWords := (b.Len() + 63) / 64
			require.Equal(t, expectedCapWords*64, b.Cap())
			require.Equal(t, len(oracle.bits), b.Len())
		}

		// Spot check invariants every 50 steps
		if step%50 == 0 {
			require.Equal(t, len(oracle.bits), b.Len())
			require.Equal(t, oracle.count(), b.Count())
			if b.Len() > 0 {
				for check := 0; check < 20; check++ {
					chkIdx := rnd.IntN(b.Len())
					require.Equal(t, oracle.test(chkIdx), b.Test(chkIdx))
				}
			}
		}
	}

	// Final full state verification
	require.Equal(t, len(oracle.bits), b.Len())
	require.Equal(t, oracle.count(), b.Count())
	for i := 0; i < b.Len(); i++ {
		require.Equal(t, oracle.test(i), b.Test(i))
	}
}

// 4. Empirical Challenge: Iterator Early Termination Stress
func TestIteratorEarlyTerminationExhaustive(t *testing.T) {
	b := New(2048)
	var expected []int
	for i := 0; i < 2048; i += 13 {
		b.Set(i)
		expected = append(expected, i)
	}

	// Break at every possible yield count from 0 to len(expected)
	for breakAt := 0; breakAt <= len(expected); breakAt++ {
		collected := make([]int, 0, breakAt)
		count := 0
		for idx := range b.Bits() {
			if count == breakAt {
				break
			}
			collected = append(collected, idx)
			count++
		}
		require.Equal(t, breakAt, len(collected))
		require.Equal(t, expected[:breakAt], collected)
	}

	// Nested loop early breaks
	bInner := New(100)
	bInner.Set(5).Set(15).Set(25)

	outerCount := 0
	for range b.Bits() {
		outerCount++
		innerCount := 0
		for range bInner.Bits() {
			innerCount++
			if innerCount == 2 {
				break
			}
		}
		require.Equal(t, 2, innerCount)
		if outerCount == 5 {
			break
		}
	}
	require.Equal(t, 5, outerCount)
}

// 5. Empirical Challenge: Multi-goroutine Read Contention and Synchronized Writes
func TestConcurrentContentionAndSynchronizedWritesStress(t *testing.T) {
	const bitsetSize = 4096
	b := New(bitsetSize)
	for i := 0; i < bitsetSize; i += 7 {
		b.Set(i)
	}

	var mu sync.RWMutex
	var wg sync.WaitGroup

	stopCh := make(chan struct{})

	// 50 concurrent reader goroutines
	for r := 0; r < 50; r++ {
		wg.Add(1)
		go func(readerId int) {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					mu.RLock()
					_ = b.Len()
					_ = b.Cap()
					_ = b.Empty()
					_ = b.Count()
					_ = b.Any()
					_ = b.None()
					_ = b.AllSet()
					_ = b.Test((readerId * 17) % bitsetSize)
					_, _ = b.NextSet((readerId * 13) % bitsetSize)
					_, _ = b.NextClear((readerId * 11) % bitsetSize)
					c := b.Clone()
					_ = c.Count()
					breakCount := 0
					for idx := range b.Bits() {
						_ = idx
						breakCount++
						if breakCount > 5 {
							break
						}
					}
					mu.RUnlock()
				}
			}
		}(r)
	}

	var writersWg sync.WaitGroup
	for w := 0; w < 10; w++ {
		writersWg.Add(1)
		go func(writerId int) {
			defer writersWg.Done()
			for iter := 0; iter < 100; iter++ {
				mu.Lock()
				switch iter % 6 {
				case 0:
					b.Set((writerId*50 + iter) % bitsetSize)
				case 1:
					b.Clear((writerId*50 + iter) % bitsetSize)
				case 2:
					b.Toggle((writerId*50 + iter) % bitsetSize)
				case 3:
					b.SetTo((writerId*50+iter)%bitsetSize, iter%2 == 0)
				case 4:
					b.Grow(bitsetSize + 500)
				case 5:
					b.ShrinkToFit()
				}
				mu.Unlock()
			}
		}(w)
	}

	writersWg.Wait()
	close(stopCh)
	wg.Wait()
}
