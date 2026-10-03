// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitset

import (
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/lemon4ksan/foundation/testing/require"
)

// refBitSet is a simple, unambiguous mathematical reference implementation
// using a boolean slice and math/big.Int to verify BitSet behavior.
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

// toBigInt converts a BitSet to a math/big.Int for independent arbitrary-precision checks.
func toBigInt(b *BitSet) *big.Int {
	z := new(big.Int)
	for i := 0; i < b.Len(); i++ {
		if b.Test(i) {
			z.SetBit(z, i, 1)
		}
	}
	return z
}

func assertBitSetMatchesRef(t *testing.T, b *BitSet, r *refBitSet, stepDesc string) {
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

	bitsets := make([]*BitSet, numPool)
	refs := make([]*refBitSet, numPool)

	for i := range bitsets {
		initLen := rng.IntN(200)
		bitsets[i] = New(initLen)
		refs[i] = newRef(initLen)
	}

	for step := 0; step < numSteps; step++ {
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

	randomBitSet := func(n int) *BitSet {
		b := New(n)
		for i := 0; i < n; i++ {
			if rng.IntN(2) == 1 {
				b.Set(i)
			}
		}
		return b
	}

	for _, n := range lengths {
		for trial := 0; trial < 10; trial++ {
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

// TestChallenge_BigIntDifferential compares BitSet boolean operations with math/big.Int
func TestChallenge_BigIntDifferential(t *testing.T) {
	rng := rand.New(rand.NewPCG(9876, 54321))

	sizes := []int{10, 64, 65, 128, 129, 256, 512, 1000}

	for _, sz := range sizes {
		b1 := New(sz)
		b2 := New(sz)

		for i := 0; i < sz; i++ {
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

	for i := 0; i < total; i++ {
		for j := 0; j < total; j++ {
			bA := New(n)
			bB := New(n)
			for bit := 0; bit < n; bit++ {
				if (i & (1 << bit)) != 0 {
					bA.Set(bit)
				}
				if (j & (1 << bit)) != 0 {
					bB.Set(bit)
				}
			}

			// Verify AND truth table
			bAnd := bA.Clone().And(bB)
			for bit := 0; bit < n; bit++ {
				expected := (i&(1<<bit) != 0) && (j&(1<<bit) != 0)
				require.Equalf(t, expected, bAnd.Test(bit), "Truth table AND bit %d for (%d, %d)", bit, i, j)
			}

			// Verify OR truth table
			bOr := bA.Clone().Or(bB)
			for bit := 0; bit < n; bit++ {
				expected := (i&(1<<bit) != 0) || (j&(1<<bit) != 0)
				require.Equalf(t, expected, bOr.Test(bit), "Truth table OR bit %d for (%d, %d)", bit, i, j)
			}

			// Verify XOR truth table
			bXor := bA.Clone().Xor(bB)
			for bit := 0; bit < n; bit++ {
				expected := (i&(1<<bit) != 0) != (j&(1<<bit) != 0)
				require.Equalf(t, expected, bXor.Test(bit), "Truth table XOR bit %d for (%d, %d)", bit, i, j)
			}

			// Verify AND NOT truth table
			bAndNot := bA.Clone().AndNot(bB)
			for bit := 0; bit < n; bit++ {
				expected := (i&(1<<bit) != 0) && !(j&(1<<bit) != 0)
				require.Equalf(t, expected, bAndNot.Test(bit), "Truth table AND NOT bit %d for (%d, %d)", bit, i, j)
			}
		}
	}
}
