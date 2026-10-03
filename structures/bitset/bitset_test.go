// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitset

import (
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/testing/require"
)

func TestConstructors(t *testing.T) {
	// New panics on negative length
	require.Panics(t, func() { New(-1) })
	require.Panics(t, func() { New(-100) })

	// New with zero length
	b0 := New(0)
	require.Equal(t, 0, b0.Len())
	require.Equal(t, 0, b0.Cap())
	require.True(t, b0.Empty())
	require.Equal(t, 0, b0.Count())

	// Standard lengths and word allocations
	for _, n := range []int{1, 63, 64, 65, 127, 128, 129, 1000} {
		b := New(n)
		require.Equal(t, n, b.Len())
		require.False(t, b.Empty())
		expectedWords := (n + 63) / 64
		require.Equal(t, expectedWords*64, b.Cap())
		require.Equal(t, 0, b.Count())
	}

	// NewFromWords panics
	require.Panics(t, func() { NewFromWords(nil, -1) })
	require.Panics(t, func() { NewFromWords([]uint64{1}, 65) })

	// NewFromWords valid zero length
	bWords0 := NewFromWords(nil, 0)
	require.Equal(t, 0, bWords0.Len())
	require.True(t, bWords0.Empty())

	// NewFromWords with trailing bit cleanup and extra words
	words := []uint64{^uint64(0), ^uint64(0)} // 2 words, all 1s
	bWords := NewFromWords(words, 10)         // only 10 bits logical length
	require.Equal(t, 10, bWords.Len())
	require.Equal(t, 10, bWords.Count())
	require.True(t, bWords.Test(9))
	require.False(t, bWords.Test(10))
	require.Equal(t, uint64(0), bWords.words[1]) // word 1 cleared by cleanTrailing

	// FromBytes panics on negative length
	require.Panics(t, func() { FromBytes(nil, -1) })

	// FromBytes with zero length
	bBytes0 := FromBytes([]byte{0xFF}, 0)
	require.Equal(t, 0, bBytes0.Len())
	require.True(t, bBytes0.Empty())

	// FromBytes little-endian byte conversion
	data := []byte{0x01, 0x80} // bit 0 and bit 15
	bBytes := FromBytes(data, 16)
	require.Equal(t, 16, bBytes.Len())
	require.True(t, bBytes.Test(0))
	require.False(t, bBytes.Test(1))
	require.True(t, bBytes.Test(15))
	require.Equal(t, 2, bBytes.Count())

	// FromBytes with byte slice shorter than nbits
	bShort := FromBytes([]byte{0x01}, 32)
	require.Equal(t, 32, bShort.Len())
	require.True(t, bShort.Test(0))
	require.False(t, bShort.Test(8))

	// FromBytes with byte slice longer than nbits and partial byte
	bMask := FromBytes([]byte{0xFF, 0xFF}, 4)
	require.Equal(t, 4, bMask.Len())
	require.Equal(t, 4, bMask.Count())
	require.True(t, bMask.Test(3))
	require.False(t, bMask.Test(4))
}

func TestNilAndZeroValueReceiver(t *testing.T) {
	var nilB *BitSet
	require.Equal(t, 0, nilB.Len())
	require.Equal(t, 0, nilB.Cap())
	require.True(t, nilB.Empty())
	require.Equal(t, 0, nilB.Count())
	require.False(t, nilB.Any())
	require.True(t, nilB.None())
	require.True(t, nilB.AllSet())
	require.False(t, nilB.Test(0))
	require.Panics(t, func() { nilB.Test(-1) })
	require.Nil(t, nilB.Clone())
	require.Nil(t, nilB.Clear(5))
	require.Nil(t, nilB.ClearAll())
	require.Nil(t, nilB.SetAll())
	require.Nil(t, nilB.Not())
	require.Nil(t, nilB.Flip())
	require.Nil(t, nilB.And(New(10)))
	require.Nil(t, nilB.Or(New(10)))
	require.Nil(t, nilB.Xor(New(10)))
	require.Nil(t, nilB.AndNot(New(10)))

	for range nilB.Bits() {
		t.Fatal("nil bitset should yield no items")
	}
	for range nilB.Values() {
		t.Fatal("nil bitset should yield no items")
	}
	for range nilB.All() {
		t.Fatal("nil bitset should yield no items")
	}

	idx, ok := nilB.NextSet(0)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	idx, ok = nilB.NextClear(0)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	var zeroB BitSet
	require.Equal(t, 0, zeroB.Len())
	require.Equal(t, 0, zeroB.Cap())
	require.True(t, zeroB.Empty())
	require.Equal(t, 0, zeroB.Count())
	require.False(t, zeroB.Any())
	require.True(t, zeroB.None())
	require.True(t, zeroB.AllSet())

	// Setting bit on zero value BitSet
	zeroB.Set(10)
	require.Equal(t, 11, zeroB.Len())
	require.True(t, zeroB.Test(10))
	require.False(t, zeroB.Test(9))
}

func TestInspection(t *testing.T) {
	b := New(128)
	require.False(t, b.Any())
	require.True(t, b.None())
	require.False(t, b.AllSet())

	b.Set(0)
	require.True(t, b.Any())
	require.False(t, b.None())
	require.False(t, b.AllSet())

	// Test AllSet with partial words and boundaries
	b.SetAll()
	require.True(t, b.Any())
	require.False(t, b.None())
	require.True(t, b.AllSet())

	// Clear bit in first word
	b.Clear(10)
	require.False(t, b.AllSet())
	b.Set(10)

	// Clear bit in boundary
	b.Clear(63)
	require.False(t, b.AllSet())
	b.Set(63)

	b.Clear(64)
	require.False(t, b.AllSet())
	b.Set(64)

	b.Clear(127)
	require.False(t, b.AllSet())
	b.Set(127)
	require.True(t, b.AllSet())

	// Non-multiple of 64 for AllSet
	bOdd := New(70).SetAll()
	require.True(t, bOdd.AllSet())
	bOdd.Clear(69)
	require.False(t, bOdd.AllSet())
	bOdd.Set(69)
	bOdd.Clear(0)
	require.False(t, bOdd.AllSet())
}

func TestEqual(t *testing.T) {
	var b1, b2 *BitSet
	require.True(t, b1.Equal(b2)) // both nil

	b1 = New(64)
	require.False(t, b1.Equal(nil))
	require.False(t, (*BitSet)(nil).Equal(b1))

	b2 = New(65)
	require.False(t, b1.Equal(b2)) // different length

	b2 = New(64)
	require.True(t, b1.Equal(b2)) // both empty length 64
	require.True(t, b1.Equal(b1)) // pointer identity

	b1.Set(10)
	require.False(t, b1.Equal(b2))
	b2.Set(10)
	require.True(t, b1.Equal(b2))

	// Multi-word difference
	b3 := New(130).Set(129)
	b4 := New(130).Set(128)
	require.False(t, b3.Equal(b4))
}

func TestBitMutations(t *testing.T) {
	b := New(50)

	require.Panics(t, func() { b.Test(-1) })
	require.Panics(t, func() { b.Set(-1) })
	require.Panics(t, func() { b.Clear(-1) })
	require.Panics(t, func() { b.Toggle(-1) })
	require.Panics(t, func() { b.SetTo(-1, true) })

	// Out of bounds queries
	require.False(t, b.Test(50))
	require.False(t, b.Test(1000))

	// Out of bounds clear is a no-op
	b.Clear(100)
	require.Equal(t, 50, b.Len())

	// Out of bounds set auto-expands length
	b.Set(100)
	require.Equal(t, 101, b.Len())
	require.True(t, b.Test(100))

	// In-bounds clear
	b.Clear(100)
	require.False(t, b.Test(100))

	// Out of bounds toggle auto-expands length
	b.Toggle(200)
	require.Equal(t, 201, b.Len())
	require.True(t, b.Test(200))

	// In-bounds toggle
	b.Toggle(200)
	require.False(t, b.Test(200))

	// SetTo
	b.SetTo(5, true)
	require.True(t, b.Test(5))
	b.SetTo(5, false)
	require.False(t, b.Test(5))
}

func TestWordBoundaries(t *testing.T) {
	boundaries := []int{0, 1, 62, 63, 64, 65, 126, 127, 128, 129, 255, 256}
	b := New(300)

	for _, idx := range boundaries {
		require.False(t, b.Test(idx))
		b.Set(idx)
		require.True(t, b.Test(idx))
		b.Clear(idx)
		require.False(t, b.Test(idx))
		b.Toggle(idx)
		require.True(t, b.Test(idx))
		b.Toggle(idx)
		require.False(t, b.Test(idx))
	}
}

func TestBulkOperations(t *testing.T) {
	// Multiple of 64
	b64 := New(64)
	b64.SetAll()
	require.Equal(t, 64, b64.Count())
	b64.ClearAll()
	require.Equal(t, 0, b64.Count())

	// Non-multiple of 64
	b70 := New(70)
	b70.SetAll()
	require.Equal(t, 70, b70.Count())
	require.True(t, b70.Test(69))
	require.False(t, b70.Test(70))

	// Not / Flip
	b70.Flip()
	require.Equal(t, 0, b70.Count())
	b70.Not()
	require.Equal(t, 70, b70.Count())

	// Zero length bulk operations
	b0 := New(0)
	b0.SetAll()
	require.Equal(t, 0, b0.Len())
	b0.ClearAll()
	b0.Flip()
	b0.Not()
	require.Equal(t, 0, b0.Len())
}

func TestClone(t *testing.T) {
	var nilB *BitSet
	require.Nil(t, nilB.Clone())

	b0 := New(0)
	c0 := b0.Clone()
	require.Equal(t, 0, c0.Len())
	require.Nil(t, c0.words)

	b := New(100).Set(10).Set(70)
	c := b.Clone()
	require.True(t, b.Equal(c))

	// Mutation independence
	c.Set(20)
	require.False(t, b.Test(20))
	require.True(t, c.Test(20))

	b.Set(30)
	require.True(t, b.Test(30))
	require.False(t, c.Test(30))
}

func TestNextSet(t *testing.T) {
	require.Panics(t, func() { New(10).NextSet(-1) })

	var nilB *BitSet
	idx, ok := nilB.NextSet(0)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	b := New(200)
	idx, ok = b.NextSet(0)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	// Set sparse bits across words
	b.Set(5).Set(63).Set(64).Set(190)

	// Hit at start
	idx, ok = b.NextSet(5)
	require.True(t, ok)
	require.Equal(t, 5, idx)

	// Hit inside first word
	idx, ok = b.NextSet(6)
	require.True(t, ok)
	require.Equal(t, 63, idx)

	// Hit at word boundary 64
	idx, ok = b.NextSet(64)
	require.True(t, ok)
	require.Equal(t, 64, idx)

	// Skip empty word to word 2
	idx, ok = b.NextSet(65)
	require.True(t, ok)
	require.Equal(t, 190, idx)

	// Past last set bit
	idx, ok = b.NextSet(191)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	// Start >= length
	idx, ok = b.NextSet(200)
	require.False(t, ok)
	require.Equal(t, -1, idx)
}

func TestNextClear(t *testing.T) {
	require.Panics(t, func() { New(10).NextClear(-1) })

	var nilB *BitSet
	idx, ok := nilB.NextClear(0)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	// Start >= length
	b := New(130).SetAll()
	idx, ok = b.NextClear(130)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	// All bits set: no clear bits
	idx, ok = b.NextClear(0)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	// Clear bits at various positions
	b.Clear(5).Clear(64).Clear(129)

	// Hit in first word
	idx, ok = b.NextClear(0)
	require.True(t, ok)
	require.Equal(t, 5, idx)

	// Hit at boundary in word 1
	idx, ok = b.NextClear(6)
	require.True(t, ok)
	require.Equal(t, 64, idx)

	// Word skip to word 2 (last partial word)
	idx, ok = b.NextClear(65)
	require.True(t, ok)
	require.Equal(t, 129, idx)

	// After last clear bit
	idx, ok = b.NextClear(130)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	// Exact multiple of 64
	b64 := New(64).SetAll()
	idx, ok = b64.NextClear(0)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	b64.Clear(63)
	idx, ok = b64.NextClear(0)
	require.True(t, ok)
	require.Equal(t, 63, idx)

	// First word is the last word with rem != 0
	bPartial := New(10).SetAll()
	idx, ok = bPartial.NextClear(0)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	bPartial.Clear(3)
	idx, ok = bPartial.NextClear(0)
	require.True(t, ok)
	require.Equal(t, 3, idx)
}

func TestBitwiseOperations(t *testing.T) {
	// And
	b1 := New(100).Set(10).Set(20).Set(70)
	b2 := New(100).Set(20).Set(30).Set(70)
	b1.And(b2)
	require.False(t, b1.Test(10))
	require.True(t, b1.Test(20))
	require.False(t, b1.Test(30))
	require.True(t, b1.Test(70))

	// Self-And
	b1.And(b1)
	require.True(t, b1.Test(20))

	// And with nil or empty
	b1.And(nil)
	require.Equal(t, 0, b1.Count())

	b1.Set(10)
	b1.And(New(0))
	require.Equal(t, 0, b1.Count())

	// And with shorter other
	bLong := New(200).Set(10).Set(150)
	bShort := New(50).Set(10)
	bLong.And(bShort)
	require.True(t, bLong.Test(10))
	require.False(t, bLong.Test(150))

	// Or
	b3 := New(50).Set(10)
	b4 := New(100).Set(70)
	b3.Or(b4) // expands b3 to 100
	require.Equal(t, 100, b3.Len())
	require.True(t, b3.Test(10))
	require.True(t, b3.Test(70))

	// Or with self, nil, or empty
	b3.Or(b3)
	require.True(t, b3.Test(10))
	b3.Or(nil)
	require.True(t, b3.Test(10))
	b3.Or(New(0))
	require.True(t, b3.Test(10))

	// Or with shorter other
	b3.Or(New(20).Set(5))
	require.True(t, b3.Test(5))

	// Xor
	b5 := New(100).Set(10).Set(20)
	b6 := New(100).Set(20).Set(30)
	b5.Xor(b6)
	require.True(t, b5.Test(10))
	require.False(t, b5.Test(20))
	require.True(t, b5.Test(30))

	// Xor with nil or empty
	b5.Xor(nil)
	require.True(t, b5.Test(10))
	b5.Xor(New(0))
	require.True(t, b5.Test(10))

	// Xor expanding length
	b5.Xor(New(150).Set(140))
	require.Equal(t, 150, b5.Len())
	require.True(t, b5.Test(140))

	// Self-Xor zeroes out
	b5.Xor(b5)
	require.Equal(t, 0, b5.Count())

	// AndNot
	b7 := New(100).Set(10).Set(20).Set(30)
	b8 := New(100).Set(20)
	b7.AndNot(b8)
	require.True(t, b7.Test(10))
	require.False(t, b7.Test(20))
	require.True(t, b7.Test(30))

	// AndNot with nil or empty
	b7.AndNot(nil)
	require.True(t, b7.Test(10))
	b7.AndNot(New(0))
	require.True(t, b7.Test(10))

	// Self-AndNot zeroes out
	b7.AndNot(b7)
	require.Equal(t, 0, b7.Count())
}

func TestIterators(t *testing.T) {
	b := New(200)
	indices := []int{0, 5, 63, 64, 65, 127, 128, 199}
	for _, idx := range indices {
		b.Set(idx)
	}

	// Bits traversal
	var yielded []int
	for idx := range b.Bits() {
		yielded = append(yielded, idx)
	}
	require.Equal(t, indices, yielded)

	// Early exit
	var early []int
	for idx := range b.Bits() {
		early = append(early, idx)
		if len(early) == 3 {
			break
		}
	}
	require.Equal(t, indices[:3], early)

	// All() and Values() alias parity
	var allYielded, valYielded []int
	for idx := range b.All() {
		allYielded = append(allYielded, idx)
	}
	for idx := range b.Values() {
		valYielded = append(valYielded, idx)
	}
	require.Equal(t, indices, allYielded)
	require.Equal(t, indices, valYielded)

	// Empty bitset iterator
	bEmpty := New(0)
	for range bEmpty.Bits() {
		t.Fatal("empty bitset should yield no items")
	}
}

func TestCapacityManagement(t *testing.T) {
	require.Panics(t, func() { New(10).Resize(-1) })
	require.Panics(t, func() { New(10).Grow(-1) })

	b := New(50)
	b.Set(10).Set(40)

	// Resize to same length (no-op)
	b.Resize(50)
	require.Equal(t, 50, b.Len())

	// Expand resize within capacity
	b.Resize(60)
	require.Equal(t, 60, b.Len())
	require.True(t, b.Test(10))
	require.True(t, b.Test(40))

	// Expand resize exceeding capacity
	b.Resize(200)
	require.Equal(t, 200, b.Len())
	require.True(t, b.Test(10))
	require.True(t, b.Test(40))

	// Shrink resize
	b.Resize(30)
	require.Equal(t, 30, b.Len())
	require.True(t, b.Test(10))
	require.False(t, b.Test(40))

	// Grow
	b.Grow(500)
	require.GreaterOrEqual(t, b.Cap(), 500)
	require.Equal(t, 30, b.Len()) // length unaltered

	// Grow with smaller capacity than current is no-op
	curCap := b.Cap()
	b.Grow(100)
	require.Equal(t, curCap, b.Cap())

	// ShrinkToFit
	b.ShrinkToFit()
	require.Equal(t, 64, b.Cap()) // exact words for 30 bits

	// ShrinkToFit when already minimal
	b.ShrinkToFit()
	require.Equal(t, 64, b.Cap())

	// ShrinkToFit on empty bitset
	bEmpty := New(100)
	bEmpty.Resize(0)
	require.Equal(t, 0, bEmpty.Len())
	bEmpty.ShrinkToFit()
	require.Equal(t, 0, bEmpty.Cap())
}

func TestEnsureWordsBranches(t *testing.T) {
	// Exercise needed <= cap(b.words) branch
	b := New(10)
	b.Grow(256)
	origCap := cap(b.words)
	require.GreaterOrEqual(t, origCap, 4)
	require.Equal(t, 1, len(b.words))

	// Expanding to 150 bits needs 3 words, which is <= 4 cap
	b.Set(150)
	require.Equal(t, 151, b.Len())
	require.True(t, b.Test(150))

	// Exercise newCap < needed branch with empty BitSet
	var bZero BitSet
	bZero.Set(500) // cap 0, needs 8 words (0*2 < 8)
	require.Equal(t, 501, bZero.Len())
	require.True(t, bZero.Test(500))
}

func TestConcurrentReaders(t *testing.T) {
	b := New(10000)
	for i := 0; i < 10000; i += 7 {
		b.Set(i)
	}

	var wg sync.WaitGroup
	for g := 0; g < 32; g++ {
		wg.Add(1)
		go func(gId int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				_ = b.Len()
				_ = b.Cap()
				_ = b.Empty()
				_ = b.Count()
				_ = b.Any()
				_ = b.None()
				_ = b.AllSet()
				_ = b.Test((gId*50 + i) % 10000)
				_, _ = b.NextSet((gId*30 + i) % 10000)
				_, _ = b.NextClear((gId*30 + i) % 10000)
				for idx := range b.Bits() {
					if idx > 100 {
						break
					}
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestConcurrentSynchronizedWriters(t *testing.T) {
	b := New(1000)
	var mu sync.RWMutex
	var wg sync.WaitGroup

	// Readers
	for r := 0; r < 16; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				mu.RLock()
				_ = b.Count()
				_ = b.Test(i % 1000)
				mu.RUnlock()
			}
		}()
	}

	// Writers
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(wId int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				mu.Lock()
				idx := (wId*100 + i) % 1000
				b.Toggle(idx)
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()
}
