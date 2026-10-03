// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitset

import (
	"iter"
	"math/bits"
)

const (
	wordShift = 6  // 2^6 = 64 bits per uint64
	wordMask  = 63 // 64 - 1
)

func bitMask(i int) uint64 {
	return uint64(1) << (i & wordMask)
}

func numWords(nbits int) int {
	if nbits <= 0 {
		return 0
	}
	return (nbits + wordMask) >> wordShift
}

// BitSet represents a dense, growable bit array backed by []uint64 words.
// Bit indices are 0-indexed. The zero value is an empty BitSet ready to use.
//
// Read-only operations are safe for concurrent use across multiple goroutines.
// Concurrent mutations or concurrent read/writes must be externally synchronized.
type BitSet struct {
	words  []uint64
	length int
}

// cleanTrailing ensures that all bits at indices >= length in the backing words are strictly 0.
func (b *BitSet) cleanTrailing() {
	nWords := numWords(b.length)
	if nWords < len(b.words) {
		clear(b.words[nWords:])
	}
	rem := b.length & wordMask
	if rem != 0 && nWords > 0 {
		b.words[nWords-1] &= (uint64(1) << rem) - 1
	}
}

func (b *BitSet) ensureWords(needed int) {
	if needed <= len(b.words) {
		return
	}
	if needed <= cap(b.words) {
		oldLen := len(b.words)
		b.words = b.words[:needed]
		clear(b.words[oldLen:needed])
		return
	}
	newCap := cap(b.words) * 2
	if newCap < needed {
		newCap = needed
	}
	newWords := make([]uint64, needed, newCap)
	copy(newWords, b.words)
	b.words = newWords
}

// New creates a BitSet with the specified bit capacity/length.
// All bits are initialized to 0. Panics if nbits < 0.
func New(nbits int) *BitSet {
	if nbits < 0 {
		panic("bitset: negative length")
	}
	n := numWords(nbits)
	var words []uint64
	if n > 0 {
		words = make([]uint64, n)
	}
	return &BitSet{
		words:  words,
		length: nbits,
	}
}

// NewFromWords constructs a BitSet directly from an existing word slice.
// Panics if nbits < 0 or if words has insufficient length for nbits.
func NewFromWords(words []uint64, nbits int) *BitSet {
	if nbits < 0 {
		panic("bitset: negative length")
	}
	needed := numWords(nbits)
	if needed > len(words) {
		panic("bitset: words slice too short")
	}
	b := &BitSet{
		words:  words,
		length: nbits,
	}
	b.cleanTrailing()
	return b
}

// FromBytes constructs a BitSet from a little-endian byte slice.
// Panics if nbits < 0.
func FromBytes(b []byte, nbits int) *BitSet {
	if nbits < 0 {
		panic("bitset: negative length")
	}
	bs := New(nbits)
	nBytes := min(len(b), (nbits+7)/8)
	for i := 0; i < nBytes; i++ {
		wIdx := i >> 3
		bOffset := (i & 7) * 8
		bs.words[wIdx] |= uint64(b[i]) << bOffset
	}
	bs.cleanTrailing()
	return bs
}

// Len returns the logical number of bits in the bitset.
func (b *BitSet) Len() int {
	if b == nil {
		return 0
	}
	return b.length
}

// Cap returns the total bit capacity currently allocated without reallocating.
func (b *BitSet) Cap() int {
	if b == nil {
		return 0
	}
	return cap(b.words) * 64
}

// Empty reports whether the bitset contains zero bits logically.
func (b *BitSet) Empty() bool {
	return b == nil || b.length == 0
}

// Count returns the number of set bits (1-bits) using hardware POPCNT.
func (b *BitSet) Count() int {
	if b == nil || b.length == 0 {
		return 0
	}
	cnt := 0
	n := numWords(b.length)
	for i := 0; i < n; i++ {
		cnt += bits.OnesCount64(b.words[i])
	}
	return cnt
}

// Any reports whether at least one bit in the bitset is set to 1.
func (b *BitSet) Any() bool {
	if b == nil || b.length == 0 {
		return false
	}
	n := numWords(b.length)
	for i := 0; i < n; i++ {
		if b.words[i] != 0 {
			return true
		}
	}
	return false
}

// None reports whether no bits in the bitset are set to 1.
func (b *BitSet) None() bool {
	return !b.Any()
}

// AllSet reports whether all bits in the bitset (0 <= i < b.Len()) are set to 1.
// If b.Len() == 0, it returns true (vacuously true).
func (b *BitSet) AllSet() bool {
	if b == nil || b.length == 0 {
		return true
	}
	fullWords := b.length >> wordShift
	for i := 0; i < fullWords; i++ {
		if b.words[i] != ^uint64(0) {
			return false
		}
	}
	rem := b.length & wordMask
	if rem != 0 {
		mask := (uint64(1) << rem) - 1
		if (b.words[fullWords] & mask) != mask {
			return false
		}
	}
	return true
}

// Equal reports whether two bitsets have the same length and identical bit values.
func (b *BitSet) Equal(other *BitSet) bool {
	if b == other {
		return true
	}
	if b == nil || other == nil {
		return false
	}
	if b.length != other.length {
		return false
	}
	n := numWords(b.length)
	for i := 0; i < n; i++ {
		if b.words[i] != other.words[i] {
			return false
		}
	}
	return true
}

// Test reports the boolean value of bit i.
// Panics if i < 0. Returns false if i >= b.Len().
func (b *BitSet) Test(i int) bool {
	if i < 0 {
		panic("bitset: index out of range")
	}
	if b == nil || i >= b.length {
		return false
	}
	return (b.words[i>>wordShift] & bitMask(i)) != 0
}

// Set sets bit i to 1 and returns b.
// Automatically expands the bitset if i >= b.Len().
// Panics if i < 0.
func (b *BitSet) Set(i int) *BitSet {
	if i < 0 {
		panic("bitset: index out of range")
	}
	if i >= b.length {
		b.ensureWords(numWords(i + 1))
		b.length = i + 1
	}
	b.words[i>>wordShift] |= bitMask(i)
	return b
}

// SetTo sets bit i to the given boolean value and returns b.
// Panics if i < 0.
func (b *BitSet) SetTo(i int, val bool) *BitSet {
	if val {
		return b.Set(i)
	}
	return b.Clear(i)
}

// Clear sets bit i to 0 and returns b.
// If i >= b.Len(), it is a no-op.
// Panics if i < 0.
func (b *BitSet) Clear(i int) *BitSet {
	if i < 0 {
		panic("bitset: index out of range")
	}
	if b == nil || i >= b.length {
		return b
	}
	b.words[i>>wordShift] &= ^bitMask(i)
	return b
}

// Toggle inverts bit i and returns b.
// If i >= b.Len(), it expands the bitset and sets bit i to 1.
// Panics if i < 0.
func (b *BitSet) Toggle(i int) *BitSet {
	if i < 0 {
		panic("bitset: index out of range")
	}
	if i >= b.length {
		b.ensureWords(numWords(i + 1))
		b.length = i + 1
		b.words[i>>wordShift] |= bitMask(i)
		return b
	}
	b.words[i>>wordShift] ^= bitMask(i)
	return b
}

// SetAll sets all bits in the bitset to 1 and returns b.
func (b *BitSet) SetAll() *BitSet {
	if b == nil || b.length == 0 {
		return b
	}
	n := numWords(b.length)
	for i := 0; i < n; i++ {
		b.words[i] = ^uint64(0)
	}
	b.cleanTrailing()
	return b
}

// ClearAll zeroes all bits in the bitset and returns b.
func (b *BitSet) ClearAll() *BitSet {
	if b == nil {
		return nil
	}
	clear(b.words)
	return b
}

// Not inverts all bits in the bitset and returns b.
func (b *BitSet) Not() *BitSet {
	if b == nil || b.length == 0 {
		return b
	}
	n := numWords(b.length)
	for i := 0; i < n; i++ {
		b.words[i] = ^b.words[i]
	}
	b.cleanTrailing()
	return b
}

// Flip inverts all bits in the bitset and returns b. Alias for Not().
func (b *BitSet) Flip() *BitSet {
	return b.Not()
}

// Clone creates a deep copy of the bitset with identical length and contents.
func (b *BitSet) Clone() *BitSet {
	if b == nil {
		return nil
	}
	n := numWords(b.length)
	var cloned []uint64
	if n > 0 {
		cloned = make([]uint64, n)
		copy(cloned, b.words[:n])
	}
	return &BitSet{
		words:  cloned,
		length: b.length,
	}
}

// NextSet returns the index of the first bit set to 1 at or after start.
// It returns (index, true) if a set bit is found, or (-1, false) if no such bit exists.
// Panics if start < 0. Returns (-1, false) if start >= b.Len().
func (b *BitSet) NextSet(start int) (int, bool) {
	if start < 0 {
		panic("bitset: index out of range")
	}
	if b == nil || start >= b.length {
		return -1, false
	}

	wIdx := start >> wordShift
	n := numWords(b.length)

	// First word: mask out bits below start & wordMask
	mask := ^uint64(0) << (start & wordMask)
	w := b.words[wIdx] & mask
	if w != 0 {
		return (wIdx << wordShift) + bits.TrailingZeros64(w), true
	}

	// Subsequent words: skip zero-words in 1 cycle
	for wIdx++; wIdx < n; wIdx++ {
		w = b.words[wIdx]
		if w != 0 {
			return (wIdx << wordShift) + bits.TrailingZeros64(w), true
		}
	}
	return -1, false
}

// NextClear returns the index of the first bit set to 0 at or after start up to b.Len() - 1.
// It returns (index, true) if a clear bit is found, or (-1, false) if no such bit exists.
// Panics if start < 0. Returns (-1, false) if start >= b.Len().
func (b *BitSet) NextClear(start int) (int, bool) {
	if start < 0 {
		panic("bitset: index out of range")
	}
	if b == nil || start >= b.length {
		return -1, false
	}

	wIdx := start >> wordShift
	n := numWords(b.length)
	rem := b.length & wordMask

	// First word: invert and mask out bits below start & wordMask
	mask := ^uint64(0) << (start & wordMask)
	if wIdx == n-1 && rem != 0 {
		mask &= (uint64(1) << rem) - 1
	}
	inv := (^b.words[wIdx]) & mask
	if inv != 0 {
		return (wIdx << wordShift) + bits.TrailingZeros64(inv), true
	}

	// Subsequent words: skip all-ones words directly in 1 cycle
	for wIdx++; wIdx < n; wIdx++ {
		inv = ^b.words[wIdx]
		if wIdx == n-1 && rem != 0 {
			inv &= (uint64(1) << rem) - 1
		}
		if inv != 0 {
			return (wIdx << wordShift) + bits.TrailingZeros64(inv), true
		}
	}
	return -1, false
}

// And computes the bitwise AND of b and other in-place on b, returning b.
// If other is nil or empty, b is cleared.
func (b *BitSet) And(other *BitSet) *BitSet {
	if b == nil {
		return nil
	}
	if other == nil || other.length == 0 {
		return b.ClearAll()
	}
	if b == other {
		return b
	}
	common := min(numWords(b.length), numWords(other.length))
	for i := 0; i < common; i++ {
		b.words[i] &= other.words[i]
	}
	n := numWords(b.length)
	if common < n {
		clear(b.words[common:n])
	}
	b.cleanTrailing()
	return b
}

// Or computes the bitwise OR of b and other in-place on b, returning b.
// If other.Len() > b.Len(), b is grown to accommodate other.Len().
func (b *BitSet) Or(other *BitSet) *BitSet {
	if b == nil {
		return nil
	}
	if other == nil || other.length == 0 || b == other {
		return b
	}
	if other.length > b.length {
		b.ensureWords(numWords(other.length))
		b.length = other.length
	}
	common := min(numWords(b.length), numWords(other.length))
	for i := 0; i < common; i++ {
		b.words[i] |= other.words[i]
	}
	b.cleanTrailing()
	return b
}

// Xor computes the bitwise XOR of b and other in-place on b, returning b.
// If other.Len() > b.Len(), b is grown to accommodate other.Len().
func (b *BitSet) Xor(other *BitSet) *BitSet {
	if b == nil {
		return nil
	}
	if other == nil || other.length == 0 {
		return b
	}
	if b == other {
		return b.ClearAll()
	}
	if other.length > b.length {
		b.ensureWords(numWords(other.length))
		b.length = other.length
	}
	common := min(numWords(b.length), numWords(other.length))
	for i := 0; i < common; i++ {
		b.words[i] ^= other.words[i]
	}
	b.cleanTrailing()
	return b
}

// AndNot computes the bitwise AND NOT (bit-clear) of b and other in-place on b, returning b.
func (b *BitSet) AndNot(other *BitSet) *BitSet {
	if b == nil {
		return nil
	}
	if other == nil || other.length == 0 {
		return b
	}
	if b == other {
		return b.ClearAll()
	}
	common := min(numWords(b.length), numWords(other.length))
	for i := 0; i < common; i++ {
		b.words[i] &^= other.words[i]
	}
	b.cleanTrailing()
	return b
}

// Bits returns an inlined, non-allocating push iterator (iter.Seq[int]) that yields
// the indices of all set bits (1-bits) in ascending order.
func (b *BitSet) Bits() iter.Seq[int] {
	return func(yield func(int) bool) {
		if b == nil || b.length == 0 {
			return
		}
		n := numWords(b.length)
		for i := 0; i < n; i++ {
			w := b.words[i]
			base := i << wordShift
			for w != 0 {
				tz := bits.TrailingZeros64(w)
				if !yield(base + tz) {
					return
				}
				w &= w - 1
			}
		}
	}
}

// Values returns an iterator yielding indices of all set bits in ascending order.
// It is an alias for Bits().
func (b *BitSet) Values() iter.Seq[int] {
	return b.Bits()
}

// All returns an iterator yielding indices of all set bits in ascending order.
// It is an alias for Bits().
func (b *BitSet) All() iter.Seq[int] {
	return b.Bits()
}

// Resize resizes the BitSet to nbits.
// If nbits < b.Len(), the bitset is truncated and trailing bits are cleared.
// If nbits > b.Len(), the bitset is grown and new bits are initialized to 0.
// Panics if nbits < 0.
func (b *BitSet) Resize(nbits int) *BitSet {
	if nbits < 0 {
		panic("bitset: negative length")
	}
	if nbits == b.length {
		return b
	}
	if nbits < b.length {
		b.length = nbits
		b.cleanTrailing()
		return b
	}
	b.ensureWords(numWords(nbits))
	b.length = nbits
	return b
}

// Grow reserves capacity for at least nbits without changing b.Len().
// Panics if nbits < 0.
func (b *BitSet) Grow(nbits int) *BitSet {
	if nbits < 0 {
		panic("bitset: negative length")
	}
	needed := numWords(nbits)
	if needed > cap(b.words) {
		newWords := make([]uint64, len(b.words), needed)
		copy(newWords, b.words)
		b.words = newWords
	}
	return b
}

// ShrinkToFit reduces the capacity of the backing word slice to fit b.Len().
func (b *BitSet) ShrinkToFit() *BitSet {
	needed := numWords(b.length)
	if cap(b.words) > needed {
		var newWords []uint64
		if needed > 0 {
			newWords = make([]uint64, needed)
			copy(newWords, b.words[:needed])
		}
		b.words = newWords
	}
	return b
}
