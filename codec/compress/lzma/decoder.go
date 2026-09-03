// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lzma

import (
	"encoding/binary"
	"errors"
	"io"
	"math/bits"
	"unsafe"
)

// ErrCorruptStream indicates invalid or corrupt LZMA compressed payload.
var ErrCorruptStream = errors.New("lzma: corrupt compressed stream")

const (
	kOutBufSize = 64 * 1024
)

// DecoderCore executes the LZMA1 / LZMA2 decoding algorithm.
type DecoderCore struct {
	lc uint
	lp uint
	pb uint

	posMask uint32

	state int
	reps  [4]uint32

	isMatch     [kNumStates][kNumPosStatesMax]uint16
	isRep       [kNumStates]uint16
	isRepG0     [kNumStates]uint16
	isRepG1     [kNumStates]uint16
	isRepG2     [kNumStates]uint16
	isRep0Long  [kNumStates][kNumPosStatesMax]uint16
	posSlot     [kNumLenToPosStates][1 << (kNumPosSlotBits + 1)]uint16
	posDecoders [10][1 << 6]uint16
	posAlign    [1 << (kNumAlignBits + 1)]uint16

	lenDecoder    LenDecoder
	repLenDecoder LenDecoder
	litProbs      [][0x300]uint16

	dictSize uint32
	win      []byte
	winPos   uint32
	winFull  bool

	outBuf    [kOutBufSize]byte
	outBufPos int

	uncompressedSize uint64
	dictMask         uint32
	lpMask           uint32
	prevShift        uint
}

// NewDecoderCore constructs a DecoderCore instance.
func NewDecoderCore(lc, lp, pb uint, dictSize uint32, uncompressedSize uint64) *DecoderCore {
	if dictSize < 4096 {
		dictSize = 4096
	}
	// Round up window allocation to nearest power of 2 for safe bitmask operations
	allocSize := max(uint32(1)<<bits.Len32(dictSize-1), 4096)
	d := &DecoderCore{
		lc:               lc,
		lp:               lp,
		pb:               pb,
		posMask:          (1 << pb) - 1,
		lpMask:           (1 << lp) - 1,
		prevShift:        8 - lc,
		dictSize:         dictSize,
		dictMask:         allocSize - 1,
		win:              make([]byte, allocSize),
		uncompressedSize: uncompressedSize,
	}
	d.InitProbs()
	return d
}

// ResetDict clears the dictionary history.
func (d *DecoderCore) ResetDict() {
	d.winPos = 0
	d.winFull = false
}

// ResetStateKeepDict resets the probability models and coder state without clearing dictionary history.
func (d *DecoderCore) ResetStateKeepDict() {
	d.state = 0
	d.reps = [4]uint32{0, 0, 0, 0}
	d.outBufPos = 0

	for i := range d.isMatch {
		for j := range d.isMatch[i] {
			d.isMatch[i][j] = probInitVal
		}
	}
	for i := range d.isRep {
		d.isRep[i] = probInitVal
		d.isRepG0[i] = probInitVal
		d.isRepG1[i] = probInitVal
		d.isRepG2[i] = probInitVal
	}
	for i := range d.isRep0Long {
		for j := range d.isRep0Long[i] {
			d.isRep0Long[i][j] = probInitVal
		}
	}
	for i := range d.posSlot {
		for j := range d.posSlot[i] {
			d.posSlot[i][j] = probInitVal
		}
	}
	for i := range d.posDecoders {
		for j := range d.posDecoders[i] {
			d.posDecoders[i][j] = probInitVal
		}
	}
	for i := range d.posAlign {
		d.posAlign[i] = probInitVal
	}

	d.lenDecoder.Init()
	d.repLenDecoder.Init()

	numLit := 1 << (d.lc + d.lp)
	if len(d.litProbs) != numLit {
		d.litProbs = make([][0x300]uint16, numLit)
	}
	for i := range d.litProbs {
		for j := range d.litProbs[i] {
			d.litProbs[i][j] = probInitVal
		}
	}
}

// InitProbs resets both dictionary history and probability models.
func (d *DecoderCore) InitProbs() {
	d.ResetDict()
	d.ResetStateKeepDict()
}

func (d *DecoderCore) getLitSubCoder(pos uint32, prevByte byte) *[0x300]uint16 {
	idx := ((pos & d.lpMask) << d.lc) + (uint32(prevByte) >> d.prevShift)
	return &d.litProbs[idx]
}

func (d *DecoderCore) putByteFast(b byte, dest io.Writer) error {
	d.win[d.winPos] = b
	d.winPos = (d.winPos + 1) & d.dictMask
	d.outBuf[d.outBufPos] = b
	d.outBufPos++
	if d.outBufPos >= kOutBufSize {
		_, err := dest.Write(d.outBuf[:])
		d.outBufPos = 0
		return err
	}
	return nil
}

func (d *DecoderCore) flushOutBuf(dest io.Writer) error {
	if d.outBufPos > 0 {
		_, err := dest.Write(d.outBuf[:d.outBufPos])
		d.outBufPos = 0
		return err
	}
	return nil
}

func (d *DecoderCore) getByte(dist uint32) byte {
	return d.win[(d.winPos-dist-1)&d.dictMask]
}

// DecodeStream decodes raw LZMA payload from rd to dest.
func (d *DecoderCore) DecodeStream(rd *RangeDecoder, dest io.Writer, maxUnpack uint64) (uint64, error) {
	var totalWritten uint64
	var prevByte byte

	defer func() {
		_ = d.flushOutBuf(dest)
	}()

	for maxUnpack == ^uint64(0) || totalWritten < maxUnpack {
		posState := int(uint32(totalWritten) & d.posMask)

		bit, err := rd.DecodeBit(&d.isMatch[d.state][posState])
		if err != nil {
			return totalWritten, err
		}

		if bit == 0 {
			// Literal
			probs := d.getLitSubCoder(uint32(totalWritten), prevByte)
			var symbol uint32 = 1

			if !stateIsChar(d.state) {
				matchByte := d.getByte(d.reps[0])
				for symbol < 0x100 {
					matchBit := int((matchByte >> 7) & 1)
					matchByte <<= 1
					probIdx := uint32(0x100 + (matchBit << 8) + int(symbol))
					b, err := rd.DecodeBit(&probs[probIdx])
					if err != nil {
						return totalWritten, err
					}
					symbol = (symbol << 1) | uint32(b)
					if matchBit != b {
						for symbol < 0x100 {
							b, err := rd.DecodeBit(&probs[symbol])
							if err != nil {
								return totalWritten, err
							}
							symbol = (symbol << 1) | uint32(b)
						}
						break
					}
				}
			} else {
				for symbol < 0x100 {
					b, err := rd.DecodeBit(&probs[symbol])
					if err != nil {
						return totalWritten, err
					}
					symbol = (symbol << 1) | uint32(b)
				}
			}

			prevByte = byte(symbol)
			if err := d.putByteFast(prevByte, dest); err != nil {
				return totalWritten, err
			}
			d.state = stateUpdateLiteral(d.state)
			totalWritten++
			continue
		}

		// Match or Repetition
		var length int
		bit, err = rd.DecodeBit(&d.isRep[d.state])
		if err != nil {
			return totalWritten, err
		}

		if bit == 1 {
			// Repetition
			bit, err = rd.DecodeBit(&d.isRepG0[d.state])
			if err != nil {
				return totalWritten, err
			}

			var dist uint32
			if bit == 0 {
				dist = d.reps[0]
				bit, err = rd.DecodeBit(&d.isRep0Long[d.state][posState])
				if err != nil {
					return totalWritten, err
				}
				if bit == 0 {
					d.state = stateUpdateShortRep(d.state)
					prevByte = d.getByte(dist)
					if err := d.putByteFast(prevByte, dest); err != nil {
						return totalWritten, err
					}
					totalWritten++
					continue
				}
			} else {
				bit, err = rd.DecodeBit(&d.isRepG1[d.state])
				if err != nil {
					return totalWritten, err
				}
				if bit == 0 {
					dist = d.reps[1]
				} else {
					bit, err = rd.DecodeBit(&d.isRepG2[d.state])
					if err != nil {
						return totalWritten, err
					}
					if bit == 0 {
						dist = d.reps[2]
					} else {
						dist = d.reps[3]
						d.reps[3] = d.reps[2]
					}
					d.reps[2] = d.reps[1]
				}
				d.reps[1] = d.reps[0]
				d.reps[0] = dist
			}

			lenVal, err := d.repLenDecoder.Decode(rd, posState)
			if err != nil {
				return totalWritten, err
			}
			length = lenVal + kMatchMinLen
			d.state = stateUpdateRep(d.state)
		} else {
			// Normal match
			d.reps[3] = d.reps[2]
			d.reps[2] = d.reps[1]
			d.reps[1] = d.reps[0]

			lenVal, err := d.lenDecoder.Decode(rd, posState)
			if err != nil {
				return totalWritten, err
			}
			length = lenVal + kMatchMinLen
			d.state = stateUpdateMatch(d.state)

			lenToPos := length - kMatchMinLen
			if lenToPos >= kNumLenToPosStates {
				lenToPos = kNumLenToPosStates - 1
			}

			posSlotVal, err := rd.DecodeTree(d.posSlot[lenToPos][:], kNumPosSlotBits)
			if err != nil {
				return totalWritten, err
			}

			if posSlotVal >= 4 {
				numDirectBits := int((posSlotVal >> 1) - 1)
				dist := (2 | (posSlotVal & 1)) << numDirectBits

				if posSlotVal < 14 {
					subProbs := d.posDecoders[posSlotVal-4][:]
					revDist, err := rd.DecodeReverseTree(subProbs, numDirectBits)
					if err != nil {
						return totalWritten, err
					}
					dist += revDist
				} else {
					directBits, err := rd.DecodeDirectBits(numDirectBits - kNumAlignBits)
					if err != nil {
						return totalWritten, err
					}
					dist += directBits << kNumAlignBits
					alignBits, err := rd.DecodeReverseTree(d.posAlign[:], kNumAlignBits)
					if err != nil {
						return totalWritten, err
					}
					dist += alignBits
				}
				d.reps[0] = dist
			} else {
				d.reps[0] = posSlotVal
			}

			if d.reps[0] == 0xFFFFFFFF {
				// End of stream marker
				break
			}
		}

		dist := d.reps[0]

		// Fast Path: Check if 8-byte chunk copy is possible
		if dist >= 8 && length >= 8 && d.winPos > dist && d.winPos+uint32(length) < d.dictSize &&
			d.outBufPos+length < kOutBufSize {
			srcPos := d.winPos - dist - 1
			rem := length
			for rem >= 8 {
				v := binary.LittleEndian.Uint64(d.win[srcPos:])
				binary.LittleEndian.PutUint64(d.win[d.winPos:], v)
				binary.LittleEndian.PutUint64(d.outBuf[d.outBufPos:], v)
				srcPos += 8
				d.winPos += 8
				d.outBufPos += 8
				rem -= 8
			}
			for rem > 0 {
				b := d.win[srcPos]
				d.win[d.winPos] = b
				d.outBuf[d.outBufPos] = b
				srcPos++
				d.winPos++
				d.outBufPos++
				rem--
			}
			prevByte = d.win[d.winPos-1]
			totalWritten += uint64(length)
		} else {
			for i := 0; i < length; i++ {
				prevByte = d.getByte(dist)
				if err := d.putByteFast(prevByte, dest); err != nil {
					return totalWritten, err
				}
				totalWritten++
			}
		}
	}

	return totalWritten, nil
}

type fastDecoder struct {
	rd     *RangeDecoder
	range_ uint32
	code   uint32
	pos    int
	limit  int
	buf    unsafe.Pointer
	err    error
}

func (fd *fastDecoder) decodeBit(prob *uint16) int {
	p := uint32(*prob)
	bound := (fd.range_ >> kNumBitModelTotalBits) * p
	if fd.code < bound {
		fd.range_ = bound
		*prob += uint16((kBitModelTotal - p) >> kNumMoveBits)
		if fd.range_ < kTopValue {
			if fd.pos < fd.limit {
				b := *(*byte)(unsafe.Add(fd.buf, fd.pos))
				fd.pos++
				fd.range_ <<= 8
				fd.code = (fd.code << 8) | uint32(b)
			} else {
				fd.normalize()
			}
		}
		return 0
	}
	fd.range_ -= bound
	fd.code -= bound
	*prob -= uint16(p >> kNumMoveBits)
	if fd.range_ < kTopValue {
		if fd.pos < fd.limit {
			b := *(*byte)(unsafe.Add(fd.buf, fd.pos))
			fd.pos++
			fd.range_ <<= 8
			fd.code = (fd.code << 8) | uint32(b)
		} else {
			fd.normalize()
		}
	}
	return 1
}

func (fd *fastDecoder) normalize() {
	fd.rd.range_ = fd.range_
	fd.rd.code = fd.code
	fd.rd.pos = fd.pos
	if err := fd.rd.normalizeSlow(); err != nil {
		fd.err = err
		return
	}
	fd.range_ = fd.rd.range_
	fd.code = fd.rd.code
	fd.pos = fd.rd.pos
	fd.limit = fd.rd.limit
}

func (fd *fastDecoder) decodeDirectBits(numBits int) uint32 {
	var val uint32
	for i := 0; i < numBits; i++ {
		fd.range_ >>= 1
		if fd.code >= fd.range_ {
			fd.code -= fd.range_
			val = (val << 1) | 1
		} else {
			val <<= 1
		}
		if fd.range_ < kTopValue {
			if fd.pos < fd.limit {
				b := *(*byte)(unsafe.Add(fd.buf, fd.pos))
				fd.pos++
				fd.range_ <<= 8
				fd.code = (fd.code << 8) | uint32(b)
			} else {
				fd.normalize()
			}
		}
	}
	return val
}

func (fd *fastDecoder) decodeTree(probs []uint16, numBits int) uint32 {
	var m uint32 = 1
	for i := 0; i < numBits; i++ {
		bit := fd.decodeBit(&probs[m])
		m = (m << 1) | uint32(bit)
	}
	return m - (1 << numBits)
}

func (fd *fastDecoder) decodeReverseTree(probs []uint16, numBits int) uint32 {
	var m uint32 = 1
	var symbol uint32
	for i := 0; i < numBits; i++ {
		bit := fd.decodeBit(&probs[m])
		m = (m << 1) | uint32(bit)
		symbol |= uint32(bit) << i
	}
	return symbol
}

func (fd *fastDecoder) decodeLen(ld *LenDecoder, posState int) int {
	if fd.decodeBit(&ld.choice) == 0 {
		probs := ld.low[posState][:]
		var m uint32 = 1
		m = (m << 1) | uint32(fd.decodeBit(&probs[m]))
		m = (m << 1) | uint32(fd.decodeBit(&probs[m]))
		m = (m << 1) | uint32(fd.decodeBit(&probs[m]))
		return int(m - 8)
	}
	if fd.decodeBit(&ld.choice2) == 0 {
		probs := ld.mid[posState][:]
		var m uint32 = 1
		m = (m << 1) | uint32(fd.decodeBit(&probs[m]))
		m = (m << 1) | uint32(fd.decodeBit(&probs[m]))
		m = (m << 1) | uint32(fd.decodeBit(&probs[m]))
		return int(m-8) + kLenNumLowSymbols
	}
	probs := ld.high[:]
	var m uint32 = 1
	for i := 0; i < kLenNumHighBits; i++ {
		m = (m << 1) | uint32(fd.decodeBit(&probs[m]))
	}
	return int(m-(1<<kLenNumHighBits)) + kLenNumLowSymbols + kLenNumMidSymbols
}

// DecodeToSlice directly decodes LZMA payload into a destination byte slice with local register pinning.
func (d *DecoderCore) DecodeToSlice(rd *RangeDecoder, dest []byte, maxUnpack uint64) (int, error) {
	var totalWritten int
	var prevByte byte
	maxLen := min(int(maxUnpack), len(dest))
	buf := rd.buf[:]
	win := d.win
	winPos := d.winPos
	dictMask := d.dictMask
	state := d.state
	reps := d.reps
	posMask := d.posMask
	lpMask := d.lpMask
	lc := d.lc
	prevShift := d.prevShift

	var winPtr, destPtr, bufPtr unsafe.Pointer
	if len(win) > 0 {
		winPtr = unsafe.Pointer(&win[0])
	}
	if len(dest) > 0 {
		destPtr = unsafe.Pointer(&dest[0])
	}
	if len(buf) > 0 {
		bufPtr = unsafe.Pointer(&buf[0])
	}

	if winPos > 0 && winPtr != nil && prevByte == 0 {
		prevByte = *(*byte)(unsafe.Add(winPtr, (winPos-1)&dictMask))
	}

	fd := fastDecoder{
		rd:     rd,
		range_: rd.range_,
		code:   rd.code,
		pos:    rd.pos,
		limit:  rd.limit,
		buf:    bufPtr,
	}

	for totalWritten < maxLen {
		if fd.err != nil {
			break
		}
		// posState isolates the lowest pb bits of uncompressed stream offset.
		// This selects the probability sub-model aligned with 2/4/8 byte struct boundaries.
		posState := int(uint32(totalWritten) & posMask)

		// isMatch determines whether the next token is a Literal (0) or a Match/Repetition (1).
		bit := fd.decodeBit(&d.isMatch[state][posState])

		if bit == 0 {
			// Literal decoding pathway:
			// Select literal sub-coder using the high lc bits of the previous byte and low lp bits of position.
			litIdx := (((winPos + uint32(totalWritten)) & lpMask) << lc) + (uint32(prevByte) >> prevShift)
			probs := &d.litProbs[litIdx]
			var symbol uint32 = 1

			if !stateIsChar(state) {
				// Matched Literal mode: The previous operation was a match that ended right before this byte.
				// Therefore, the byte at the previous match distance (matchByte) is the most likely candidate.
				// We compare decoded bits against matchByte bits. While they match, we use the specialized
				// probability branch 0x100 + (matchBit << 8). Once a bit differs, we fall back to standard tree.
				matchByte := *(*byte)(unsafe.Add(winPtr, ((winPos - reps[0] - 1) & dictMask)))
				for {
					matchBit := int((matchByte >> 7) & 1)
					matchByte <<= 1
					probIdx := uint32(0x100 + (matchBit << 8) + int(symbol))
					b := fd.decodeBit(&probs[probIdx])
					symbol = (symbol << 1) | uint32(b)
					if matchBit != b {
						for symbol < 0x100 {
							symbol = (symbol << 1) | uint32(fd.decodeBit(&probs[symbol]))
						}
						break
					}
					if symbol >= 0x100 {
						break
					}
				}
			} else {
				// Pure Literal mode: Unrolled 8-step binary tree descent without loop branching.
				// Each step shifts the accumulated symbol and resolves 1 bit using local register pinned range state.
				symbol = (symbol << 1) | uint32(fd.decodeBit(&probs[symbol]))
				symbol = (symbol << 1) | uint32(fd.decodeBit(&probs[symbol]))
				symbol = (symbol << 1) | uint32(fd.decodeBit(&probs[symbol]))
				symbol = (symbol << 1) | uint32(fd.decodeBit(&probs[symbol]))
				symbol = (symbol << 1) | uint32(fd.decodeBit(&probs[symbol]))
				symbol = (symbol << 1) | uint32(fd.decodeBit(&probs[symbol]))
				symbol = (symbol << 1) | uint32(fd.decodeBit(&probs[symbol]))
				symbol = (symbol << 1) | uint32(fd.decodeBit(&probs[symbol]))
			}

			prevByte = byte(symbol)
			*(*byte)(unsafe.Add(winPtr, winPos)) = prevByte
			winPos = (winPos + 1) & dictMask
			*(*byte)(unsafe.Add(destPtr, totalWritten)) = prevByte
			state = stateUpdateLiteral(state)
			totalWritten++
			continue
		}

		// Match or Repetition pathway:
		var length int
		bit = fd.decodeBit(&d.isRep[state])

		if bit == 1 {
			// Repetition Match: Reuses one of the last 4 observed match distances (reps[0..3]).
			bit = fd.decodeBit(&d.isRepG0[state])

			var dist uint32
			if bit == 0 {
				dist = reps[0]
				bit = fd.decodeBit(&d.isRep0Long[state][posState])
				if bit == 0 {
					// ShortRep: Exactly 1 byte repeated at distance reps[0].
					// Encoded in only ~4 bits total, bypassing length and distance parsing completely.
					state = stateUpdateShortRep(state)
					prevByte = *(*byte)(unsafe.Add(winPtr, ((winPos - dist - 1) & dictMask)))
					*(*byte)(unsafe.Add(winPtr, winPos)) = prevByte
					winPos = (winPos + 1) & dictMask
					*(*byte)(unsafe.Add(destPtr, totalWritten)) = prevByte
					totalWritten++
					continue
				}
			} else {
				// Repetition distance selection (reps[1], reps[2], or reps[3]) with LRU cycling.
				bit = fd.decodeBit(&d.isRepG1[state])
				if bit == 0 {
					dist = reps[1]
				} else {
					bit = fd.decodeBit(&d.isRepG2[state])
					if bit == 0 {
						dist = reps[2]
					} else {
						dist = reps[3]
						reps[3] = reps[2]
					}
					reps[2] = reps[1]
				}
				reps[1] = reps[0]
				reps[0] = dist
			}

			lenVal := fd.decodeLen(&d.repLenDecoder, posState)
			length = lenVal + kMatchMinLen
			state = stateUpdateRep(state)
		} else {
			// Normal match
			reps[3] = reps[2]
			reps[2] = reps[1]
			reps[1] = reps[0]

			lenVal := fd.decodeLen(&d.lenDecoder, posState)
			length = lenVal + kMatchMinLen
			state = stateUpdateMatch(state)

			lenToPos := length - kMatchMinLen
			if lenToPos >= kNumLenToPosStates {
				lenToPos = kNumLenToPosStates - 1
			}

			posSlotVal := fd.decodeTree(d.posSlot[lenToPos][:], kNumPosSlotBits)

			if posSlotVal >= 4 {
				numDirectBits := int((posSlotVal >> 1) - 1)
				dist := (2 | (posSlotVal & 1)) << numDirectBits

				if posSlotVal < 14 {
					subProbs := d.posDecoders[posSlotVal-4][:]
					revDist := fd.decodeReverseTree(subProbs, numDirectBits)
					dist += revDist
				} else {
					directBits := fd.decodeDirectBits(numDirectBits - kNumAlignBits)
					dist += directBits << kNumAlignBits
					alignBits := fd.decodeReverseTree(d.posAlign[:], kNumAlignBits)
					dist += alignBits
				}
				reps[0] = dist
			} else {
				reps[0] = posSlotVal
			}

			if reps[0] == 0xFFFFFFFF {
				// End of stream marker
				break
			}
		}

		dist := reps[0]

		switch {
		case dist >= 8 && length >= 8 && winPos > dist && winPos+uint32(length) < d.dictSize &&
			totalWritten+length <= maxLen:
			// Fast Path: Check if 8-byte chunk copy is possible
			srcPos := winPos - dist - 1
			rem := length
			for rem >= 8 {
				srcP := unsafe.Add(winPtr, srcPos)
				wDstP := unsafe.Add(winPtr, winPos)
				dDstP := unsafe.Add(destPtr, totalWritten)
				v := *(*uint64)(srcP)
				*(*uint64)(wDstP) = v
				*(*uint64)(dDstP) = v
				srcPos += 8
				winPos += 8
				totalWritten += 8
				rem -= 8
			}
			for rem > 0 {
				b := *(*byte)(unsafe.Add(winPtr, srcPos))
				*(*byte)(unsafe.Add(winPtr, winPos)) = b
				*(*byte)(unsafe.Add(destPtr, totalWritten)) = b
				srcPos++
				winPos++
				totalWritten++
				rem--
			}
			prevByte = *(*byte)(unsafe.Add(winPtr, winPos-1))

		case dist == 0 && length >= 4 && winPos+uint32(length) < d.dictSize &&
			totalWritten+length <= maxLen:
			// RLE Fast Path: single-byte run expanded via 64-bit register store
			b := *(*byte)(unsafe.Add(winPtr, (winPos-1)&dictMask))
			pattern := uint64(b) * 0x0101010101010101
			rem := length
			for rem >= 8 {
				wDstP := unsafe.Add(winPtr, winPos)
				dDstP := unsafe.Add(destPtr, totalWritten)
				*(*uint64)(wDstP) = pattern
				*(*uint64)(dDstP) = pattern
				winPos += 8
				totalWritten += 8
				rem -= 8
			}
			for rem > 0 {
				*(*byte)(unsafe.Add(winPtr, winPos)) = b
				*(*byte)(unsafe.Add(destPtr, totalWritten)) = b
				winPos++
				totalWritten++
				rem--
			}
			prevByte = b

		default:
			for i := 0; i < length; i++ {
				prevByte = *(*byte)(unsafe.Add(winPtr, ((winPos - dist - 1) & dictMask)))
				*(*byte)(unsafe.Add(winPtr, winPos)) = prevByte
				winPos = (winPos + 1) & dictMask
				*(*byte)(unsafe.Add(destPtr, totalWritten)) = prevByte
				totalWritten++
			}
		}
	}

	// Write back local register state to structs
	rd.range_ = fd.range_
	rd.code = fd.code
	rd.pos = fd.pos
	d.winPos = winPos
	d.state = state
	d.reps = reps

	if fd.err != nil {
		return totalWritten, fd.err
	}
	return totalWritten, nil
}
