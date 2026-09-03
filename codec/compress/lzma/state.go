// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lzma

// The 12-state Markov automaton tracks stream recency profiles:
//   - States 0..3:  Pure literal characters.
//   - States 4..6:  Literal character emitted immediately after a match or repetition.
//   - State 7:      Regular dictionary match.
//   - State 8:      Repetition match using distance reps[1..3].
//   - State 9:      1-byte repetition match using reps[0] (ShortRep).
//   - State 10:     Repetition match following a literal.
//   - State 11:     1-byte repetition match following a literal.
//
// Match lengths are partitioned into three tiered probability trees:
//   - Low Tier:  lengths 2..9   (8 symbols, 3 bits)
//   - Mid Tier:  lengths 10..17 (8 symbols, 3 bits)
//   - High Tier: lengths 18..273 (256 symbols, 8 bits)

const (
	kNumStates = 12

	kNumPosSlotBits    = 6
	kNumLenToPosStates = 4
	kNumAlignBits      = 4
	kAlignTableSize    = 1 << kNumAlignBits

	kMatchMinLen = 2
	kMatchMaxLen = kMatchMinLen + 271

	kNumPosBitsMax     = 4
	kNumPosStatesMax   = 1 << kNumPosBitsMax
	kLenNumLowBits     = 3
	kLenNumLowSymbols  = 1 << kLenNumLowBits
	kLenNumMidBits     = 3
	kLenNumMidSymbols  = 1 << kLenNumMidBits
	kLenNumHighBits    = 8
	kLenNumHighSymbols = 1 << kLenNumHighBits
)

// stateUpdateLiteral transitions state after emitting a single literal byte.
func stateUpdateLiteral(state int) int {
	if state < 4 {
		return 0
	}
	if state < 10 {
		return state - 3
	}
	return state - 6
}

// stateUpdateMatch transitions state after emitting a regular dictionary match.
func stateUpdateMatch(state int) int {
	if state < 7 {
		return 7
	}
	return 10
}

// stateUpdateRep transitions state after emitting a repetition match.
func stateUpdateRep(state int) int {
	if state < 7 {
		return 8
	}
	return 11
}

// stateUpdateShortRep transitions state after emitting a 1-byte repetition match (ShortRep).
func stateUpdateShortRep(state int) int {
	if state < 7 {
		return 9
	}
	return 11
}

// stateIsChar reports whether state represents a stream currently in a literal-dominant sequence.
func stateIsChar(state int) bool {
	return state < 7
}

// LenDecoder encodes and decodes match lengths partitioned into 3 tiered probability trees:
// Low (2..9), Mid (10..17), and High (18..273).
type LenDecoder struct {
	choice  uint16
	choice2 uint16
	low     [kNumPosStatesMax][1 << (kLenNumLowBits + 1)]uint16
	mid     [kNumPosStatesMax][1 << (kLenNumMidBits + 1)]uint16
	high    [1 << (kLenNumHighBits + 1)]uint16
}

// Init initializes all probability tables in the length decoder to neutral probability (1024).
func (ld *LenDecoder) Init() {
	ld.choice = probInitVal
	ld.choice2 = probInitVal
	for i := range ld.low {
		for j := range ld.low[i] {
			ld.low[i][j] = probInitVal
		}
	}
	for i := range ld.mid {
		for j := range ld.mid[i] {
			ld.mid[i][j] = probInitVal
		}
	}
	for i := range ld.high {
		ld.high[i] = probInitVal
	}
}

// Decode decodes a match length from the arithmetic range stream.
func (ld *LenDecoder) Decode(rd *RangeDecoder, posState int) (int, error) {
	bit, err := rd.DecodeBit(&ld.choice)
	if err != nil {
		return 0, err
	}
	if bit == 0 {
		val, err := rd.DecodeTree(ld.low[posState][:], kLenNumLowBits)
		if err != nil {
			return 0, err
		}
		return int(val), nil
	}

	bit, err = rd.DecodeBit(&ld.choice2)
	if err != nil {
		return 0, err
	}
	if bit == 0 {
		val, err := rd.DecodeTree(ld.mid[posState][:], kLenNumMidBits)
		if err != nil {
			return 0, err
		}
		return int(val) + kLenNumLowSymbols, nil
	}

	val, err := rd.DecodeTree(ld.high[:], kLenNumHighBits)
	if err != nil {
		return 0, err
	}
	return int(val) + kLenNumLowSymbols + kLenNumMidSymbols, nil
}

// Encode encodes a match length into the arithmetic range stream.
func (ld *LenDecoder) Encode(re *RangeEncoder, lenVal, posState int) error {
	if lenVal < kLenNumLowSymbols {
		if err := re.EncodeBit(&ld.choice, 0); err != nil {
			return err
		}
		return re.EncodeTree(ld.low[posState][:], kLenNumLowBits, uint32(lenVal))
	}
	if err := re.EncodeBit(&ld.choice, 1); err != nil {
		return err
	}
	lenVal -= kLenNumLowSymbols
	if lenVal < kLenNumMidSymbols {
		if err := re.EncodeBit(&ld.choice2, 0); err != nil {
			return err
		}
		return re.EncodeTree(ld.mid[posState][:], kLenNumMidBits, uint32(lenVal))
	}
	if err := re.EncodeBit(&ld.choice2, 1); err != nil {
		return err
	}
	lenVal -= kLenNumMidSymbols
	return re.EncodeTree(ld.high[:], kLenNumHighBits, uint32(lenVal))
}
