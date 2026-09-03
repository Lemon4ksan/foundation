package lzma

import (
	"encoding/binary"
	"math/bits"
)

// matchCandidate represents a match or repetition candidate found in the dictionary history.
type matchCandidate struct {
	length  int
	dist    int // >= 0 for normal match (dist-1), < 0 for repetition match (-1 - repIdx)
	savings int // net entropy bit savings over literals
}

// matchSavings calculates the entropy savings in bits of a match over literal encoding,
// subtracting the bit cost of encoding distance and position slots.
func matchSavings(length, dist int, isRep bool, repIdx int) int {
	if length < kMatchMinLen {
		return 0
	}
	if isRep {
		// Repetition distances cost only 2-6 bits in LZMA
		cost := 2 + repIdx*2
		return (length * 8) - cost
	}
	// Distance slot costs in LZMA
	var distCost int
	switch {
	case dist < 64:
		distCost = 8
	case dist < 2048:
		distCost = 12
	case dist < 32768:
		distCost = 16
	case dist < 524288:
		distCost = 20
	default:
		distCost = 24
	}
	// 2-byte normal match only profitable if distance < 64
	if length == 2 && dist >= 64 {
		return 0
	}
	// 3-byte normal match only profitable if distance < 32768
	if length == 3 && dist >= 32768 {
		return 0
	}
	return (length * 8) - distCost
}

// findMatchLength determines the length of matching bytes between a and b using a 32-byte SWAR pipeline.
func findMatchLength(a, b []byte) int {
	maxLen := min(len(a), len(b))
	if maxLen > kMatchMaxLen {
		maxLen = kMatchMaxLen
	}
	var i int
	for i+32 <= maxLen {
		diff0 := binary.LittleEndian.Uint64(a[i:]) ^ binary.LittleEndian.Uint64(b[i:])
		if diff0 != 0 {
			return i + (bits.TrailingZeros64(diff0) >> 3)
		}
		diff1 := binary.LittleEndian.Uint64(a[i+8:]) ^ binary.LittleEndian.Uint64(b[i+8:])
		if diff1 != 0 {
			return i + 8 + (bits.TrailingZeros64(diff1) >> 3)
		}
		diff2 := binary.LittleEndian.Uint64(a[i+16:]) ^ binary.LittleEndian.Uint64(b[i+16:])
		if diff2 != 0 {
			return i + 16 + (bits.TrailingZeros64(diff2) >> 3)
		}
		diff3 := binary.LittleEndian.Uint64(a[i+24:]) ^ binary.LittleEndian.Uint64(b[i+24:])
		if diff3 != 0 {
			return i + 24 + (bits.TrailingZeros64(diff3) >> 3)
		}
		i += 32
	}
	for i+8 <= maxLen {
		diff := binary.LittleEndian.Uint64(a[i:]) ^ binary.LittleEndian.Uint64(b[i:])
		if diff != 0 {
			return i + (bits.TrailingZeros64(diff) >> 3)
		}
		i += 8
	}
	for i < maxLen && a[i] == b[i] {
		i++
	}
	return i
}

// findBestMatch searches for the highest entropy savings candidate across reps, 2-byte mini table, and hash chains.
func (e *EncoderCore) findBestMatch(src []byte, pos, endPos int) matchCandidate {
	var best matchCandidate
	dictSize := e.dictSize
	dictMask := e.dictMask
	chainLimit := e.chainLimit
	if chainLimit <= 0 {
		chainLimit = 16
	}
	fastBytesLimit := e.fastBytesLimit
	if fastBytesLimit <= 0 {
		fastBytesLimit = 32
	}

	// 1. Repetition matches
	for repIdx, dist := range e.reps {
		d := int(dist) + 1
		if pos >= d {
			matchLen := findMatchLength(src[pos:endPos], src[pos-d:])
			if matchLen >= kMatchMinLen {
				savings := matchSavings(matchLen, 0, true, repIdx)
				if savings > best.savings || (savings == best.savings && matchLen > best.length) {
					best.length = matchLen
					best.dist = -1 - repIdx
					best.savings = savings
				}
			}
		}
	}

	// 2. Dual-Hash 2-Byte Mini Table
	if pos+2 <= endPos {
		u16 := binary.LittleEndian.Uint16(src[pos:])
		m2 := e.head2[u16]
		e.head2[u16] = uint32(pos)
		if m2 != 0xFFFFFFFF && uint32(pos)-m2 < 64 {
			m2Pos := int(m2)
			if src[pos] == src[m2Pos] && src[pos+1] == src[m2Pos+1] {
				d := pos - m2Pos
				savings := matchSavings(2, d, false, 0)
				if savings > best.savings {
					best.length = 2
					best.dist = d - 1
					best.savings = savings
				}
			}
		}
	}

	// 3. 4-Byte Hash Chain Dictionary Search
	if pos+4 <= endPos {
		u32 := binary.LittleEndian.Uint32(src[pos:])
		h := (u32 * 0x1E35A7BD) >> e.headShift
		matchPos := e.head[h]
		e.head[h] = uint32(pos)
		e.prev[uint32(pos)&dictMask] = matchPos

		chainLen := 0
		for matchPos != 0xFFFFFFFF && uint32(pos)-matchPos < dictSize && chainLen < chainLimit {
			d := int(uint32(pos) - matchPos)
			mPos := int(matchPos)
			if best.length >= 4 {
				if src[pos+best.length-1] != src[mPos+best.length-1] || src[pos] != src[mPos] {
					matchPos = e.prev[matchPos&dictMask]
					chainLen++
					continue
				}
			} else {
				if binary.LittleEndian.Uint32(src[pos:]) != binary.LittleEndian.Uint32(src[mPos:]) {
					matchPos = e.prev[matchPos&dictMask]
					chainLen++
					continue
				}
			}
			matchLen := findMatchLength(src[pos:endPos], src[mPos:])
			if matchLen >= 4 {
				savings := matchSavings(matchLen, d, false, 0)
				if savings > best.savings || (savings == best.savings && matchLen > best.length) {
					best.length = matchLen
					best.dist = d - 1
					best.savings = savings
					if best.length >= fastBytesLimit {
						break
					}
				}
			}
			matchPos = e.prev[matchPos&dictMask]
			chainLen++
		}
	} else if pos+kMatchMinLen <= endPos {
		h := (int(src[pos]) | (int(src[pos+1]) << 8)) & 0xFFFF
		matchPos := e.head[h]
		e.head[h] = uint32(pos)
		e.prev[uint32(pos)&dictMask] = matchPos

		chainLen := 0
		for matchPos != 0xFFFFFFFF && uint32(pos)-matchPos < dictSize && chainLen < chainLimit {
			d := int(uint32(pos) - matchPos)
			mPos := int(matchPos)
			matchLen := findMatchLength(src[pos:endPos], src[mPos:])
			if matchLen >= kMatchMinLen {
				savings := matchSavings(matchLen, d, false, 0)
				if savings > best.savings || (savings == best.savings && matchLen > best.length) {
					best.length = matchLen
					best.dist = d - 1
					best.savings = savings
					if best.length >= fastBytesLimit {
						break
					}
				}
			}
			matchPos = e.prev[matchPos&dictMask]
			chainLen++
		}
	}

	return best
}
