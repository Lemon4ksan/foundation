// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitset

import (
	"math/big"
	"math/rand/v2"
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
	var nilB *Set
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

	var zeroB Set
	require.Equal(t, 0, zeroB.Len())
	require.Equal(t, 0, zeroB.Cap())
	require.True(t, zeroB.Empty())
	require.Equal(t, 0, zeroB.Count())
	require.False(t, zeroB.Any())
	require.True(t, zeroB.None())
	require.True(t, zeroB.AllSet())

	// Setting bit on zero value Set
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
	var b1, b2 *Set
	require.True(t, b1.Equal(b2)) // both nil

	b1 = New(64)
	require.False(t, b1.Equal(nil))
	require.False(t, (*Set)(nil).Equal(b1))

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
	var nilB *Set
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

	var nilB *Set
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

	var nilB *Set
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

	// All() and Values() parity
	var allOrds, allYielded, valYielded []int
	for ord, idx := range b.All() {
		allOrds = append(allOrds, ord)
		allYielded = append(allYielded, idx)
	}
	for idx := range b.Values() {
		valYielded = append(valYielded, idx)
	}
	require.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7}, allOrds)
	require.Equal(t, indices, allYielded)
	require.Equal(t, indices, valYielded)

	// Early exit on All()
	var earlyAll []int
	for _, idx := range b.All() {
		earlyAll = append(earlyAll, idx)
		if len(earlyAll) == 2 {
			break
		}
	}
	require.Equal(t, indices[:2], earlyAll)

	// Empty bitset iterator
	bEmpty := New(0)
	for range bEmpty.Bits() {
		t.Fatal("empty bitset should yield no items")
	}
	for range bEmpty.All() {
		t.Fatal("empty bitset All() should yield no items")
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

	// Exercise newCap < needed branch with empty Set
	var bZero Set
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
	for g := range 32 {
		wg.Add(1)
		go func(gId int) {
			defer wg.Done()
			for i := range 500 {
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
	for range 16 {
		wg.Go(func() {
			for i := range 200 {
				mu.RLock()
				_ = b.Count()
				_ = b.Test(i % 1000)
				mu.RUnlock()
			}
		})
	}

	// Writers
	for w := range 8 {
		wg.Add(1)
		go func(wId int) {
			defer wg.Done()
			for i := range 100 {
				mu.Lock()
				idx := (wId*100 + i) % 1000
				b.Toggle(idx)
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()
}

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

// refBitSet is a simple, unambiguous mathematical reference implementation
// using a boolean slice and math/big.Int to verify Set behavior.
type refBitSet struct {
	bits   []bool
	length int
}

func newRef(length int) *refBitSet {
	return &refBitSet{
		bits:   make([]bool, length),
		length: length,
	}
}

func (r *refBitSet) clone() *refBitSet {
	c := &refBitSet{
		bits:   make([]bool, r.length),
		length: r.length,
	}
	copy(c.bits, r.bits)
	return c
}

func (r *refBitSet) set(i int) {
	if i >= r.length {
		newBits := make([]bool, i+1)
		copy(newBits, r.bits)
		r.bits = newBits
		r.length = i + 1
	}
	r.bits[i] = true
}

func (r *refBitSet) clear(i int) {
	if i < r.length {
		r.bits[i] = false
	}
}

func (r *refBitSet) toggle(i int) {
	if i >= r.length {
		r.set(i)
		return
	}
	r.bits[i] = !r.bits[i]
}

func (r *refBitSet) test(i int) bool {
	if i >= r.length {
		return false
	}
	return r.bits[i]
}

func (r *refBitSet) count() int {
	cnt := 0
	for i := 0; i < r.length; i++ {
		if r.bits[i] {
			cnt++
		}
	}
	return cnt
}

func (r *refBitSet) any() bool {
	for i := 0; i < r.length; i++ {
		if r.bits[i] {
			return true
		}
	}
	return false
}

func (r *refBitSet) none() bool {
	return !r.any()
}

func (r *refBitSet) allSet() bool {
	for i := 0; i < r.length; i++ {
		if !r.bits[i] {
			return false
		}
	}
	return true
}

func (r *refBitSet) setAll() {
	for i := 0; i < r.length; i++ {
		r.bits[i] = true
	}
}

func (r *refBitSet) clearAll() {
	for i := 0; i < r.length; i++ {
		r.bits[i] = false
	}
}

func (r *refBitSet) not() {
	for i := 0; i < r.length; i++ {
		r.bits[i] = !r.bits[i]
	}
}

func (r *refBitSet) nextSet(start int) (int, bool) {
	for i := start; i < r.length; i++ {
		if r.bits[i] {
			return i, true
		}
	}
	return -1, false
}

func (r *refBitSet) nextClear(start int) (int, bool) {
	for i := start; i < r.length; i++ {
		if !r.bits[i] {
			return i, true
		}
	}
	return -1, false
}

func (r *refBitSet) resize(nbits int) {
	if nbits == r.length {
		return
	}
	newBits := make([]bool, nbits)
	copy(newBits, r.bits[:min(r.length, nbits)])
	r.bits = newBits
	r.length = nbits
}

func (r *refBitSet) and(other *refBitSet) {
	if other == nil || other.length == 0 {
		r.clearAll()
		return
	}
	for i := 0; i < r.length; i++ {
		if i < other.length {
			r.bits[i] = r.bits[i] && other.bits[i]
		} else {
			r.bits[i] = false
		}
	}
}

func (r *refBitSet) or(other *refBitSet) {
	if other == nil || other.length == 0 {
		return
	}
	if other.length > r.length {
		r.resize(other.length)
	}
	for i := 0; i < other.length; i++ {
		r.bits[i] = r.bits[i] || other.bits[i]
	}
}

func (r *refBitSet) xor(other *refBitSet) {
	if other == nil || other.length == 0 {
		return
	}
	if other.length > r.length {
		r.resize(other.length)
	}
	for i := 0; i < other.length; i++ {
		r.bits[i] = r.bits[i] != other.bits[i]
	}
}

func (r *refBitSet) andNot(other *refBitSet) {
	if other == nil || other.length == 0 {
		return
	}
	for i := 0; i < min(r.length, other.length); i++ {
		if other.bits[i] {
			r.bits[i] = false
		}
	}
}

// toBigInt converts a Set to a math/big.Int for independent arbitrary-precision checks.
func toBigInt(b *Set) *big.Int {
	z := new(big.Int)
	for i := 0; i < b.Len(); i++ {
		if b.Test(i) {
			z.SetBit(z, i, 1)
		}
	}
	return z
}

func assertBitSetMatchesRef(t *testing.T, b *Set, r *refBitSet, stepDesc string) {
	t.Helper()
	require.Equalf(t, r.length, b.Len(), "%s: length mismatch", stepDesc)
	require.Equalf(t, r.count(), b.Count(), "%s: count mismatch", stepDesc)
	require.Equalf(t, r.any(), b.Any(), "%s: any mismatch", stepDesc)
	require.Equalf(t, r.none(), b.None(), "%s: none mismatch", stepDesc)
	require.Equalf(t, r.allSet(), b.AllSet(), "%s: allSet mismatch", stepDesc)

	// Sample bits check and iterators check
	var setIndices []int
	for i := 0; i < r.length; i++ {
		expected := r.test(i)
		actual := b.Test(i)
		require.Equalf(t, expected, actual, "%s: bit %d mismatch", stepDesc, i)
		if expected {
			setIndices = append(setIndices, i)
		}
	}

	// Verify iterator
	var iterIndices []int
	for idx := range b.Bits() {
		iterIndices = append(iterIndices, idx)
	}
	require.Equalf(t, setIndices, iterIndices, "%s: Bits() iterator mismatch", stepDesc)

	// Verify NextSet & NextClear at strategic probe points
	probePoints := []int{0, 1, r.length / 2, max(0, r.length-2), max(0, r.length-1)}
	for _, p := range probePoints {
		if p >= r.length {
			continue
		}
		expNSIdx, expNSOk := r.nextSet(p)
		actNSIdx, actNSOk := b.NextSet(p)
		require.Equalf(t, expNSOk, actNSOk, "%s: NextSet(%d) ok mismatch", stepDesc, p)
		require.Equalf(t, expNSIdx, actNSIdx, "%s: NextSet(%d) index mismatch", stepDesc, p)

		expNCIdx, expNCOk := r.nextClear(p)
		actNCIdx, actNCOk := b.NextClear(p)
		require.Equalf(t, expNCOk, actNCOk, "%s: NextClear(%d) ok mismatch", stepDesc, p)
		require.Equalf(t, expNCIdx, actNCIdx, "%s: NextClear(%d) index mismatch", stepDesc, p)
	}
}

// TestChallenge_DifferentialFuzz against reference boolean array and math/big.Int
func TestChallenge_DifferentialFuzz(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 9999))

	const numPool = 5
	const numSteps = 5000

	bitsets := make([]*Set, numPool)
	refs := make([]*refBitSet, numPool)

	for i := range bitsets {
		initLen := rng.IntN(200)
		bitsets[i] = New(initLen)
		refs[i] = newRef(initLen)
	}

	for range numSteps {
		idx := rng.IntN(numPool)
		bs := bitsets[idx]
		ref := refs[idx]

		op := rng.IntN(15)
		switch op {
		case 0: // Set
			bit := rng.IntN(bs.Len() + 50)
			bs.Set(bit)
			ref.set(bit)
		case 1: // Clear
			bit := rng.IntN(bs.Len() + 50)
			bs.Clear(bit)
			ref.clear(bit)
		case 2: // Toggle
			bit := rng.IntN(bs.Len() + 50)
			bs.Toggle(bit)
			ref.toggle(bit)
		case 3: // SetAll
			bs.SetAll()
			ref.setAll()
		case 4: // ClearAll
			bs.ClearAll()
			ref.clearAll()
		case 5: // Not / Flip
			bs.Not()
			ref.not()
		case 6: // Resize shrink or grow
			newLen := rng.IntN(300)
			bs.Resize(newLen)
			ref.resize(newLen)
		case 7: // Grow
			capBits := rng.IntN(500)
			bs.Grow(capBits)
		case 8: // ShrinkToFit
			bs.ShrinkToFit()
		case 9: // And with another bitset
			otherIdx := rng.IntN(numPool)
			bs.And(bitsets[otherIdx])
			ref.and(refs[otherIdx])
		case 10: // Or with another bitset
			otherIdx := rng.IntN(numPool)
			bs.Or(bitsets[otherIdx])
			ref.or(refs[otherIdx])
		case 11: // Xor with another bitset
			otherIdx := rng.IntN(numPool)
			bs.Xor(bitsets[otherIdx])
			ref.xor(refs[otherIdx])
		case 12: // AndNot with another bitset
			otherIdx := rng.IntN(numPool)
			bs.AndNot(bitsets[otherIdx])
			ref.andNot(refs[otherIdx])
		case 13: // Clone replace
			bitsets[idx] = bs.Clone()
			refs[idx] = ref.clone()
		case 14: // Self-op (And, Or, Xor, AndNot)
			subOp := rng.IntN(4)
			switch subOp {
			case 0:
				bs.And(bs)
				ref.and(ref)
			case 1:
				bs.Or(bs)
				ref.or(ref)
			case 2:
				bs.Xor(bs)
				ref.xor(ref)
			case 3:
				bs.AndNot(bs)
				ref.andNot(ref)
			}
		}

		assertBitSetMatchesRef(t, bitsets[idx], refs[idx], "Step")
	}
}

// TestChallenge_AlgebraicLaws verifies mathematical laws of boolean algebra.
func TestChallenge_AlgebraicLaws(t *testing.T) {
	rng := rand.New(rand.NewPCG(1234, 5678))

	lengths := []int{0, 1, 63, 64, 65, 127, 128, 129, 255, 256, 500}

	randomBitSet := func(n int) *Set {
		b := New(n)
		for i := range n {
			if rng.IntN(2) == 1 {
				b.Set(i)
			}
		}
		return b
	}

	for _, n := range lengths {
		for range 10 {
			a := randomBitSet(n)
			b := randomBitSet(n)
			c := randomBitSet(n)

			// 1. Involution: Not(Not(A)) == A
			inv := a.Clone().Not().Not()
			require.Truef(t, a.Equal(inv), "Involution failed for length %d", n)

			// 2. Commutativity:
			// A OR B == B OR A
			or1 := a.Clone().Or(b)
			or2 := b.Clone().Or(a)
			require.Truef(t, or1.Equal(or2), "Commutativity of OR failed for length %d", n)

			// A XOR B == B XOR A
			xor1 := a.Clone().Xor(b)
			xor2 := b.Clone().Xor(a)
			require.Truef(t, xor1.Equal(xor2), "Commutativity of XOR failed for length %d", n)

			// A AND B == B AND A (equal lengths)
			and1 := a.Clone().And(b)
			and2 := b.Clone().And(a)
			require.Truef(t, and1.Equal(and2), "Commutativity of AND failed for length %d", n)

			// 3. Associativity:
			// (A OR B) OR C == A OR (B OR C)
			assocOr1 := a.Clone().Or(b).Or(c)
			assocOr2 := b.Clone().Or(c).Or(a)
			require.Truef(t, assocOr1.Equal(assocOr2), "Associativity of OR failed for length %d", n)

			// (A AND B) AND C == A AND (B AND C)
			assocAnd1 := a.Clone().And(b).And(c)
			assocAnd2 := b.Clone().And(c).And(a)
			require.Truef(t, assocAnd1.Equal(assocAnd2), "Associativity of AND failed for length %d", n)

			// (A XOR B) XOR C == A XOR (B XOR C)
			assocXor1 := a.Clone().Xor(b).Xor(c)
			assocXor2 := b.Clone().Xor(c).Xor(a)
			require.Truef(t, assocXor1.Equal(assocXor2), "Associativity of XOR failed for length %d", n)

			// 4. De Morgan's Laws:
			// Not(A AND B) == Not(A) OR Not(B)
			deMorgan1Left := a.Clone().And(b).Not()
			deMorgan1Right := a.Clone().Not().Or(b.Clone().Not())
			require.Truef(t, deMorgan1Left.Equal(deMorgan1Right), "De Morgan 1 failed for length %d", n)

			// Not(A OR B) == Not(A) AND Not(B)
			deMorgan2Left := a.Clone().Or(b).Not()
			deMorgan2Right := a.Clone().Not().And(b.Clone().Not())
			require.Truef(t, deMorgan2Left.Equal(deMorgan2Right), "De Morgan 2 failed for length %d", n)

			// 5. Distributivity:
			// A AND (B OR C) == (A AND B) OR (A AND C)
			distrib1Left := a.Clone().And(b.Clone().Or(c))
			distrib1Right := a.Clone().And(b).Or(a.Clone().And(c))
			require.Truef(t, distrib1Left.Equal(distrib1Right), "Distributivity AND-over-OR failed for length %d", n)

			// A OR (B AND C) == (A OR B) AND (A OR C)
			distrib2Left := a.Clone().Or(b.Clone().And(c))
			distrib2Right := a.Clone().Or(b).And(a.Clone().Or(c))
			require.Truef(t, distrib2Left.Equal(distrib2Right), "Distributivity OR-over-AND failed for length %d", n)

			// 6. Idempotence:
			// A AND A == A, A OR A == A
			idempAnd := a.Clone().And(a)
			require.Truef(t, a.Equal(idempAnd), "Idempotence of AND failed for length %d", n)
			idempOr := a.Clone().Or(a)
			require.Truef(t, a.Equal(idempOr), "Idempotence of OR failed for length %d", n)

			// 7. Absorption:
			// A AND (A OR B) == A
			absorp1 := a.Clone().And(a.Clone().Or(b))
			require.Truef(t, a.Equal(absorp1), "Absorption 1 failed for length %d", n)

			// A OR (A AND B) == A
			absorp2 := a.Clone().Or(a.Clone().And(b))
			require.Truef(t, a.Equal(absorp2), "Absorption 2 failed for length %d", n)

			// 8. Complements & Zero/One:
			// A AND Not(A) == 0
			compAnd := a.Clone().And(a.Clone().Not())
			require.Equalf(t, 0, compAnd.Count(), "A AND Not(A) must have 0 set bits for length %d", n)

			// A OR Not(A) == 1
			compOr := a.Clone().Or(a.Clone().Not())
			require.Truef(t, compOr.AllSet(), "A OR Not(A) must be AllSet for length %d", n)

			// A XOR A == 0
			selfXor := a.Clone().Xor(a)
			require.Equalf(t, 0, selfXor.Count(), "A XOR A must be 0 for length %d", n)

			// A AND NOT A == 0
			selfAndNot := a.Clone().AndNot(a)
			require.Equalf(t, 0, selfAndNot.Count(), "A AND NOT A must be 0 for length %d", n)
		}
	}
}

// TestChallenge_BigIntDifferential compares Set boolean operations with math/big.Int
func TestChallenge_BigIntDifferential(t *testing.T) {
	rng := rand.New(rand.NewPCG(9876, 54321))

	sizes := []int{10, 64, 65, 128, 129, 256, 512, 1000}

	for _, sz := range sizes {
		b1 := New(sz)
		b2 := New(sz)

		for i := range sz {
			if rng.IntN(2) == 1 {
				b1.Set(i)
			}
			if rng.IntN(2) == 1 {
				b2.Set(i)
			}
		}

		big1 := toBigInt(b1)
		big2 := toBigInt(b2)

		// Test AND
		andBS := b1.Clone().And(b2)
		andBig := new(big.Int).And(big1, big2)
		require.Equalf(t, andBig.String(), toBigInt(andBS).String(), "BigInt AND mismatch at size %d", sz)

		// Test OR
		orBS := b1.Clone().Or(b2)
		orBig := new(big.Int).Or(big1, big2)
		require.Equalf(t, orBig.String(), toBigInt(orBS).String(), "BigInt OR mismatch at size %d", sz)

		// Test XOR
		xorBS := b1.Clone().Xor(b2)
		xorBig := new(big.Int).Xor(big1, big2)
		require.Equalf(t, xorBig.String(), toBigInt(xorBS).String(), "BigInt XOR mismatch at size %d", sz)

		// Test AND NOT
		andNotBS := b1.Clone().AndNot(b2)
		andNotBig := new(big.Int).AndNot(big1, big2)
		require.Equalf(t, andNotBig.String(), toBigInt(andNotBS).String(), "BigInt AND NOT mismatch at size %d", sz)

		// Test NOT
		notBS := b1.Clone().Not()
		// For big.Int, not of bitset of size sz is: ((1 << sz) - 1) XOR big1
		mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(sz)), big.NewInt(1))
		notBig := new(big.Int).Xor(big1, mask)
		require.Equalf(t, notBig.String(), toBigInt(notBS).String(), "BigInt NOT mismatch at size %d", sz)
	}
}

// TestChallenge_WordBoundaryTransitions tests bit manipulation precisely around 64-bit boundaries.
func TestChallenge_WordBoundaryTransitions(t *testing.T) {
	boundaries := []int{
		0, 1, 2,
		61, 62, 63, 64, 65, 66,
		125, 126, 127, 128, 129, 130,
		191, 192, 193,
		255, 256, 257,
	}

	for _, bIdx := range boundaries {
		b := New(bIdx + 20)

		// Test single bit set at boundary
		b.Set(bIdx)
		require.Truef(t, b.Test(bIdx), "bit %d should be set", bIdx)
		require.Equalf(t, 1, b.Count(), "count should be 1 for single bit %d", bIdx)

		// NextSet search
		idx, ok := b.NextSet(0)
		require.True(t, ok)
		require.Equal(t, bIdx, idx)

		idx, ok = b.NextSet(bIdx)
		require.True(t, ok)
		require.Equal(t, bIdx, idx)

		idx, ok = b.NextSet(bIdx + 1)
		require.False(t, ok)
		require.Equal(t, -1, idx)

		// NextClear search
		bAll := New(bIdx + 10).SetAll()
		bAll.Clear(bIdx)
		cIdx, cOk := bAll.NextClear(0)
		require.True(t, cOk)
		require.Equal(t, bIdx, cIdx)

		cIdx, cOk = bAll.NextClear(bIdx)
		require.True(t, cOk)
		require.Equal(t, bIdx, cIdx)

		cIdx, cOk = bAll.NextClear(bIdx + 1)
		require.False(t, cOk)
		require.Equal(t, -1, cIdx)
	}
}

// TestChallenge_NextClear_SkipMultipleWords specifically stress tests NextClear word skipping.
func TestChallenge_NextClear_SkipMultipleWords(t *testing.T) {
	// A bitset of 1000 bits, all set
	b := New(1000).SetAll()

	// Initially no clear bits
	idx, ok := b.NextClear(0)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	// Clear bit 500 (inside word 7)
	b.Clear(500)

	// Starting from 0 should skip words 0..6 (all ones) and land directly on 500
	idx, ok = b.NextClear(0)
	require.True(t, ok)
	require.Equal(t, 500, idx)

	// Starting from 64, 128, 256, 499 should all return 500
	for _, start := range []int{64, 128, 256, 448, 499, 500} {
		idx, ok = b.NextClear(start)
		require.True(t, ok)
		require.Equal(t, 500, idx)
	}

	// Starting from 501 should find no clear bits
	idx, ok = b.NextClear(501)
	require.False(t, ok)
	require.Equal(t, -1, idx)

	// Clear the very last bit (999)
	b.Clear(999)
	idx, ok = b.NextClear(501)
	require.True(t, ok)
	require.Equal(t, 999, idx)
}

// TestChallenge_NextSet_SkipMultipleWords specifically stress tests NextSet word skipping.
func TestChallenge_NextSet_SkipMultipleWords(t *testing.T) {
	// Empty bitset of 1000 bits
	b := New(1000)

	// Set only bit 700 (word 10)
	b.Set(700)

	// Search from 0 skips words 0..9 and lands on 700
	idx, ok := b.NextSet(0)
	require.True(t, ok)
	require.Equal(t, 700, idx)

	for _, start := range []int{0, 63, 64, 127, 128, 640, 699, 700} {
		idx, ok = b.NextSet(start)
		require.True(t, ok)
		require.Equal(t, 700, idx)
	}

	// Past 700
	idx, ok = b.NextSet(701)
	require.False(t, ok)
	require.Equal(t, -1, idx)
}

// TestChallenge_AsymmetricBitwiseOperations tests bitwise ops between bitsets of very different lengths.
func TestChallenge_AsymmetricBitwiseOperations(t *testing.T) {
	lengths := []struct {
		l1 int
		l2 int
	}{
		{0, 0},
		{0, 64},
		{64, 0},
		{10, 200},
		{200, 10},
		{63, 65},
		{65, 63},
		{127, 257},
		{257, 127},
	}

	for _, tc := range lengths {
		b1 := New(tc.l1)
		b2 := New(tc.l2)
		r1 := newRef(tc.l1)
		r2 := newRef(tc.l2)

		for i := 0; i < tc.l1; i += 2 {
			b1.Set(i)
			r1.set(i)
		}
		for i := 0; i < tc.l2; i += 3 {
			b2.Set(i)
			r2.set(i)
		}

		// AND
		andB := b1.Clone().And(b2)
		andR := r1.clone()
		andR.and(r2)
		assertBitSetMatchesRef(t, andB, andR, "Asymmetric AND")

		// OR
		orB := b1.Clone().Or(b2)
		orR := r1.clone()
		orR.or(r2)
		assertBitSetMatchesRef(t, orB, orR, "Asymmetric OR")

		// XOR
		xorB := b1.Clone().Xor(b2)
		xorR := r1.clone()
		xorR.xor(r2)
		assertBitSetMatchesRef(t, xorB, xorR, "Asymmetric XOR")

		// AND NOT
		andNotB := b1.Clone().AndNot(b2)
		andNotR := r1.clone()
		andNotR.andNot(r2)
		assertBitSetMatchesRef(t, andNotB, andNotR, "Asymmetric AND NOT")
	}
}

// TestChallenge_FullTruthTableSmall tests all 2^(2*N) combinations for small N to prove exhaustive boolean truth.
func TestChallenge_FullTruthTableSmall(t *testing.T) {
	const n = 8
	const total = 1 << n // 256 combinations

	for i := range total {
		for j := range total {
			bA := New(n)
			bB := New(n)
			for bit := range n {
				if (i & (1 << bit)) != 0 {
					bA.Set(bit)
				}
				if (j & (1 << bit)) != 0 {
					bB.Set(bit)
				}
			}

			// Verify AND truth table
			bAnd := bA.Clone().And(bB)
			for bit := range n {
				expected := (i&(1<<bit) != 0) && (j&(1<<bit) != 0)
				require.Equalf(t, expected, bAnd.Test(bit), "Truth table AND bit %d for (%d, %d)", bit, i, j)
			}

			// Verify OR truth table
			bOr := bA.Clone().Or(bB)
			for bit := range n {
				expected := (i&(1<<bit) != 0) || (j&(1<<bit) != 0)
				require.Equalf(t, expected, bOr.Test(bit), "Truth table OR bit %d for (%d, %d)", bit, i, j)
			}

			// Verify XOR truth table
			bXor := bA.Clone().Xor(bB)
			for bit := range n {
				expected := (i&(1<<bit) != 0) != (j&(1<<bit) != 0)
				require.Equalf(t, expected, bXor.Test(bit), "Truth table XOR bit %d for (%d, %d)", bit, i, j)
			}

			// Verify AND NOT truth table
			bAndNot := bA.Clone().AndNot(bB)
			for bit := range n {
				expected := (i&(1<<bit) != 0) && !(j&(1<<bit) != 0)
				require.Equalf(t, expected, bAndNot.Test(bit), "Truth table AND NOT bit %d for (%d, %d)", bit, i, j)
			}
		}
	}
}

// Reference oracle using a simple bool slice to verify Set behavior bit-by-bit.
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
		// Test Set of size bIdx + 1
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

		// Auto-expansion across boundary: Set initialized to bIdx, set at bIdx
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
		for i := range len1 {
			if (i*7+3)%5 == 0 {
				b1.Set(i)
				o1.set(i)
			}
		}
		for i := range len2 {
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
			for i := range len1 {
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
			for i := range expectedLen {
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
			for i := range expectedLen {
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
			for i := range len1 {
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

	for step := range 1000 {
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
				for range 20 {
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
	for r := range 50 {
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
	for w := range 10 {
		writersWg.Add(1)
		go func(writerId int) {
			defer writersWg.Done()
			for iter := range 100 {
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
