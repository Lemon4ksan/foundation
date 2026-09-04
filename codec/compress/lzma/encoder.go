// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lzma

import (
	"encoding/binary"
	"io"
	"math/bits"
	"unsafe"
)

// EncoderCore implements LZMA1 / LZMA2 stream compression.
type EncoderCore struct {
	lc uint
	lp uint
	pb uint

	posMask    uint32
	lpMask     uint32
	prevShift  uint
	dictMask   uint32
	isPowerOf2 bool

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

	dictSize  uint32
	head      []uint32
	headMask  uint32
	headShift uint
	head2     [1 << 16]uint32
	prev      []uint32

	chainLimit     int
	fastBytesLimit int
}

// NewEncoderCore creates an LZMA encoder core.
func NewEncoderCore(lc, lp, pb uint, dictSize uint32) *EncoderCore {
	return NewEncoderCoreWithOptions(Options{
		Lc:          lc,
		Lp:          lp,
		Pb:          pb,
		DictSize:    dictSize,
		ChainLength: 16,
		FastBytes:   32,
	})
}

// NewEncoderCoreWithOptions creates an LZMA encoder core configured with custom options.
func NewEncoderCoreWithOptions(opts Options) *EncoderCore {
	dictSize := opts.DictSize
	if dictSize < 4096 {
		dictSize = 4096
	}
	prevSize := uint32(1) << bits.Len32(dictSize-1)
	if prevSize < 65536 {
		prevSize = 65536
	}
	maxPrev := uint32(16 * 1024 * 1024)
	switch {
	case opts.Level <= LevelFastest:
		maxPrev = 1 * 1024 * 1024
	case opts.Level <= LevelFast:
		maxPrev = 2 * 1024 * 1024
	case opts.Level <= LevelNormal:
		maxPrev = 8 * 1024 * 1024
	case opts.Level >= LevelUltra:
		maxPrev = 32 * 1024 * 1024
	}
	if prevSize > maxPrev {
		prevSize = maxPrev
	}
	dictMask := prevSize - 1
	isPowerOf2 := (prevSize & dictMask) == 0

	var headBits int
	switch {
	case opts.Level <= LevelFastest:
		headBits = 16
	case opts.Level <= LevelNormal:
		headBits = 18
	default:
		headBits = 20
	}
	headSize := 1 << headBits
	headMask := uint32(headSize - 1)
	headShift := uint(32 - headBits)

	chainLen := opts.ChainLength
	if chainLen <= 0 {
		chainLen = 16
	}
	fastBytes := opts.FastBytes
	if fastBytes <= 0 {
		fastBytes = 32
	}

	e := &EncoderCore{
		lc:             opts.Lc,
		lp:             opts.Lp,
		pb:             opts.Pb,
		posMask:        (1 << opts.Pb) - 1,
		lpMask:         (1 << opts.Lp) - 1,
		prevShift:      8 - opts.Lc,
		dictSize:       prevSize,
		dictMask:       dictMask,
		isPowerOf2:     isPowerOf2,
		head:           make([]uint32, headSize),
		headMask:       headMask,
		headShift:      headShift,
		prev:           make([]uint32, prevSize),
		chainLimit:     chainLen,
		fastBytesLimit: fastBytes,
	}
	e.InitProbs()
	return e
}

// InitProbs initializes probability tables and clears dictionary history.
func (e *EncoderCore) InitProbs() {
	e.ResetStateKeepDict()
	for i := range e.head {
		e.head[i] = 0xFFFFFFFF
	}
	for i := range e.head2 {
		e.head2[i] = 0xFFFFFFFF
	}
	for i := range e.prev {
		e.prev[i] = 0xFFFFFFFF
	}
}

// ResetStateKeepDict resets probability models and reps while preserving dictionary history across chunks.
func (e *EncoderCore) ResetStateKeepDict() {
	e.state = 0
	e.reps = [4]uint32{0, 0, 0, 0}

	for i := range e.isMatch {
		for j := range e.isMatch[i] {
			e.isMatch[i][j] = probInitVal
		}
	}
	for i := range e.isRep {
		e.isRep[i] = probInitVal
		e.isRepG0[i] = probInitVal
		e.isRepG1[i] = probInitVal
		e.isRepG2[i] = probInitVal
	}
	for i := range e.isRep0Long {
		for j := range e.isRep0Long[i] {
			e.isRep0Long[i][j] = probInitVal
		}
	}
	for i := range e.posSlot {
		for j := range e.posSlot[i] {
			e.posSlot[i][j] = probInitVal
		}
	}
	for i := range e.posDecoders {
		for j := range e.posDecoders[i] {
			e.posDecoders[i][j] = probInitVal
		}
	}
	for i := range e.posAlign {
		e.posAlign[i] = probInitVal
	}

	e.lenDecoder.Init()
	e.repLenDecoder.Init()

	numLit := 1 << (e.lc + e.lp)
	if len(e.litProbs) != numLit {
		e.litProbs = make([][0x300]uint16, numLit)
	}
	for i := range e.litProbs {
		for j := range e.litProbs[i] {
			e.litProbs[i][j] = probInitVal
		}
	}
}

// Reset clears state, probabilities, and dictionary history.
func (e *EncoderCore) Reset() {
	e.InitProbs()
}

func (e *EncoderCore) getLitSubCoder(pos uint32, prevByte byte) *[0x300]uint16 {
	idx := ((pos & e.lpMask) << e.lc) + (uint32(prevByte) >> e.prevShift)
	return &e.litProbs[idx]
}

func getPosSlot(dist uint32) uint32 {
	if dist < 4 {
		return dist
	}
	slot := uint32(4)
	for {
		numDirectBits := (slot >> 1) - 1
		base := (2 | (slot & 1)) << numDirectBits
		limit := base + (1 << numDirectBits)
		if dist < limit {
			return slot
		}
		slot++
	}
}

type fastRangeEncoder struct {
	re        *RangeEncoder
	low       uint64
	range_    uint32
	cacheSize uint64
	cache     byte
	bufPos    int
	buf       []byte
	w         io.Writer
	err       error
}

func (fe *fastRangeEncoder) writeByte(b byte) {
	if fe.bufPos < len(fe.buf) {
		fe.buf[fe.bufPos] = b
		fe.bufPos++
		return
	}
	if fe.bufPos > 0 {
		if _, err := fe.w.Write(fe.buf[:fe.bufPos]); err != nil {
			fe.err = err
			return
		}
		fe.bufPos = 0
	}
	fe.buf[0] = b
	fe.bufPos = 1
}

func (fe *fastRangeEncoder) shiftLow() {
	l := uint32(fe.low)
	high := uint32(fe.low >> 32)
	if l < 0xFF000000 || high != 0 {
		temp := fe.cache
		for {
			fe.writeByte(temp + byte(high))
			if fe.err != nil {
				return
			}
			temp = 0xFF
			fe.cacheSize--
			if fe.cacheSize == 0 {
				break
			}
		}
		fe.cache = byte(l >> 24)
	}
	fe.cacheSize++
	fe.low = uint64(l << 8)
}

func (fe *fastRangeEncoder) encodeBit(prob *uint16, bit int) {
	p := uint32(*prob)
	bound := (fe.range_ >> kNumBitModelTotalBits) * p
	if bit == 0 {
		fe.range_ = bound
		*prob += uint16((kBitModelTotal - p) >> kNumMoveBits)
	} else {
		fe.low += uint64(bound)
		fe.range_ -= bound
		*prob -= uint16(p >> kNumMoveBits)
	}
	if fe.range_ < kTopValue {
		fe.range_ <<= 8
		fe.shiftLow()
	}
}

func (fe *fastRangeEncoder) encodeDirectBits(val uint32, numBits int) {
	for i := numBits - 1; i >= 0; i-- {
		fe.range_ >>= 1
		if ((val >> i) & 1) != 0 {
			fe.low += uint64(fe.range_)
		}
		if fe.range_ < kTopValue {
			fe.range_ <<= 8
			fe.shiftLow()
		}
	}
}

func (fe *fastRangeEncoder) encodeTree(probs []uint16, numBits int, symbol uint32) {
	var m uint32 = 1
	for i := numBits - 1; i >= 0; i-- {
		bit := int((symbol >> i) & 1)
		fe.encodeBit(&probs[m], bit)
		m = (m << 1) | uint32(bit)
	}
}

func (fe *fastRangeEncoder) encodeReverseTree(probs []uint16, numBits int, symbol uint32) {
	var m uint32 = 1
	for i := 0; i < numBits; i++ {
		bit := int((symbol >> i) & 1)
		fe.encodeBit(&probs[m], bit)
		m = (m << 1) | uint32(bit)
	}
}

func (fe *fastRangeEncoder) encodeLen(ld *LenDecoder, lenVal, posState int) {
	if lenVal < kLenNumLowSymbols {
		fe.encodeBit(&ld.choice, 0)
		fe.encodeTree(ld.low[posState][:], kLenNumLowBits, uint32(lenVal))
	} else {
		fe.encodeBit(&ld.choice, 1)
		if lenVal < kLenNumLowSymbols+kLenNumMidSymbols {
			fe.encodeBit(&ld.choice2, 0)
			fe.encodeTree(ld.mid[posState][:], kLenNumMidBits, uint32(lenVal-kLenNumLowSymbols))
		} else {
			fe.encodeBit(&ld.choice2, 1)
			fe.encodeTree(ld.high[:], kLenNumHighBits, uint32(lenVal-kLenNumLowSymbols-kLenNumMidSymbols))
		}
	}
}

func (e *EncoderCore) encodeLiteral(
	fe *fastRangeEncoder,
	isMatchProb *uint16,
	pos uint32,
	prevByte, curByte byte,
	src []byte,
) {
	fe.encodeBit(isMatchProb, 0)
	probs := e.getLitSubCoder(pos, prevByte)

	var m uint32 = 1
	if !stateIsChar(e.state) && pos >= e.reps[0]+1 {
		matchByte := src[pos-e.reps[0]-1]
		offs := uint32(0x100)
		for i := 7; i >= 0; i-- {
			matchBit := int((matchByte >> i) & 1)
			bit := int((curByte >> i) & 1)
			probIdx := offs + (uint32(matchBit) << 8) + m
			fe.encodeBit(&probs[probIdx], bit)
			m = (m << 1) | uint32(bit)
			if matchBit != bit {
				for j := i - 1; j >= 0; j-- {
					bit := int((curByte >> j) & 1)
					fe.encodeBit(&probs[m], bit)
					m = (m << 1) | uint32(bit)
				}
				break
			}
		}
	} else {
		for i := 7; i >= 0; i-- {
			bit := int((curByte >> i) & 1)
			fe.encodeBit(&probs[m], bit)
			m = (m << 1) | uint32(bit)
		}
	}
}

func (e *EncoderCore) encodeShortRep(
	fe *fastRangeEncoder,
	isMatchProb, isRepProb, isRepG0Prob, isRep0LongProb *uint16,
) {
	fe.encodeBit(isMatchProb, 1)
	fe.encodeBit(isRepProb, 1)
	fe.encodeBit(isRepG0Prob, 0)
	fe.encodeBit(isRep0LongProb, 0)
	e.state = stateUpdateShortRep(e.state)
}

func (e *EncoderCore) encodeRepMatch(
	fe *fastRangeEncoder,
	isRepProb, isRepG0Prob, isRepG1Prob, isRepG2Prob *uint16,
	repIdx, bestLen, posState int,
) {
	fe.encodeBit(isRepProb, 1)

	if repIdx == 0 {
		fe.encodeBit(isRepG0Prob, 0)
		isRep0LongProb := (*uint16)(unsafe.Add(unsafe.Pointer(&e.isRep0Long[0][0]), (e.state<<4+posState)*2))
		fe.encodeBit(isRep0LongProb, 1)
	} else {
		fe.encodeBit(isRepG0Prob, 1)
		if repIdx == 1 {
			fe.encodeBit(isRepG1Prob, 0)
		} else {
			fe.encodeBit(isRepG1Prob, 1)
			if repIdx == 2 {
				fe.encodeBit(isRepG2Prob, 0)
			} else {
				fe.encodeBit(isRepG2Prob, 1)
			}
		}
	}

	lenVal := bestLen - kMatchMinLen
	fe.encodeLen(&e.repLenDecoder, lenVal, posState)

	dist := e.reps[repIdx]
	for i := repIdx; i > 0; i-- {
		e.reps[i] = e.reps[i-1]
	}
	e.reps[0] = dist
	e.state = stateUpdateRep(e.state)
}

func (e *EncoderCore) encodeNormalMatch(
	fe *fastRangeEncoder,
	isRepProb *uint16,
	posSlotPtr unsafe.Pointer,
	bestDist, bestLen, posState int,
) {
	fe.encodeBit(isRepProb, 0)

	lenVal := bestLen - kMatchMinLen
	fe.encodeLen(&e.lenDecoder, lenVal, posState)

	dist := uint32(bestDist)
	posSlot := getPosSlot(dist)
	lenToPos := lenVal
	if lenToPos >= kNumLenToPosStates {
		lenToPos = kNumLenToPosStates - 1
	}

	subPosSlot := (*[1 << kNumPosSlotBits]uint16)(unsafe.Add(posSlotPtr, (lenToPos<<kNumPosSlotBits)*2))[:]
	fe.encodeTree(subPosSlot, kNumPosSlotBits, posSlot)

	if posSlot >= 4 {
		numDirectBits := int((posSlot >> 1) - 1)
		base := (2 | (posSlot & 1)) << numDirectBits
		directBits := dist - base

		if posSlot < 14 {
			subProbs := e.posDecoders[posSlot-4][:]
			fe.encodeReverseTree(subProbs, numDirectBits, directBits)
		} else {
			alignBits := directBits & ((1 << kNumAlignBits) - 1)
			directHigh := directBits >> kNumAlignBits
			fe.encodeDirectBits(directHigh, numDirectBits-kNumAlignBits)
			fe.encodeReverseTree(e.posAlign[:], kNumAlignBits, alignBits)
		}
	}

	e.reps[3] = e.reps[2]
	e.reps[2] = e.reps[1]
	e.reps[1] = e.reps[0]
	e.reps[0] = dist
	e.state = stateUpdateMatch(e.state)
}

// EncodeChunk compresses a slice of src from startPos to endPos into re with dictionary continuity.
func (e *EncoderCore) EncodeChunk(src []byte, startPos, endPos int, re *RangeEncoder) error {
	var prevByte byte
	if startPos > 0 && startPos <= len(src) {
		prevByte = src[startPos-1]
	}

	fe := fastRangeEncoder{
		re:        re,
		low:       re.low,
		range_:    re.range_,
		cacheSize: re.cacheSize,
		cache:     re.cache,
		bufPos:    re.pos,
		buf:       re.buf[:],
		w:         re.w,
	}

	isMatchPtr := unsafe.Pointer(&e.isMatch[0][0])
	isRepPtr := unsafe.Pointer(&e.isRep[0])
	isRepG0Ptr := unsafe.Pointer(&e.isRepG0[0])
	isRepG1Ptr := unsafe.Pointer(&e.isRepG1[0])
	isRepG2Ptr := unsafe.Pointer(&e.isRepG2[0])
	isRep0LongPtr := unsafe.Pointer(&e.isRep0Long[0][0])
	posSlotPtr := unsafe.Pointer(&e.posSlot[0][0])

	srcLen := len(src)
	srcPtr := unsafe.Pointer(unsafe.SliceData(src))
	if endPos > srcLen {
		endPos = srcLen
	}

	for pos := startPos; pos < endPos && fe.err == nil; {
		posState := int(uint32(pos-startPos) & e.posMask)
		isMatchProb := (*uint16)(unsafe.Add(isMatchPtr, (e.state<<4+posState)*2))

		cand := e.findBestMatch(src, pos, endPos)

		if cand.length < kMatchMinLen || cand.savings <= 0 {
			// ShortRep
			if pos >= int(e.reps[0])+1 && src[pos] == src[pos-int(e.reps[0])-1] {
				isRepProb := (*uint16)(unsafe.Add(isRepPtr, e.state*2))
				isRepG0Prob := (*uint16)(unsafe.Add(isRepG0Ptr, e.state*2))
				isRep0LongProb := (*uint16)(unsafe.Add(isRep0LongPtr, (e.state<<4+posState)*2))
				e.encodeShortRep(&fe, isMatchProb, isRepProb, isRepG0Prob, isRep0LongProb)
				prevByte = src[pos]
				pos++
				continue
			}

			// Literal
			e.encodeLiteral(&fe, isMatchProb, uint32(pos), prevByte, src[pos], src)
			prevByte = src[pos]
			e.state = stateUpdateLiteral(e.state)
			pos++
			continue
		}

		// 1-Step Lookahead
		if cand.length < 32 && pos+5 <= endPos {
			nextU32 := binary.LittleEndian.Uint32(src[pos+1:])
			nextH := (nextU32 * 0x1E35A7BD) >> e.headShift
			nextMatchPos := e.head[nextH]
			if nextMatchPos != 0xFFFFFFFF && uint32(pos+1)-nextMatchPos < e.dictSize {
				nextLen := findMatchLengthPtr(unsafe.Add(srcPtr, pos+1), unsafe.Add(srcPtr, int(nextMatchPos)), min(endPos-(pos+1), srcLen-int(nextMatchPos)))
				if nextLen > cand.length+1 {
					e.encodeLiteral(&fe, isMatchProb, uint32(pos), prevByte, src[pos], src)
					prevByte = src[pos]
					e.state = stateUpdateLiteral(e.state)
					pos++
					continue
				}
			}
		}

		// Encode Match / Repetition
		fe.encodeBit(isMatchProb, 1)
		isRepProb := (*uint16)(unsafe.Add(isRepPtr, e.state*2))

		if cand.dist < 0 {
			repIdx := -cand.dist - 1
			isRepG0Prob := (*uint16)(unsafe.Add(isRepG0Ptr, e.state*2))
			isRepG1Prob := (*uint16)(unsafe.Add(isRepG1Ptr, e.state*2))
			isRepG2Prob := (*uint16)(unsafe.Add(isRepG2Ptr, e.state*2))
			e.encodeRepMatch(&fe, isRepProb, isRepG0Prob, isRepG1Prob, isRepG2Prob, repIdx, cand.length, posState)
		} else {
			e.encodeNormalMatch(&fe, isRepProb, posSlotPtr, cand.dist, cand.length, posState)
		}

		prevByte = src[pos+cand.length-1]
		pos += cand.length
	}

	re.low = fe.low
	re.range_ = fe.range_
	re.cacheSize = fe.cacheSize
	re.cache = fe.cache
	re.pos = fe.bufPos

	return fe.err
}

// EncodeStream compresses src into re with local register pinning.
func (e *EncoderCore) EncodeStream(src []byte, re *RangeEncoder) error {
	return e.EncodeChunk(src, 0, len(src), re)
}
